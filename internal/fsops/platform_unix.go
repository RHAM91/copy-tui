//go:build unix

package fsops

import (
	"errors"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// setLinkTimes sets the times of a symlink itself, not of its target.
func setLinkTimes(path string, atime, mtime time.Time) {
	ts := []unix.Timespec{unix.NsecToTimespec(atime.UnixNano()), unix.NsecToTimespec(mtime.UnixNano())}
	_ = unix.UtimesNanoAt(unix.AT_FDCWD, path, ts, unix.AT_SYMLINK_NOFOLLOW)
}

// isCrossDevice reports whether a rename failed because source and
// destination are on different file systems.
func isCrossDevice(err error) bool { return errors.Is(err, syscall.EXDEV) }
