package ui

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// osMeta holds the unix stat fields the Meta tab shows; osStat fills it per OS
// (osstat_{darwin,linux}.go).
type osMeta struct {
	uid, gid uint32
	nlink    uint64
	inode    uint64
	atime    time.Time
	ctime    time.Time
	btime    time.Time // zero when unavailable (Linux)
}

// metaLines is panel [3]'s Meta tab: rich metadata for the cursor item.
func metaLines(it fileItem, parent string) []string {
	if it.name == "" {
		return []string{"(no selection)"}
	}
	full := filepath.Join(parent, it.name)
	fi, err := os.Lstat(full)
	if err != nil {
		return []string{"(unreadable)"}
	}

	kind := "file"
	switch {
	case fi.IsDir():
		kind = "dir"
	case fi.Mode()&os.ModeSymlink != 0:
		kind = "symlink"
		if tgt, e := os.Readlink(full); e == nil {
			kind = "symlink → " + tgt
		}
	}

	var rows []string
	key := lipgloss.NewStyle().Foreground(focusColor) // blue
	add := func(name, value string) {
		if value == "" {
			return
		}
		rows = append(rows, key.Render(fmt.Sprintf("%-9s", name))+value)
	}

	add("Name", it.name)
	add("Path", shortPath(parent))
	add("Type", kind)
	if fi.IsDir() {
		if entries, e := os.ReadDir(full); e == nil {
			add("Items", strconv.Itoa(len(entries)))
		}
	} else {
		add("Size", fmt.Sprintf("%s (%d bytes)", humanSize(fi.Size()), fi.Size()))
	}

	meta, hasMeta := osStat(fi)
	if hasMeta {
		add("Owner", userName(meta.uid))
		add("Group", groupName(meta.gid))
		add("Links", strconv.FormatUint(meta.nlink, 10))
		add("Inode", strconv.FormatUint(meta.inode, 10))
	}

	add("Perm", fi.Mode().String())
	add("Octal", fmt.Sprintf("%#o", fi.Mode().Perm()))

	const tf = "2006-01-02 15:04:05"
	add("Modified", fi.ModTime().Format(tf))
	if hasMeta {
		add("Accessed", timeOrEmpty(meta.atime, tf))
		add("Changed", timeOrEmpty(meta.ctime, tf))
		add("Created", timeOrEmpty(meta.btime, tf))
	}
	return rows
}

func timeOrEmpty(t time.Time, layout string) string {
	if t.IsZero() || t.Unix() <= 0 {
		return ""
	}
	return t.Format(layout)
}

// userName / groupName resolve an id to a name, falling back to the number when
// lookup fails (e.g. a CGO-free static build on macOS, where names aren't in
// /etc/passwd).
func userName(uid uint32) string {
	if u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10)); err == nil {
		return u.Username
	}
	return strconv.FormatUint(uint64(uid), 10)
}

func groupName(gid uint32) string {
	if g, err := user.LookupGroupId(strconv.FormatUint(uint64(gid), 10)); err == nil {
		return g.Name
	}
	return strconv.FormatUint(uint64(gid), 10)
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
