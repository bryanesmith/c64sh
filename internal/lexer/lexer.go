// Package lexer turns a line of C64 BASIC into tokens.
//
// Each token rule of the grammar is written in EBNF, in the notation of the
// Go language specification, above the code in Lex that scans it. The
// parser's rule comments are the rest of the grammar.
package lexer

import (
	"strings"
	"unicode/utf8"

	"github.com/bryanesmith/c64sh/internal/token"
)

// keywords are matched case-sensitively, at any position outside a string,
// whatever follows them.
var keywords = []struct {
	text string
	kind token.Kind
}{
	{"PRINT", token.Print},
	{"REM", token.Rem},
	{"LET", token.Let},
}

// symbols maps single-character tokens to their kinds.
var symbols = map[byte]token.Kind{
	'?': token.Print,
	':': token.Colon,
	';': token.Semicolon,
	',': token.Comma,
	'+': token.Plus,
	'-': token.Minus,
	'*': token.Star,
	'/': token.Slash,
	'(': token.LParen,
	')': token.RParen,
	'^': token.Caret,
	'=': token.Equal,
}

// upArrow is the C64's exponentiation key, read as "^".
const upArrow = "\u2191"

// Lex returns the tokens of line. The last token is always EOL.
//
// @spec LEXER-001, LEXER-002, LEXER-003, LEXER-004, LEXER-005, LEXER-006, LEXER-007
// @spec LEXER-008, LEXER-009, LEXER-010, LEXER-011, LEXER-012, LEXER-013, LEXER-014
// @spec LEXER-015, LEXER-016, LEXER-017, LEXER-018, LEXER-019, LEXER-020, LEXER-021
// @spec LEXER-022
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
		// string    = `"` { character } [ `"` ] .
		// character = /* any character except `"` and a line feed */ .
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
		// number = ( digit { digit } [ "." { digit } ] | "." { digit } )
		//          [ "E" [ "+" | "-" ] { digit } ] .   /* spaces inside are ignored */
		// digit  = "0" … "9" .
		if isDigit(c) || c == '.' {
			text, end := scanNumber(line, i)
			emit(token.Number, text, i)
			i = end
			continue
		}
		// print = "PRINT" | "?" .   ("?" is scanned with the symbols below)
		// rem   = "REM" { character | `"` } .
		// let   = "LET" .
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
		// name   = letter { letter | digit } [ "$" | "%" ] .   /* spaces inside are ignored; a keyword ends it */
		// letter = "A" … "Z" .
		if isLetter(c) {
			text, end := scanName(line, i)
			emit(token.Name, text, i)
			i = end
			continue
		}
		if strings.HasPrefix(line[i:], upArrow) {
			emit(token.Caret, upArrow, i)
			i += len(upArrow)
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

func isDigit(c byte) bool {
	return '0' <= c && c <= '9'
}

// scanNumber reads the number literal starting at line[start], which is a
// digit or ".". It returns the literal with its spaces and tabs removed, and
// the index just after its last character. As on a C64, spaces and tabs
// inside the literal are skipped, a second "." ends it, and an "E" that
// begins a keyword is left for the keyword.
func scanNumber(line string, start int) (string, int) {
	var text strings.Builder
	end := start
	// next returns the index of the next non-blank character from end.
	next := func() int {
		j := end
		for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
			j++
		}
		return j
	}
	take := func(j int) {
		text.WriteByte(line[j])
		end = j + 1
	}

	take(start)
	seenPoint := line[start] == '.'
	for {
		j := next()
		if j < len(line) && isDigit(line[j]) {
			take(j)
		} else if j < len(line) && line[j] == '.' && !seenPoint {
			seenPoint = true
			take(j)
		} else {
			break
		}
	}

	if j := next(); j < len(line) && line[j] == 'E' {
		if _, _, isKeyword := matchKeyword(line[j:]); !isKeyword {
			take(j)
			if k := next(); k < len(line) && (line[k] == '+' || line[k] == '-') {
				take(k)
			}
			for k := next(); k < len(line) && isDigit(line[k]); k = next() {
				take(k)
			}
		}
	}
	return text.String(), end
}

func isLetter(c byte) bool {
	return 'A' <= c && c <= 'Z'
}

// scanName reads the variable name starting at line[start], an uppercase
// letter where no keyword begins, with any "$" or "%" suffix. It returns
// the name with its spaces and
// tabs removed, and the index just after its last character. As on a C64,
// spaces and tabs inside the name are skipped, and a keyword ends the name,
// since the C64 reads keywords before names.
func scanName(line string, start int) (string, int) {
	var text strings.Builder
	end := start
	next := func() int {
		j := end
		for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
			j++
		}
		return j
	}
	take := func(j int) {
		text.WriteByte(line[j])
		end = j + 1
	}

	take(start)
	for {
		j := next()
		if j >= len(line) {
			break
		}
		if isDigit(line[j]) {
			take(j)
			continue
		}
		if !isLetter(line[j]) {
			break
		}
		if _, _, isKeyword := matchKeyword(line[j:]); isKeyword {
			break
		}
		take(j)
	}
	if j := next(); j < len(line) && (line[j] == '$' || line[j] == '%') {
		take(j)
	}
	return text.String(), end
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
