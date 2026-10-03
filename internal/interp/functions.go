package interp

import (
	"strings"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// maxCalls is the number of user-defined function calls that can be in
// progress at once, roughly what the C64's stack holds while their bodies
// are evaluated.
const maxCalls = 9

// execDef records a function definition ($B3B3).
//
// @spec INTERP-091
func (in *Interp) execDef(s *ast.DefStmt) error {
	switch {
	case strings.HasSuffix(s.Name.Name, "$"):
		return &basicerr.Error{Kind: basicerr.TypeMismatch}
	case in.cur.line == directLine:
		return &basicerr.Error{Kind: basicerr.IllegalDirect}
	case strings.HasSuffix(s.Param.Name, "$"):
		return &basicerr.Error{Kind: basicerr.TypeMismatch}
	}
	in.fns[s.Name.Name] = s
	return nil
}

// callFn evaluates a call of a user-defined function ($B3F4): evaluate
// the argument, give it to the parameter, evaluate the body, and restore
// the parameter. If the call fails, the parameter keeps the argument.
//
// @spec INTERP-092, INTERP-093, INTERP-094
func (in *Interp) callFn(e *ast.FnExpr) (value, error) {
	if strings.HasSuffix(e.Name.Name, "$") {
		return value{}, &basicerr.Error{Kind: basicerr.TypeMismatch}
	}
	arg, err := in.evalNumber(e.Arg)
	if err != nil {
		return value{}, err
	}
	def, ok := in.fns[e.Name.Name]
	if !ok {
		return value{}, &basicerr.Error{Kind: basicerr.UndefdFunction}
	}
	if in.calls >= maxCalls {
		return value{}, &basicerr.Error{Kind: basicerr.OutOfMemory}
	}
	if def.BodyErr != nil {
		in.vars[def.Param.Name] = numberValue(arg)
		return value{}, def.BodyErr
	}
	param := def.Param.Name
	saved, wasSet := in.vars[param]
	in.vars[param] = numberValue(arg)
	in.calls++
	result, err := in.evalNumber(def.Body)
	in.calls--
	if err != nil {
		return value{}, err
	}
	if wasSet {
		in.vars[param] = saved
	} else {
		delete(in.vars, param)
	}
	return numberValue(result), nil
}
