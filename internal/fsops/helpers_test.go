package fsops

import (
	"os"
	"runtime"
	"testing"
)

// unixPerms reports whether the platform enforces Unix permission bits.
func unixPerms() bool { return runtime.GOOS != "windows" }

// requireEnforcedPermissions skips tests that rely on denied access, which
// root and Windows do not honour.
func requireEnforcedPermissions(t *testing.T) {
	t.Helper()
	if !unixPerms() || os.Geteuid() == 0 {
		t.Skip("permission bits are not enforced here")
	}
}
