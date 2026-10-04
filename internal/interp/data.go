package interp

import (
	"strings"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// dataPos is the data pointer: a program line (an index), a statement in
// it, and a place in that statement's DATA text; off beyond the text's end
// means the statement's items are used up.
type dataPos struct{ line, stmt, off int }

// atLine marks a BASIC error to be reported with a given program line
// rather than the line of the statement executing.
type atLineErr struct {
	err  error
	line int
}

func (e *atLineErr) Error() string { return e.err.Error() }
func (e *atLineErr) Unwrap() error { return e.err }

// execRead assigns each variable the next DATA item ($AC06).
//
// @spec INTERP-139, INTERP-140, INTERP-141
func (in *Interp) execRead(s *ast.ReadStmt) error {
	for _, v := range s.Vars {
		text, number, err := in.nextData()
		if err != nil {
			return err
		}
		var val value
		var ok bool
		var i int
		if strings.HasSuffix(v.Name, "$") {
			val, i, ok = inputString(text, in.data.off)
		} else {
			var n float64
			n, i, ok = inputNumber(text, in.data.off)
			if ok {
				if val, err = inRange(n); err != nil {
					return err
				}
			}
		}
		if !ok {
			// The ROM reports a bad item at its DATA line ($AB57).
			return &atLineErr{err: &basicerr.Error{Kind: basicerr.Syntax}, line: number}
		}
		i = skipSpaces(text, i)
		if i < len(text) { // a comma: another item follows
			in.data.off = i + 1
		} else {
			in.data.off = len(text) + 1
		}
		if err := in.assign(v, val); err != nil {
			return err
		}
	}
	return nil
}

// nextData moves the data pointer to the next DATA item and returns that
// statement's text and the number of its line, or fails with OUT OF DATA.
func (in *Interp) nextData() (string, int, error) {
	for p := &in.data; p.line < len(in.program); p.line, p.stmt, p.off = p.line+1, 0, 0 {
		stmts := in.program[p.line].tree.Statements
		for ; p.stmt < len(stmts); p.stmt, p.off = p.stmt+1, 0 {
			if d, ok := stmts[p.stmt].(*ast.DataStmt); ok && p.off <= len(d.Text) {
				return d.Text, in.program[p.line].number, nil
			}
		}
	}
	return "", 0, &basicerr.Error{Kind: basicerr.OutOfData}
}

// restore moves the data pointer to the first item.
//
// @spec INTERP-142
func (in *Interp) restore() {
	in.data = dataPos{}
}
