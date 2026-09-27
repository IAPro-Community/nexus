package testutil

import (
	"errors"
	"os"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

// SymlinkOrSkip creates a symbolic link for a capability-sensitive test.
// Windows can legitimately deny symlink creation when Developer Mode/admin
// privileges are unavailable. In that case the test is capability-skipped;
// all other errors remain hard failures.
func SymlinkOrSkip(t testing.TB, oldname, newname string) {
	t.Helper()
	if err := os.Symlink(oldname, newname); err != nil {
		if runtime.GOOS == "windows" && symlinkCapabilityUnavailable(err) {
			t.Skipf("symbolic-link capability unavailable on this Windows runner: %v", err)
		}
		t.Fatalf("create symbolic link %q -> %q: %v", newname, oldname, err)
	}
}

func symlinkCapabilityUnavailable(err error) bool {
	if errors.Is(err, os.ErrPermission) {
		return true
	}
	var errno syscall.Errno
	if errors.As(err, &errno) && errno == syscall.Errno(1314) { // ERROR_PRIVILEGE_NOT_HELD
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "privilege") ||
		strings.Contains(msg, "not supported") ||
		strings.Contains(msg, "operation not permitted")
}
