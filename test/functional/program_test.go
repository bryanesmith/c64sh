package functional_test

import (
	"os"
	"testing"
)

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

// TestKeyboardInput checks INPUT and GET reading stdin end to end.
//
// @spec SHELL-KEY-001, SHELL-KEY-006
func TestKeyboardInput(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  result
	}{
		{"piped script answers itself", "10 INPUT \"NAME\";N$\n20 PRINT \"HI \";N$\nRUN\nALICE\nPRINT \"DONE\"\n", result{"NAME? ALICE\nHI ALICE\nDONE\n", "", 0}},
		{"redo from start", "10 INPUT A:PRINT A*2\nRUN\nX\n21\n", result{"? X\n?REDO FROM START\n? 21\n 42 \n", "", 0}},
		{"GET reads characters", "10 GET A$,B$:PRINT \"[\";A$;B$;\"]\"\nRUN\nX\n", result{"[X\r]\n", "", 0}},
		{"end of input", "10 INPUT A\n", result{"? \n", "c64sh: stdin: end of input\n", 1}},
		{"direct", "INPUT \"HI\";A\n", result{"HI\n", "?ILLEGAL DIRECT  ERROR\n", 1}},
	}
	for _, c := range cases {
		check(t, c.name, runMain(t, c.input), c.want)
	}
	path := writeFile(t, "double.bas", "10 INPUT A:PRINT A*2\n")
	check(t, "file reads stdin", runMain(t, "42\n", path), result{"? 42\n 84 \n", "", 0})
	check(t, "interactive", runInteractive(t, "10 INPUT A$\n20 PRINT A$\nRUN\nHELLO\n"),
		result{"? HELLO\nHELLO\n", banner + "READY.\n\n", 0})
}

// TestUserFunctions checks DEF FN end to end.
//
// @spec INTERP-091, INTERP-092
func TestUserFunctions(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  result
	}{
		{"define and call", "10 DEF FN SQ(X)=X*X\n20 PRINT FN SQ(4)\n", result{" 16 \n", "", 0}},
		{"direct DEF", "DEF FN A(X)=X\n", result{"", "?ILLEGAL DIRECT  ERROR\n", 1}},
		{"undefined", "10 PRINT FN Q(1)\n", result{"", "?UNDEF'D FUNCTION  ERROR IN 10\n", 1}},
	}
	for _, c := range cases {
		check(t, c.name, runMain(t, c.input), c.want)
	}
}

// TestProgramFiles checks SAVE, LOAD, and VERIFY end to end, in a
// temporary current directory.
//
// @spec SHELL-FILE-001, SHELL-FILE-002, SHELL-FILE-003
func TestProgramFiles(t *testing.T) {
	t.Chdir(t.TempDir())
	check(t, "save", runMain(t, "10 PRINT \"HELLO\"\nSAVE \"HELLO\",8\nNEW\n"), result{"", "", 0})
	data, err := os.ReadFile("HELLO.bas")
	if err != nil || string(data) != "#!/usr/bin/env c64sh\n10 PRINT \"HELLO\"\n" {
		t.Errorf("HELLO.bas = %q, %v", data, err)
	}
	check(t, "load and run", runMain(t, "LOAD \"HELLO\"\nRUN\n"), result{"HELLO\n", "", 0})
	check(t, "saved program is a script", runMain(t, "", "HELLO.bas"), result{"HELLO\n", "", 0})
	check(t, "disk refuses to replace", runMain(t, "10 REM\nSAVE \"HELLO\",8\nPRINT \"NOT REACHED\"\n"),
		result{"", "c64sh: HELLO.bas: file exists (use SAVE \"@0:HELLO\" to replace it)\n", 1})
	check(t, "replace with @0:", runMain(t, "10 PRINT \"NEW\"\nSAVE \"@0:HELLO\",8\nLOAD \"HELLO\"\nRUN\n"), result{"NEW\n", "", 0})
	check(t, "not found", runMain(t, "LOAD \"NONE\",8\n"), result{"", "?FILE NOT FOUND  ERROR\n", 1})
	check(t, "interactive messages", runInteractive(t, "10 REM\nSAVE \"TWO\",8\nLOAD \"TWO\",8\nSAVE \"TWO\",8\n"),
		result{"", banner + "SAVING TWO\nREADY.\nSEARCHING FOR TWO\nLOADING\nREADY.\nSAVING TWO\nc64sh: TWO.bas: file exists (use SAVE \"@0:TWO\" to replace it)\nREADY.\n\n", 0})
}

// TestDataFiles checks OPEN, PRINT#, INPUT#, CMD, and CLOSE end to end,
// in a temporary current directory.
//
// @spec SHELL-FILE-003, SHELL-FILE-004
func TestDataFiles(t *testing.T) {
	t.Chdir(t.TempDir())
	check(t, "write", runMain(t, "OPEN 2,8,2,\"SCORES,S,W\"\nPRINT#2,\"ALICE\";\",\";12\nCLOSE 2\n"), result{"", "", 0})
	if data, _ := os.ReadFile("SCORES"); string(data) != "ALICE, 12 \n" {
		t.Errorf("SCORES = %q", data)
	}
	check(t, "read", runMain(t, "10 OPEN 2,8,2,\"SCORES\":INPUT#2,N$,S:CLOSE 2:PRINT N$;S\n"), result{"ALICE 12 \n", "", 0})
	check(t, "left open, written at the end", runMain(t, "OPEN 3,8,3,\"OPEN,S,W\"\nPRINT#3,\"KEPT\"\n"), result{"", "", 0})
	if data, _ := os.ReadFile("OPEN"); string(data) != "KEPT\n" {
		t.Errorf("OPEN = %q", data)
	}
	check(t, "disk refuses to replace", runMain(t, "OPEN 2,8,2,\"SCORES,S,W\"\n"),
		result{"", "c64sh: SCORES: file exists (use \"@0:SCORES,S,W\" to replace it)\n", 1})
	check(t, "CMD to the printer", runMain(t, "10 REM HI\nOPEN 4,4:CMD 4:LIST\n"), result{"\n\n10 REM HI\n", "", 0})
	check(t, "missing file", runMain(t, "OPEN 2,8,2,\"NONE\"\n"), result{"", "?FILE NOT FOUND  ERROR\n", 1})
}
