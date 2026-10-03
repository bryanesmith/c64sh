package functional_test

import "testing"

// @spec SHELL-PROG-001, SHELL-MODE-004
func TestNumberedLinesAreStored(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  result
	}{
		{"stored, then run", "10 PRINT \"A\"\nPRINT \"B\"\nRUN\n", result{"B\nA\n", "", 0}},
		{"listed as typed", "20 PRINT \"B\"\n10 ?\"A\"\nLIST\nRUN\n", result{"\n10 PRINT\"A\"\n20 PRINT \"B\"\nA\nB\n", "", 0}},
		{"leading spaces", "  10 PRINT \"A\"\nLIST\nRUN\n", result{"\n10 PRINT \"A\"\nA\n", "", 0}},
		{"number alone deletes", "10 PRINT \"A\"\n20 PRINT \"B\"\n10\nRUN\n", result{"B\n", "", 0}},
		{"storing clears variables", "A=5\n10 PRINT A\nPRINT A\n", result{" 0 \n 0 \n", "", 0}},
		{"RUN n", "10 PRINT \"A\"\n20 PRINT \"B\"\nRUN 20\n", result{"B\n", "", 0}},
		{"NEW", "10 PRINT \"A\"\nNEW\nLIST\n", result{"", "", 0}},
		{"END", "10 PRINT \"A\"\n20 END\n30 PRINT \"B\"\n", result{"A\n", "", 0}},
	}
	for _, c := range cases {
		check(t, c.name, runMain(t, c.input), c.want)
	}
}

// @spec SHELL-PROG-002
func TestLineNumberTooLarge(t *testing.T) {
	check(t, "script", runMain(t, "64000 PRINT \"A\"\nPRINT \"B\"\n"), result{"", "?SYNTAX  ERROR\n", 1})
	check(t, "interactive", runInteractive(t, "64000 PRINT \"A\"\nLIST\n"),
		result{"", banner + "?SYNTAX  ERROR\nREADY.\nREADY.\n\n", 0})
}

// @spec SHELL-INT-002
func TestNoReadyAfterStoredLine(t *testing.T) {
	check(t, "stored lines", runInteractive(t, "10 PRINT \"A\"\n20 PRINT \"B\"\nRUN\n"),
		result{"A\nB\n", banner + "READY.\n\n", 0})
	check(t, "LIST", runInteractive(t, "10 PRINT \"A\"\nLIST\n"),
		result{"\n10 PRINT \"A\"\n", banner + "READY.\n\n", 0})
	check(t, "program error", runInteractive(t, "10 PRINT \"A\";\n20 @\nRUN\nPRINT \"B\"\n"),
		result{"A\nB\n", banner + "?SYNTAX  ERROR IN 20\nREADY.\nREADY.\n\n", 0})
	check(t, "not run when input ends", runInteractive(t, "10 PRINT \"A\"\n"),
		result{"", banner + "\n", 0})
}

// @spec SHELL-SCRIPT-009, SHELL-SCRIPT-008
func TestScriptRunsProgramItNeverRan(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  result
	}{
		{"numbered lines only", "20 PRINT \"B\"\n10 PRINT \"A\"\n", result{"A\nB\n", "", 0}},
		{"mixed", "30 PRINT \"1\"\nPRINT \"2\"\n10 PRINT \"3\"\nPRINT \"4\"\n40 PRINT \"5\"\n", result{"2\n4\n3\n1\n5\n", "", 0}},
		{"mixed with RUN", "30 PRINT \"1\"\nPRINT \"2\"\n10 PRINT \"3\"\nPRINT \"4\"\n40 PRINT \"5\"\nRUN\n", result{"2\n4\n3\n1\n5\n", "", 0}},
		{"RUN twice", "10 PRINT \"A\"\nRUN\nRUN\n", result{"A\nA\n", "", 0}},
		{"lines stored after RUN", "10 PRINT \"A\"\nRUN\n20 PRINT \"B\"\n", result{"A\n", "", 0}},
		{"after LIST", "10 PRINT \"A\"\nLIST\n", result{"\n10 PRINT \"A\"\nA\n", "", 0}},
		{"no numbered lines", "PRINT \"A\"\n", result{"A\n", "", 0}},
		{"all deleted", "10 PRINT \"A\"\n10\n", result{"", "", 0}},
		{"ends mid-line", "10 PRINT \"A\";\n", result{"A", "", 0}},
		{"error in the program", "10 PRINT \"A\"\n20 PRINT 1/0\n", result{"A\n", "?DIVISION BY ZERO  ERROR IN 20\n", 1}},
		{"error before input ends", "10 PRINT \"A\"\n@\n", result{"", "?SYNTAX  ERROR\n", 1}},
		{"piped stdin", "10 PRINT \"A\"\n", result{"A\n", "", 0}},
	}
	for _, c := range cases {
		check(t, c.name, runMain(t, c.input), c.want)
	}
	path := writeFile(t, "prog.bas", "#!/usr/bin/env c64sh\n10 PRINT \"A\"\n")
	check(t, "file", runMain(t, "", path), result{"A\n", "", 0})
}

