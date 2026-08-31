package ui

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSuggestZipName: the prompt opens with the name that needs the least
// typing — the item's own name, the directory the set came from, or a timestamp
// when the set spans directories.
func TestSuggestZipName(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "assets")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name  string
		items []string
		want  string // "" = expect the filu-<stamp>.zip fallback
	}{
		{"single file drops its extension", []string{"/tmp/pics/report.pdf"}, "report.zip"},
		{"single dir keeps its name", []string{sub}, "assets.zip"},
		{"one directory names the set", []string{"/tmp/pics/a.png", "/tmp/pics/b.png"}, "pics.zip"},
		{"across directories falls back", []string{"/tmp/pics/a.png", "/tmp/docs/b.md"}, ""},
		{"empty falls back", nil, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := suggestZipName(c.items)
			if c.want == "" {
				if !strings.HasPrefix(got, "filu-") || !strings.HasSuffix(got, ".zip") {
					t.Fatalf("suggestZipName(%v) = %q, want a filu-<stamp>.zip fallback", c.items, got)
				}
				return
			}
			if got != c.want {
				t.Fatalf("suggestZipName(%v) = %q, want %q", c.items, got, c.want)
			}
		})
	}
}

// TestZipFileName: the typed name is reduced to a basename with a .zip suffix,
// so nothing can be written outside the temp dir.
func TestZipFileName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"stuff", "stuff.zip"},
		{"stuff.zip", "stuff.zip"},
		{"STUFF.ZIP", "STUFF.ZIP"},
		{"  spaced  ", "spaced.zip"},
		{"../../etc/passwd", "passwd.zip"},
		{"/", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := zipFileName(c.in); got != c.want {
			t.Errorf("zipFileName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestRunZipPacksItems: a file and a directory pack into one archive — the dir
// keeps its inner structure — and the finish message reports the archive it
// produced.
func TestRunZipPacksItems(t *testing.T) {
	src, out := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(src, "d", "inner")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "b.txt"), []byte("beta"), 0o644); err != nil {
		t.Fatal(err)
	}

	zipPath := filepath.Join(out, "bundle.zip")
	ch := make(chan landMsg, 8)
	go runZip(1, []string{filepath.Join(src, "a.txt"), filepath.Join(src, "d")}, zipPath, ch)

	var last landMsg
	for {
		msg := <-ch
		last = msg
		if msg.finished {
			break
		}
	}
	if last.failed != 0 || last.done != 2 || last.total != 2 {
		t.Fatalf("finish msg = %+v, want 2/2 with no failures", last)
	}
	if last.produced != zipPath {
		t.Fatalf("produced = %q, want %q", last.produced, zipPath)
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	got := map[string]bool{}
	for _, f := range r.File {
		got[f.Name] = true
	}
	for _, want := range []string{"a.txt", "d/inner/b.txt"} {
		if !got[want] {
			t.Errorf("archive is missing %q; has %v", want, got)
		}
	}
}

// TestRunZipRenamesCollisions: two picks with the same basename both survive.
func TestRunZipRenamesCollisions(t *testing.T) {
	one, two, out := t.TempDir(), t.TempDir(), t.TempDir()
	for _, d := range []string{one, two} {
		if err := os.WriteFile(filepath.Join(d, "note.txt"), []byte(d), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	zipPath := filepath.Join(out, "notes.zip")
	ch := make(chan landMsg, 8)
	go runZip(1, []string{filepath.Join(one, "note.txt"), filepath.Join(two, "note.txt")}, zipPath, ch)
	for {
		if (<-ch).finished {
			break
		}
	}
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if len(r.File) != 2 {
		t.Fatalf("want 2 entries, got %d", len(r.File))
	}
	if r.File[0].Name != "note.txt" || r.File[1].Name != "note (2).txt" {
		t.Errorf("entries = %q/%q, want note.txt / note (2).txt", r.File[0].Name, r.File[1].Name)
	}
}

// TestRunZipAllFailedRemovesArchive: nothing readable went in, so no half-made
// archive is left behind and nothing is produced to carry.
func TestRunZipAllFailedRemovesArchive(t *testing.T) {
	out := t.TempDir()
	zipPath := filepath.Join(out, "empty.zip")
	ch := make(chan landMsg, 8)
	go runZip(1, []string{filepath.Join(out, "missing.txt")}, zipPath, ch)
	var last landMsg
	for {
		last = <-ch
		if last.finished {
			break
		}
	}
	if last.failed != 1 || last.produced != "" {
		t.Fatalf("finish msg = %+v, want 1 failure and nothing produced", last)
	}
	if _, err := os.Stat(zipPath); err == nil {
		t.Error("a fully failed zip should not be left on disk")
	}
}

// TestZipResultBecomesSolePick: a finished zip joins the bucket and takes over
// the pick, so the next Copy/Move here lands the archive.
func TestZipResultBecomesSolePick(t *testing.T) {
	m := minModel()
	m.marks.items = []string{"/tmp/a", "/tmp/b"}
	m.marks.picked = map[string]bool{"/tmp/a": true}
	m.tasks = []landTask{{id: 7, action: "zip", dest: "bundle.zip", destPath: "/tmp/z", total: 2, status: taskRunning}}

	m.handleLandMsg(landMsg{taskID: 7, done: 2, total: 2, finished: true, produced: "/tmp/z/bundle.zip"})

	if m.tasks[0].status != taskDone {
		t.Fatalf("task status = %v, want done", m.tasks[0].status)
	}
	want := []string{"/tmp/a", "/tmp/b", "/tmp/z/bundle.zip"}
	if len(m.marks.items) != 3 || m.marks.items[2] != want[2] {
		t.Fatalf("bucket = %v, want %v", m.marks.items, want)
	}
	if len(m.marks.picked) != 1 || !m.marks.picked["/tmp/z/bundle.zip"] {
		t.Fatalf("picked = %v, want only the zip", m.marks.picked)
	}
	if items := m.marks.landItems(); len(items) != 1 || items[0] != "/tmp/z/bundle.zip" {
		t.Fatalf("landItems = %v, want just the zip", items)
	}
}

// TestMarksZipOpensPrompt: Z opens the name prompt pre-filled, and packs nothing
// until it is committed.
func TestMarksZipOpensPrompt(t *testing.T) {
	m := minModel()
	m.inputPopup = newInputPopup()
	m.focus = panelMarks
	m.marks.items = []string{"/tmp/pics/a.png", "/tmp/pics/b.png"}

	if cmd := m.handleMarksKey("Z"); cmd == nil {
		t.Fatal("Z should open the zip name prompt")
	}
	if m.inputPopup.kind != inputZip {
		t.Fatalf("input kind = %v, want inputZip", m.inputPopup.kind)
	}
	if m.inputPopup.buffer != "pics.zip" {
		t.Errorf("prompt should be pre-filled with pics.zip, got %q", m.inputPopup.buffer)
	}
	if m.inputPopup.prompt != "Zip 2 items" {
		t.Errorf("prompt title = %q, want \"Zip 2 items\"", m.inputPopup.prompt)
	}
	if len(m.tasks) != 0 {
		t.Errorf("Z must not start a task before the name is committed, got %d", len(m.tasks))
	}
}

// TestMarksZipNoopWhenEmpty: nothing marked, nothing to pack.
func TestMarksZipNoopWhenEmpty(t *testing.T) {
	m := minModel()
	m.inputPopup = newInputPopup()
	m.focus = panelMarks
	if cmd := m.handleMarksKey("Z"); cmd != nil {
		t.Error("Z on an empty bucket should do nothing")
	}
}
