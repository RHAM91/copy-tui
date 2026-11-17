package fsops

import (
	"io/fs"
	"syscall"
	"time"
)

// accessTime returns the last access time recorded by the file system.
func accessTime(info fs.FileInfo) time.Time {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return time.Unix(st.Atimespec.Unix())
	}
	return info.ModTime()
}
