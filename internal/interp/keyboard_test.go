package interp

import (
	"errors"
	"io"
	"testing"

	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// fakeConsole supplies lines and keys from lists. A key "" means no key
// is waiting; when a list runs out, the console is at the end of input.
type fakeConsole struct {
	lines  []string
	keys   []string
	echoed bool
	err    error // returned instead, if set
}

func (c *fakeConsole) ReadLine(stop func() bool) (string, bool, error) {
	if c.err != nil {
		return "", false, c.err
	}
	if len(c.lines) == 0 {
		return "", false, io.EOF
	}
	l := c.lines[0]
	c.lines = c.lines[1:]
	return l, c.echoed, nil
}

func (c *fakeConsole) ReadKey(stop func() bool) (string, error) {
	if c.err != nil {
		return "", c.err
	}
	if len(c.keys) == 0 {
		return "", io.EOF
	}
	k := c.keys[0]
	c.keys = c.keys[1:]
	return k, nil
}

type inputCase struct {
	name    string
	console *fakeConsole
	lines   []string
	want    string
	wantErr error
}

func runInputCases(t *testing.T, cases []inputCase) {
	t.Helper()
	for _, c := range cases {
		rec := &recorder{}
		in := New(rec)
		if c.console != nil {
			in.SetConsole(c.console)
		}
		var err error
		for _, l := range c.lines {
			err = enter(in, l)
		}
		if got := rec.String(); got != c.want {
			t.Errorf("%s: output %q, want %q", c.name, got, c.want)
		}
		if errors.Is(c.wantErr, ErrEndOfInput) {
			if !errors.Is(err, ErrEndOfInput) {
				t.Errorf("%s: error %v, want ErrEndOfInput", c.name, err)
			}
		} else if !sameErr(err, c.wantErr) {
			t.Errorf("%s: error %#v, want %#v", c.name, err, c.wantErr)
		}
	}
}

func typed(l ...string) *fakeConsole   { return &fakeConsole{lines: l} }
func pressed(k ...string) *fakeConsole { return &fakeConsole{keys: k} }

// @spec INTERP-081
func TestInputGetIllegalDirect(t *testing.T) {
	runInputCases(t, []inputCase{
		{"INPUT with prompt", typed("5"), lines(`INPUT "HI";A`), "HI", &basicerr.Error{Kind: basicerr.IllegalDirect}},
		{"INPUT", typed("5"), lines(`INPUT A`), "", &basicerr.Error{Kind: basicerr.IllegalDirect}},
		{"GET", pressed("X"), lines(`GET A$`), "", &basicerr.Error{Kind: basicerr.IllegalDirect}},
	})
}

// @spec INTERP-082
func TestInputPromptAndEcho(t *testing.T) {
	runInputCases(t, []inputCase{
		{"no prompt", typed("5"), lines(`10 INPUT A:PRINT A`, "RUN"), "? 5\n 5 \n", nil},
		{"prompt", typed("ALICE"), lines(`10 INPUT "NAME";N$:PRINT "HI ";N$`, "RUN"), "NAME? ALICE\nHI ALICE\n", nil},
		{"echoed by console", &fakeConsole{lines: []string{"ALICE"}, echoed: true}, lines(`10 INPUT "NAME";N$:PRINT N$`, "RUN"), "NAME? \nALICE\n", nil},
		{"trailing spaces removed", typed("ALICE   "), lines(`10 INPUT N$:PRINT N$;"."`, "RUN"), "? ALICE\nALICE.\n", nil},
		{"after unfinished output", typed("1"), lines(`10 PRINT "X";:INPUT A`, "RUN"), "X? 1\n", nil},
	})
}

// @spec INTERP-083
func TestInputEmptyLine(t *testing.T) {
	runInputCases(t, []inputCase{
		{"keeps variables", typed(""), lines(`10 A=7:B$="X":INPUT A,B$:PRINT A;B$`, "RUN"), "? \n 7 X\n", nil},
		{"only spaces", typed("   "), lines(`10 A=7:INPUT A:PRINT A`, "RUN"), "? \n 7 \n", nil},
	})
}

// @spec INTERP-084
func TestInputValues(t *testing.T) {
	runInputCases(t, []inputCase{
		{"two values", typed("1,X"), lines(`10 INPUT A,B$:PRINT A;B$`, "RUN"), "? 1,X\n 1 X\n", nil},
		{"quoted string", typed(`"A,B"`), lines(`10 INPUT A$:PRINT A$`, "RUN"), "? \"A,B\"\nA,B\n", nil},
		{"unclosed quote", typed(`"A,B`), lines(`10 INPUT A$:PRINT A$`, "RUN"), "? \"A,B\nA,B\n", nil},
		{"spaces inside kept", typed("  HELLO WORLD"), lines(`10 INPUT A$:PRINT A$;"."`, "RUN"), "?   HELLO WORLD\nHELLO WORLD.\n", nil},
		{"space before comma kept", typed("A ,B"), lines(`10 INPUT A$,B$:PRINT A$;".";B$`, "RUN"), "? A ,B\nA .B\n", nil},
		{"number with spaces", typed("1 2"), lines(`10 INPUT A:PRINT A`, "RUN"), "? 1 2\n 12 \n", nil},
		{"negative fraction", typed("-3.5"), lines(`10 INPUT A:PRINT A`, "RUN"), "? -3.5\n-3.5 \n", nil},
		{"exponent", typed("1E3"), lines(`10 INPUT A:PRINT A`, "RUN"), "? 1E3\n 1000 \n", nil},
		{"no digits", typed("."), lines(`10 INPUT A:PRINT A`, "RUN"), "? .\n 0 \n", nil},
		{"integer variable", typed("3.7"), lines(`10 INPUT A%:PRINT A%`, "RUN"), "? 3.7\n 3 \n", nil},
		{"integer out of range", typed("40000"), lines(`10 INPUT A%`, "RUN"), "? 40000\n", errIn(basicerr.IllegalQuantity, 10)},
		{"overflow", typed("1E39"), lines(`10 INPUT A`, "RUN"), "? 1E39\n", errIn(basicerr.Overflow, 10)},
	})
}

// @spec INTERP-085
func TestInputRedoFromStart(t *testing.T) {
	runInputCases(t, []inputCase{
		{"not a number", typed("ABC", "5"), lines(`10 INPUT "N";A:PRINT A`, "RUN"), "N? ABC\n?REDO FROM START\nN? 5\n 5 \n", nil},
		{"second value bad", typed("1,X", "3,4"), lines(`10 INPUT A,B:PRINT A;B`, "RUN"), "? 1,X\n?REDO FROM START\n? 3,4\n 3  4 \n", nil},
		{"keeps assigned values", typed("5,X", ""), lines(`10 INPUT A,B:PRINT A;B`, "RUN"), "? 5,X\n?REDO FROM START\n? \n 5  0 \n", nil},
		{"junk after quotes", typed(`"AB"C`, "D"), lines(`10 INPUT A$:PRINT A$`, "RUN"), "? \"AB\"C\n?REDO FROM START\n? D\nD\n", nil},
		{"second point", typed("1.2.3", "1"), lines(`10 INPUT A:PRINT A`, "RUN"), "? 1.2.3\n?REDO FROM START\n? 1\n 1 \n", nil},
	})
}

// @spec INTERP-086
func TestInputAsksForMore(t *testing.T) {
	runInputCases(t, []inputCase{
		{"more values", typed("1", "2,3"), lines(`10 INPUT A,B,C:PRINT A;B;C`, "RUN"), "? 1\n?? 2,3\n 1  2  3 \n", nil},
		{"colon ends the values", typed("X:Y", "Z"), lines(`10 INPUT A$,B$:PRINT A$;B$`, "RUN"), "? X:Y\n?? Z\nXZ\n", nil},
		{"empty answer", typed("1", ""), lines(`10 B=9:INPUT A,B:PRINT A;B`, "RUN"), "? 1\n?? \n 1  0 \n", nil},
	})
}

// @spec INTERP-087
func TestInputExtraIgnored(t *testing.T) {
	runInputCases(t, []inputCase{
		{"extra value", typed("1,2"), lines(`10 INPUT A:PRINT A`, "RUN"), "? 1,2\n?EXTRA IGNORED\n 1 \n", nil},
		{"colon", typed("X:Y"), lines(`10 INPUT A$:PRINT A$`, "RUN"), "? X:Y\n?EXTRA IGNORED\nX\n", nil},
		{"trailing comma", typed("1,"), lines(`10 INPUT A`, "RUN"), "? 1,\n?EXTRA IGNORED\n", nil},
	})
}

// @spec INTERP-088
func TestGetString(t *testing.T) {
	runInputCases(t, []inputCase{
		{"key", pressed("Q"), lines(`10 GET K$:PRINT "[";K$;"]"`, "RUN"), "[Q]\n", nil},
		{"no key", pressed(""), lines(`10 GET K$:PRINT "[";K$;"]"`, "RUN"), "[]\n", nil},
		{"two keys", pressed("X", "Y"), lines(`10 GET A$,B$:PRINT A$;B$`, "RUN"), "XY\n", nil},
		{"wait loop", pressed("", "", "Z"), lines(`10 GET K$:IF K$="" THEN 10`, `20 PRINT K$`, "RUN"), "Z\n", nil},
	})
}

// @spec INTERP-089
func TestGetNumber(t *testing.T) {
	runInputCases(t, []inputCase{
		{"digit", pressed("7"), lines(`10 GET A:PRINT A`, "RUN"), " 7 \n", nil},
		{"no key", pressed(""), lines(`10 A=5:GET A:PRINT A`, "RUN"), " 0 \n", nil},
		{"space", pressed(" "), lines(`10 GET A:PRINT A`, "RUN"), " 0 \n", nil},
		{"sign", pressed("-"), lines(`10 GET A:PRINT A`, "RUN"), " 0 \n", nil},
		{"letter", pressed("X"), lines(`10 GET A`, "RUN"), "", &basicerr.Error{Kind: basicerr.Syntax}},
	})
}

// @spec INTERP-090
func TestInputErrors(t *testing.T) {
	runInputCases(t, []inputCase{
		{"no console", nil, lines(`10 INPUT A`, "RUN"), "? ", ErrEndOfInput},
		{"input ends", typed(), lines(`10 INPUT A`, "RUN"), "? ", ErrEndOfInput},
		{"input ends during ??", typed("1"), lines(`10 INPUT A,B`, "RUN"), "? 1\n?? ", ErrEndOfInput},
		{"GET at end", pressed(), lines(`10 GET A$`, "RUN"), "", ErrEndOfInput},
		{"interrupted", &fakeConsole{err: ErrInterrupted}, lines(`10 INPUT A`, "RUN"), "? ", errIn(basicerr.Break, 10)},
		{"interrupted GET", &fakeConsole{err: ErrInterrupted}, lines(`10 GET A$`, "RUN"), "", errIn(basicerr.Break, 10)},
	})
}
