// Package lexer turns a line of C64 BASIC into tokens.
package lexer

import (
	"strings"
	"unicode/utf8"

	"github.com/bryanesmith/c64sh/internal/token"
)

// grammarTokens maps each lexical grammar rule, and each literal token used
// in a syntactic grammar rule (written with its quotes), to its token kind.
//
// @spec GRAMMAR-009
var grammarTokens = map[string]token.Kind{
	"print":  token.Print,
	"rem":    token.Rem,
	"string": token.String,
	`":"`:    token.Colon,
	`";"`:    token.Semicolon,
	`","`:    token.Comma,
	`"+"`:    token.Plus,
}

// grammarHelpers lists lexical grammar rules that are used only inside
// other lexical rules and produce no token of their own.
var grammarHelpers = map[string]bool{
	"character": true,
}

// keywords are matched case-sensitively, at any position outside a string,
// whatever follows them.
var keywords = []struct {
	text string
	kind token.Kind
}{
	{"PRINT", token.Print},
	{"REM", token.Rem},
}

// symbols maps single-character tokens to their kinds.
var symbols = map[byte]token.Kind{
	'?': token.Print,
	':': token.Colon,
	';': token.Semicolon,
	',': token.Comma,
	'+': token.Plus,
}

// Lex returns the tokens of line. The last token is always EOL.
//
// @spec LEXER-001, LEXER-002, LEXER-003, LEXER-004, LEXER-005, LEXER-006, LEXER-007
// @spec LEXER-008, LEXER-009, LEXER-010, LEXER-011, LEXER-012, LEXER-013, LEXER-014
func Lex(line string) []token.Token {
	var toks []token.Token
	emit := func(k token.Kind, value string, pos int) {
		toks = append(toks, token.Token{Kind: k, Value: value, Pos: pos})
	}
	for i := 0; i < len(line); {
		c := line[i]
		if c == ' ' || c == '\t' {
			i++
			continue
		}
		if c == '"' {
			body := line[i+1:]
			end := strings.IndexByte(body, '"')
			if end < 0 {
				emit(token.String, body, i) // unclosed: runs to end of line
				i = len(line)
			} else {
				emit(token.String, body[:end], i)
				i += end + 2
			}
			continue
		}
		if text, kind, ok := matchKeyword(line[i:]); ok {
			if kind == token.Rem {
				// A comment runs to the end of the line, untokenized.
				emit(kind, line[i+len(text):], i)
				i = len(line)
				continue
			}
			emit(kind, text, i)
			i += len(text)
			continue
		}
		if kind, ok := symbols[c]; ok {
			emit(kind, line[i:i+1], i)
			i++
			continue
		}
		// One UTF-8 character, or one byte if it is not valid UTF-8.
		_, size := utf8.DecodeRuneInString(line[i:])
		emit(token.Illegal, line[i:i+size], i)
		i += size
	}
	emit(token.EOL, "", len(line))
	return toks
}

// matchKeyword returns the longest keyword that s begins with.
func matchKeyword(s string) (string, token.Kind, bool) {
	best := -1
	for i, kw := range keywords {
		if strings.HasPrefix(s, kw.text) && (best < 0 || len(kw.text) > len(keywords[best].text)) {
			best = i
		}
	}
	if best < 0 {
		return "", 0, false
	}
	return keywords[best].text, keywords[best].kind, true
}
