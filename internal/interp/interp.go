// Package interp executes the AST of a line of C64 BASIC.
package interp

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// maxStringLen is the longest string, in characters, that a C64 can
// produce at run time.
const maxStringLen = 255

// zoneWidth is the width of a C64 print zone, the columns "," moves between.
const zoneWidth = 10

// Interp executes lines, writing program output to its writer.
type Interp struct {
	out    io.Writer
	column int              // cursor column: characters written since the last newline
	vars   map[string]value // variables, by identity (ast.VarRef.Name)
}

// New returns an interpreter that writes program output to out.
//
// @spec INTERP-002
func New(out io.Writer) *Interp {
	return &Interp{out: out, vars: map[string]value{}}
}

// Exec runs the statements of line in order. It stops at the first
// statement that fails and returns that error; statements after it do
// not run. Output from statements before the failure has already been
// written.
//
// @spec INTERP-001, INTERP-013
func (in *Interp) Exec(line *ast.Line) error {
	for _, s := range line.Statements {
		if err := in.execStmt(s); err != nil {
			return err
		}
	}
	return nil
}

// Column returns the cursor column: the number of characters written since
// the last newline. It is 0 at the start of a line.
//
// @spec INTERP-016
func (in *Interp) Column() int {
	return in.column
}

// FreshLine ends the current output line if it is unfinished.
//
// @spec INTERP-017
func (in *Interp) FreshLine() error {
	if in.column == 0 {
		return nil
	}
	return in.write("\n")
}

// execLet assigns a value to a variable. A string variable (a name ending
// in "$") takes only strings, up to 255 characters, and a number variable
// only numbers. On any error the variable keeps its old value.
//
// @spec INTERP-032, INTERP-034, INTERP-035, INTERP-036
func (in *Interp) execLet(s *ast.LetStmt) error {
	v, err := in.eval(s.Value)
	if err != nil {
		return err
	}
	isString := strings.HasSuffix(s.Var.Name, "$")
	if isString == v.isNum {
		return &basicerr.Error{Kind: basicerr.TypeMismatch}
	}
	if isString && utf8.RuneCountInString(v.str) > maxStringLen {
		return &basicerr.Error{Kind: basicerr.StringTooLong}
	}
	in.vars[s.Var.Name] = v
	return nil
}

// @spec INTERP-003, INTERP-014
func (in *Interp) execStmt(s ast.Stmt) error {
	switch s := s.(type) {
	case *ast.PrintStmt:
		return in.execPrint(s)
	case *ast.RemStmt:
		return nil // a comment does nothing
	case *ast.LetStmt:
		return in.execLet(s)
	default:
		panic(fmt.Sprintf("interp: unhandled statement %T", s))
	}
}

// execPrint writes a PRINT statement's output in one call. If an item
// fails, it writes the output of the items before it and returns the
// item's error, as a C64 prints each item as it goes.
//
// @spec INTERP-004, INTERP-005, INTERP-006, INTERP-007, INTERP-008, INTERP-015
func (in *Interp) execPrint(s *ast.PrintStmt) error {
	var buf strings.Builder
	column := in.column // where the cursor will be once buf is written
	newline := true
	for _, item := range s.Items {
		switch item := item.(type) {
		case *ast.ExprItem:
			v, err := in.eval(item.Expr)
			if err != nil {
				return in.fail(buf.String(), err)
			}
			text := v.str
			if v.isNum {
				text = formatNumber(v.num) + " "
			}
			buf.WriteString(text)
			column += utf8.RuneCountInString(text)
			newline = true
		case *ast.Semicolon:
			newline = false
		case *ast.Comma:
			// Move to the next print zone; never 0 spaces, as in the C64 ROM.
			n := zoneWidth - column%zoneWidth
			buf.WriteString(strings.Repeat(" ", n))
			column += n
			newline = false
		case *ast.BadItem:
			return in.fail(buf.String(), item.Err)
		default:
			panic(fmt.Sprintf("interp: unhandled print item %T", item))
		}
	}
	if newline {
		buf.WriteByte('\n')
	}
	return in.write(buf.String())
}

// fail writes partial output and returns err, or the write error if the
// write fails.
func (in *Interp) fail(partial string, err error) error {
	if werr := in.write(partial); werr != nil {
		return werr
	}
	return err
}

// write writes s and advances the cursor column by what was written.
//
// @spec INTERP-016, INTERP-018
func (in *Interp) write(s string) error {
	if s == "" {
		return nil
	}
	n, err := io.WriteString(in.out, s)
	written := s[:n]
	// RuneCountInString counts each invalid UTF-8 byte as one character.
	if i := strings.LastIndexByte(written, '\n'); i >= 0 {
		in.column = utf8.RuneCountInString(written[i+1:])
	} else {
		in.column += utf8.RuneCountInString(written)
	}
	return err
}

// @spec INTERP-003, INTERP-009, INTERP-010, INTERP-011, INTERP-012
// @spec INTERP-020, INTERP-021, INTERP-022, INTERP-025, INTERP-026, INTERP-027, INTERP-028
// @spec INTERP-029, INTERP-030, INTERP-031
func (in *Interp) eval(e ast.Expr) (value, error) {
	switch e := e.(type) {
	case *ast.StringLit:
		return stringValue(e.Value), nil
	case *ast.NumberLit:
		return inRange(e.Value)
	case *ast.VarRef:
		// @spec INTERP-033
		if v, ok := in.vars[e.Name]; ok {
			return v, nil
		}
		if strings.HasSuffix(e.Name, "$") {
			return stringValue(""), nil
		}
		return numberValue(0), nil
	case *ast.NegExpr:
		x, err := in.eval(e.X)
		if err != nil {
			return value{}, err
		}
		if !x.isNum {
			return value{}, &basicerr.Error{Kind: basicerr.TypeMismatch}
		}
		return inRange(-x.num)
	case *ast.BinaryExpr:
		// Left operand, then right, then the operator: the first error wins.
		l, err := in.eval(e.Left)
		if err != nil {
			return value{}, err
		}
		r, err := in.eval(e.Right)
		if err != nil {
			return value{}, err
		}
		return binary(e.Op, l, r)
	default:
		panic(fmt.Sprintf("interp: unhandled expression %T", e))
	}
}

// binary applies op to two evaluated operands.
func binary(op ast.Op, l, r value) (value, error) {
	switch op {
	case ast.Add:
		switch {
		case l.isNum && r.isNum:
			return inRange(l.num + r.num)
		case !l.isNum && !r.isNum:
			// RuneCountInString counts each invalid UTF-8 byte as one character.
			if utf8.RuneCountInString(l.str)+utf8.RuneCountInString(r.str) > maxStringLen {
				return value{}, &basicerr.Error{Kind: basicerr.StringTooLong}
			}
			return stringValue(l.str + r.str), nil
		default:
			return value{}, &basicerr.Error{Kind: basicerr.TypeMismatch}
		}
	case ast.Sub, ast.Mul, ast.Div, ast.Pow:
		// Only "+" accepts strings; types are checked before the divisor.
		if !l.isNum || !r.isNum {
			return value{}, &basicerr.Error{Kind: basicerr.TypeMismatch}
		}
		switch op {
		case ast.Sub:
			return inRange(l.num - r.num)
		case ast.Mul:
			return inRange(l.num * r.num)
		case ast.Pow:
			return power(l.num, r.num)
		default:
			if r.num == 0 {
				return value{}, &basicerr.Error{Kind: basicerr.DivisionByZero}
			}
			return inRange(l.num / r.num)
		}
	default:
		panic(fmt.Sprintf("interp: unhandled operator %d", op))
	}
}
