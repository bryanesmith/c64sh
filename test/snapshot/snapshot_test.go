// Package snapshot_test runs every script in examples/ and compares its
// result with a recorded snapshot, and checks that the examples follow
// their conventions. Run `make update-snapshots` to rewrite the snapshots
// after an intended change, then review the diff.
package snapshot_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bryanesmith/c64sh/internal/lexer"
	"github.com/bryanesmith/c64sh/internal/shell"
	"github.com/bryanesmith/c64sh/internal/token"
)

const (
	examplesDir = "../../examples"
	snapshotDir = "testdata"
	shebang     = "#!/usr/bin/env c64sh"
)

// examplePattern matches an example file name and captures its number.
var examplePattern = regexp.MustCompile(`^(\d{3})-[a-z0-9]+(-[a-z0-9]+)*\.bas$`)

// updating reports whether snapshots should be rewritten rather than checked.
func updating() bool {
	return os.Getenv("UPDATE_SNAPS") == "true"
}

// entries returns the non-hidden entries of examples/.
func entries(t *testing.T) []os.DirEntry {
	t.Helper()
	all, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Fatalf("reading examples: %v", err)
	}
	var visible []os.DirEntry
	for _, e := range all {
		if !strings.HasPrefix(e.Name(), ".") {
			visible = append(visible, e)
		}
	}
	return visible
}

// examples returns the file names in examples/ that match the example
// naming pattern, in order.
func examples(t *testing.T) []string {
	t.Helper()
	var names []string
	for _, e := range entries(t) {
		if !e.IsDir() && examplePattern.MatchString(e.Name()) {
			names = append(names, e.Name())
		}
	}
	return names
}

// lines returns the lines of an example, without line terminators.
func lines(t *testing.T, file string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(examplesDir, file))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
}

// format renders a run's result as a snapshot.
func format(status int, stdout, stderr string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "exit status: %d\n", status)
	section(&b, "stdout", stdout)
	section(&b, "stderr", stderr)
	return b.String()
}

// section writes one output stream. Output that does not end with a newline
// is marked in the header and followed by one, so the next header starts on
// its own line.
func section(b *strings.Builder, name, out string) {
	if out != "" && !strings.HasSuffix(out, "\n") {
		fmt.Fprintf(b, "--- %s (no newline at end) ---\n%s\n", name, out)
		return
	}
	fmt.Fprintf(b, "--- %s ---\n%s", name, out)
}

// snapshotFile returns the file holding an example's snapshot.
func snapshotFile(name string) string {
	return name + ".snap"
}

