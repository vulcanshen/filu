package ui

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Zip packs the Marks panel's land subset into one archive. It lands in a fresh
// temp dir rather than any working directory: the point of a Zip is to carry the
// result somewhere, so the archive joins the marks bucket as the only pick and
// the usual Copy/Move here puts it where the user wants it.

// suggestZipName is the name the Zip prompt opens with: one item keeps its own
// name, a set from one directory takes that directory's name, and a set spanning
// directories falls back to a timestamp.
func suggestZipName(items []string) string {
	stamp := "filu-" + time.Now().Format("20060102-150405") + ".zip"
	switch len(items) {
	case 0:
		return stamp
	case 1:
		base := filepath.Base(items[0])
		if fi, err := os.Stat(items[0]); err == nil && fi.IsDir() {
			return base + ".zip"
		}
		return strings.TrimSuffix(base, filepath.Ext(base)) + ".zip"
	}
	dir := filepath.Dir(items[0])
	for _, p := range items[1:] {
		if filepath.Dir(p) != dir {
			return stamp
		}
	}
	if base := filepath.Base(dir); base != "/" && base != "." {
		return base + ".zip"
	}
	return stamp
}

// zipPrompt titles the Zip input popup with what it is about to pack.
func zipPrompt(n int) string {
	if n == 1 {
		return "Zip 1 item"
	}
	return fmt.Sprintf("Zip %d items", n)
}

// zipFileName sanitises what the user typed: basename only (no escaping the temp
// dir) and a .zip suffix.
func zipFileName(input string) string {
	name := filepath.Base(strings.TrimSpace(input))
	if name == "." || name == "/" || name == "" {
		return ""
	}
	if !strings.HasSuffix(strings.ToLower(name), ".zip") {
		name += ".zip"
	}
	return name
}

// startZip packs items into name.zip inside a fresh temp dir, as an async task
// in the Tasks tab (same shape as a land — long work off the UI goroutine).
func (m *AppModel) startZip(items []string, name string) tea.Cmd {
	if len(items) == 0 || name == "" {
		return nil
	}
	dir, err := os.MkdirTemp("", "filu-zip-")
	if err != nil {
		return m.toast.show("Could not create a temp dir")
	}
	m.nextTaskID++
	m.tasks = append(m.tasks, landTask{
		id: m.nextTaskID, action: "zip",
		dest: name, destPath: dir, srcs: items,
		total: len(items), at: time.Now(), status: taskRunning,
	})
	m.capTasks()
	go runZip(m.nextTaskID, items, filepath.Join(dir, name), m.taskCh)
	saveState(m.snapshotState())
	if !m.spinning {
		m.spinning = true
		return spinnerTick()
	}
	return nil
}

// runZip writes items into zipPath on a goroutine, streaming progress to ch. It
// never touches model state — the main loop applies the results.
func runZip(id int, items []string, zipPath string, ch chan<- landMsg) {
	fail := func(n int) {
		ch <- landMsg{taskID: id, done: len(items), total: len(items), finished: true, failed: n}
	}
	f, err := os.Create(zipPath)
	if err != nil {
		fail(len(items))
		return
	}
	zw := zip.NewWriter(f)
	used := map[string]bool{}
	failed := 0
	for i, src := range items {
		if err := zipAdd(zw, src, used); err != nil {
			failed++
		}
		ch <- landMsg{taskID: id, done: i + 1, total: len(items)}
	}
	cerr := zw.Close()
	f.Close()
	if cerr != nil || failed == len(items) { // nothing usable came out
		os.Remove(zipPath)
		fail(max(failed, 1))
		return
	}
	ch <- landMsg{taskID: id, done: len(items), total: len(items), finished: true, failed: failed, produced: zipPath}
}

// zipAdd writes src into the archive under its basename — a directory keeps its
// inner structure below that. Picks from different directories that share a name
// are renamed rather than overwritten.
func zipAdd(zw *zip.Writer, src string, used map[string]bool) error {
	info, err := os.Stat(src) // follow a picked symlink, like Copy does
	if err != nil {
		return err
	}
	name := uniqueEntry(filepath.Base(src), used)
	if !info.IsDir() {
		return zipRegular(zw, src, name, info)
	}
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		entry := path.Join(name, filepath.ToSlash(rel))
		if d.IsDir() {
			_, err := zw.Create(entry + "/")
			return err
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		return zipRegular(zw, p, entry, fi)
	})
}

// zipRegular writes one regular file. Anything else inside a walked tree
// (symlink, socket, device) is skipped — a zip has nowhere to put it, and
// following links while walking risks a cycle.
func zipRegular(zw *zip.Writer, src, name string, info os.FileInfo) error {
	if !info.Mode().IsRegular() {
		return nil
	}
	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	hdr.Name = name
	hdr.Method = zip.Deflate
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	_, err = io.Copy(w, in)
	return err
}

// uniqueEntry keeps archive entry names distinct: "a.txt", "a (2).txt", …
func uniqueEntry(name string, used map[string]bool) string {
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	cand := name
	for i := 2; used[cand]; i++ {
		cand = fmt.Sprintf("%s (%d)%s", stem, i, ext)
	}
	used[cand] = true
	return cand
}
