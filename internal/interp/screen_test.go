package interp

import "testing"

// screenOutput runs a line with the screen set as given.
func screenOutput(t *testing.T, terminal, color bool, l string) (string, int) {
	t.Helper()
	rec := &recorder{}
	in := New(rec)
	in.SetScreen(terminal, color)
	if err := enter(in, l); err != nil {
		t.Fatal(err)
	}
	return rec.String(), in.Column()
}

// @spec INTERP-146
func TestReturnAndCursorRight(t *testing.T) {
	for _, terminal := range []bool{false, true} {
		out, col := screenOutput(t, terminal, true, `PRINT "A";CHR$(13);"B";CHR$(29);"C";`)
		want := "A\nB C"
		if out != want || col != 3 {
			t.Errorf("terminal %v: %q, column %d; want %q, 3", terminal, out, col, want)
		}
	}
	if out, _ := screenOutput(t, false, true, `PRINT "A";CHR$(141);"B"`); out != "A\nB\n" {
		t.Errorf("CHR$(141): %q", out)
	}
	// Every Return turns reverse off, including the one ending a PRINT.
	for _, tc := range []struct{ src, want string }{
		{`PRINT CHR$(18);"A":PRINT "B"`, "\x1b[7mA\x1b[27m\nB\n"},
		{`PRINT CHR$(18);"A";CHR$(13);"B";`, "\x1b[7mA\x1b[27m\nB"},
		{`PRINT CHR$(18);"A";CHR$(141);CHR$(13);`, "\x1b[7mA\x1b[27m\n\n"},
	} {
		if out, _ := screenOutput(t, true, true, tc.src); out != tc.want {
			t.Errorf("%s: %q; want %q", tc.src, out, tc.want)
		}
	}
}

// @spec INTERP-147
func TestTerminalCursorCodes(t *testing.T) {
	cases := []struct {
		line string
		want string
		col  int
	}{
		{`PRINT "AB";CHR$(157);`, "AB\x1b[D", 1},
		{`PRINT CHR$(157);`, "\x1b[D", 0},
		{`PRINT "A";CHR$(17);CHR$(145);`, "A\x1b[B\x1b[A", 1},
		{`PRINT "AB";CHR$(19);`, "AB\x1b[H", 0},
		{`PRINT "AB";CHR$(147);`, "AB\x1b[2J\x1b[H", 0},
		{`PRINT CHR$(18);"R";CHR$(146);`, "\x1b[7mR\x1b[27m", 1},
	}
	for _, c := range cases {
		if out, col := screenOutput(t, true, true, c.line); out != c.want || col != c.col {
			t.Errorf("%s: %q, column %d; want %q, %d", c.line, out, col, c.want, c.col)
		}
	}
}

// @spec INTERP-148
func TestTerminalColors(t *testing.T) {
	cases := map[string]string{
		`PRINT CHR$(28);"X";`: "\x1b[38;2;104;55;43mX",
		`PRINT CHR$(5);`:      "\x1b[38;2;255;255;255m",
		`PRINT CHR$(144);`:    "\x1b[38;2;0;0;0m",
		`PRINT CHR$(158);`:    "\x1b[38;2;184;199;111m",
		`PRINT CHR$(154);`:    "\x1b[38;2;108;94;181m",
		`PRINT CHR$(155);`:    "\x1b[38;2;149;149;149m",
	}
	for line, want := range cases {
		if out, _ := screenOutput(t, true, true, line); out != want {
			t.Errorf("%s: %q, want %q", line, out, want)
		}
	}
	if out, col := screenOutput(t, true, true, `PRINT "A";CHR$(30);"B";`); col != 2 || out != "A\x1b[38;2;88;141;67mB" {
		t.Errorf("color in a line: %q, column %d", out, col)
	}
}

// @spec INTERP-149
func TestScreenCodesLeftOut(t *testing.T) {
	if out, col := screenOutput(t, false, true, `PRINT CHR$(147);CHR$(28);"A";CHR$(18);"B";CHR$(17);CHR$(157);`); out != "AB" || col != 2 {
		t.Errorf("not a terminal: %q, column %d; want \"AB\", 2", out, col)
	}
	if out, _ := screenOutput(t, true, false, `PRINT CHR$(28);CHR$(18);"A"`); out != "\x1b[7mA\x1b[27m\n" {
		t.Errorf("colors off: %q, want reverse kept and color left out", out)
	}
	if out, _ := screenOutput(t, false, true, `PRINT "A"+CHR$(9)+"B"`); out != "A\tB\n" {
		t.Errorf("other characters: %q", out)
	}
}

// @spec INTERP-150
func TestScreenState(t *testing.T) {
	cases := []struct {
		terminal, color bool
		src, want       string
	}{
		{true, true, `PRINT "A"`, ""},
		{true, true, `PRINT CHR$(28);"A"`, "\x1b[38;2;104;55;43m"},
		{true, true, `PRINT CHR$(28);CHR$(158);`, "\x1b[38;2;184;199;111m"},
		{true, true, `PRINT CHR$(28);CHR$(18);`, "\x1b[38;2;104;55;43m\x1b[7m"},
		{true, true, `PRINT CHR$(18);"A"`, ""}, // the Return turned reverse off
		{true, false, `PRINT CHR$(28);CHR$(18);`, "\x1b[7m"},
		{false, true, `PRINT CHR$(28);CHR$(18);`, ""},
	}
	for _, c := range cases {
		in := New(&recorder{})
		in.SetScreen(c.terminal, c.color)
		if err := enter(in, c.src); err != nil {
			t.Fatal(err)
		}
		if got := in.ScreenState(); got != c.want {
			t.Errorf("terminal %v, color %v, %s: %q; want %q", c.terminal, c.color, c.src, got, c.want)
		}
	}
}
