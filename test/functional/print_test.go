package functional_test

import "testing"

// TestPrintForms checks the PRINT forms listed in the HLD's Goals, end to
// end through the shell in script mode.
func TestPrintForms(t *testing.T) {
	cases := []struct {
		input string
		want  result
	}{
		{`PRINT "sometext"`, result{"sometext\n", "", 0}},
		{`PRINT"sometext"`, result{"sometext\n", "", 0}},
		{`PRINT "sometext";"BAR"`, result{"sometextBAR\n", "", 0}},
		{`PRINT "sometext"+"BAR"`, result{"sometextBAR\n", "", 0}},
		{`PRINT "sometext","BAR"`, result{"sometext  BAR\n", "", 0}},
		{`PRINT "foo":PRINT "bar"`, result{"foo\nbar\n", "", 0}},
		{`PRINT "sometext" "BAR"`, result{"sometextBAR\n", "", 0}},
		{`PRINT`, result{"\n", "", 0}},
		{`?"HI"`, result{"HI\n", "", 0}},
		{`PRINT "HI`, result{"HI\n", "", 0}},
		{`PRINT "A";:PRINT "B"`, result{"AB\n", "", 0}},
	}
	for _, c := range cases {
		check(t, c.input, runMain(t, c.input+"\n"), c.want)
	}
}

// TestUnsupportedInputIsSyntaxError checks that input c64sh does not accept
// fails the way a C64 reports it.
func TestUnsupportedInputIsSyntaxError(t *testing.T) {
	cases := []struct {
		input string
		want  result
	}{
		{`print "hi"`, result{"", "?SYNTAX  ERROR\n", 1}},
		{`10 PRINT "FOO"`, result{"", "?SYNTAX  ERROR\n", 1}},
		{`PRINT CHR$(34)`, result{"", "?SYNTAX  ERROR\n", 1}},
		{`PRINT (1+2`, result{"", "?SYNTAX  ERROR\n", 1}},
		{`PRINT "HELLO"@`, result{"HELLO\n", "?SYNTAX  ERROR\n", 1}},
	}
	for _, c := range cases {
		check(t, c.input, runMain(t, c.input+"\n"), c.want)
	}
}

// TestPrintZones checks that "," moves to the next 10-column print zone,
// across statements and lines, as on a C64.
func TestPrintZones(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  result
	}{
		{"table", "PRINT \"NAME\",\"SCORE\"\nPRINT \"ALICE\",\"12\"\n", result{"NAME      SCORE\nALICE     12\n", "", 0}},
		{"zone start moves a full zone", "PRINT \"0123456789\",\"X\"\n", result{"0123456789          X\n", "", 0}},
		{"column carries across lines", "PRINT \"AB\";\nPRINT ,\"X\"\n", result{"AB        X\n", "", 0}},
		{"column carries across statements", "PRINT \"AB\";:PRINT ,\"X\"\n", result{"AB        X\n", "", 0}},
		{"error resets the column", "PRINT \"AB\";:@\n", result{"AB\n", "?SYNTAX  ERROR\n", 1}},
	}
	for _, c := range cases {
		check(t, c.name, runMain(t, c.input), c.want)
	}
	check(t, "READY resets the column", runInteractive(t, "PRINT \"AB\";\nPRINT ,\"X\"\n"),
		result{"AB\n" + "          X\n", banner + "READY.\nREADY.\n\n", 0})
}

