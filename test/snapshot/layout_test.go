package snapshot_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// writeTree creates files under root; a name ending in "/" is a directory.
func writeTree(t *testing.T, root string, names ...string) {
	t.Helper()
	for _, name := range names {
		path := filepath.Join(root, name)
		if name[len(name)-1] == '/' {
			if err := os.MkdirAll(path, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// @spec SNAPSHOT-002, SNAPSHOT-014
func TestFeatureDirectoryNames(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, "001-hello.bas", "README.md", "README", "notes.txt", "sub/", ".DS_Store", ".hidden/")
	got := nameProblems(dir)
	want := []string{"README", "notes.txt", "sub"}
	if !slices.Equal(got, want) {
		t.Errorf("nameProblems = %v, want %v (only README.md is allowed besides examples; hidden entries are ignored)", got, want)
	}
	// In a directory of its own: a filesystem that ignores case cannot hold
	// both README.md and readme.md.
	lower := t.TempDir()
	writeTree(t, lower, "readme.md")
	if got := nameProblems(lower); !slices.Equal(got, []string{"readme.md"}) {
		t.Errorf("nameProblems = %v, want [readme.md] (only the exact name README.md is allowed)", got)
	}
}

// @spec SNAPSHOT-012
func TestMissingOrEmptyFeatureDirectory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "features")
	if got, err := examplesIn(missing); err != nil || len(got) != 0 {
		t.Errorf("missing directory: examples %v, error %v; want none and no error", got, err)
	}
	if got := nameProblems(missing); len(got) != 0 {
		t.Errorf("missing directory: name problems %v, want none", got)
	}
	onlyReadme := t.TempDir()
	writeTree(t, onlyReadme, "README.md")
	if got, err := examplesIn(onlyReadme); err != nil || len(got) != 0 {
		t.Errorf("only README.md: examples %v, error %v; want none", got, err)
	}
}

// @spec SNAPSHOT-014
func TestOutsideFeatureDirectoryIgnored(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root,
		"examples/README.md", "examples/notes.txt", "examples/games/battleship.bas", "examples/033-misplaced.bas",
		"examples/features/001-hello.bas", "examples/features/README.md")
	features := filepath.Join(root, "examples", "features")
	got, err := examplesIn(features)
	if err != nil || !slices.Equal(got, []string{"001-hello.bas"}) {
		t.Errorf("examples = %v, error %v; want only [001-hello.bas] (README.md and everything outside features/ ignored)", got, err)
	}
	if problems := nameProblems(features); len(problems) != 0 {
		t.Errorf("name problems %v, want none", problems)
	}
}

// @spec SNAPSHOT-013
func TestStaleSnapshots(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, "001-hello.snap", "002-gone.snap", "001-hello.input")
	got, err := staleSnapshots(dir, map[string]bool{"001-hello.snap": true})
	if err != nil || !slices.Equal(got, []string{filepath.Join(dir, "002-gone.snap")}) {
		t.Errorf("staleSnapshots = %v, %v; want only 002-gone.snap", got, err)
	}
	got, _ = staleSnapshots(dir, map[string]bool{})
	if len(got) != 2 {
		t.Errorf("with no examples, staleSnapshots = %v; want both snapshots (leftovers fail)", got)
	}
}