// @spec SHELL-ERR-001
func TestProgramErrorsNameTheLine(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  result
	}{
		{"syntax", "10 PRINT \"A\"\n20 PRINT \"B\"@\nRUN\n", result{"A\nB\n", "?SYNTAX  ERROR IN 20\n", 1}},
		{"line 0", "0 @\n", result{"", "?SYNTAX  ERROR IN 0\n", 1}},
		{"undefined line, direct", "10 PRINT \"A\"\nRUN 20\n", result{"", "?UNDEF'D STATEMENT  ERROR\n", 1}},
		{"undefined line, in program", "10 RUN 20\n", result{"", "?UNDEF'D STATEMENT  ERROR IN 10\n", 1}},
		{"skipped by IF", "10 IF 0 THEN @\n20 PRINT \"OK\"\n", result{"OK\n", "", 0}},
		{"LIST junk", "10 PRINT \"A\"\nLIST 10\n", result{"", "?SYNTAX  ERROR\n", 1}},
	}
	for _, c := range cases {
		check(t, c.name, runMain(t, c.input), c.want)
	}
}

// @spec SHELL-SCRIPT-009
func TestGotoInScripts(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  result
	}{
		{"GOTO starts the program, keeping variables", "10 PRINT A\nA=5\nGOTO 10\n", result{" 5 \n", "", 0}},
		{"loop", "10 A=A+1:PRINT A;\n20 IF A<3 THEN 10\n", result{" 1  2  3 ", "", 0}},
		{"IF GOTO", "10 IF 1 GOTO 30\n20 PRINT \"B\"\n30 PRINT \"C\"\n", result{"C\n", "", 0}},
		{"GO TO", "10 GO TO 30\n20 PRINT \"B\"\n30 PRINT \"C\"\n", result{"C\n", "", 0}},
		{"missing line", "10 GOTO 99\n", result{"", "?UNDEF'D STATEMENT  ERROR IN 10\n", 1}},
		{"GO without TO", "GO 10\n", result{"", "?SYNTAX  ERROR\n", 1}},
		{"GO in a name", "GOLD=1\n", result{"", "?SYNTAX  ERROR\n", 1}},
	}
	for _, c := range cases {
		check(t, c.name, runMain(t, c.input), c.want)
	}
}

// TestLoops checks FOR … NEXT end to end.
//
// @spec INTERP-068, INTERP-074
func TestLoops(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  result
	}{
		{"program", "10 FOR I=1 TO 3\n20 PRINT I;\n30 NEXT\n", result{" 1  2  3 ", "", 0}},
		{"direct", "FOR I=1 TO 3:PRINT I;:NEXT:PRINT\n", result{" 1  2  3 \n", "", 0}},
		{"NEXT without FOR", "10 NEXT\n", result{"", "?NEXT WITHOUT FOR  ERROR IN 10\n", 1}},
		{"integer counter", "FOR I%=1 TO 3\n", result{"", "?SYNTAX  ERROR\n", 1}},
		{"FOR in a name", "FORM=1\n", result{"", "?SYNTAX  ERROR\n", 1}},
	}
	for _, c := range cases {
		check(t, c.name, runMain(t, c.input), c.want)
	}
}

// TestSubroutines checks GOSUB and RETURN end to end.
//
// @spec INTERP-077, INTERP-079
func TestSubroutines(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  result
	}{
		{"program", "10 GOSUB 100\n20 PRINT \"BACK\"\n30 END\n100 PRINT \"SUB\"\n110 RETURN\n", result{"SUB\nBACK\n", "", 0}},
		{"direct GOSUB runs the program", "100 PRINT \"SUB\":RETURN\nGOSUB 100\n", result{"SUB\n", "", 0}},
		{"RETURN WITHOUT GOSUB", "10 RETURN\n", result{"", "?RETURN WITHOUT GOSUB  ERROR IN 10\n", 1}},
	}
	for _, c := range cases {
		check(t, c.name, runMain(t, c.input), c.want)
	}
}
