//go:build linux

package python

import (
	"os"
	"path"
	"syscall"
	"testing"
)

func TestCleanupTempDirRemovesOnlyFilesOwnedBySandboxUID(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("requires root to chown temp files")
	}

	dir := t.TempDir()
	ownedPath := path.Join(dir, "owned.txt")
	otherPath := path.Join(dir, "other.txt")

	if err := os.WriteFile(ownedPath, []byte("owned"), 0600); err != nil {
		t.Fatalf("write owned file: %v", err)
	}
	if err := os.WriteFile(otherPath, []byte("other"), 0600); err != nil {
		t.Fatalf("write other file: %v", err)
	}

	const sandboxUID = 10000
	if err := syscall.Chown(ownedPath, sandboxUID, 0); err != nil {
		t.Fatalf("chown owned file: %v", err)
	}

	cleanupTempDir(dir, sandboxUID)

	if _, err := os.Stat(ownedPath); !os.IsNotExist(err) {
		t.Fatalf("expected owned file to be removed, stat err=%v", err)
	}
	if _, err := os.Stat(otherPath); err != nil {
		t.Fatalf("expected other file to remain, stat err=%v", err)
	}
}
