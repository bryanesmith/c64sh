---
parent: high-level-design
prefix: LEXER
---

# Lexer

## Context and Design Philosophy

The lexer turns one line of input into a sequence of tokens for the parser. It implements the grammar's token rules (see *Token Rules*) and the punctuation tokens the parser's rules use.

The lexer follows how the C64 reads a line, including its tolerances: keywords are recognized with or without surrounding spaces (`PRINT"X"`), and a string literal may be left unclosed. It never fails. Any character it does not recognize becomes an `Illegal` token, and the parser turns that into `?SYNTAX  ERROR` at the point where it is reached. This keeps every syntax decision in one place (the parser) and preserves the C64 order of events, where statements before a bad one still run.

## Input

The input is one line as a Go string, without its line terminator. The shell removes the trailing `\n` and any `\r` before it.

## Tokens

Token types live in `internal/token`:

```go
type Kind int

const (
    EOL       Kind = iota // end of line; always the last token
    Illegal               // a character no rule accepts
    Print                 // PRINT or ?
    Rem                   // REM and the rest of the line
    String                // "…"
    Colon                 // :
    Semicolon             // ;
    Comma                 // ,
    Plus                  // +
)

type Token struct {
    Kind  Kind
    Value string // String: contents without quotes; Illegal: the character; otherwise the source text
    Pos   int    // byte offset of the token's first character in the line
}
```

`Pos` is not shown to users (C64 error messages carry no position) but is available to parser tests and future diagnostics.

## Scanning Rules

At each position the lexer applies the first matching rule:

| Input at position | Result |
|---|---|
| End of line | `EOL` token; scanning stops. |
| Space or tab | Skipped. |
| `"` | `String` token. Its value is every character up to the next `"`, or to the end of the line if there is none. The closing quote, if present, is consumed. |
| `REM` (see *Keywords*) | `Rem` token whose value is every byte after `REM` to the end of the line, exactly as written (including a leading space, quotes, colons, and keywords). Scanning stops; the next token is `EOL`. |
| Other keyword text (see *Keywords*) | Keyword token (`Print`). |
| `?` | `Print` token (the C64 abbreviation for `PRINT`). |
| `:` `;` `,` `+` | `Colon`, `Semicolon`, `Comma`, `Plus`. |
| Any other character | `Illegal` token holding that one character (a full UTF-8 character, not a single byte). |
| A byte that is not valid UTF-8 | `Illegal` token holding that one byte. |

Only space and tab count as whitespace. Other Unicode spacing characters (non-breaking space, form feed, …) outside a string are `Illegal`.

Inside a string literal, the bytes of the line are kept exactly as they are, including spaces, tabs, `:`, `;`, non-ASCII characters, and bytes that are not valid UTF-8. Letter case inside strings is always preserved.

## Keywords

The keywords are `PRINT` and `REM`.

- **Recognition is by prefix, without word boundaries**, as on the C64: at any position outside a string, if the upcoming characters spell a keyword, the keyword token is produced, whatever follows. `PRINT"X"` is `Print String`; `PRINTX` is `Print Illegal(X)`; `REMARK` is a `Rem` token with the comment `ARK`.
- **Spaces inside a keyword break it.** `PR INT` is not `PRINT`; it scans as `Illegal(P) Illegal(R) Illegal(I) Illegal(N) Illegal(T)`.
- **Case-sensitive.** Keywords are recognized only in uppercase, exactly as written in their token rules. `print` and `Print` are not keywords; their letters scan as `Illegal` tokens, so `print "HI"` is a syntax error, as it is on a C64, where lowercase letters are different characters from uppercase ones.

Keywords are kept in a table, so future keywords are added by extending the table. When more than one keyword could match at a position, the longest match wins.

## Token Rules

The lexer's half of the grammar is its token rules. Each is written in EBNF, in the notation of the Go language specification, as a comment directly above the code in `lexer.go` that scans it:

```
print     = "PRINT" | "?" .
rem       = "REM" { character | `"` } .
string    = `"` { character } [ `"` ] .
character = /* any character except `"` and a line feed */ .
```

A line never contains a line feed, because the shell splits input into lines at line feeds, so `character` is every character the lexer can see except the double quote; a carriage return inside a line is an ordinary character. The punctuation tokens `:`, `;`, `,`, and `+` are written as literal tokens in the parser's rules.

Uppercase rule names belong to the parser; lowercase ones are these token rules. Together, the parser's rule comments and these comments are the complete grammar of the language.

## API

```go
package lexer

// Lex returns the tokens of line. The last token is always EOL.
func Lex(line string) []token.Token
```

Scanning a whole line up front is sufficient: lines are short, and the parser benefits from being able to look ahead freely.

## Decisions & Alternatives

| Decision | Chosen | Alternatives Considered | Rationale |
|---|---|---|---|
| Unknown characters | `Illegal` token; parser reports the error | Lexer returns an error | One place decides syntax errors, and statements before the bad character still run, matching the C64 order of events. |
| Keyword boundaries | Prefix match, no word boundary | Require a non-letter after a keyword | Matches C64 tokenization, which is what makes `PRINT"X"` valid. Needed for fidelity once variables exist (`PRINTA` is `PRINT A` on a C64). |
| Keyword case | Uppercase only | Case-insensitive | Matches C64 BASIC V2, where keywords are uppercase and lowercase letters are distinct characters. Keeps the token rules literal: the keyword text in each rule is exactly what the lexer accepts. |
| Comment text | One `Rem` token carrying the rest of the line | Discard the comment in the lexer; tokenize the comment's contents | A C64 ignores everything after `REM`, so its contents must not be tokenized. Keeping the text in the token lets the parser place a node in the AST, which program mode needs for `LIST`, and keeps `PRINT "A" REM` an error as on a C64. |
| `?` abbreviation | Scanned as `Print` | Not supported until later | It is how the C64 itself tokenizes `?`, and it costs one table entry. |
| Token rule documentation | EBNF comment above the code that scans each rule | A separate grammar file | The rule sits beside its implementation, so there is one description of each token to keep current (see HLD *Where the syntax is defined*). |
| Output shape | Slice of all tokens for the line | Streaming `Next()` iterator | Lines are short; a slice is simpler to test and gives the parser unlimited lookahead. |
| Whitespace | Space and tab skipped between tokens | Space only | A tab outside a string has no meaning in BASIC V2; treating it like a space avoids surprising errors from pasted or indented scripts. |

## Open Questions & Future Decisions

### Deferred
1. Digits are currently `Illegal`. When numbers are added, a `number` token is introduced; when program mode is added, a leading number is read as a line number.
2. Letters that do not begin a keyword are currently `Illegal`. When variables are added, they become identifier tokens, following the C64 rule that only the first two characters of a name are significant.

## References

- `docs/intent/parser/parser-design.md` (the parser's grammar rules)
- [The Go Programming Language Specification — Notation](https://go.dev/ref/spec#Notation)
- *Commodore 64 Programmer's Reference Guide*, chapter 2 (keyword abbreviations)
