// Package tutorial_test runs the programs in the tutorial: each chapter's
// complete listing, with recorded input, compared with a recorded result.
package tutorial_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/bryanesmith/c64sh/internal/shell"
)

const (
	tutorialDir = "../../docs/tutorial"
	testdataDir = "testdata"
)

// chapterPattern matches the tutorial's numbered pages, chapters and
// projects alike.
var chapterPattern = regexp.MustCompile(`^(\d\d-[a-z0-9-]+|projects/[a-z0-9-]+)\.md$`)

// fixedClock is the time the programs see, so their output does not change.
var fixedClock = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

func updating() bool { return os.Getenv("UPDATE_SNAPS") == "true" }

// pages returns the tutorial's chapter and project pages, relative to
// docs/tutorial.
func pages(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, pattern := range []string{"[0-9][0-9]-*.md", "projects/*.md"} {
		matches, err := filepath.Glob(filepath.Join(tutorialDir, pattern))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range matches {
			rel, _ := filepath.Rel(tutorialDir, m)
			out = append(out, filepath.ToSlash(rel))
		}
	}
	return out
}

// basicBlocks returns the contents of a page's ```basic code blocks.
func basicBlocks(page string) []string {
	var blocks []string
	var cur *strings.Builder
	for _, line := range strings.Split(page, "\n") {
		switch {
		case cur == nil && strings.TrimSpace(line) == "```basic":
			cur = &strings.Builder{}
		case cur != nil && strings.TrimSpace(line) == "```":
			blocks = append(blocks, cur.String())
			cur = nil
		case cur != nil:
			cur.WriteString(line + "\n")
		}
	}
	return blocks
}

// name returns the test name for a page: its path without ".md", with
// "/" as "-".
func name(page string) string {
	return strings.ReplaceAll(strings.TrimSuffix(page, ".md"), "/", "-")
}

// @spec TUTORIAL-002, TUTORIAL-003
func TestListings(t *testing.T) {
	for _, page := range pages(t) {
		t.Run(name(page), func(t *testing.T) {
			if !chapterPattern.MatchString(page) {
				t.Fatalf("%s: name must look like 01-lowercase-words.md or projects/words.md", page)
			}
			data, err := os.ReadFile(filepath.Join(tutorialDir, page))
			if err != nil {
				t.Fatal(err)
			}
			blocks := basicBlocks(string(data))
			if len(blocks) == 0 {
				return // a page without a program
			}
			listing := blocks[len(blocks)-1]
			lines := map[string]bool{}
			for _, l := range strings.Split(listing, "\n") {
				lines[l] = true
			}
			for _, b := range blocks[:len(blocks)-1] {
				for _, l := range strings.Split(strings.TrimRight(b, "\n"), "\n") {
					if !lines[l] {
						t.Errorf("%s: excerpt line %q is not in the page's complete listing", page, l)
					}
				}
			}
			got := run(t, listing, name(page))
			match(t, filepath.Join(testdataDir, name(page)+".snap"), got)
		})
	}
}

// run runs a listing as a program file, in a new temporary directory,
// with the page's recorded input, and returns the formatted result.
func run(t *testing.T, listing, name string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "adventure.bas")
	if err := os.WriteFile(path, []byte(listing), 0o644); err != nil {
		t.Fatal(err)
	}
	stdin, _ := os.ReadFile(filepath.Join(testdataDir, name+".input"))
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	var stdout, stderr bytes.Buffer
	cfg := shell.Config{File: path, Clock: func() time.Time { return fixedClock }}
	status := shell.Run(cfg, bytes.NewReader(stdin), &stdout, &stderr)
	return fmt.Sprintf("exit status: %d\n--- stdout ---\n%s\n--- stderr ---\n%s", status, stdout.String(), stderr.String())
}

// match compares got with the recorded result, or records it when
// updating.
func match(t *testing.T, path, got string) {
	t.Helper()
	if updating() {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: %v; run `make update-snapshots` to create it", path, err)
	}
	if string(want) != got {
		t.Errorf("%s differs; run `make update-snapshots` if the change is intended, then review the diff\n got:\n%s", path, got)
	}
}

// @spec TUTORIAL-001
func TestNavigation(t *testing.T) {
	index, err := os.ReadFile(filepath.Join(tutorialDir, "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	var chapters []string
	for _, page := range pages(t) {
		if !strings.Contains(string(index), "("+page+")") {
			t.Errorf("index.md does not link %s", page)
		}
		if !strings.HasPrefix(page, "projects/") {
			chapters = append(chapters, page)
		}
	}
	matches, _ := filepath.Glob(filepath.Join(tutorialDir, "[0-9][0-9]-*.md"))
	for _, m := range matches {
		chapters = appendUnique(chapters, filepath.Base(m))
	}
	for i, ch := range chapters {
		data, err := os.ReadFile(filepath.Join(tutorialDir, ch))
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"(index.md)"}
		if i > 0 {
			want = append(want, "("+chapters[i-1]+")")
		}
		if i+1 < len(chapters) {
			want = append(want, "("+chapters[i+1]+")")
		}
		for _, link := range want {
			if !strings.Contains(string(data), link) {
				t.Errorf("%s does not link %s", ch, link)
			}
		}
	}
}

func appendUnique(list []string, s string) []string {
	for _, x := range list {
		if x == s {
			return list
		}
	}
	return append(list, s)
}

// @spec TUTORIAL-004
func TestTestdataHasPages(t *testing.T) {
	names := map[string]bool{}
	for _, page := range pages(t) {
		names[name(page)] = true
	}
	files, err := filepath.Glob(filepath.Join(testdataDir, "*"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		base := strings.TrimSuffix(strings.TrimSuffix(filepath.Base(f), ".snap"), ".input")
		if !names[base] {
			t.Errorf("%s belongs to no tutorial page; delete it", f)
		}
	}
}
