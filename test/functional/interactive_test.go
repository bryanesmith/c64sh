package functional_test

import "testing"

// @spec SHELL-INT-001
func TestInteractiveBanner(t *testing.T) {
	r := runInteractive(t, "")
	if len(r.stderr) < len(banner) || r.stderr[:len(banner)] != banner {
		t.Errorf("stderr %q does not start with banner %q", r.stderr, banner)
	}
}

// @spec SHELL-INT-002
func TestReadyAfterEachLine(t *testing.T) {
	check(t, "successful line", runInteractive(t, "PRINT \"A\"\n"),
		result{"A\n", banner + "READY.\n\n", 0})
	check(t, "line with error", runInteractive(t, "@\n"),
		result{"", banner + "?SYNTAX  ERROR\nREADY.\n\n", 0})
	check(t, "empty statement", runInteractive(t, ":\n"),
		result{"", banner + "READY.\n\n", 0})
	check(t, "two lines", runInteractive(t, "PRINT \"A\"\nPRINT \"B\"\n"),
		result{"A\nB\n", banner + "READY.\nREADY.\n\n", 0})
}

// @spec SHELL-INT-003
func TestReadyStartsOnFreshLine(t *testing.T) {
	check(t, "trailing semicolon", runInteractive(t, "PRINT \"A\";\n"),
		result{"A\n", banner + "READY.\n\n", 0})
	check(t, "trailing comma", runInteractive(t, "PRINT \"A\",\n"),
		result{"A\t\n", banner + "READY.\n\n", 0})
}

// @spec SHELL-INT-004
func TestInteractiveBlankLineDoesNothing(t *testing.T) {
	check(t, "blank lines", runInteractive(t, "\n  \t\n"),
		result{"", banner + "\n", 0})
}

// @spec SHELL-INT-005
func TestInteractiveContinuesAfterError(t *testing.T) {
	check(t, "error then print", runInteractive(t, "@\nPRINT \"B\"\n"),
		result{"B\n", banner + "?SYNTAX  ERROR\nREADY.\nREADY.\n\n", 0})
}

// @spec SHELL-INT-006
func TestInteractiveEndOfInput(t *testing.T) {
	check(t, "immediate end of input", runInteractive(t, ""),
		result{"", banner + "\n", 0})
}
