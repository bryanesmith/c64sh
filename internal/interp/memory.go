package interp

import (
	"unicode/utf8"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/lexer"
	"github.com/bryanesmith/c64sh/internal/token"
)

// basicFree is the memory free for BASIC with no program and no variables:
// the 38911 bytes a C64 reports at start-up, less the two bytes that end
// an empty program.
const basicFree = 38909

// fre evaluates FRE: X, of either type, is evaluated and ignored, as the
// ROM ignores it, and the result is the memory left, counted as a C64
// counts it, shown as a signed 16-bit number, as a C64 shows it.
//
// @spec INTERP-157
func (in *Interp) fre(arg ast.Expr) (value, error) {
	if _, err := in.eval(arg); err != nil {
		return value{}, err
	}
	free := max(basicFree-in.memoryUsed(), 0)
	if free > 32767 {
		free -= 65536
	}
	return numberValue(float64(free)), nil
}

// memoryUsed returns the bytes the program, variables, function
// definitions, arrays, and strings would take on a C64.
func (in *Interp) memoryUsed() int {
	used := 0
	for _, l := range in.program {
		used += 5 + tokenizedLen(l.text) // link, line number, and end byte
	}
	used += 7 * (len(in.vars) + len(in.fns))
	for _, v := range in.vars {
		if !v.isNum {
			used += utf8.RuneCountInString(v.str)
		}
	}
	for name, a := range in.arrays {
		used += size(name, a.tops)
		for _, v := range a.data {
			if !v.isNum {
				used += utf8.RuneCountInString(v.str)
			}
		}
	}
	return used
}

// tokenizedLen returns the length of a line's text as a C64 stores it:
// each keyword one byte, everything else, spaces included, one byte per
// character.
func tokenizedLen(text string) int {
	n := utf8.RuneCountInString(text)
	for _, tok := range lexer.Lex(text) {
		switch tok.Kind {
		case token.Rem:
			n -= len("REM") - 1 // the token's value is the comment
		case token.Data:
			n -= len("DATA") - 1
		case token.String, token.Number, token.Name, token.Illegal, token.EOL:
		default:
			n -= max(utf8.RuneCountInString(tok.Value)-1, 0)
		}
	}
	return n
}
