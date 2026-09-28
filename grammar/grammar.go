// Package grammar holds the EBNF grammar of the C64 BASIC V2 subset that
// c64sh accepts, for use by tests that keep the lexer and parser aligned
// with it.
package grammar

import (
	_ "embed"
	"strings"

	"golang.org/x/exp/ebnf"
)

// Source is the text of c64basic.ebnf.
//
// @spec GRAMMAR-006
//
//go:embed c64basic.ebnf
var Source string

// Start is the grammar's start rule.
const Start = "Line"

// Load parses and verifies Source.
//
// @spec GRAMMAR-007
func Load() (ebnf.Grammar, error) {
	g, err := ebnf.Parse("c64basic.ebnf", strings.NewReader(Source))
	if err != nil {
		return nil, err
	}
	if err := ebnf.Verify(g, Start); err != nil {
		return nil, err
	}
	return g, nil
}
