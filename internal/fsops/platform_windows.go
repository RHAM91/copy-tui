//go:build windows

package fsops

import (
	"errors"
	"io/fs"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// accessTime returns the last access time recorded by the file system.
func accessTime(info fs.FileInfo) time.Time {
	if d, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		return time.Unix(0, d.LastAccessTime.Nanoseconds())
	}
	return info.ModTime()
}

// setLinkTimes is a no-op: Go cannot set the times of a Windows symlink
// without following it.
func setLinkTimes(string, time.Time, time.Time) {}

// isCrossDevice reports whether a rename failed because source and
// destination are on different volumes.
func isCrossDevice(err error) bool {
	return errors.Is(err, windows.ERROR_NOT_SAME_DEVICE) || errors.Is(err, syscall.EXDEV)
}
