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
		{`PRINT "sometext","BAR"`, result{"sometext\tBAR\n", "", 0}},
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
		{`PRINT 1+2`, result{"", "?SYNTAX  ERROR\n", 1}},
		{`PRINT "HELLO"@`, result{"HELLO\n", "?SYNTAX  ERROR\n", 1}},
	}
	for _, c := range cases {
		check(t, c.input, runMain(t, c.input+"\n"), c.want)
	}
}
