package functional_test

import (
	"io"
	"strings"
	"testing"

	"github.com/bryanesmith/c64sh/internal/shell"
)

// shellMain runs shell.Main with no arguments on the given streams.
func shellMain(stdin io.Reader, stdout, stderr io.Writer) int {
	return shell.Main(nil, stdin, stdout, stderr)
}

// @spec SHELL-ERR-001
func TestErrorFormat(t *testing.T) {
	check(t, "syntax", runMain(t, "@\n"), result{"", "?SYNTAX  ERROR\n", 1})
	long := strings.Repeat("x", 200)
	check(t, "string too long", runMain(t, "PRINT \""+long+"\"+\""+long+"\"\n"),
		result{"", "?STRING TOO LONG  ERROR\n", 1})
}

// @spec SHELL-ERR-002
func TestNewlineBeforeErrorWhenMidLine(t *testing.T) {
	check(t, "mid-line on same line", runMain(t, "PRINT \"A\";:@\n"),
		result{"A\n", "?SYNTAX  ERROR\n", 1})
	check(t, "mid-line from earlier line", runMain(t, "PRINT \"A\";\n@\n"),
		result{"A\n", "?SYNTAX  ERROR\n", 1})
	check(t, "already at line start", runMain(t, "PRINT \"A\"\n@\n"),
		result{"A\n", "?SYNTAX  ERROR\n", 1})
	check(t, "no output yet", runMain(t, "@\n"),
		result{"", "?SYNTAX  ERROR\n", 1})
	check(t, "interactive", runInteractive(t, "PRINT \"A\";:@\n"),
		result{"A\n", banner + "?SYNTAX  ERROR\nREADY.\n\n", 0})
}