// TestNumbers checks number literals and number printing end to end, as
// listed in the HLD's Goals.
func TestNumbers(t *testing.T) {
	cases := []struct {
		input string
		want  result
	}{
		{`PRINT 5`, result{" 5 \n", "", 0}},
		{`? "5*9=";45`, result{"5*9= 45 \n", "", 0}},
		{`? 2,3,4,5,6`, result{" 2 " + "       " + " 3 " + "       " + " 4 " + "       " + " 5 " + "       " + " 6 \n", "", 0}},
		{`PRINT 3.14;.5;1E3;1.5E-3`, result{" 3.14  .5  1000  1.5E-03 \n", "", 0}},
		{`PRINT 1/3`, result{" .333333333 \n", "", 0}},
		{`PRINT 1;2`, result{" 1  2 \n", "", 0}},
		{`PRINT 1 2`, result{" 12 \n", "", 0}},
		{`PRINT "A"1`, result{"A 1 \n", "", 0}},
		{`PRINT 1+1`, result{" 2 \n", "", 0}},
		{`PRINT 1.5+2.25+3`, result{" 6.75 \n", "", 0}},
		{`PRINT .`, result{" 0 \n", "", 0}},
		{`PRINT 1.2.3`, result{" 1.2  .3 \n", "", 0}},
		{`PRINT 1E9;999999999;.01;.001`, result{" 1E+09  999999999  .01  1E-03 \n", "", 0}},
		{`PRINT "A"+1`, result{"", "?TYPE MISMATCH  ERROR\n", 1}},
		{`PRINT "SUM:";1+"A"`, result{"SUM:\n", "?TYPE MISMATCH  ERROR\n", 1}},
		{`PRINT 1E39`, result{"", "?OVERFLOW  ERROR\n", 1}},
		{`PRINT 1E39+"A"`, result{"", "?OVERFLOW  ERROR\n", 1}},
		{`PRINT -5`, result{"-5 \n", "", 0}},
		{`10 PRINT "HI"`, result{"", "?SYNTAX  ERROR\n", 1}},
	}
	for _, c := range cases {
		check(t, c.input, runMain(t, c.input+"\n"), c.want)
	}
}

// TestArithmetic checks arithmetic end to end, as listed in the HLD's Goals.
func TestArithmetic(t *testing.T) {
	cases := []struct {
		input string
		want  result
	}{
		{`? "5*9=";5*9`, result{"5*9= 45 \n", "", 0}},
		{`PRINT 2+3*4`, result{" 14 \n", "", 0}},
		{`PRINT (2+3)*4`, result{" 20 \n", "", 0}},
		{`PRINT 10-4-3`, result{" 3 \n", "", 0}},
		{`PRINT 100/10/2`, result{" 5 \n", "", 0}},
		{`PRINT 7/2`, result{" 3.5 \n", "", 0}},
		{`PRINT -2*3`, result{"-6 \n", "", 0}},
		{`PRINT 2*-3`, result{"-6 \n", "", 0}},
		{`PRINT --5`, result{" 5 \n", "", 0}},
		{`PRINT 5--5`, result{" 10 \n", "", 0}},
		{`PRINT +5`, result{" 5 \n", "", 0}},
		{`PRINT +"A"`, result{"A\n", "", 0}},
		{`PRINT ((1+2)*(3+4))`, result{" 21 \n", "", 0}},
		{`PRINT 1 -1`, result{" 0 \n", "", 0}},
		{`PRINT 2(3)`, result{" 2  3 \n", "", 0}},
		{`PRINT 1/3*3`, result{" 1 \n", "", 0}},
		{`PRINT -0`, result{" 0 \n", "", 0}},
		{`PRINT 1/0`, result{"", "?DIVISION BY ZERO  ERROR\n", 1}},
		{`PRINT "RESULT:";1/0`, result{"RESULT:\n", "?DIVISION BY ZERO  ERROR\n", 1}},
		{`PRINT "A"-1`, result{"", "?TYPE MISMATCH  ERROR\n", 1}},
		{`PRINT -"A"`, result{"", "?TYPE MISMATCH  ERROR\n", 1}},
		{`PRINT "A"/0`, result{"", "?TYPE MISMATCH  ERROR\n", 1}},
		{`PRINT 1E38*10`, result{"", "?OVERFLOW  ERROR\n", 1}},
		{`PRINT (1`, result{"", "?SYNTAX  ERROR\n", 1}},
		{`PRINT 1)`, result{" 1 \n", "?SYNTAX  ERROR\n", 1}},
		{`PRINT 2*`, result{"", "?SYNTAX  ERROR\n", 1}},
	}
	for _, c := range cases {
		check(t, c.input, runMain(t, c.input+"\n"), c.want)
	}
}
