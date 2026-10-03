// Package lexer turns a line of C64 BASIC into tokens.
//
// Each token rule of the grammar is written in EBNF, in the notation of the
// Go language specification, above the code in Lex that scans it. The
// parser's rule comments are the rest of the grammar.
package lexer

import (
	"strings"
	"unicode/utf8"

	"github.com/bryanesmith/c64sh/internal/basicerr"
	"github.com/bryanesmith/c64sh/internal/token"
)

// maxLineNumber is the largest line number the C64 accepts.
const maxLineNumber = 63999

// keywords are matched case-sensitively, at any position outside a string,
// whatever follows them.
var keywords = []struct {
	text string
	kind token.Kind
}{
	{"PRINT", token.Print},
	{"REM", token.Rem},
	{"LET", token.Let},
	{"AND", token.And},
	{"OR", token.Or},
	{"NOT", token.Not},
	{"IF", token.If},
	{"THEN", token.Then},
	{"RUN", token.Run},
	{"LIST", token.List},
	{"NEW", token.New},
	{"END", token.End},
	{"GOTO", token.Goto},
	{"GO", token.Go},
	{"TO", token.To},
	{"FOR", token.For},
	{"NEXT", token.Next},
	{"STEP", token.Step},
	{"GOSUB", token.Gosub},
	{"RETURN", token.Return},
	{"INPUT", token.Input},
	{"GET", token.Get},
	{"DEF", token.Def},
	{"FN", token.Fn},
	{"LOAD", token.Load},
	{"SAVE", token.Save},
	{"VERIFY", token.Verify},
	{"PRINT#", token.PrintFile},
	{"INPUT#", token.InputFile},
	{"OPEN", token.Open},
	{"CLOSE", token.Close},
	{"CMD", token.Cmd},
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
	'<': token.Less,
	'>': token.Greater,
	'#': token.Hash,
}

// upArrow is the C64's exponentiation key, read as "^".
const upArrow = "\u2191"

// Lex returns the tokens of line. The last token is always EOL.
//
// @spec LEXER-001, LEXER-002, LEXER-003, LEXER-004, LEXER-005, LEXER-006, LEXER-007
// @spec LEXER-008, LEXER-009, LEXER-010, LEXER-011, LEXER-012, LEXER-013, LEXER-014
// @spec LEXER-015, LEXER-016, LEXER-017, LEXER-018, LEXER-019, LEXER-020, LEXER-021
// @spec LEXER-022, LEXER-023, LEXER-024, LEXER-025, LEXER-026, LEXER-030, LEXER-031, LEXER-032, LEXER-033, LEXER-034, LEXER-035, LEXER-036
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
		// and   = "AND" .
		// or    = "OR" .
		// not   = "NOT" .
		// if    = "IF" .
		// then  = "THEN" .
		// run   = "RUN" .
		// list  = "LIST" .
		// new   = "NEW" .
		// end   = "END" .
		// goto  = "GOTO" .
		// go    = "GO" .
		// to    = "TO" .
		// for   = "FOR" .
		// next  = "NEXT" .
		// step  = "STEP" .
		// gosub = "GOSUB" .
		// return = "RETURN" .
		// input = "INPUT" .
		// get   = "GET" .
		// def   = "DEF" .
		// fn    = "FN" .
		// load  = "LOAD" .
		// save  = "SAVE" .
		// verify = "VERIFY" .
		// printfile = "PRINT#" .
		// inputfile = "INPUT#" .
		// open  = "OPEN" .
		// close = "CLOSE" .
		// cmd   = "CMD" .
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

// LineNumber reads the line number at the start of s, after any spaces
// and tabs, as the C64 ROM reads one ($A96B). ok is false if s does not
// begin with a digit. rest is the text after the number and the spaces
// and tabs after it. err is a SYNTAX error if the number exceeds 63999.
//
//	line_number = digit { digit } .   /* spaces inside are ignored */
//
// @spec LEXER-027, LEXER-028, LEXER-029
func LineNumber(s string) (n int, rest string, ok bool, err error) {
	i := skipBlanks(s, 0)
	if i == len(s) || !isDigit(s[i]) {
		return 0, s, false, nil
	}
	for i < len(s) && isDigit(s[i]) {
		n = n*10 + int(s[i]-'0')
		if n > maxLineNumber {
			return 0, "", true, &basicerr.Error{Kind: basicerr.Syntax}
		}
		i = skipBlanks(s, i+1)
	}
	return n, s[i:], true, nil
}

// skipBlanks returns the index of the first character at or after i that
// is not a space or tab.
func skipBlanks(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return i
}
