package interp

import (
	"strings"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// frame is an entry of the control stack: a FOR or a GOSUB.
type frame struct {
	gosub       bool    // a GOSUB entry; otherwise a FOR entry
	name        string  // FOR: the variable's identity
	limit, step float64 // FOR: the end value and the step
	resume      pos     // where execution continues: just after the FOR or GOSUB
}

// The C64 stack space each entry takes, and the space in use at which a
// FOR or GOSUB finds no room: the ROM's check at $A3FB, with the stack
// depth of a running program. A GOSUB's 7 bytes include the statement
// loop's return address that it leaves behind its 5-byte entry.
const (
	forBytes   = 18
	gosubBytes = 7
	forRoom    = 169
	gosubRoom  = 179
)

// stackBytes returns the C64 stack space the control stack takes.
func (in *Interp) stackBytes() int {
	n := 0
	for _, f := range in.stack {
		if f.gosub {
			n += gosubBytes
		} else {
			n += forBytes
		}
	}
	return n
}

// findFor searches the control stack from the top for the FOR entry of
// the variable name, or the topmost FOR entry if name is empty, stopping
// at the first entry that is not a FOR ($A38A).
//
// @spec INTERP-080
func (in *Interp) findFor(name string) (int, bool) {
	for i := len(in.stack) - 1; i >= 0 && !in.stack[i].gosub; i-- {
		if name == "" || in.stack[i].name == name {
			return i, true
		}
	}
	return 0, false
}

// execGosub pushes a GOSUB entry and jumps to the subroutine ($A883).
//
// @spec INTERP-077
func (in *Interp) execGosub(s *ast.GosubStmt) error {
	if in.stackBytes() >= gosubRoom {
		return &basicerr.Error{Kind: basicerr.OutOfMemory}
	}
	in.ran = true
	in.stack = append(in.stack, frame{gosub: true, resume: in.after()})
	return &jump{line: s.Line, hasLine: true}
}

// execOn jumps to, or calls, the line its index picks from its list, or
// does nothing if the index is 0 or past the end of the list ($A94B).
//
// @spec INTERP-120, INTERP-121
func (in *Interp) execOn(s *ast.OnStmt) error {
	n, err := in.evalByte(s.Index)
	if err != nil || n == 0 || n > len(s.Lines) {
		return err
	}
	if s.Gosub {
		return in.execGosub(&ast.GosubStmt{Line: s.Lines[n-1]})
	}
	return in.execGoto(&ast.GotoStmt{Line: s.Lines[n-1]})
}

// execReturn passes over FOR entries to the topmost other entry, which
// must be a GOSUB, removes it and everything above it, and continues
// after its GOSUB ($A8D2).
//
// @spec INTERP-078, INTERP-079
func (in *Interp) execReturn() error {
	for i := len(in.stack) - 1; i >= 0; i-- {
		if f := in.stack[i]; f.gosub {
			in.stack = in.stack[:i]
			return &resume{at: f.resume}
		}
	}
	return &basicerr.Error{Kind: basicerr.ReturnWithoutGosub}
}

// execFor starts a loop as the ROM does ($A742): assign the start value,
// drop any loop already using the variable, check for stack room, then
// evaluate the end value and step once and push the entry.
//
// @spec INTERP-068, INTERP-069, INTERP-070, INTERP-075
func (in *Interp) execFor(s *ast.ForStmt) error {
	start, err := in.eval(s.From)
	if err != nil {
		return err
	}
	name := s.Var.Name
	if err := in.assign(name, start); err != nil {
		return err
	}
	if strings.HasSuffix(name, "$") {
		return &basicerr.Error{Kind: basicerr.TypeMismatch}
	}
	if i, found := in.findFor(name); found {
		in.stack = in.stack[:i]
	}
	if in.stackBytes() >= forRoom {
		return &basicerr.Error{Kind: basicerr.OutOfMemory}
	}
	limit, err := in.evalNumber(s.To)
	if err != nil {
		return err
	}
	step := 1.0
	if s.Step != nil {
		if step, err = in.evalNumber(s.Step); err != nil {
			return err
		}
	}
	in.stack = append(in.stack, frame{name: name, limit: limit, step: step, resume: in.after()})
	return nil
}

// execNext steps the loop of each of its variables, innermost first
// ($AD1E): it continues a loop that is not complete, and moves on to the
// next variable when one is.
//
// @spec INTERP-071, INTERP-072, INTERP-073
func (in *Interp) execNext(s *ast.NextStmt) error {
	names := []string{""}
	if len(s.Vars) > 0 {
		names = names[:0]
		for _, v := range s.Vars {
			names = append(names, v.Name)
		}
	}
	for _, name := range names {
		i, found := in.findFor(name)
		if !found {
			return &basicerr.Error{Kind: basicerr.NextWithoutFor}
		}
		in.stack = in.stack[:i+1] // loops left unfinished inside this one
		f := in.stack[i]
		v, err := inRange(in.vars[f.name].num + f.step)
		if err != nil {
			return err
		}
		in.vars[f.name] = v
		n := v.num
		done := (f.step > 0 && n > f.limit) || (f.step < 0 && n < f.limit) || (f.step == 0 && n == f.limit)
		if !done {
			return &resume{at: f.resume}
		}
		in.stack = in.stack[:i]
	}
	return nil
}

// evalNumber evaluates e, which must be a number.
func (in *Interp) evalNumber(e ast.Expr) (float64, error) {
	v, err := in.eval(e)
	if err != nil {
		return 0, err
	}
	if !v.isNum {
		return 0, &basicerr.Error{Kind: basicerr.TypeMismatch}
	}
	return v.num, nil
}
