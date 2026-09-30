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
		{"all punctuation", `:;,+-*/()`, []token.Token{
			tok(token.Colon, ":", 0), tok(token.Semicolon, ";", 1), tok(token.Comma, ",", 2),
			tok(token.Plus, "+", 3), tok(token.Minus, "-", 4), tok(token.Star, "*", 5),
			tok(token.Slash, "/", 6), tok(token.LParen, "(", 7), tok(token.RParen, ")", 8), eol(9),
		}},
		{"minus between numbers", `1-2`, []token.Token{
			tok(token.Number, "1", 0), tok(token.Minus, "-", 1), tok(token.Number, "2", 2), eol(3),
		}},
		{"minus in an exponent", `1E-2`, []token.Token{tok(token.Number, "1E-2", 0), eol(4)}},
		{"expression", `(2+3)*4/-5`, []token.Token{
			tok(token.LParen, "(", 0), tok(token.Number, "2", 1), tok(token.Plus, "+", 2), tok(token.Number, "3", 3),
			tok(token.RParen, ")", 4), tok(token.Star, "*", 5), tok(token.Number, "4", 6), tok(token.Slash, "/", 7),
			tok(token.Minus, "-", 8), tok(token.Number, "5", 9), eol(10),
		}},
	})
}

// @spec LEXER-012
func TestUnrecognizedCharactersAreIllegal(t *testing.T) {
	runLexCases(t, []lexCase{
		{"letter, accented letter, non-breaking space", "Xé\u00a0", []token.Token{
			tok(token.Illegal, "X", 0), tok(token.Illegal, "é", 1), tok(token.Illegal, "\u00a0", 3), eol(5),
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

func num(v string, pos int) token.Token { return tok(token.Number, v, pos) }

// @spec LEXER-015
func TestNumberLiterals(t *testing.T) {
	runLexCases(t, []lexCase{
		{"integer", `45`, []token.Token{num("45", 0), eol(2)}},
		{"decimal", `3.14`, []token.Token{num("3.14", 0), eol(4)}},
		{"leading point", `.5`, []token.Token{num(".5", 0), eol(2)}},
		{"trailing point", `5.`, []token.Token{num("5.", 0), eol(2)}},
		{"lone point", `.`, []token.Token{num(".", 0), eol(1)}},
		{"exponent", `1E3`, []token.Token{num("1E3", 0), eol(3)}},
		{"signed exponents", `1.5E-3:2E+4`, []token.Token{num("1.5E-3", 0), tok(token.Colon, ":", 6), num("2E+4", 7), eol(11)}},
		{"exponent without digits", `1E`, []token.Token{num("1E", 0), eol(2)}},
		{"exponent sign without digits", `1E+`, []token.Token{num("1E+", 0), eol(3)}},
		{"leading zeros", `007`, []token.Token{num("007", 0), eol(3)}},
		{"in PRINT", `PRINT 5;"A"`, []token.Token{
			tok(token.Print, "PRINT", 0), num("5", 6), tok(token.Semicolon, ";", 7), tok(token.String, "A", 8), eol(11),
		}},
		{"after a string", `"A"1`, []token.Token{tok(token.String, "A", 0), num("1", 3), eol(4)}},
		{"followed by a letter", `1A`, []token.Token{num("1", 0), tok(token.Illegal, "A", 1), eol(2)}},
		{"second E ends the number", `1E5E5`, []token.Token{
			num("1E5", 0), tok(token.Illegal, "E", 3), num("5", 4), eol(5),
		}},
	})
}

// @spec LEXER-016
func TestSpacesInsideNumbersAreSkipped(t *testing.T) {
	runLexCases(t, []lexCase{
		{"digits", `1 2`, []token.Token{num("12", 0), eol(3)}},
		{"every part", "1 . 5\tE - 3", []token.Token{num("1.5E-3", 0), eol(11)}},
		{"trailing whitespace excluded", `PRINT 1  ;`, []token.Token{
			tok(token.Print, "PRINT", 0), num("1", 6), tok(token.Semicolon, ";", 9), eol(10),
		}},
		{"space then keyword", `1 REM X`, []token.Token{num("1", 0), tok(token.Rem, " X", 2), eol(7)}},
	})
}

// @spec LEXER-017
func TestSecondPointStartsANewNumber(t *testing.T) {
	runLexCases(t, []lexCase{
		{"two numbers", `1.2.3`, []token.Token{num("1.2", 0), num(".3", 3), eol(5)}},
		{"three numbers", `...`, []token.Token{num(".", 0), num(".", 1), num(".", 2), eol(3)}},
	})
}
