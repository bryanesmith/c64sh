package interp

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// AST helpers.
func str(v string) *ast.StringLit                     { return &ast.StringLit{Value: v} }
func cat(l, r ast.Expr) *ast.Concat                   { return &ast.Concat{Left: l, Right: r} }
func item(e ast.Expr) *ast.ExprItem                   { return &ast.ExprItem{Expr: e} }
func printStmt(items ...ast.PrintItem) *ast.PrintStmt { return &ast.PrintStmt{Items: items} }
func line(stmts ...ast.Stmt) *ast.Line                { return &ast.Line{Statements: stmts} }

var (
	semi  = &ast.Semicolon{}
	comma = &ast.Comma{}
)

func spaces(n int) string { return strings.Repeat(" ", n) }

func syntaxErr() error { return &basicerr.Error{Kind: basicerr.Syntax} }

func isKind(err error, k basicerr.Kind) bool {
	var be *basicerr.Error
	return errors.As(err, &be) && be.Kind == k
}

// recorder records each call to Write.
type recorder struct{ writes []string }

func (r *recorder) Write(p []byte) (int, error) {
	r.writes = append(r.writes, string(p))
	return len(p), nil
}

func (r *recorder) String() string { return strings.Join(r.writes, "") }

// exec runs l and returns the output and error.
func exec(l *ast.Line) (*recorder, error) {
	rec := &recorder{}
	err := New(rec).Exec(l)
	return rec, err
}

type printCase struct {
	name string
	line *ast.Line
	want string
}

func runPrintCases(t *testing.T, cases []printCase) {
	t.Helper()
	for _, c := range cases {
		rec, err := exec(c.line)
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.name, err)
		}
		if got := rec.String(); got != c.want {
			t.Errorf("%s: output %q, want %q", c.name, got, c.want)
		}
	}
}

// @spec INTERP-001
func TestStatementsRunInOrderAndStopAtFailure(t *testing.T) {
	runPrintCases(t, []printCase{
		{"two statements", line(printStmt(item(str("A"))), printStmt(item(str("B")))), "A\nB\n"},
	})

	long := strings.Repeat("x", 200)
	rec, err := exec(line(
		printStmt(item(str("A"))),
		printStmt(item(cat(str(long), str(long)))),
		printStmt(item(str("B"))),
	))
	if !isKind(err, basicerr.StringTooLong) {
		t.Errorf("error = %v, want STRING TOO LONG", err)
	}
	if got := rec.String(); got != "A\n" {
		t.Errorf("output %q, want %q (statement after the failure must not run)", got, "A\n")
	}
}

// @spec INTERP-002
func TestOutputGoesToGivenWriter(t *testing.T) {
	runPrintCases(t, []printCase{
		{"hello", line(printStmt(item(str("HELLO")))), "HELLO\n"},
	})
}

// @spec INTERP-003
func TestUnhandledNodeTypesPanic(t *testing.T) {
	cases := map[string]*ast.Line{
		"statement":  line(nil),
		"print item": line(&ast.PrintStmt{Items: []ast.PrintItem{nil}}),
		"expression": line(printStmt(&ast.ExprItem{Expr: nil})),
	}
	for name, l := range cases {
		func() {
			defer func() {
				r := recover()
				if r == nil {
					t.Errorf("%s: nil node did not panic", name)
					return
				}
				if msg := fmt.Sprint(r); !strings.Contains(msg, "<nil>") {
					t.Errorf("%s: panic message %q does not name the node type", name, msg)
				}
			}()
			exec(l)
		}()
	}
}

// @spec INTERP-014
func TestRemDoesNothing(t *testing.T) {
	rem := &ast.RemStmt{Text: " HELLO"}
	runPrintCases(t, []printCase{
		{"comment alone", line(rem), ""},
		{"comment after a print", line(printStmt(item(str("A")), semi), rem), "A"},
		{"statement after a comment runs", line(rem, printStmt(item(str("B")))), "B\n"},
	})
	rec, _ := exec(line(rem))
	if len(rec.writes) != 0 {
		t.Errorf("REM made %d writes, want 0", len(rec.writes))
	}
}

// @spec INTERP-004
func TestPrintItems(t *testing.T) {
	runPrintCases(t, []printCase{
		{"semicolon joins", line(printStmt(item(str("A")), semi, item(str("B")))), "AB\n"},
		{"comma moves to the next zone", line(printStmt(item(str("A")), comma, item(str("B")))), "A" + spaces(9) + "B\n"},
		{"adjacent items", line(printStmt(item(str("A")), item(str("B")))), "AB\n"},
		{"leading comma", line(printStmt(comma, item(str("A")))), spaces(10) + "A\n"},
		{"mixed", line(printStmt(item(str("A")), semi, item(str("B")), comma, item(str("C")))), "AB" + spaces(8) + "C\n"},
	})
}

