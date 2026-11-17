//go:build unix && !linux && !darwin

package fsops

import (
	"io/fs"
	"time"
)

// accessTime falls back to the modification time where the access time is
// not portable.
func accessTime(info fs.FileInfo) time.Time { return info.ModTime() }
