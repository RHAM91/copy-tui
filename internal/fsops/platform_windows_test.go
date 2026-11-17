package fsops

import (
	"os"
	"syscall"
	"testing"

	"golang.org/x/sys/windows"
)

func TestIsCrossDeviceWindows(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "different drive", err: &os.LinkError{Op: "rename", Err: windows.ERROR_NOT_SAME_DEVICE}, want: true},
		{name: "EXDEV", err: &os.LinkError{Op: "rename", Err: syscall.EXDEV}, want: true},
		{name: "access denied is not cross device", err: &os.LinkError{Op: "rename", Err: windows.ERROR_ACCESS_DENIED}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCrossDevice(tt.err); got != tt.want {
				t.Fatalf("isCrossDevice(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
