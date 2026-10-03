// Package interp executes the AST of a line of C64 BASIC, and holds and
// runs the stored program.
package interp

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"
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
	out         io.Writer
	column      int                     // cursor column: characters written since the last newline
	vars        map[string]value        // variables, by identity (ast.VarRef.Name)
	program     []progLine              // stored lines, in ascending order of number
	ran         bool                    // whether RUN or GOTO has been executed
	direct      *ast.Line               // the line Exec is running, in direct mode
	stack       []frame                 // the control stack: FOR and GOSUB entries
	cur         pos                     // the position of the statement executing
	console     Console                 // where INPUT and GET read
	fns         map[string]*ast.DefStmt // user-defined functions, by name identity
	calls       int                     // user-defined function calls in progress
	storage     Storage                 // where LOAD, SAVE, and VERIFY find files
	messages    io.Writer               // where tape and disk messages go; nil for none
	files       map[int]*ioFile         // open logical files, by number
	cmd         int                     // the file CMD sends output to; 0 for the screen
	status      int                     // ST: the status of the last file operation
	arrays      map[string]*array       // arrays, by identity
	data        dataPos                 // the data pointer: the next DATA item
	printing    bool                    // a PRINT to the screen is evaluating an item
	printColumn int                     // while printing: the column its output so far reaches
	seed        uint64                  // the RND seed
	clock       func() time.Time        // the clock RND(0) reads; nil: the system clock

	interrupted atomic.Bool // set by Interrupt, checked after each statement
}

// New returns an interpreter that writes program output to out.
//
// @spec INTERP-002
func New(out io.Writer) *Interp {
	return &Interp{out: out, vars: map[string]value{}, arrays: map[string]*array{}, fns: map[string]*ast.DefStmt{}, files: map[int]*ioFile{}, seed: initialSeed}
}

