package lexer

import (
	"reflect"
	"testing"

	"github.com/bryanesmith/c64sh/internal/token"
)

func tok(k token.Kind, value string, pos int) token.Token {
	return token.Token{Kind: k, Value: value, Pos: pos}
}

func eol(pos int) token.Token { return tok(token.EOL, "", pos) }

type lexCase struct {
	name string
	line string
	want []token.Token
}

func runLexCases(t *testing.T, cases []lexCase) {
	t.Helper()
	for _, c := range cases {
		got := Lex(c.line)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: Lex(%q)\n got: %v\nwant: %v", c.name, c.line, got, c.want)
		}
	}
}

// @spec LEXER-001
func TestLexEndsWithExactlyOneEOL(t *testing.T) {
	runLexCases(t, []lexCase{
		{"empty line", "", []token.Token{eol(0)}},
		{"print statement", `PRINT "A"`, []token.Token{
			tok(token.Print, "PRINT", 0), tok(token.String, "A", 6), eol(9),
		}},
	})
}

// @spec LEXER-002
func TestTokenPositionsAreByteOffsets(t *testing.T) {
	runLexCases(t, []lexCase{
		{"positions", `PRINT "A";`, []token.Token{
			tok(token.Print, "PRINT", 0), tok(token.String, "A", 6), tok(token.Semicolon, ";", 9), eol(10),
		}},
		{"multi-byte string contents", `"é":`, []token.Token{
			tok(token.String, "é", 0), tok(token.Colon, ":", 4), eol(5),
		}},
	})
}

// @spec LEXER-003
func TestSpacesAndTabsOutsideStringsAreSkipped(t *testing.T) {
	runLexCases(t, []lexCase{
		{"leading, inner, trailing whitespace", "  PRINT\t\"A\"  ", []token.Token{
			tok(token.Print, "PRINT", 2), tok(token.String, "A", 8), eol(13),
		}},
		{"only whitespace", " \t ", []token.Token{eol(3)}},
	})
}

// @spec LEXER-004
func TestClosedStringLiteral(t *testing.T) {
	runLexCases(t, []lexCase{
		{"closed string then colon", `"HELLO":`, []token.Token{
			tok(token.String, "HELLO", 0), tok(token.Colon, ":", 7), eol(8),
		}},
		{"empty string", `""`, []token.Token{tok(token.String, "", 0), eol(2)}},
		{"adjacent strings", `"A""B"`, []token.Token{
			tok(token.String, "A", 0), tok(token.String, "B", 3), eol(6),
		}},
	})
}

// @spec LEXER-005
func TestUnclosedStringLiteralRunsToEndOfLine(t *testing.T) {
	runLexCases(t, []lexCase{
		{"unclosed with trailing spaces", `PRINT "HI  `, []token.Token{
			tok(token.Print, "PRINT", 0), tok(token.String, "HI  ", 6), eol(11),
		}},
		{"lone quote", `"`, []token.Token{tok(token.String, "", 0), eol(1)}},
	})
}

// @spec LEXER-006
func TestStringContentsAreKeptExactly(t *testing.T) {
	contents := "a: ;,+?\tb PRINT print é\xff"
	runLexCases(t, []lexCase{
		{"special characters inside string", `"` + contents + `"`, []token.Token{
			tok(token.String, contents, 0), eol(len(contents) + 2),
		}},
	})
}

// @spec LEXER-007
func TestPrintKeywordNeedsNoFollowingSpace(t *testing.T) {
	runLexCases(t, []lexCase{
		{"PRINT then string", `PRINT"X"`, []token.Token{
			tok(token.Print, "PRINT", 0), tok(token.String, "X", 5), eol(8),
		}},
		{"PRINT then letter", `PRINTX`, []token.Token{
			tok(token.Print, "PRINT", 0), tok(token.Illegal, "X", 5), eol(6),
		}},
		{"PRINT twice", `PRINTPRINT`, []token.Token{
			tok(token.Print, "PRINT", 0), tok(token.Print, "PRINT", 5), eol(10),
		}},
	})
}

// @spec LEXER-008
func TestQuestionMarkIsPrint(t *testing.T) {
	runLexCases(t, []lexCase{
		{"? then string", `?"A"`, []token.Token{
			tok(token.Print, "?", 0), tok(token.String, "A", 1), eol(4),
		}},
	})
}

