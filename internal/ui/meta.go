package ui

import (
	"fmt"
	"os/user"
	"strconv"
	"time"
)

// osMeta holds the unix stat fields osStat fills per OS (osstat_{darwin,linux}.go).
// The owner (uid/gid) feeds the list's owner column; the rest are kept for the
// platform stat contract.
type osMeta struct {
	uid, gid uint32
	nlink    uint64
	inode    uint64
	atime    time.Time
	ctime    time.Time
	btime    time.Time // zero when unavailable (Linux)
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