// Interrupt asks the interpreter to stop, as the C64's STOP key does: the
// statement running now finishes, and Exec returns a BREAK error. It may
// be called from another goroutine.
//
// @spec INTERP-067
func (in *Interp) Interrupt() {
	in.interrupted.Store(true)
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

// execLet assigns a value to a variable.
func (in *Interp) execLet(s *ast.LetStmt) error {
	v, err := in.eval(s.Value)
	if err != nil {
		return err
	}
	return in.assign(s.Var, v)
}

// assign stores v in a variable or array element. A string variable (a
// name ending in "$") takes only strings, up to 255 characters; number
// and integer variables take only numbers, and an integer variable (a name
// ending in "%") stores the number rounded down, within -32768..32767. On
// any error the variable keeps its old value.
//
// @spec INTERP-032, INTERP-034, INTERP-035, INTERP-036, INTERP-037, INTERP-038, INTERP-136
func (in *Interp) assign(ref *ast.VarRef, v value) error {
	name := ref.Name
	var elem *value
	if len(ref.Subs) > 0 {
		var err error
		if elem, err = in.element(ref); err != nil {
			return err
		}
	}
	isString := strings.HasSuffix(name, "$")
	if isString == v.isNum {
		return &basicerr.Error{Kind: basicerr.TypeMismatch}
	}
	if isString && utf8.RuneCountInString(v.str) > maxStringLen {
		return &basicerr.Error{Kind: basicerr.StringTooLong}
	}
	if strings.HasSuffix(name, "%") {
		n, err := toInt16(v)
		if err != nil {
			return err
		}
		v = numberValue(float64(n))
	}
	if elem != nil {
		*elem = v
		return nil
	}
	in.vars[name] = v
	return nil
}

// errSkipLine is returned by execIf when its condition is false, ending
// the line without error.
var errSkipLine = errors.New("skip the rest of the line")

// execIf evaluates an IF's condition: a number is true when it is not 0, a
// string when it is not empty, as the C64 ROM tests the byte holding a
// string's length ($A928, $B4D5).
//
// @spec INTERP-045, INTERP-046
func (in *Interp) execIf(s *ast.IfStmt) error {
	v, err := in.eval(s.Cond)
	if err != nil {
		return err
	}
	if (v.isNum && v.num != 0) || (!v.isNum && v.str != "") {
		return nil
	}
	return errSkipLine
}

// @spec INTERP-003, INTERP-014, INTERP-047
func (in *Interp) execStmt(s ast.Stmt) error {
	switch s := s.(type) {
	case *ast.PrintStmt:
		return in.execPrint(s)
	case *ast.RemStmt:
		return nil // a comment does nothing
	case *ast.LetStmt:
		return in.execLet(s)
	case *ast.IfStmt:
		return in.execIf(s)
	case *ast.BadStmt:
		return s.Err // a syntax error, now that execution has reached it
	case *ast.RunStmt:
		return in.execRun(s)
	case *ast.GotoStmt:
		return in.execGoto(s)
	case *ast.ForStmt:
		return in.execFor(s)
	case *ast.NextStmt:
		return in.execNext(s)
	case *ast.GosubStmt:
		return in.execGosub(s)
	case *ast.ReturnStmt:
		return in.execReturn()
	case *ast.InputStmt:
		return in.execInput(s)
	case *ast.GetStmt:
		return in.execGet(s)
	case *ast.DefStmt:
		return in.execDef(s)
	case *ast.LoadStmt:
		return in.execLoad(s)
	case *ast.SaveStmt:
		return in.execSave(s)
	case *ast.VerifyStmt:
		return in.execVerify(s)
	case *ast.OpenStmt:
		return in.execOpen(s)
	case *ast.CloseStmt:
		return in.execClose(s)
	case *ast.CmdStmt:
		return in.execCmd(s)
	case *ast.OnStmt:
		return in.execOn(s)
	case *ast.DimStmt:
		return in.execDim(s)
	case *ast.DataStmt:
		return nil // DATA is read by READ
	case *ast.ReadStmt:
		return in.execRead(s)
	case *ast.RestoreStmt:
		in.restore()
		return nil
	case *ast.ListStmt:
		return in.execList()
	case *ast.NewStmt:
		return in.execNew()
	case *ast.EndStmt:
		return errEnd
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
	if s.File == nil {
		return in.printItems(s.Items, in.cmdFile())
	}
	f, err := in.outputFile(s.File)
	if err != nil {
		return err
	}
	defer in.endCmd() // PRINT# returns output to the screen ($ABB5)
	return in.printItems(s.Items, f)
}

// printItems writes PRINT's items to f, or to the screen if f is nil. A
// comma moves to the next print zone of the screen's cursor column, as the
// C64 reads it even when writing to a file ($AAE8), so output to a storage
// file does not move the zones.
//
// @spec INTERP-114
func (in *Interp) printItems(items []ast.PrintItem, f *ioFile) error {
	var buf strings.Builder
	column := in.column // where the cursor will be once buf is written
	screen := f == nil || f.screen
	newline := true
	for _, item := range items {
		switch item := item.(type) {
		case *ast.ExprItem:
			if screen {
				// POS sees the column the items so far have reached.
				in.printColumn, in.printing = column, true
			}
			v, err := in.eval(item.Expr)
			in.printing = false
			if err != nil {
				return in.fail(f, buf.String(), err)
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
			if !screen {
				n = zoneWidth - in.column%zoneWidth
			}
			buf.WriteString(strings.Repeat(" ", n))
			column += n
			newline = false
		case *ast.TabItem, *ast.SpcItem:
			// @spec INTERP-130, INTERP-131
			var x ast.Expr
			if tab, ok := item.(*ast.TabItem); ok {
				x = tab.X
			} else {
				x = item.(*ast.SpcItem).X
			}
			n, err := in.evalByte(x)
			if err != nil {
				return in.fail(f, buf.String(), err)
			}
			if _, ok := item.(*ast.TabItem); ok {
				from := column
				if !screen {
					from = in.column
				}
				n = max(n-from, 0)
			}
			buf.WriteString(strings.Repeat(" ", n))
			column += n
			newline = false
		case *ast.BadItem:
			return in.fail(f, buf.String(), item.Err)
		default:
			panic(fmt.Sprintf("interp: unhandled print item %T", item))
		}
	}
	if newline {
		buf.WriteByte('\n')
	}
	return in.emit(f, buf.String())
}

// fail writes partial output to f and returns err, or the write error if
// the write fails.
func (in *Interp) fail(f *ioFile, partial string, err error) error {
	if werr := in.emit(f, partial); werr != nil {
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
// @spec INTERP-029, INTERP-030, INTERP-031, INTERP-039, INTERP-040, INTERP-041
// @spec INTERP-042, INTERP-043, INTERP-044
func (in *Interp) eval(e ast.Expr) (value, error) {
	switch e := e.(type) {
	case *ast.StringLit:
		return stringValue(e.Value), nil
	case *ast.NumberLit:
		return inRange(e.Value)
	case *ast.VarRef:
		if len(e.Subs) > 0 {
			elem, err := in.element(e)
			if err != nil {
				return value{}, err
			}
			return *elem, nil
		}
		if e.Name == "ST" {
			return numberValue(float64(in.status)), nil // @spec INTERP-119
		}
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
	case *ast.NotExpr:
		x, err := in.eval(e.X)
		if err != nil {
			return value{}, err
		}
		n, err := toInt16(x)
		if err != nil {
			return value{}, err
		}
		return numberValue(float64(^n)), nil
	case *ast.FnExpr:
		return in.callFn(e)
	case *ast.CallExpr:
		return in.call(e)
	case *ast.CompareExpr:
		l, err := in.eval(e.Left)
		if err != nil {
			return value{}, err
		}
		r, err := in.eval(e.Right)
		if err != nil {
			return value{}, err
		}
		return compare(e.Rel, l, r)
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
	case ast.And, ast.Or:
		a, err := toInt16(l)
		if err != nil {
			return value{}, err
		}
		b, err := toInt16(r)
		if err != nil {
			return value{}, err
		}
		if op == ast.And {
			return numberValue(float64(a & b)), nil
		}
		return numberValue(float64(a | b)), nil
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
