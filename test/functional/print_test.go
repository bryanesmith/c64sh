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

// TestExponentiation checks "^" end to end, as listed in the HLD's Goals.
func TestExponentiation(t *testing.T) {
	cases := []struct {
		input string
		want  result
	}{
		{`PRINT 2^3`, result{" 8 \n", "", 0}},
		{"PRINT 2\u21913", result{" 8 \n", "", 0}},
		{`PRINT 2^3^2`, result{" 64 \n", "", 0}},
		{`PRINT -2^2`, result{"-4 \n", "", 0}},
		{`PRINT (-2)^2`, result{" 4 \n", "", 0}},
		{`PRINT 2*3^2`, result{" 18 \n", "", 0}},
		{`PRINT 2^-1`, result{" .5 \n", "", 0}},
		{`PRINT 2^-1^2`, result{" .5 \n", "", 0}},
		{`PRINT 2^-3*4`, result{" .5 \n", "", 0}},
		{`PRINT 9^.5`, result{" 3 \n", "", 0}},
		{`PRINT 0^0`, result{" 1 \n", "", 0}},
		{`PRINT 0^-1`, result{" 0 \n", "", 0}},
		{`PRINT (-2)^3`, result{"-8 \n", "", 0}},
		{`PRINT -8^(1/3)`, result{"-2 \n", "", 0}},
		{`PRINT (-8)^(1/3)`, result{"", "?ILLEGAL QUANTITY  ERROR\n", 1}},
		{`PRINT 10^39`, result{"", "?OVERFLOW  ERROR\n", 1}},
		{`PRINT "A"^2`, result{"", "?TYPE MISMATCH  ERROR\n", 1}},
		{`PRINT 2^`, result{"", "?SYNTAX  ERROR\n", 1}},
		{`PRINT "2^3"`, result{"2^3\n", "", 0}},
	}
	for _, c := range cases {
		check(t, c.input, runMain(t, c.input+"\n"), c.want)
	}
}

// TestVariables checks variables end to end, as listed in the HLD's Goals.
func TestVariables(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  result
	}{
		{"assign and print", "A=5\nPRINT A\n", result{" 5 \n", "", 0}},
		{"LET", "LET A=5:PRINT A\n", result{" 5 \n", "", 0}},
		{"string", "N$=\"ALICE\":PRINT \"HELLO, \";N$\n", result{"HELLO, ALICE\n", "", 0}},
		{"expression", "A=5:B=A*2+1:PRINT \"B IS\";B\n", result{"B IS 11 \n", "", 0}},
		{"increment", "A=1\nA=A+1\nPRINT A\n", result{" 2 \n", "", 0}},
		{"unset number", "PRINT X\n", result{" 0 \n", "", 0}},
		{"unset string", "PRINT X$;\"|\"\n", result{"|\n", "", 0}},
		{"two characters count", "HEIGHT=10:PRINT HE\n", result{" 10 \n", "", 0}},
		{"OR inside a name", "SCORE=10\n", result{"", "?SYNTAX  ERROR\n", 1}},
		{"spaces inside names", "A B=3:PRINT AB\n", result{" 3 \n", "", 0}},
		{"number and string separate", "A=1:A$=\"X\":PRINT A;A$\n", result{" 1 X\n", "", 0}},
		{"PRINTER prints ER", "ER=4:PRINTER\n", result{" 4 \n", "", 0}},
		{"LETTER assigns TE", "LETTER=9:PRINT TE\n", result{" 9 \n", "", 0}},
		{"keyword in a name", "PREMIUM=1\n", result{"", "?SYNTAX  ERROR\n", 1}},
		{"wrong type", "A=\"HI\"\n", result{"", "?TYPE MISMATCH  ERROR\n", 1}},
		{"wrong type for string", "A$=5\n", result{"", "?TYPE MISMATCH  ERROR\n", 1}},
		{"number then variable", "A=2:PRINT 1A\n", result{" 1  2 \n", "", 0}},
		{"TI not supported", "PRINT TI\n", result{"", "?SYNTAX  ERROR\n", 1}},
		{"arrays not supported", "PRINT A(1)\n", result{"", "?SYNTAX  ERROR\n", 1}},
		{"lowercase", "a=1\n", result{"", "?SYNTAX  ERROR\n", 1}},
	}
	for _, c := range cases {
		check(t, c.name, runMain(t, c.input), c.want)
	}
	check(t, "interactive persistence", runInteractive(t, "A=5\nPRINT A*2\n"),
		result{" 10 \n", banner + "READY.\nREADY.\n\n", 0})
}

// TestIntegerVariables checks integer variables end to end, as listed in
// the HLD's Goals.
func TestIntegerVariables(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  result
	}{
		{"assign and print", "C%=5:PRINT C%\n", result{" 5 \n", "", 0}},
		{"rounds down", "C%=3.7:PRINT C%\n", result{" 3 \n", "", 0}},
		{"rounds down when negative", "C%=-3.7:PRINT C%\n", result{"-4 \n", "", 0}},
		{"three separate variables", "A=1.5:A%=2:A$=\"X\":PRINT A;A%;A$\n", result{" 1.5  2 X\n", "", 0}},
		{"two characters count", "COUNT%=7:PRINT CO%\n", result{" 7 \n", "", 0}},
		{"counting", "C%=0\nC%=C%+1\nC%=C%+1\nPRINT C%\n", result{" 2 \n", "", 0}},
		{"largest", "C%=32767:PRINT C%\n", result{" 32767 \n", "", 0}},
		{"smallest", "C%=-32768:PRINT C%\n", result{"-32768 \n", "", 0}},
		{"too large", "C%=32768\n", result{"", "?ILLEGAL QUANTITY  ERROR\n", 1}},
		{"too small", "C%=-32769\n", result{"", "?ILLEGAL QUANTITY  ERROR\n", 1}},
		{"string", "C%=\"X\"\n", result{"", "?TYPE MISMATCH  ERROR\n", 1}},
		{"TI% is ordinary", "TI%=3:PRINT TI%\n", result{" 3 \n", "", 0}},
	}
	for _, c := range cases {
		check(t, c.name, runMain(t, c.input), c.want)
	}
}

