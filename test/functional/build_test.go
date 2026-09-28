package functional_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// makeCmd returns a make command run at the repository root, skipping the
// test if make is not installed.
func makeCmd(t *testing.T, env []string, args ...string) *exec.Cmd {
	t.Helper()
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make not found on PATH")
	}
	cmd := exec.Command("make", args...)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), env...)
	return cmd
}

// dryRun returns the commands make would run for args.
func dryRun(t *testing.T, env []string, args ...string) string {
	t.Helper()
	// "make ... -n" runs Make in dry run. E.g.,
	//    % make build -n
        //    go build -o bin/c64sh ./cmd/c64sh
	out, err := makeCmd(t, env, append([]string{"-n"}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("make -n %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func wantContains(t *testing.T, name, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("%s: output %q does not contain %q", name, got, want)
	}
}

// @spec BUILD-001
func TestModulePath(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	first, _, _ := strings.Cut(string(data), "\n")
	if first != "module github.com/bryanesmith/c64sh" {
		t.Errorf("go.mod first line %q, want %q", first, "module github.com/bryanesmith/c64sh")
	}
}

// @spec BUILD-002
func TestMakeBuild(t *testing.T) {
	want := "go build -o bin/c64sh ./cmd/c64sh"
	wantContains(t, "make", dryRun(t, nil), want)
	wantContains(t, "make build", dryRun(t, nil, "build"), want)
}

// @spec BUILD-003, BUILD-008
func TestMakeRun(t *testing.T) {
	script := writeFile(t, "hello.bas", "PRINT \"A\"\n")
	cmd := makeCmd(t, nil, "run", "ARGS="+script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("make run: %v\n%s", err, stderr.String())
	}
	if stdout.String() != "A\n" {
		t.Errorf("make run stdout %q, want only the program output %q", stdout.String(), "A\n")
	}
	wantContains(t, "make -n run", dryRun(t, nil, "run", "ARGS=foo.bas"), "bin/c64sh foo.bas")
}

// @spec BUILD-004
func TestMakeInstall(t *testing.T) {
	cases := map[string]bool{"new directory": false, "replace existing file": true}
	for name, existing := range cases {
		dir := filepath.Join(t.TempDir(), "a", "b")
		target := filepath.Join(dir, "c64sh")
		if existing {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		cmd := makeCmd(t, nil, "install", "INSTALL_DIR="+dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s: make install: %v\n%s", name, err, out)
		}
		info, err := os.Stat(target)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if perm := info.Mode().Perm(); perm != 0o755 {
			t.Errorf("%s: installed mode %v, want 0755", name, perm)
		}
		run := exec.Command(target)
		run.Stdin = strings.NewReader("PRINT \"HELLO\"\n")
		out, err := run.Output()
		if err != nil || string(out) != "HELLO\n" {
			t.Errorf("%s: installed c64sh printed %q (err %v), want %q", name, out, err, "HELLO\n")
		}
	}
}

// @spec BUILD-005
func TestMakeVariables(t *testing.T) {
	wantContains(t, "default INSTALL_DIR", dryRun(t, []string{"HOME=/fakehome"}, "install"), "/fakehome/bin/c64sh")
	wantContains(t, "INSTALL_DIR from environment", dryRun(t, []string{"INSTALL_DIR=/x/y"}, "install"), "/x/y/c64sh")
	wantContains(t, "INSTALL_DIR on command line", dryRun(t, nil, "install", "INSTALL_DIR=/p/q"), "/p/q/c64sh")
	wantContains(t, "GO and BIN on command line", dryRun(t, nil, "build", "GO=mygo", "BIN=out/prog"), "mygo build -o out/prog ./cmd/c64sh")
	wantContains(t, "GO from environment", dryRun(t, []string{"GO=envgo"}, "build"), "envgo build")
	if got := dryRun(t, nil, "run"); !strings.Contains(got, "bin/c64sh") {
		t.Errorf("default run: output %q does not run bin/c64sh", got)
	}
}

// @spec BUILD-006
func TestMakeTest(t *testing.T) {
	wantContains(t, "make -n test", dryRun(t, nil, "test"), "go test ./...")
}

// @spec BUILD-007
func TestMakeClean(t *testing.T) {
	wantContains(t, "make -n clean", dryRun(t, nil, "clean"), "rm -rf bin")
}

// @spec BUILD-009
func TestGitignoreExcludesBin(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(data), "\n")
	if !slices.Contains(lines, "bin/") && !slices.Contains(lines, "/bin/") {
		t.Errorf(".gitignore does not exclude bin/")
	}
}
