package ui

import (
	"fmt"
	"os"
	"path/filepath"
)

// infoLines is panel [3]'s Info tab: metadata for the cursor item.
func infoLines(it fileItem, parent string) []string {
	if it.name == "" {
		return []string{"(無選取)"}
	}
	full := filepath.Join(parent, it.name)
	fi, err := os.Lstat(full)
	if err != nil {
		return []string{"(無法讀取)"}
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

	lines := []string{
		"Name  " + it.name,
		"Type  " + kind,
	}
	if !fi.IsDir() {
		lines = append(lines, "Size  "+humanSize(fi.Size()))
	}
	lines = append(lines,
		"Perm  "+fi.Mode().String(),
		"Oct   "+fmt.Sprintf("%#o", fi.Mode().Perm()),
		"Mtime "+fi.ModTime().Format("2006-01-02 15:04"),
	)
	return lines
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