// @spec INTERP-005
func TestPrintEndsWithNewline(t *testing.T) {
	runPrintCases(t, []printCase{
		{"PRINT alone", line(printStmt()), "\n"},
		{"one item", line(printStmt(item(str("A")))), "A\n"},
		{"separator then item", line(printStmt(item(str("A")), semi, item(str("B")))), "AB\n"},
	})
}

// @spec INTERP-006
func TestTrailingSeparatorSuppressesNewline(t *testing.T) {
	runPrintCases(t, []printCase{
		{"trailing semicolon", line(printStmt(item(str("A")), semi)), "A"},
		{"trailing comma", line(printStmt(item(str("A")), comma)), "A" + spaces(9)},
		{"only a semicolon", line(printStmt(semi)), ""},
	})
}

// @spec INTERP-007
func TestPrintWritesOnce(t *testing.T) {
	rec, err := exec(line(printStmt(item(str("A")), semi, item(str("B")), comma, item(str("C")))))
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if len(rec.writes) != 1 {
		t.Errorf("PRINT made %d writes %q, want 1", len(rec.writes), rec.writes)
	}
}

// @spec INTERP-008
func TestFailingItemWritesEarlierItems(t *testing.T) {
	long := strings.Repeat("x", 200)
	cases := []struct {
		name    string
		line    *ast.Line
		want    string
		wantErr basicerr.Kind
	}{
		{"bad item after items",
			line(printStmt(item(str("A")), semi, &ast.BadItem{Err: syntaxErr()})),
			"A", basicerr.Syntax},
		{"bad item first",
			line(printStmt(&ast.BadItem{Err: syntaxErr()})),
			"", basicerr.Syntax},
		{"failing expression after items",
			line(printStmt(item(str("A")), semi, item(cat(str(long), str(long))), item(str("B")))),
			"A", basicerr.StringTooLong},
	}
	for _, c := range cases {
		rec, err := exec(c.line)
		if !isKind(err, c.wantErr) {
			t.Errorf("%s: error = %v, want kind %d", c.name, err, c.wantErr)
		}
		if got := rec.String(); got != c.want {
			t.Errorf("%s: output %q, want %q", c.name, got, c.want)
		}
		if len(rec.writes) > 1 {
			t.Errorf("%s: made %d writes, want at most 1", c.name, len(rec.writes))
		}
	}
}

// @spec INTERP-009
func TestStringLiteralValueUnchanged(t *testing.T) {
	v := "a:\t;é\xff"
	runPrintCases(t, []printCase{
		{"special contents", line(printStmt(item(str(v)))), v + "\n"},
	})
}

// @spec INTERP-010
func TestConcat(t *testing.T) {
	runPrintCases(t, []printCase{
		{"two", line(printStmt(item(cat(str("FOO"), str("BAR"))))), "FOOBAR\n"},
		{"nested", line(printStmt(item(cat(cat(str("A"), str("B")), str("C"))))), "ABC\n"},
	})
}

