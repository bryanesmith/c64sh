package basicerr

import "testing"

// @spec SHELL-ERR-003
func TestErrorReturnsC64Name(t *testing.T) {
	cases := map[Kind]string{
		Syntax:             "SYNTAX",
		StringTooLong:      "STRING TOO LONG",
		TypeMismatch:       "TYPE MISMATCH",
		Overflow:           "OVERFLOW",
		DivisionByZero:     "DIVISION BY ZERO",
		IllegalQuantity:    "ILLEGAL QUANTITY",
		UndefdStatement:    "UNDEF'D STATEMENT",
		Break:              "BREAK",
		NextWithoutFor:     "NEXT WITHOUT FOR",
		OutOfMemory:        "OUT OF MEMORY",
		ReturnWithoutGosub: "RETURN WITHOUT GOSUB",
		IllegalDirect:      "ILLEGAL DIRECT",
		UndefdFunction:     "UNDEF'D FUNCTION",
	}
	for kind, want := range cases {
		if got := (&Error{Kind: kind}).Error(); got != want {
			t.Errorf("(&Error{Kind: %d}).Error() = %q, want %q", kind, got, want)
		}
	}
}
