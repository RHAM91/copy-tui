package fsops

import (
	"context"
	"os"
)

// Remove permanently deletes each path (directories recursively, symlinks
// without touching their target) and returns how many were removed.
func Remove(ctx context.Context, paths []string, onProgress func(Progress)) (int, error) {
	prog := Progress{FilesTotal: len(paths)}
	report := func() {
		if onProgress != nil {
			onProgress(prog)
		}
	}
	report()
	for _, p := range paths {
		if err := ctx.Err(); err != nil {
			return prog.FilesDone, err
		}
		prog.Current = p
		if err := os.RemoveAll(p); err != nil {
			return prog.FilesDone, err
		}
		prog.FilesDone++
		report()
	}
	return prog.FilesDone, nil
}
