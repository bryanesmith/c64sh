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

// Interp executes lines, writing program output to its writer.
type Interp struct {
	out io.Writer
}

// New returns an interpreter that writes program output to out.
//
// @spec INTERP-002
func New(out io.Writer) *Interp {
	return &Interp{out: out}
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

// @spec INTERP-003, INTERP-014
func (in *Interp) execStmt(s ast.Stmt) error {
	switch s := s.(type) {
	case *ast.PrintStmt:
		return in.execPrint(s)
	case *ast.RemStmt:
		return nil // a comment does nothing
	default:
		panic(fmt.Sprintf("interp: unhandled statement %T", s))
	}
}

// execPrint writes a PRINT statement's output in one call. If an item
// fails, it writes the output of the items before it and returns the
// item's error, as a C64 prints each item as it goes.
//
// @spec INTERP-004, INTERP-005, INTERP-006, INTERP-007, INTERP-008
func (in *Interp) execPrint(s *ast.PrintStmt) error {
	var buf strings.Builder
	newline := true
	for _, item := range s.Items {
		switch item := item.(type) {
		case *ast.ExprItem:
			v, err := in.eval(item.Expr)
			if err != nil {
				return in.fail(buf.String(), err)
			}
			buf.WriteString(v)
			newline = true
		case *ast.Semicolon:
			newline = false
		case *ast.Comma:
			buf.WriteByte('\t')
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

func (in *Interp) write(s string) error {
	if s == "" {
		return nil
	}
	_, err := io.WriteString(in.out, s)
	return err
}

// @spec INTERP-003, INTERP-009, INTERP-010, INTERP-011, INTERP-012
func (in *Interp) eval(e ast.Expr) (string, error) {
	switch e := e.(type) {
	case *ast.StringLit:
		return e.Value, nil
	case *ast.Concat:
		l, err := in.eval(e.Left)
		if err != nil {
			return "", err
		}
		r, err := in.eval(e.Right)
		if err != nil {
			return "", err
		}
		// RuneCountInString counts each invalid UTF-8 byte as one character.
		if utf8.RuneCountInString(l)+utf8.RuneCountInString(r) > maxStringLen {
			return "", &basicerr.Error{Kind: basicerr.StringTooLong}
		}
		return l + r, nil
	default:
		panic(fmt.Sprintf("interp: unhandled expression %T", e))
	}
}
