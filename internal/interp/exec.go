package interp

import (
	"errors"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// pos is an execution position: a line, either an index into the program
// or directLine, and the index of a statement in it.
type pos struct{ line, stmt int }

// directLine is the line of a position in the line Exec is running.
const directLine = -1

// resume is returned by a statement that continues execution at a
// position, such as NEXT looping back.
type resume struct{ at pos }

func (*resume) Error() string { return "resume" }

// Exec runs the statements of line in order, in direct mode. It stops at
// the first statement that fails and returns that error; statements after
// it do not run. Output from statements before the failure has already
// been written. A RUN among the statements runs the stored program, and
// an error in it is returned with its line number.
//
// @spec INTERP-001, INTERP-013, INTERP-057, INTERP-066, INTERP-076
func (in *Interp) Exec(line *ast.Line) error {
	in.interrupted.Store(false)
	in.direct = line
	err := in.execute(pos{directLine, 0})
	if err != nil {
		in.cmd = 0 // an error returns output to the screen ($A447)
	}
	if be, ok := errors.AsType[*basicerr.Error](err); ok && be.Kind != basicerr.Break {
		in.stack = nil // an error flushes the stack, as on a C64 ($A462)
	}
	// Entries that would resume in this line are gone with it.
	for i, f := range in.stack {
		if f.resume.line == directLine {
			in.stack = in.stack[:i]
			break
		}
	}
	in.direct = nil
	return err
}

// execute runs statements from p until execution ends: at the end of the
// direct-mode line or of the program, at END, LIST, or NEW, or at an
// error, which it returns with the program line it occurred in. After
// each statement that finishes normally (including a false IF, and a
// jump before it is made), it checks for an interrupt, as the C64 checks
// its STOP key between statements ($A7AE, $A82C).
//
// @spec INTERP-052, INTERP-053, INTERP-054, INTERP-055, INTERP-056, INTERP-058, INTERP-061
// @spec INTERP-065, INTERP-074
func (in *Interp) execute(p pos) error {
	for {
		stmts := in.statements(p.line)
		if p.stmt >= len(stmts) {
			if p.line == directLine || p.line+1 >= len(in.program) {
				return nil
			}
			p = pos{p.line + 1, 0}
			continue
		}
		in.cur = p
		err := in.execStmt(stmts[p.stmt])
		next := pos{p.line, p.stmt + 1}

		var j *jump
		var r *resume
		isJump := errors.As(err, &j)
		isResume := errors.As(err, &r)
		if (err == nil || err == errSkipLine || isJump || isResume) && in.interrupted.Swap(false) {
			return in.atCur(&basicerr.Error{Kind: basicerr.Break})
		}
		switch {
		case err == nil:
		case err == errSkipLine:
			next = pos{p.line, len(stmts)} // a false IF: the rest of the line does not run
		case err == errEnd:
			return nil
		case isResume:
			next = r.at
		case isJump && !j.hasLine:
			if len(in.program) == 0 {
				return nil
			}
			next = pos{0, 0}
		case isJump:
			i, found := in.find(j.line)
			if !found {
				return in.atCur(&basicerr.Error{Kind: basicerr.UndefdStatement})
			}
			next = pos{i, 0}
		default:
			return in.atCur(err)
		}
		p = next
	}
}

// statements returns the statements of a position's line.
func (in *Interp) statements(line int) []ast.Stmt {
	if line == directLine {
		return in.direct.Statements
	}
	return in.program[line].tree.Statements
}

// atCur returns err with the number of the program line holding the
// statement executing, or unchanged in direct mode.
func (in *Interp) atCur(err error) error {
	if nl, ok := err.(*noLine); ok {
		return nl.err
	}
	if al, ok := err.(*atLineErr); ok {
		return atLine(al.err, al.line)
	}
	if in.cur.line == directLine {
		return err
	}
	return atLine(err, in.program[in.cur.line].number)
}

// after returns the position of the statement after the one executing.
func (in *Interp) after() pos {
	return pos{in.cur.line, in.cur.stmt + 1}
}

// clr clears the variables, function definitions, and the control stack,
// as the C64's CLR does.
//
// @spec INTERP-095, INTERP-138
func (in *Interp) clr() {
	in.closeAll()
	clear(in.vars)
	clear(in.fns)
	clear(in.arrays)
	in.restore()
	in.stack = nil
}
