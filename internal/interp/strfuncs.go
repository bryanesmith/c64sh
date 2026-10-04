package interp

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// callString evaluates a call of a string function: LEN, LEFT$, RIGHT$,
// MID$, CHR$, ASC, STR$, or VAL.
//
// @spec INTERP-126, INTERP-127, INTERP-128, INTERP-129
func (in *Interp) callString(e *ast.CallExpr) (value, error) {
	switch e.Name {
	case "ENVIRON$":
		return in.environ(e.Args[0])
	case "CHR$":
		n, err := in.evalByte(e.Args[0])
		if err != nil {
			return value{}, err
		}
		return stringValue(string(rune(n))), nil
	case "STR$":
		x, err := in.evalNumber(e.Args[0])
		if err != nil {
			return value{}, err
		}
		return stringValue(formatNumber(x)), nil
	}
	s, err := in.evalString(e.Args[0])
	if err != nil {
		return value{}, err
	}
	runes := []rune(s)
	var nums []int
	for _, a := range e.Args[1:] {
		n, err := in.evalPosition(a)
		if err != nil {
			return value{}, err
		}
		nums = append(nums, n)
	}
	switch e.Name {
	case "LEN":
		return numberValue(float64(len(runes))), nil
	case "ASC":
		if len(runes) == 0 {
			return value{}, &basicerr.Error{Kind: basicerr.IllegalQuantity}
		}
		r, _ := utf8.DecodeRuneInString(s)
		return numberValue(float64(r)), nil
	case "VAL":
		n, _, _ := inputNumber(strings.TrimRight(s, " "), 0)
		return inRange(n)
	case "LEFT$":
		return stringValue(string(runes[:min(nums[0], len(runes))])), nil
	case "RIGHT$":
		return stringValue(string(runes[len(runes)-min(nums[0], len(runes)):])), nil
	case "MID$":
		start := nums[0]
		if start == 0 {
			return value{}, &basicerr.Error{Kind: basicerr.IllegalQuantity}
		}
		if start > len(runes) {
			return stringValue(""), nil
		}
		end := len(runes)
		if len(nums) > 1 {
			end = min(start-1+nums[1], len(runes))
		}
		return stringValue(string(runes[start-1 : end])), nil
	}
	panic("interp: unhandled function " + e.Name)
}

// evalPosition evaluates a position or length in a string, rounded down:
// a byte while strings are limited to 255 characters or fewer, as on a
// C64, and otherwise any number from 0 up to the string limit, if any.
func (in *Interp) evalPosition(e ast.Expr) (int, error) {
	if in.stringLimit >= 0 && in.stringLimit <= c64StringLimit {
		return in.evalByte(e)
	}
	x, err := in.evalNumber(e)
	if err != nil {
		return 0, err
	}
	x = math.Floor(x)
	if x < 0 || (in.stringLimit >= 0 && x > float64(in.stringLimit)) {
		return 0, &basicerr.Error{Kind: basicerr.IllegalQuantity}
	}
	return int(min(x, math.MaxInt32)), nil
}

// evalString evaluates e, which must be a string.
func (in *Interp) evalString(e ast.Expr) (string, error) {
	v, err := in.eval(e)
	if err != nil {
		return "", err
	}
	if v.isNum {
		return "", &basicerr.Error{Kind: basicerr.TypeMismatch}
	}
	return v.str, nil
}
