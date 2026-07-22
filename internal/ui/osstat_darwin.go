package ui

import (
	"os"
	"syscall"
	"time"
)

// osStat pulls the unix stat fields the Meta tab shows. Split by build tag: the
// timespec field names differ per OS and only darwin exposes a birth time.
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
		atime: time.Unix(int64(st.Atimespec.Sec), int64(st.Atimespec.Nsec)),
		ctime: time.Unix(int64(st.Ctimespec.Sec), int64(st.Ctimespec.Nsec)),
		btime: time.Unix(int64(st.Birthtimespec.Sec), int64(st.Birthtimespec.Nsec)),
	}, true
}
