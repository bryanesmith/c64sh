package functional_test

import "testing"

// TestComments checks REM comments end to end, as listed in the HLD's Goals.
func TestComments(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  result
	}{
		{"comment line", "REM THIS DOES NOTHING\n", result{"", "", 0}},
		{"comment after a statement", "PRINT \"A\":REM SHOW A\n", result{"A\n", "", 0}},
		{"colon inside a comment", "REM A:PRINT \"X\"\n", result{"", "", 0}},
		{"no space after REM", "REMARK\nPRINT \"B\"\n", result{"B\n", "", 0}},
		{"REM inside a string", "PRINT \"REM\"\n", result{"REM\n", "", 0}},
		{"REM without a colon inside PRINT", "PRINT \"A\" REM NOTE\n", result{"A\n", "?SYNTAX  ERROR\n", 1}},
		{"lowercase rem", "rem hello\n", result{"", "?SYNTAX  ERROR\n", 1}},
		{"trailing semicolon then comment", "PRINT \"A\";:REM KEEP GOING\nPRINT \"B\"\n", result{"AB\n", "", 0}},
		{"commented script", "#!/usr/bin/env c64sh\nREM GREET THE USER\nPRINT \"HELLO\" : REM SAY HELLO\n", result{"HELLO\n", "", 0}},
	}
	for _, c := range cases {
		check(t, c.name, runMain(t, c.input), c.want)
	}
}

// TestCommentsInteractive checks that a comment-only line is a command:
// it prints READY., as on a C64.
func TestCommentsInteractive(t *testing.T) {
	check(t, "comment line", runInteractive(t, "REM HELLO\n"),
		result{"", banner + "READY.\n\n", 0})
	check(t, "trailing semicolon then comment", runInteractive(t, "PRINT \"A\";:REM X\n"),
		result{"A\n", banner + "READY.\n\n", 0})
}