// @spec INTERP-011
func TestConcatLongerThan255IsStringTooLong(t *testing.T) {
	cases := []struct {
		name        string
		left, right string
		wantErr     bool
	}{
		{"255 ASCII", strings.Repeat("x", 200), strings.Repeat("y", 55), false},
		{"256 ASCII", strings.Repeat("x", 200), strings.Repeat("y", 56), true},
		{"255 two-byte characters", strings.Repeat("é", 128), strings.Repeat("é", 127), false},
		{"256 two-byte characters", strings.Repeat("é", 128), strings.Repeat("é", 128), true},
		{"255 with invalid bytes", strings.Repeat("\xff", 200), strings.Repeat("y", 55), false},
		{"256 with invalid bytes", strings.Repeat("\xff", 200), strings.Repeat("y", 56), true},
	}
	for _, c := range cases {
		rec, err := exec(line(printStmt(item(cat(str(c.left), str(c.right))))))
		if c.wantErr {
			if !isKind(err, basicerr.StringTooLong) {
				t.Errorf("%s: error = %v, want STRING TOO LONG", c.name, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.name, err)
		}
		if want := c.left + c.right + "\n"; rec.String() != want {
			t.Errorf("%s: output has %d bytes, want %d", c.name, len(rec.String()), len(want))
		}
	}
}

// @spec INTERP-012
func TestLongStringLiteralIsNotLimited(t *testing.T) {
	long := strings.Repeat("x", 1000)
	runPrintCases(t, []printCase{
		{"1000-character literal", line(printStmt(item(str(long)))), long + "\n"},
	})
}

// failWriter fails every write.
type failWriter struct{ calls int }

var errBoom = errors.New("boom")

func (w *failWriter) Write(p []byte) (int, error) {
	w.calls++
	return 0, errBoom
}

// @spec INTERP-013
func TestWriteErrorIsReturnedUnchanged(t *testing.T) {
	w := &failWriter{}
	err := New(w).Exec(line(printStmt(item(str("A"))), printStmt(item(str("B")))))
	if err != errBoom {
		t.Errorf("error = %v, want the writer's error unchanged", err)
	}
	if w.calls != 1 {
		t.Errorf("writer called %d times, want 1 (no statements after the failure)", w.calls)
	}
}

// @spec INTERP-015
func TestCommaMovesToNextPrintZone(t *testing.T) {
	runPrintCases(t, []printCase{
		{"from column 0, a full zone", line(printStmt(comma, item(str("X")))), spaces(10) + "X\n"},
		{"from column 1", line(printStmt(item(str("A")), comma, item(str("X")))), "A" + spaces(9) + "X\n"},
		{"from column 9", line(printStmt(item(str("123456789")), comma, item(str("X")))), "123456789 X\n"},
		{"from a zone start, a full zone", line(printStmt(item(str("0123456789")), comma, item(str("X")))), "0123456789" + spaces(10) + "X\n"},
		{"from column 11", line(printStmt(item(str("01234567890")), comma, item(str("X")))), "01234567890" + spaces(9) + "X\n"},
		{"columns 0, 10, 20", line(printStmt(item(str("A")), comma, item(str("B")), comma, item(str("C")))), "A" + spaces(9) + "B" + spaces(9) + "C\n"},
		{"two commas", line(printStmt(item(str("A")), comma, comma, item(str("C")))), "A" + spaces(19) + "C\n"},
		{"past column 40", line(printStmt(item(str(strings.Repeat("x", 45))), comma, item(str("X")))), strings.Repeat("x", 45) + spaces(5) + "X\n"},
		{"multi-byte characters count once", line(printStmt(item(str("éé")), comma, item(str("X")))), "éé" + spaces(8) + "X\n"},
	})
}

// @spec INTERP-016
func TestColumnCarriesAcrossStatementsAndLines(t *testing.T) {
	rec := &recorder{}
	in := New(rec)
	if got := in.Column(); got != 0 {
		t.Errorf("initial column %d, want 0", got)
	}
	steps := []struct {
		line *ast.Line
		want int
	}{
		{line(printStmt(item(str("AB")), semi)), 2},
		{line(printStmt(item(str("é\xff")), semi)), 4},
		{line(printStmt(item(str("C")))), 0},
		{line(printStmt(item(str("XY")), semi), printStmt(item(str("Z")), comma)), 10},
	}
	for i, s := range steps {
		if err := in.Exec(s.line); err != nil {
			t.Fatal(err)
		}
		if got := in.Column(); got != s.want {
			t.Errorf("after step %d: column %d, want %d", i+1, got, s.want)
		}
	}

	// A comma on a later line uses the column left by an earlier line.
	rec2 := &recorder{}
	in2 := New(rec2)
	in2.Exec(line(printStmt(item(str("AB")), semi)))
	in2.Exec(line(printStmt(comma, item(str("X")))))
	if got, want := rec2.String(), "AB"+spaces(8)+"X\n"; got != want {
		t.Errorf("output %q, want %q", got, want)
	}
}

// @spec INTERP-017
func TestFreshLine(t *testing.T) {
	rec := &recorder{}
	in := New(rec)
	if err := in.FreshLine(); err != nil || rec.String() != "" {
		t.Errorf("at column 0: wrote %q (err %v), want nothing", rec.String(), err)
	}
	in.Exec(line(printStmt(item(str("A")), semi)))
	if err := in.FreshLine(); err != nil {
		t.Fatal(err)
	}
	if got := rec.String(); got != "A\n" {
		t.Errorf("mid-line: output %q, want %q", got, "A\n")
	}
	if got := in.Column(); got != 0 {
		t.Errorf("column after FreshLine %d, want 0", got)
	}
	if err := New(&failWriter{}).FreshLine(); err != nil {
		t.Errorf("FreshLine at column 0 with a failing writer returned %v, want nil (nothing written)", err)
	}
	failing := New(&failWriter{})
	failing.column = 3
	if err := failing.FreshLine(); err != errBoom {
		t.Errorf("FreshLine write error = %v, want the writer's error", err)
	}
}

// shortWriter writes at most n bytes, then fails.
type shortWriter struct {
	n   int
	got string
}

func (w *shortWriter) Write(p []byte) (int, error) {
	if len(p) <= w.n {
		w.n -= len(p)
		w.got += string(p)
		return len(p), nil
	}
	written := w.n
	w.got += string(p[:written])
	w.n = 0
	return written, errBoom
}

// @spec INTERP-018
func TestColumnAdvancesOnlyByWrittenCharacters(t *testing.T) {
	failing := New(&failWriter{})
	failing.Exec(line(printStmt(item(str("ABC")), semi)))
	if got := failing.Column(); got != 0 {
		t.Errorf("after a failed write: column %d, want 0", got)
	}
	w := &shortWriter{n: 2}
	short := New(w)
	short.Exec(line(printStmt(item(str("ABCD")), semi)))
	if got := short.Column(); got != 2 {
		t.Errorf("after a 2-byte partial write: column %d, want 2", got)
	}
}
