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

// @spec SHELL-FILE-001
func TestDirStorageFiles(t *testing.T) {
	dir := t.TempDir()
	s := dirStorage{dir: dir}
	for name, data := range map[string]string{"B": "12345", "A.bas": "", ".hidden": "x"} {
		os.WriteFile(filepath.Join(dir, name), []byte(data), 0o644)
	}
	os.Mkdir(filepath.Join(dir, "SUB"), 0o755)
	files, err := s.Files()
	if err != nil || len(files) != 2 || files[0].Name != "A.bas" || files[1].Name != "B" || files[1].Size != 5 {
		t.Errorf("Files = %+v, %v; want A.bas and B (5 bytes), without the hidden file or directory", files, err)
	}
	if err := s.Rename("B", "A.bas"); !errors.Is(err, fs.ErrExist) {
		t.Errorf("Rename onto an existing file: %v, want fs.ErrExist", err)
	}
	if err := s.Rename("B", "C"); err != nil {
		t.Fatal(err)
	}
	if err := s.Remove("C"); err != nil {
		t.Fatal(err)
	}
	if files, _ := s.Files(); len(files) != 1 {
		t.Errorf("after rename and remove: %+v, want only A.bas", files)
	}
}
