// Package token defines the tokens the lexer produces from a line of
// C64 BASIC input.
package token

import "fmt"

// Kind identifies the type of a token.
type Kind int

const (
	EOL       Kind = iota // end of line; always the last token
	Illegal               // a character no rule accepts
	Print                 // PRINT or ?
	Rem                   // REM and the rest of the line
	Number                // 12, 3.5, .5, 1E3
	String                // "…"
	Colon                 // :
	Semicolon             // ;
	Comma                 // ,
	Plus                  // +
	Minus                 // -
	Star                  // *
	Slash                 // /
	LParen                // (
	RParen                // )
	Caret                 // ^ or ↑ (exponentiation)
)

var kindNames = [...]string{
	EOL:       "EOL",
	Illegal:   "Illegal",
	Print:     "Print",
	Rem:       "Rem",
	Number:    "Number",
	String:    "String",
	Colon:     "Colon",
	Semicolon: "Semicolon",
	Comma:     "Comma",
	Plus:      "Plus",
	Minus:     "Minus",
	Star:      "Star",
	Slash:     "Slash",
	LParen:    "LParen",
	RParen:    "RParen",
	Caret:     "Caret",
}

func (k Kind) String() string {
	if k >= 0 && int(k) < len(kindNames) {
		return kindNames[k]
	}
	return fmt.Sprintf("Kind(%d)", int(k))
}

// Token is one token of a line.
type Token struct {
	Kind  Kind
	Value string // String: contents without quotes; Rem: the text after REM; Number: the literal without spaces; Illegal: the character; otherwise the source text
	Pos   int    // byte offset of the token's first character in the line
}
