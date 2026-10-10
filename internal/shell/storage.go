package shell

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bryanesmith/c64sh/internal/interp"
)

// dirStorage holds LOAD, SAVE, and VERIFY files in a directory: the
// current directory when dir is empty.
//
// @spec SHELL-FILE-001
type dirStorage struct{ dir string }

func (s dirStorage) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(s.dir, name))
}

// WriteFile writes a file with permissions 0644. Without replace, it
// creates the file only if it does not exist, in one step.
func (s dirStorage) WriteFile(name string, data []byte, replace bool) error {
	path := filepath.Join(s.dir, name)
	if replace {
		return os.WriteFile(path, data, 0o644)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// Files returns the directory's regular files that are not hidden, in
// order of name, with their sizes.
func (s dirStorage) Files() ([]interp.StoredFile, error) {
	dir := s.dir
	if dir == "" {
		dir = "."
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []interp.StoredFile
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") || !e.Type().IsRegular() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue // removed since it was listed
		}
		files = append(files, interp.StoredFile{Name: e.Name(), Size: info.Size()})
	}
	return files, nil
}

// Remove deletes a file.
func (s dirStorage) Remove(name string) error {
	return os.Remove(filepath.Join(s.dir, name))
}

// Rename renames a file, refusing to replace one.
func (s dirStorage) Rename(oldName, newName string) error {
	newPath := filepath.Join(s.dir, newName)
	if _, err := os.Lstat(newPath); err == nil {
		return &os.PathError{Op: "rename", Path: newName, Err: fs.ErrExist}
	}
	return os.Rename(filepath.Join(s.dir, oldName), newPath)
}
