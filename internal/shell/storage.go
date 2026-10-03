package shell

import (
	"os"
	"path/filepath"
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
