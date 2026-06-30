package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWithTempDirCreatesWritableTemporaryDirectories(t *testing.T) {
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalCwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	baseDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(baseDir, "tmp"), 0755); err != nil {
		t.Fatalf("create base tmp: %v", err)
	}

	runner := TempDirRunner{}
	err = runner.WithTempDir(baseDir, nil, func(rootPath string) error {
		assertStickyWorldWritableDir(t, filepath.Join(rootPath, "tmp"))
		assertStickyWorldWritableDir(t, filepath.Join(rootPath, "var", "tmp"))
		return nil
	})
	if err != nil {
		t.Fatalf("with temp dir: %v", err)
	}
}

func assertStickyWorldWritableDir(t *testing.T, dir string) {
	t.Helper()

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat %s: %v", dir, err)
	}

	if !info.IsDir() {
		t.Fatalf("expected %s to be a directory", dir)
	}

	if info.Mode().Perm() != 0777 {
		t.Fatalf("expected %s permissions 0777, got %03o", dir, info.Mode().Perm())
	}

	if info.Mode()&os.ModeSticky == 0 {
		t.Fatalf("expected %s to have sticky bit set, got %s", dir, info.Mode())
	}
}