// matchSnapshot compares got with the snapshot stored in file, or writes it
// when updating.
func matchSnapshot(t *testing.T, name, file, got string) {
	t.Helper()
	path := filepath.Join(snapshotDir, file)
	if updating() {
		if err := os.MkdirAll(snapshotDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		t.Fatalf("%s: snapshot %s is missing; run `make update-snapshots` to record it, then review it", name, path)
	}
	if err != nil {
		t.Fatal(err)
	}
	if want := string(data); got != want {
		n, gotLine, wantLine := firstDifference(got, want)
		t.Errorf("%s: snapshot differs at line %d:\n     got: %s\n    want: %s\nrun `make update-snapshots` if the change is intended, then review the diff",
			name, n, gotLine, wantLine)
	}
}

// firstDifference returns the 1-based number of the first line where got and
// want differ, and both versions of that line, Go-quoted.
func firstDifference(got, want string) (int, string, string) {
	g, w := strings.SplitAfter(got, "\n"), strings.SplitAfter(want, "\n")
	line := func(lines []string, i int) string {
		if i < len(lines) {
			return strconv.Quote(lines[i])
		}
		return "(end of snapshot)"
	}
	for i := 0; ; i++ {
		if i >= len(g) || i >= len(w) || g[i] != w[i] {
			return i + 1, line(g, i), line(w, i)
		}
	}
}

// @spec SNAPSHOT-001, SNAPSHOT-008, SNAPSHOT-009, SNAPSHOT-010, SNAPSHOT-011
func TestExamples(t *testing.T) {
	files := examples(t)
	want := map[string]bool{}
	for _, file := range files {
		name := strings.TrimSuffix(file, ".bas")
		want[snapshotFile(name)] = true
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			stdin, _ := os.ReadFile(filepath.Join(snapshotDir, name+".input")) // none: empty stdin
			path, err := filepath.Abs(filepath.Join(examplesDir, file))
			if err != nil {
				t.Fatal(err)
			}
			status := inTempDir(t, func() int {
				cfg := shell.Config{File: path, Clock: func() time.Time { return fixedClock }}
				return shell.Run(cfg, bytes.NewReader(stdin), &stdout, &stderr)
			})
			matchSnapshot(t, name, snapshotFile(name), format(status, stdout.String(), stderr.String()))
		})
	}

	// Snapshots whose example no longer exists.
	stored, err := filepath.Glob(filepath.Join(snapshotDir, "*.snap"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range stored {
		if want[filepath.Base(path)] {
			continue
		}
		if updating() {
			if err := os.Remove(path); err != nil {
				t.Error(err)
			}
			continue
		}
		t.Errorf("snapshot %s has no example in examples/; run `make update-snapshots` to delete it", path)
	}

	// Input files, written by hand, whose example no longer exists.
	inputs, err := filepath.Glob(filepath.Join(snapshotDir, "*.input"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range inputs {
		if !want[strings.TrimSuffix(filepath.Base(path), ".input")+".snap"] {
			t.Errorf("input file %s has no example in examples/; delete it", path)
		}
	}
}

// fixedClock is the time examples see, so their output does not change.
var fixedClock = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

// inTempDir runs f with a new, empty temporary directory as the current
// directory, so that files an example saves are discarded, and returns
// what f returns.
func inTempDir(t *testing.T, f func() int) int {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(old); err != nil {
			t.Fatal(err)
		}
	}()
	return f()
}

// @spec SNAPSHOT-002
func TestExampleNames(t *testing.T) {
	for _, e := range entries(t) {
		switch {
		case e.IsDir():
			t.Errorf("examples/%s: directories do not belong in examples/", e.Name())
		case !examplePattern.MatchString(e.Name()):
			t.Errorf("examples/%s: name must look like 001-lowercase-words.bas", e.Name())
		}
	}
}

// @spec SNAPSHOT-003
func TestExampleNumbering(t *testing.T) {
	var numbers []int
	for _, file := range examples(t) {
		n, _ := strconv.Atoi(examplePattern.FindStringSubmatch(file)[1])
		numbers = append(numbers, n)
	}
	slices.Sort(numbers)
	for i, n := range numbers {
		if n != i+1 {
			t.Errorf("example numbers are %v; want 001 through %03d with no gaps or duplicates", numbers, len(numbers))
			return
		}
	}
}

// @spec SNAPSHOT-004
func TestExampleShebang(t *testing.T) {
	for _, file := range examples(t) {
		if first := lines(t, file)[0]; first != shebang {
			t.Errorf("examples/%s: first line is %q, want %q", file, first, shebang)
		}
	}
}

// @spec SNAPSHOT-005
func TestExampleDescription(t *testing.T) {
	for _, file := range examples(t) {
		ls := lines(t, file)
		if len(ls) < 2 || !strings.HasPrefix(ls[1], "REM") || strings.TrimSpace(ls[1][3:]) == "" {
			t.Errorf("examples/%s: second line must be a REM comment explaining the file", file)
		}
	}
}

// @spec SNAPSHOT-006
func TestExampleIsExecutable(t *testing.T) {
	for _, file := range examples(t) {
		info, err := os.Stat(filepath.Join(examplesDir, file))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0o100 == 0 {
			t.Errorf("examples/%s: not executable; run chmod +x examples/%s", file, file)
		}
	}
}

// isCommentLine reports whether tokens form a line holding only a comment.
func isCommentLine(tokens []token.Token) bool {
	return len(tokens) == 2 && tokens[0].Kind == token.Rem
}

// @spec SNAPSHOT-007
func TestExamplePrintComments(t *testing.T) {
	for _, file := range examples(t) {
		ls := lines(t, file)
		for i := 1; i < len(ls); i++ {
			tokens := lexer.Lex(ls[i])
			hasPrint := slices.ContainsFunc(tokens, func(tok token.Token) bool { return tok.Kind == token.Print })
			if !hasPrint {
				continue
			}
			endsWithComment := tokens[len(tokens)-2].Kind == token.Rem
			if endsWithComment || isCommentLine(lexer.Lex(ls[i-1])) {
				continue
			}
			t.Errorf("examples/%s:%d: end the line with :REM and what it prints, or put a REM line before it", file, i+1)
		}
	}
}
