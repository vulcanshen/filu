//go:build darwin || linux

package ui

import "golang.org/x/sys/unix"

// diskUsage returns the bytes available to the user and the total bytes of the
// filesystem holding path. It is a single statfs syscall reading the filesystem
// superblock — no directory traversal — so it is cheap enough to refresh every
// time a directory loads (unlike a recursive content-size sum, which is not).
func diskUsage(path string) (free, total int64, ok bool) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, 0, false
	}
	bsize := int64(st.Bsize)
	return int64(st.Bavail) * bsize, int64(st.Blocks) * bsize, true
}
