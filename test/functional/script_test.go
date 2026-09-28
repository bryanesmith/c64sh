package functional_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// @spec SHELL-SCRIPT-001
func TestScriptModeHasNoBannerOrReady(t *testing.T) {
	check(t, "stdin script", runMain(t, "PRINT \"A\"\nPRINT \"B\"\n"), result{"A\nB\n", "", 0})
}

// @spec SHELL-SCRIPT-002
func TestShebangLineIsSkipped(t *testing.T) {
	src := "#!/usr/bin/env c64sh\nPRINT \"A\"\n"
	check(t, "stdin", runMain(t, src), result{"A\n", "", 0})
	check(t, "file", runMain(t, "", writeFile(t, "s.bas", src)), result{"A\n", "", 0})
}

// @spec SHELL-SCRIPT-003
func TestShebangAfterFirstLineIsInput(t *testing.T) {
	check(t, "second-line shebang", runMain(t, "PRINT \"A\"\n#!/usr/bin/env c64sh\n"),
		result{"A\n", "?SYNTAX  ERROR\n", 1})
}

// @spec SHELL-SCRIPT-004
func TestScriptSkipsBlankLines(t *testing.T) {
	check(t, "blank lines", runMain(t, "\nPRINT \"A\"\n  \t\n\nPRINT \"B\"\n"),
		result{"A\nB\n", "", 0})
}

// @spec SHELL-SCRIPT-005
func TestScriptStopsAtFirstError(t *testing.T) {
	check(t, "error in middle", runMain(t, "PRINT \"A\"\n@\nPRINT \"B\"\n"),
		result{"A\n", "?SYNTAX  ERROR\n", 1})
	check(t, "error in file", runMain(t, "", writeFile(t, "s.bas", "@\nPRINT \"B\"\n")),
		result{"", "?SYNTAX  ERROR\n", 1})
}

// @spec SHELL-SCRIPT-006
func TestScriptSuccessExitsZero(t *testing.T) {
	check(t, "all lines succeed", runMain(t, "PRINT \"A\"\n"), result{"A\n", "", 0})
	check(t, "empty script", runMain(t, ""), result{"", "", 0})
}

// @spec SHELL-SCRIPT-007
func TestShebangScriptRunsThroughPath(t *testing.T) {
	bin := binary(t)
	script := filepath.Join(t.TempDir(), "hello.bas")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env c64sh\nPRINT \"HELLO\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(script)
	cmd.Env = append(os.Environ(), "PATH="+filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	code := exitCode(t, cmd.Run())
	check(t, "executable script", result{stdout.String(), stderr.String(), code}, result{"HELLO\n", "", 0})
}

// @spec SHELL-SCRIPT-008
func TestScriptEndAddsNothing(t *testing.T) {
	check(t, "trailing semicolon", runMain(t, "PRINT \"A\";\n"), result{"A", "", 0})
	check(t, "two partial lines", runMain(t, "PRINT \"A\";\nPRINT \"B\","), result{"AB\t", "", 0})
}
