package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	terminu "github.com/vulcanshen/terminu-go"
)

// metaRow is one fact about a file: a label and its value.
type metaRow struct{ label, value string }

// fileFacts reads everything the information box shows about path.
func fileFacts(path string) []metaRow {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	rows := []metaRow{{"Path", terminu.Escape(abs)}}
	fi, err := os.Lstat(path)
	if err != nil {
		return append(rows, metaRow{"Error", opFailedText("read it", err)})
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		target, lerr := os.Readlink(path)
		if lerr != nil {
			target = opFailedText("read the link", lerr)
		}
		rows = append(rows, metaRow{"Link to", terminu.Escape(target)})
	}
	rows = append(rows,
		metaRow{"Type", fileType(path, fi)},
		metaRow{"Size", humanSize(fi.Size()) + " (" + groupDigits(fi.Size()) + " bytes)"})
	const stamp = "2006-01-02 15:04:05"
	rows = append(rows, metaRow{"Modified", fi.ModTime().Format(stamp)})
	st, ok := osStat(fi)
	if ok {
		rows = append(rows, metaRow{"Accessed", st.atime.Format(stamp)})
		if runtime.GOOS == "darwin" && !st.btime.IsZero() {
			rows = append(rows, metaRow{"Created", st.btime.Format(stamp)})
		} else {
			rows = append(rows, metaRow{"Changed", st.ctime.Format(stamp)})
		}
	}
	rows = append(rows, metaRow{"Permissions", fi.Mode().String() + " (" + fmt.Sprintf("%04o", fi.Mode().Perm()|specialBits(fi.Mode())) + ")"})
	if ok {
		rows = append(rows, metaRow{"Owner", userName(st.uid) + ":" + groupName(st.gid)})
	} else {
		rows = append(rows, metaRow{"Owner", "unknown on this system"})
	}
	return rows
}

// specialBits maps the setuid / setgid / sticky mode flags onto their octal
// digit, so the octal reads as chmod would take it (e.g. 4755).
func specialBits(m os.FileMode) os.FileMode {
	var b os.FileMode
	if m&os.ModeSetuid != 0 {
		b |= 0o4000
	}
	if m&os.ModeSetgid != 0 {
		b |= 0o2000
	}
	if m&os.ModeSticky != 0 {
		b |= 0o1000
	}
	return b
}

// fileType names what the file is, the way the preview tells them apart: the
// file kind, then images, archives and PDFs by name, then text or binary by its
// first bytes.
func fileType(path string, fi os.FileInfo) string {
	mode := fi.Mode()
	switch {
	case mode&os.ModeSymlink != 0:
		if t, err := os.Stat(path); err == nil && t.IsDir() {
			return "Symbolic link to a directory"
		} else if err != nil {
			return "Symbolic link (broken)"
		}
		return "Symbolic link to a file"
	case mode.IsDir():
		return "Directory"
	case mode&os.ModeNamedPipe != 0:
		return "Named pipe"
	case mode&os.ModeSocket != 0:
		return "Socket"
	case mode&os.ModeDevice != 0:
		return "Device"
	}
	name := strings.ToLower(fi.Name())
	exec := ""
	if mode&0o111 != 0 {
		exec = ", executable"
	}
	switch {
	case isImage(name):
		return "Image (" + strings.TrimPrefix(filepath.Ext(name), ".") + ")" + exec
	case strings.HasSuffix(name, ".pdf"):
		return "PDF document" + exec
	}
	if _, ok := archiveEntries(path, fi.Name()); ok {
		return "Archive (" + strings.TrimPrefix(filepath.Ext(name), ".") + ")" + exec
	}
	data, err := readCapped(path, 8192)
	switch {
	case err != nil:
		return opFailedText("read it", err)
	case len(data) == 0:
		return "Empty file" + exec
	case isText(data):
		return "Text" + exec
	}
	return "Binary" + exec
}

// groupDigits writes n with thousands separators: 12601 → 12,601.
func groupDigits(n int64) string {
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
