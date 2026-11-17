package fsops

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// SHFileOperationW constants (shellapi.h).
const (
	foDelete           = 0x0003
	fofSilent          = 0x0004
	fofNoConfirmation  = 0x0010
	fofAllowUndo       = 0x0040
	fofNoErrorUI       = 0x0400
	fofWantNukeWarning = 0x4000
)

// shFileOpStruct mirrors SHFILEOPSTRUCTW with the default 64-bit packing.
type shFileOpStruct struct {
	hwnd                  uintptr
	wFunc                 uint32
	pFrom                 *uint16
	pTo                   *uint16
	fFlags                uint16
	fAnyOperationsAborted int32
	hNameMappings         uintptr
	lpszProgressTitle     *uint16
}

var procSHFileOperationW = windows.NewLazySystemDLL("shell32.dll").NewProc("SHFileOperationW")

// recycle sends one path to the Recycle Bin. FOF_WANTNUKEWARNING makes
// Windows ask instead of silently deleting when the item cannot be recycled.
func recycle(path string) error {
	// pFrom is a list of paths ending with an extra NUL.
	from, err := windows.UTF16FromString(path)
	if err != nil {
		return err
	}
	from = append(from, 0)
	op := shFileOpStruct{
		wFunc:  foDelete,
		pFrom:  &from[0],
		fFlags: fofAllowUndo | fofNoConfirmation | fofSilent | fofNoErrorUI | fofWantNukeWarning,
	}
	r, _, _ := procSHFileOperationW.Call(uintptr(unsafe.Pointer(&op)))
	if r != 0 {
		return fmt.Errorf("moving %s to the Recycle Bin failed (code 0x%x)", path, r)
	}
	if op.fAnyOperationsAborted != 0 {
		return fmt.Errorf("moving %s to the Recycle Bin was cancelled", path)
	}
	return nil
}

// Trash sends each path to the Windows Recycle Bin, from where it can be
// restored. It returns how many entries were trashed.
func Trash(ctx context.Context, paths []string, onProgress func(Progress)) (int, error) {
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
		abs, err := filepath.Abs(p)
		if err != nil {
			return prog.FilesDone, err
		}
		if _, err := os.Lstat(abs); err != nil {
			return prog.FilesDone, err
		}
		prog.Current = abs
		if err := recycle(abs); err != nil {
			return prog.FilesDone, err
		}
		prog.FilesDone++
		report()
	}
	return prog.FilesDone, nil
}
