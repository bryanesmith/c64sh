package shell

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// @spec SHELL-FILE-001
func TestDirStorage(t *testing.T) {
	dir := t.TempDir()
	s := dirStorage{dir: dir}
	if _, err := s.ReadFile("NONE.bas"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("ReadFile of a missing file: %v, want fs.ErrNotExist", err)
	}
	if err := s.WriteFile("A.bas", []byte("one"), false); err != nil {
		t.Fatal(err)
	}
	if err := s.WriteFile("A.bas", []byte("two"), false); !errors.Is(err, fs.ErrExist) {
		t.Errorf("WriteFile over a file without replace: %v, want fs.ErrExist", err)
	}
	if data, _ := s.ReadFile("A.bas"); string(data) != "one" {
		t.Errorf("file %q after a refused write, want \"one\"", data)
	}
	if err := s.WriteFile("A.bas", []byte("three"), true); err != nil {
		t.Fatal(err)
	}
	if data, _ := s.ReadFile("A.bas"); string(data) != "three" {
		t.Errorf("file %q after replacing, want \"three\"", data)
	}
	if info, err := os.Stat(filepath.Join(dir, "A.bas")); err != nil || info.Mode().Perm() != 0o644 {
		t.Errorf("permissions %v, %v; want 0644", info.Mode().Perm(), err)
	}
}
