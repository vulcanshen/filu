package ui

import (
	"os"
	"syscall"
	"time"
)

// osStat — see osstat_darwin.go. Linux has no birth time without statx, so btime
// stays zero and the Meta tab omits the "Created" row.
func osStat(fi os.FileInfo) (osMeta, bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return osMeta{}, false
	}
	return osMeta{
		uid:   st.Uid,
		gid:   st.Gid,
		nlink: uint64(st.Nlink),
		inode: st.Ino,
		atime: time.Unix(int64(st.Atim.Sec), int64(st.Atim.Nsec)),
		ctime: time.Unix(int64(st.Ctim.Sec), int64(st.Ctim.Nsec)),
	}, true
}
