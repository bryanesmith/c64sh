// Package basicerr defines the BASIC errors that c64sh reports the way a
// C64 does, such as ?SYNTAX  ERROR.
package basicerr

// Kind identifies a BASIC error.
type Kind int

const (
	Syntax          Kind = iota // SYNTAX
	StringTooLong               // STRING TOO LONG
	TypeMismatch                // TYPE MISMATCH
	Overflow                    // OVERFLOW
	DivisionByZero              // DIVISION BY ZERO
	IllegalQuantity             // ILLEGAL QUANTITY
	UndefdStatement             // UNDEF'D STATEMENT
	Break                       // BREAK: execution stopped by Ctrl-C
)

// names holds each kind's name as the C64 prints it.
var names = [...]string{
	Syntax:          "SYNTAX",
	StringTooLong:   "STRING TOO LONG",
	TypeMismatch:    "TYPE MISMATCH",
	Overflow:        "OVERFLOW",
	DivisionByZero:  "DIVISION BY ZERO",
	IllegalQuantity: "ILLEGAL QUANTITY",
	UndefdStatement: "UNDEF'D STATEMENT",
	Break:           "BREAK",
}

// Error is a BASIC error.
type Error struct {
	Kind    Kind
	Line    int  // the program line where the error occurred, if HasLine
	HasLine bool // false for an error in direct mode
}

// Error returns the error's C64 name, such as "SYNTAX".
//
// @spec SHELL-ERR-003
func (e *Error) Error() string {
	return names[e.Kind]
}
