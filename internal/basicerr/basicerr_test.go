package basicerr

import "testing"

// @spec SHELL-ERR-003
func TestErrorReturnsC64Name(t *testing.T) {
	cases := map[Kind]string{
		Syntax:        "SYNTAX",
		StringTooLong: "STRING TOO LONG",
	}
	for kind, want := range cases {
		if got := (&Error{Kind: kind}).Error(); got != want {
			t.Errorf("(&Error{Kind: %d}).Error() = %q, want %q", kind, got, want)
		}
	}
}