// TestComparisons checks comparisons end to end, as listed in the HLD's
// Goals.
func TestComparisons(t *testing.T) {
	cases := []struct {
		input string
		want  result
	}{
		{`PRINT 1<2`, result{"-1 \n", "", 0}},
		{`PRINT 5=6`, result{" 0 \n", "", 0}},
		{`PRINT 1+1=2`, result{"-1 \n", "", 0}},
		{`PRINT 3<>4;3><4;4<=4;4=<4;5>=6;5=>6`, result{"-1 -1 -1 -1  0  0 \n", "", 0}},
		{`PRINT 1<=>2`, result{"-1 \n", "", 0}},
		{`PRINT 1 < > 2`, result{"-1 \n", "", 0}},
		{`PRINT "APPLE"<"BANANA"`, result{"-1 \n", "", 0}},
		{`PRINT "A"<"AB"`, result{"-1 \n", "", 0}},
		{`PRINT 1<2<3`, result{"-1 \n", "", 0}},
		{`A=5:PRINT A=5`, result{"-1 \n", "", 0}},
		{`B=1:C=1:A=B=C:PRINT A`, result{"-1 \n", "", 0}},
		{`PRINT 1==1`, result{"", "?SYNTAX  ERROR\n", 1}},
		{`PRINT "1"=1`, result{"", "?TYPE MISMATCH  ERROR\n", 1}},
	}
	for _, c := range cases {
		check(t, c.input, runMain(t, c.input+"\n"), c.want)
	}
}

// TestLogic checks AND, OR, and NOT end to end, as listed in the HLD's
// Goals.
func TestLogic(t *testing.T) {
	cases := []struct {
		input string
		want  result
	}{
		{`PRINT 1<2 AND 3<4`, result{"-1 \n", "", 0}},
		{`PRINT 1<2 AND 3>4`, result{" 0 \n", "", 0}},
		{`PRINT 1>2 OR 3<4`, result{"-1 \n", "", 0}},
		{`PRINT NOT 0;NOT -1;NOT 5`, result{"-1  0 -6 \n", "", 0}},
		{`PRINT 12 AND 10;12 OR 10`, result{" 8  14 \n", "", 0}},
		{`PRINT NOT 1=2`, result{"-1 \n", "", 0}},
		{`PRINT 0 OR 1 AND 0`, result{" 0 \n", "", 0}},
		{`PRINT 1+NOT 0+1`, result{"-1 \n", "", 0}},
		{`PRINT 40000 AND 1`, result{"", "?ILLEGAL QUANTITY  ERROR\n", 1}},
		{`PRINT "A" AND 1`, result{"", "?TYPE MISMATCH  ERROR\n", 1}},
		{`SCORE=1`, result{"", "?SYNTAX  ERROR\n", 1}},
	}
	for _, c := range cases {
		check(t, c.input, runMain(t, c.input+"\n"), c.want)
	}
}

// TestIf checks IF ... THEN end to end, as listed in the HLD's Goals.
func TestIf(t *testing.T) {
	cases := []struct {
		input string
		want  result
	}{
		{`A=5:IF A>3 THEN PRINT "BIG"`, result{"BIG\n", "", 0}},
		{`A=1:IF A>3 THEN PRINT "BIG"`, result{"", "", 0}},
		{`IF 1 THEN PRINT "A":PRINT "B"`, result{"A\nB\n", "", 0}},
		{`IF 0 THEN PRINT "A":PRINT "B"`, result{"", "", 0}},
		{`PRINT "X";:IF 0 THEN PRINT "A"`, result{"X", "", 0}},
		{`IF 0 THEN PRINT "A"@`, result{"", "", 0}},
		{`IF 1 THEN PRINT "A"@`, result{"A\n", "?SYNTAX  ERROR\n", 1}},
		{`IF "X" THEN PRINT "NON-EMPTY"`, result{"NON-EMPTY\n", "", 0}},
		{`IF "" THEN PRINT "EMPTY"`, result{"", "", 0}},
		{`IF 1<2 AND 3<4 THEN PRINT "BOTH"`, result{"BOTH\n", "", 0}},
		{`IF 1 THEN IF 0 THEN PRINT "NO"`, result{"", "", 0}},
		{`IF 1 THEN A=7:PRINT A`, result{" 7 \n", "", 0}},
		{`IF 1 PRINT "X"`, result{"", "?SYNTAX  ERROR\n", 1}},
		{`IF 1 THEN 100`, result{"", "?SYNTAX  ERROR\n", 1}},
	}
	for _, c := range cases {
		check(t, c.input, runMain(t, c.input+"\n"), c.want)
	}
}