// @spec LEXER-009
func TestKeywordsAreUppercaseOnly(t *testing.T) {
	runLexCases(t, []lexCase{
		{"lowercase", `print`, []token.Token{
			tok(token.Illegal, "p", 0), tok(token.Illegal, "r", 1), tok(token.Illegal, "i", 2),
			tok(token.Illegal, "n", 3), tok(token.Illegal, "t", 4), eol(5),
		}},
		{"mixed case", `Print`, []token.Token{
			tok(token.Illegal, "P", 0), tok(token.Illegal, "r", 1), tok(token.Illegal, "i", 2),
			tok(token.Illegal, "n", 3), tok(token.Illegal, "t", 4), eol(5),
		}},
		{"lowercase rem", `rem`, []token.Token{
			tok(token.Illegal, "r", 0), tok(token.Illegal, "e", 1), tok(token.Illegal, "m", 2), eol(3),
		}},
	})
}

// @spec LEXER-010
func TestWhitespaceInsideKeywordBreaksIt(t *testing.T) {
	runLexCases(t, []lexCase{
		{"PR INT", `PR INT`, []token.Token{
			tok(token.Illegal, "P", 0), tok(token.Illegal, "R", 1), tok(token.Illegal, "I", 3),
			tok(token.Illegal, "N", 4), tok(token.Illegal, "T", 5), eol(6),
		}},
	})
}

// @spec LEXER-011
func TestPunctuation(t *testing.T) {
	runLexCases(t, []lexCase{
		{"all punctuation", `:;,+`, []token.Token{
			tok(token.Colon, ":", 0), tok(token.Semicolon, ";", 1), tok(token.Comma, ",", 2),
			tok(token.Plus, "+", 3), eol(4),
		}},
	})
}

// @spec LEXER-012
func TestUnrecognizedCharactersAreIllegal(t *testing.T) {
	runLexCases(t, []lexCase{
		{"digit, accented letter, non-breaking space", "1é ", []token.Token{
			tok(token.Illegal, "1", 0), tok(token.Illegal, "é", 1), tok(token.Illegal, " ", 3), eol(5),
		}},
		{"symbols", `@#`, []token.Token{
			tok(token.Illegal, "@", 0), tok(token.Illegal, "#", 1), eol(2),
		}},
		{"form feed", "\f", []token.Token{tok(token.Illegal, "\f", 0), eol(1)}},
	})
}

// @spec LEXER-013
func TestInvalidUTF8BytesAreIllegal(t *testing.T) {
	runLexCases(t, []lexCase{
		{"invalid bytes", "\xff\xfe", []token.Token{
			tok(token.Illegal, "\xff", 0), tok(token.Illegal, "\xfe", 1), eol(2),
		}},
	})
}

// @spec LEXER-014
func TestRemTakesRestOfLine(t *testing.T) {
	runLexCases(t, []lexCase{
		{"comment", `REM HELLO`, []token.Token{tok(token.Rem, " HELLO", 0), eol(9)}},
		{"bare REM", `REM`, []token.Token{tok(token.Rem, "", 0), eol(3)}},
		{"no space after REM", `REMARK`, []token.Token{tok(token.Rem, "ARK", 0), eol(6)}},
		{"colon, quote, and keyword in comment", `REM A:PRINT "X"`, []token.Token{
			tok(token.Rem, ` A:PRINT "X"`, 0), eol(15),
		}},
		{"after a statement", `PRINT "A":REM X`, []token.Token{
			tok(token.Print, "PRINT", 0), tok(token.String, "A", 6), tok(token.Colon, ":", 9),
			tok(token.Rem, " X", 10), eol(15),
		}},
		{"after print items", `PRINT "A" REM X`, []token.Token{
			tok(token.Print, "PRINT", 0), tok(token.String, "A", 6), tok(token.Rem, " X", 10), eol(15),
		}},
		{"trailing spaces, tab, CR, invalid bytes", "REM \t\r\xffé  ", []token.Token{
			tok(token.Rem, " \t\r\xffé  ", 0), eol(11),
		}},
		{"REM inside a string is text", `PRINT "REM"`, []token.Token{
			tok(token.Print, "PRINT", 0), tok(token.String, "REM", 6), eol(11),
		}},
	})
}
