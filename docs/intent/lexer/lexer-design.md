---
parent: high-level-design
prefix: LEXER
---

# Lexer

## Context and Design Philosophy

The lexer turns one line of input into a sequence of tokens for the parser. It implements the lowercase (lexical) rules of `grammar/c64basic.ebnf` and the literal tokens used in its syntactic rules.

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
| Keyword text (see *Keywords*) | Keyword token (`Print`). |
| `?` | `Print` token (the C64 abbreviation for `PRINT`). |
| `:` `;` `,` `+` | `Colon`, `Semicolon`, `Comma`, `Plus`. |
| Any other character | `Illegal` token holding that one character (a full UTF-8 character, not a single byte). |
| A byte that is not valid UTF-8 | `Illegal` token holding that one byte. |

Only space and tab count as whitespace. Other Unicode spacing characters (non-breaking space, form feed, …) outside a string are `Illegal`.

Inside a string literal, the bytes of the line are kept exactly as they are, including spaces, tabs, `:`, `;`, non-ASCII characters, and bytes that are not valid UTF-8. Letter case inside strings is always preserved.

## Keywords

The only keyword is `PRINT`.

- **Recognition is by prefix, without word boundaries**, as on the C64: at any position outside a string, if the upcoming characters spell a keyword, the keyword token is produced, whatever follows. `PRINT"X"` is `Print String`; `PRINTX` is `Print Illegal(X)`.
- **Spaces inside a keyword break it.** `PR INT` is not `PRINT`; it scans as `Illegal(P) Illegal(R) Illegal(I) Illegal(N) Illegal(T)`.
- **Case-sensitive.** Keywords are recognized only in uppercase, exactly as written in the grammar. `print` and `Print` are not keywords; their letters scan as `Illegal` tokens, so `print "HI"` is a syntax error, as it is on a C64, where lowercase letters are different characters from uppercase ones.

Keywords are kept in a table, so future keywords are added by extending the table. When more than one keyword could match at a position, the longest match wins.

## Conformance Table

The lexer package keeps a table from each lexical rule name and each literal token in the grammar's syntactic rules to its token kind:

| Grammar item | Token kind |
|---|---|
| `print` | `Print` |
| `string` | `String` |
| `":"` | `Colon` |
| `";"` | `Semicolon` |
| `","` | `Comma` |
| `"+"` | `Plus` |

`character` is a helper rule used only inside `string` and produces no token of its own; the table lists it as a helper so the both-ways check still passes.

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
| Keyword case | Uppercase only | Case-insensitive | Matches C64 BASIC V2, where keywords are uppercase and lowercase letters are distinct characters. Keeps the grammar literal: the keyword text in `c64basic.ebnf` is exactly what the lexer accepts. |
| `?` abbreviation | Scanned as `Print` | Not supported until later | It is how the C64 itself tokenizes `?`, and it costs one table entry. |
| Output shape | Slice of all tokens for the line | Streaming `Next()` iterator | Lines are short; a slice is simpler to test and gives the parser unlimited lookahead. |
| Whitespace | Space and tab skipped between tokens | Space only | A tab outside a string has no meaning in BASIC V2; treating it like a space avoids surprising errors from pasted or indented scripts. |

## Open Questions & Future Decisions

### Deferred
1. Digits are currently `Illegal`. When numbers are added, a `number` token is introduced; when program mode is added, a leading number is read as a line number.
2. Letters that do not begin a keyword are currently `Illegal`. When variables are added, they become identifier tokens, following the C64 rule that only the first two characters of a name are significant.

## References

- `docs/intent/grammar/grammar-design.md`
- *Commodore 64 Programmer's Reference Guide*, chapter 2 (keyword abbreviations)
