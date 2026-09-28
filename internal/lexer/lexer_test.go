package lexer

import (
	"reflect"
	"slices"
	"testing"
	"unicode"

	"github.com/bryanesmith/c64sh/grammar"
	"github.com/bryanesmith/c64sh/internal/token"
	"golang.org/x/exp/ebnf"
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

// literalTokens adds to out every literal token in e, written with quotes.
func literalTokens(e ebnf.Expression, out map[string]bool) {
	switch e := e.(type) {
	case nil:
	case ebnf.Alternative:
		for _, x := range e {
			literalTokens(x, out)
		}
	case ebnf.Sequence:
		for _, x := range e {
			literalTokens(x, out)
		}
	case *ebnf.Group:
		literalTokens(e.Body, out)
	case *ebnf.Option:
		literalTokens(e.Body, out)
	case *ebnf.Repetition:
		literalTokens(e.Body, out)
	case *ebnf.Token:
		out[`"`+e.String+`"`] = true
	}
}

// @spec GRAMMAR-009
func TestTokenTableMatchesGrammar(t *testing.T) {
	g, err := grammar.Load()
	if err != nil {
		t.Fatalf("grammar.Load() error: %v", err)
	}
	items := map[string]bool{}
	for name, prod := range g {
		if unicode.IsLower(rune(name[0])) {
			items[name] = true
		} else {
			literalTokens(prod.Expr, items)
		}
	}
	for item := range items {
		_, isToken := grammarTokens[item]
		if !isToken && !grammarHelpers[item] {
			t.Errorf("grammar item %s has no entry in the lexer's token table", item)
		}
	}
	var tableItems []string
	for item := range grammarTokens {
		tableItems = append(tableItems, item)
	}
	for item := range grammarHelpers {
		tableItems = append(tableItems, item)
	}
	slices.Sort(tableItems)
	for _, item := range tableItems {
		if !items[item] {
			t.Errorf("lexer token table entry %s names no lexical rule or literal token in the grammar", item)
		}
	}
	if len(tableItems) == 0 {
		t.Errorf("lexer token table is empty")
	}
}
