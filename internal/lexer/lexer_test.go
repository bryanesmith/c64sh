package lexer

import (
	"errors"
	"reflect"
	"testing"

	"github.com/bryanesmith/c64sh/internal/basicerr"
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
			tok(token.Print, "PRINT", 0), tok(token.Name, "X", 5), eol(6),
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
			tok(token.Name, "P", 0), tok(token.Illegal, "r", 1), tok(token.Illegal, "i", 2),
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
		{"PRI NT", `PRI NT`, []token.Token{
			tok(token.Name, "PRINT", 0), eol(6),
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
		{"lowercase letter, accented letter, non-breaking space", "xé\u00a0", []token.Token{
			tok(token.Illegal, "x", 0), tok(token.Illegal, "é", 1), tok(token.Illegal, "\u00a0", 3), eol(5),
		}},
		{"symbols", `@&`, []token.Token{
			tok(token.Illegal, "@", 0), tok(token.Illegal, "&", 1), eol(2),
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
		{"followed by a letter", `1A`, []token.Token{num("1", 0), tok(token.Name, "A", 1), eol(2)}},
		{"second E ends the number", `1E5E5`, []token.Token{
			num("1E5", 0), tok(token.Name, "E5", 3), eol(5),
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

// @spec LEXER-019
func TestCaret(t *testing.T) {
	runLexCases(t, []lexCase{
		{"caret", `2^3`, []token.Token{num("2", 0), tok(token.Caret, "^", 1), num("3", 2), eol(3)}},
		{"up arrow", "2\u21913", []token.Token{num("2", 0), tok(token.Caret, "\u2191", 1), num("3", 4), eol(5)}},
		{"up arrow in a string is text", "\"\u2191\"", []token.Token{tok(token.String, "\u2191", 0), eol(5)}},
		{"negative exponent", `2^-1`, []token.Token{num("2", 0), tok(token.Caret, "^", 1), tok(token.Minus, "-", 2), num("1", 3), eol(4)}},
	})
}

func name(v string, pos int) token.Token { return tok(token.Name, v, pos) }

// @spec LEXER-020
func TestNames(t *testing.T) {
	runLexCases(t, []lexCase{
		{"single letter", `A`, []token.Token{name("A", 0), eol(1)}},
		{"long name", `HEIGHT`, []token.Token{name("HEIGHT", 0), eol(6)}},
		{"letters and digits", `A1B2`, []token.Token{name("A1B2", 0), eol(4)}},
		{"string variable", `N$`, []token.Token{name("N$", 0), eol(2)}},
		{"spaces inside are skipped", `A B`, []token.Token{name("AB", 0), eol(3)}},
		{"space before $", `N $`, []token.Token{name("N$", 0), eol(3)}},
		{"integer variable", `C%`, []token.Token{name("C%", 0), eol(2)}},
		{"space before %", `COUNT %`, []token.Token{name("COUNT%", 0), eol(7)}},
		{"% ends the name", `A%B`, []token.Token{name("A%", 0), name("B", 2), eol(3)}},
		{"% alone is illegal", `5%`, []token.Token{num("5", 0), tok(token.Illegal, "%", 1), eol(2)}},
		{"assignment", `A=5`, []token.Token{name("A", 0), tok(token.Equal, "=", 1), num("5", 2), eol(3)}},
		{"in PRINT", `PRINT A;B$`, []token.Token{
			tok(token.Print, "PRINT", 0), name("A", 6), tok(token.Semicolon, ";", 7), name("B$", 8), eol(10),
		}},
		{"name then string", `A"X"`, []token.Token{name("A", 0), tok(token.String, "X", 1), eol(4)}},
		{"$ ends the name", `A$B`, []token.Token{name("A$", 0), name("B", 2), eol(3)}},
		{"trailing spaces excluded", `A  :`, []token.Token{name("A", 0), tok(token.Colon, ":", 3), eol(4)}},
		{"lowercase is not a name", `a`, []token.Token{tok(token.Illegal, "a", 0), eol(1)}},
	})
}

// @spec LEXER-021
func TestKeywordEndsName(t *testing.T) {
	runLexCases(t, []lexCase{
		{"LET inside a name", `OUTLET`, []token.Token{name("OUT", 0), tok(token.Let, "LET", 3), eol(6)}},
		{"REM inside a name", `PREMIUM`, []token.Token{name("P", 0), tok(token.Rem, "IUM", 1), eol(7)}},
		{"PRINT inside a name", `APRINT`, []token.Token{name("A", 0), tok(token.Print, "PRINT", 1), eol(6)}},
		{"keyword after a space", `A PRINT`, []token.Token{name("A", 0), tok(token.Print, "PRINT", 2), eol(7)}},
		{"keyword first", `PRINTER`, []token.Token{tok(token.Print, "PRINT", 0), name("ER", 5), eol(7)}},
		{"LET first", `LETTER`, []token.Token{tok(token.Let, "LET", 0), name("TER", 3), eol(6)}},
	})
}

// @spec LEXER-022
func TestLetAndEqual(t *testing.T) {
	runLexCases(t, []lexCase{
		{"LET statement", `LET A=1`, []token.Token{
			tok(token.Let, "LET", 0), name("A", 4), tok(token.Equal, "=", 5), num("1", 6), eol(7),
		}},
		{"lowercase let", `let`, []token.Token{
			tok(token.Illegal, "l", 0), tok(token.Illegal, "e", 1), tok(token.Illegal, "t", 2), eol(3),
		}},
	})
}

// @spec LEXER-023
func TestComparisonSymbols(t *testing.T) {
	runLexCases(t, []lexCase{
		{"less", `1<2`, []token.Token{num("1", 0), tok(token.Less, "<", 1), num("2", 2), eol(3)}},
		{"not equal is two tokens", `<>`, []token.Token{tok(token.Less, "<", 0), tok(token.Greater, ">", 1), eol(2)}},
		{"all three", `<=>`, []token.Token{tok(token.Less, "<", 0), tok(token.Equal, "=", 1), tok(token.Greater, ">", 2), eol(3)}},
		{"inside a string", `"<>"`, []token.Token{tok(token.String, "<>", 0), eol(4)}},
	})
}

// @spec LEXER-024
func TestLogicKeywords(t *testing.T) {
	runLexCases(t, []lexCase{
		{"AND", `1AND2`, []token.Token{num("1", 0), tok(token.And, "AND", 1), num("2", 4), eol(5)}},
		{"OR and NOT", `A OR NOT B`, []token.Token{
			name("A", 0), tok(token.Or, "OR", 2), tok(token.Not, "NOT", 5), name("B", 9), eol(10),
		}},
		{"OR inside a name", `SCORE`, []token.Token{name("SC", 0), tok(token.Or, "OR", 2), name("E", 4), eol(5)}},
		{"NOT inside a name", `NOTE`, []token.Token{tok(token.Not, "NOT", 0), name("E", 3), eol(4)}},
		{"AND inside a name", `BAND`, []token.Token{name("B", 0), tok(token.And, "AND", 1), eol(4)}},
	})
}

// @spec LEXER-025
func TestIfThenKeywords(t *testing.T) {
	runLexCases(t, []lexCase{
		{"IF THEN", `IFA=1THEN`, []token.Token{
			tok(token.If, "IF", 0), name("A", 2), tok(token.Equal, "=", 3), num("1", 4), tok(token.Then, "THEN", 5), eol(9),
		}},
		{"THEN ends a name", `IF A THEN`, []token.Token{tok(token.If, "IF", 0), name("A", 3), tok(token.Then, "THEN", 5), eol(9)}},
		{"IF inside a name", `DIFF`, []token.Token{name("D", 0), tok(token.If, "IF", 1), name("F", 3), eol(4)}},
	})
}

// @spec LEXER-026
func TestProgramKeywords(t *testing.T) {
	runLexCases(t, []lexCase{
		{"RUN", "RUN", []token.Token{tok(token.Run, "RUN", 0), eol(3)}},
		{"RUN n", "RUN 20", []token.Token{tok(token.Run, "RUN", 0), num("20", 4), eol(6)}},
		{"LIST", "LIST", []token.Token{tok(token.List, "LIST", 0), eol(4)}},
		{"NEW", "NEW", []token.Token{tok(token.New, "NEW", 0), eol(3)}},
		{"END", "END", []token.Token{tok(token.End, "END", 0), eol(3)}},
		{"keyword in a name", "FRIEND", []token.Token{name("FRI", 0), tok(token.End, "END", 3), eol(6)}},
		{"lowercase", "run", []token.Token{
			tok(token.Illegal, "r", 0), tok(token.Illegal, "u", 1), tok(token.Illegal, "n", 2), eol(3),
		}},
	})
}

// @spec LEXER-027
func TestLineNumber(t *testing.T) {
	cases := []struct {
		in   string
		n    int
		rest string
	}{
		{`10 PRINT "A"`, 10, `PRINT "A"`},
		{`10PRINT"A"`, 10, `PRINT"A"`},
		{"  10   PRINT", 10, "PRINT"},
		{"\t10\tPRINT", 10, "PRINT"},
		{"1 0X", 10, "X"},
		{"10.5", 10, ".5"},
		{"1E3", 1, "E3"},
		{"5+5", 5, "+5"},
		{"010", 10, ""},
		{"10", 10, ""},
		{"10   ", 10, ""},
		{"0", 0, ""},
		{"63999 END", 63999, "END"},
		{`10 PRINT "A  "  `, 10, `PRINT "A  "  `},
	}
	for _, c := range cases {
		n, rest, ok, err := LineNumber(c.in)
		if n != c.n || rest != c.rest || !ok || err != nil {
			t.Errorf("LineNumber(%q) = %d, %q, %v, %v; want %d, %q, true, nil", c.in, n, rest, ok, err, c.n, c.rest)
		}
	}
}

// @spec LEXER-028
func TestNoLineNumber(t *testing.T) {
	for _, in := range []string{"", "   ", `PRINT 10`, ".5", "-10", "A10", `"10"`} {
		if _, _, ok, err := LineNumber(in); ok || err != nil {
			t.Errorf("LineNumber(%q): ok %v, err %v; want false, nil", in, ok, err)
		}
	}
}

// @spec LEXER-029
func TestLineNumberTooLarge(t *testing.T) {
	for _, in := range []string{"64000 PRINT", "6400 0", "99999", "100000000000000000000000"} {
		_, _, ok, err := LineNumber(in)
		var be *basicerr.Error
		if !ok || !errors.As(err, &be) || be.Kind != basicerr.Syntax {
			t.Errorf("LineNumber(%q): ok %v, err %v; want true, a SYNTAX error", in, ok, err)
		}
	}
}

// @spec LEXER-030
func TestGotoKeywords(t *testing.T) {
	runLexCases(t, []lexCase{
		{"GOTO", "GOTO 10", []token.Token{tok(token.Goto, "GOTO", 0), num("10", 5), eol(7)}},
		{"GO TO", "GO TO 10", []token.Token{tok(token.Go, "GO", 0), tok(token.To, "TO", 3), num("10", 6), eol(8)}},
		{"GOTO without spaces", "GOTO10", []token.Token{tok(token.Goto, "GOTO", 0), num("10", 4), eol(6)}},
		{"GO in a name", "GOLD", []token.Token{tok(token.Go, "GO", 0), name("LD", 2), eol(4)}},
		{"TO in a name", "TOTAL", []token.Token{tok(token.To, "TO", 0), name("TAL", 2), eol(5)}},
	})
}

// @spec LEXER-031
func TestLoopKeywords(t *testing.T) {
	runLexCases(t, []lexCase{
		{"FOR TO STEP", "FORI=1TO9STEP2", []token.Token{
			tok(token.For, "FOR", 0), name("I", 3), tok(token.Equal, "=", 4), num("1", 5),
			tok(token.To, "TO", 6), num("9", 8), tok(token.Step, "STEP", 9), num("2", 13), eol(14),
		}},
		{"NEXT", "NEXT I", []token.Token{tok(token.Next, "NEXT", 0), name("I", 5), eol(6)}},
		{"FOR in a name", "FORM", []token.Token{tok(token.For, "FOR", 0), name("M", 3), eol(4)}},
	})
}

// @spec LEXER-032
func TestSubroutineKeywords(t *testing.T) {
	runLexCases(t, []lexCase{
		{"GOSUB", "GOSUB 100", []token.Token{tok(token.Gosub, "GOSUB", 0), num("100", 6), eol(9)}},
		{"RETURN", "RETURN", []token.Token{tok(token.Return, "RETURN", 0), eol(6)}},
		{"GO SUB", "GO SUB", []token.Token{tok(token.Go, "GO", 0), name("SUB", 3), eol(6)}},
	})
}

// @spec LEXER-033
func TestInputKeywords(t *testing.T) {
	runLexCases(t, []lexCase{
		{"INPUT", `INPUT"N";A`, []token.Token{
			tok(token.Input, "INPUT", 0), tok(token.String, "N", 5), tok(token.Semicolon, ";", 8), name("A", 9), eol(10),
		}},
		{"GET", "GET K$", []token.Token{tok(token.Get, "GET", 0), name("K$", 4), eol(6)}},
		{"GET in a name", "TARGET", []token.Token{name("TAR", 0), tok(token.Get, "GET", 3), eol(6)}},
	})
}

// @spec LEXER-034
func TestFunctionKeywords(t *testing.T) {
	runLexCases(t, []lexCase{
		{"DEF FN", "DEFFNSQ(X)", []token.Token{
			tok(token.Def, "DEF", 0), tok(token.Fn, "FN", 3), name("SQ", 5), tok(token.LParen, "(", 7), name("X", 8), tok(token.RParen, ")", 9), eol(10),
		}},
		{"FN in an expression", "FN A(1)", []token.Token{tok(token.Fn, "FN", 0), name("A", 3), tok(token.LParen, "(", 4), num("1", 5), tok(token.RParen, ")", 6), eol(7)}},
	})
}

// @spec LEXER-035
func TestFileKeywords(t *testing.T) {
	runLexCases(t, []lexCase{
		{"LOAD", `LOAD"X",8`, []token.Token{tok(token.Load, "LOAD", 0), tok(token.String, "X", 4), tok(token.Comma, ",", 7), num("8", 8), eol(9)}},
		{"SAVE", "SAVE", []token.Token{tok(token.Save, "SAVE", 0), eol(4)}},
		{"VERIFY", "VERIFY", []token.Token{tok(token.Verify, "VERIFY", 0), eol(6)}},
	})
}

// @spec LEXER-036
func TestDataFileKeywords(t *testing.T) {
	runLexCases(t, []lexCase{
		{"PRINT#", `PRINT#1,"X"`, []token.Token{tok(token.PrintFile, "PRINT#", 0), num("1", 6), tok(token.Comma, ",", 7), tok(token.String, "X", 8), eol(11)}},
		{"PRINT #", "PRINT #1", []token.Token{tok(token.Print, "PRINT", 0), tok(token.Hash, "#", 6), num("1", 7), eol(8)}},
		{"INPUT#", "INPUT#1,A", []token.Token{tok(token.InputFile, "INPUT#", 0), num("1", 6), tok(token.Comma, ",", 7), name("A", 8), eol(9)}},
		{"GET#", "GET#1", []token.Token{tok(token.Get, "GET", 0), tok(token.Hash, "#", 3), num("1", 4), eol(5)}},
		{"OPEN CLOSE CMD", "OPEN CLOSE CMD", []token.Token{tok(token.Open, "OPEN", 0), tok(token.Close, "CLOSE", 5), tok(token.Cmd, "CMD", 11), eol(14)}},
	})
}

// @spec LEXER-037
func TestOnKeyword(t *testing.T) {
	runLexCases(t, []lexCase{
		{"ON GOTO", "ON X GOTO 10,20", []token.Token{
			tok(token.On, "ON", 0), name("X", 3), tok(token.Goto, "GOTO", 5), num("10", 10), tok(token.Comma, ",", 12), num("20", 13), eol(15),
		}},
		{"ON in a name", "MONEY", []token.Token{name("M", 0), tok(token.On, "ON", 1), name("EY", 3), eol(5)}},
	})
}

// @spec LEXER-038
func TestNumberFunctionKeywords(t *testing.T) {
	for _, f := range []string{"ABS", "INT", "SGN", "SQR", "RND", "LOG", "EXP", "SIN", "COS", "TAN", "ATN"} {
		runLexCases(t, []lexCase{{f, f + "(1)", []token.Token{
			tok(token.Function, f, 0), tok(token.LParen, "(", 3), num("1", 4), tok(token.RParen, ")", 5), eol(6),
		}}})
	}
	runLexCases(t, []lexCase{
		{"pi", "2*π", []token.Token{num("2", 0), tok(token.Star, "*", 1), tok(token.Pi, "π", 2), eol(4)}},
		{"INT in a name", "POINT", []token.Token{name("PO", 0), tok(token.Function, "INT", 2), eol(5)}},
	})
}

// @spec LEXER-039
func TestStringFunctionKeywords(t *testing.T) {
	for _, f := range []string{"LEN", "LEFT$", "RIGHT$", "MID$", "CHR$", "ASC", "STR$", "VAL"} {
		runLexCases(t, []lexCase{{f, f + "(", []token.Token{tok(token.Function, f, 0), tok(token.LParen, "(", len(f)), eol(len(f) + 1)}}})
	}
	runLexCases(t, []lexCase{
		{"LEFT without $", "LEFT", []token.Token{name("LEFT", 0), eol(4)}},
		{"VAL in a name", "VALUE", []token.Token{tok(token.Function, "VAL", 0), name("UE", 3), eol(5)}},
	})
}

// @spec LEXER-040
func TestPrintFormattingKeywords(t *testing.T) {
	runLexCases(t, []lexCase{
		{"TAB(", "TAB(5)", []token.Token{tok(token.Tab, "TAB(", 0), num("5", 4), tok(token.RParen, ")", 5), eol(6)}},
		{"SPC(", "SPC(2)", []token.Token{tok(token.Spc, "SPC(", 0), num("2", 4), tok(token.RParen, ")", 5), eol(6)}},
		{"TAB with a space", "TAB (5)", []token.Token{name("TAB", 0), tok(token.LParen, "(", 4), num("5", 5), tok(token.RParen, ")", 6), eol(7)}},
		{"POS", "POS(0)", []token.Token{tok(token.Function, "POS", 0), tok(token.LParen, "(", 3), num("0", 4), tok(token.RParen, ")", 5), eol(6)}},
	})
}

// @spec LEXER-041
func TestDimKeyword(t *testing.T) {
	runLexCases(t, []lexCase{{"DIM", "DIM A(9)", []token.Token{
		tok(token.Dim, "DIM", 0), name("A", 4), tok(token.LParen, "(", 5), num("9", 6), tok(token.RParen, ")", 7), eol(8),
	}}})
}

// @spec LEXER-042
func TestReservedKeywords(t *testing.T) {
	for _, k := range []string{"STOP", "CONT", "CLR", "PEEK", "POKE", "SYS", "WAIT", "USR"} {
		runLexCases(t, []lexCase{{k, k, []token.Token{tok(token.Reserved, k, 0), eol(len(k))}}})
	}
	runLexCases(t, []lexCase{{"in a name", "STOPS", []token.Token{tok(token.Reserved, "STOP", 0), name("S", 4), eol(5)}}})
}

// @spec LEXER-043
func TestDataText(t *testing.T) {
	data := func(v string, pos int) token.Token { return tok(token.Data, v, pos) }
	runLexCases(t, []lexCase{
		{"items", "DATA 1,2,ABC", []token.Token{data(" 1,2,ABC", 0), eol(12)}},
		{"keywords are text", "DATA PRINT,TO", []token.Token{data(" PRINT,TO", 0), eol(13)}},
		{"colon ends it", `DATA 1,2:PRINT"X"`, []token.Token{data(" 1,2", 0), tok(token.Colon, ":", 8), tok(token.Print, "PRINT", 9), tok(token.String, "X", 14), eol(17)}},
		{"colon in quotes", `DATA "A:B",C`, []token.Token{data(` "A:B",C`, 0), eol(12)}},
		{"trailing spaces removed", "DATA A, B   ", []token.Token{data(" A, B", 0), eol(12)}},
		{"no space", "DATA1", []token.Token{data("1", 0), eol(5)}},
	})
}

// @spec LEXER-045
func TestEnvironKeywords(t *testing.T) {
	runLexCases(t, []lexCase{
		{"ENVIRON$", `ENVIRON$("HOME")`, []token.Token{tok(token.Function, "ENVIRON$", 0), tok(token.LParen, "(", 8), tok(token.String, "HOME", 9), tok(token.RParen, ")", 15), eol(16)}},
		{"ENVIRON", `ENVIRON"A=1"`, []token.Token{tok(token.Environ, "ENVIRON", 0), tok(token.String, "A=1", 7), eol(12)}},
		{"ENVIRON then a name", "ENVIRONX", []token.Token{tok(token.Environ, "ENVIRON", 0), name("X", 7), eol(8)}},
	})
}

// @spec LEXER-018
func TestKeywordEndsNumberAtE(t *testing.T) {
	runLexCases(t, []lexCase{
		{"END after a number", "1END", []token.Token{num("1", 0), tok(token.End, "END", 1), eol(4)}},
		{"EXP after a number", "2EXP(0)", []token.Token{num("2", 0), tok(token.Function, "EXP", 1), tok(token.LParen, "(", 4), num("0", 5), tok(token.RParen, ")", 6), eol(7)}},
		{"an exponent still works", "1E2", []token.Token{num("1E2", 0), eol(3)}},
	})
}

// @spec LEXER-046
func TestFreKeyword(t *testing.T) {
	runLexCases(t, []lexCase{
		{"FRE", "FRE(0)", []token.Token{tok(token.Function, "FRE", 0), tok(token.LParen, "(", 3), num("0", 4), tok(token.RParen, ")", 5), eol(6)}},
		{"FRE in a name", "FREE", []token.Token{tok(token.Function, "FRE", 0), name("E", 3), eol(4)}},
	})
}

// @spec LEXER-044
func TestReadRestoreKeywords(t *testing.T) {
	runLexCases(t, []lexCase{
		{"READ", "READ A", []token.Token{tok(token.Read, "READ", 0), name("A", 5), eol(6)}},
		{"RESTORE", "RESTORE", []token.Token{tok(token.Restore, "RESTORE", 0), eol(7)}},
		{"READ in a name", "READY", []token.Token{tok(token.Read, "READ", 0), name("Y", 4), eol(5)}},
	})
}
