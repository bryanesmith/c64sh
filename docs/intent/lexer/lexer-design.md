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
    Number                // 12, 3.5, .5, 1E3 (Value: the literal without spaces)
    String                // "…"
    Colon                 // :
    Semicolon             // ;
    Comma                 // ,
    Plus                  // +
    Minus                 // -
    Star                  // *
    Slash                 // /
    LParen                // (
    RParen                // )
    Caret                 // ^ or ↑ (exponentiation)
    Let                   // LET
    Equal                 // =
    Less                  // <
    Greater               // >
    Name                  // a variable name: A, HEIGHT, N$ (Value: the name without spaces)
    And                   // AND
    Or                    // OR
    Not                   // NOT
    If                    // IF
    Then                  // THEN
    Run                   // RUN
    List                  // LIST
    New                   // NEW
    End                   // END
    Goto                  // GOTO
    Go                    // GO (as in GO TO)
    To                    // TO
    For                   // FOR
    Next                  // NEXT
    Step                  // STEP
    Gosub                 // GOSUB
    Return                // RETURN
    Input                 // INPUT
    Get                   // GET
    Def                   // DEF
    Fn                    // FN
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
| Other keyword text (see *Keywords*) | Keyword token (`Print`, `Let`, `If`, …). |
| `?` | `Print` token (the C64 abbreviation for `PRINT`). |
| `:` `;` `,` `+` `-` `*` `/` `(` `)` | `Colon`, `Semicolon`, `Comma`, `Plus`, `Minus`, `Star`, `Slash`, `LParen`, `RParen`. |
| `^` or `↑` (U+2191) | `Caret`, whose value is the character as written. |
| A digit, or `.` | `Number` token (see *Numbers*). |
| An uppercase letter that does not begin a keyword | `Name` token (see *Names*). |
| `=` `<` `>` | `Equal`, `Less`, `Greater`, one token per character; the parser combines them into comparison operators (`<>`, `<=`, …). |
| Any other character | `Illegal` token holding that one character (a full UTF-8 character, not a single byte). |
| A byte that is not valid UTF-8 | `Illegal` token holding that one byte. |

Only space and tab count as whitespace. Other Unicode spacing characters (non-breaking space, form feed, …) outside a string are `Illegal`.

Inside a string literal, the bytes of the line are kept exactly as they are, including spaces, tabs, `:`, `;`, non-ASCII characters, and bytes that are not valid UTF-8. Letter case inside strings is always preserved.

## Keywords

The keywords are `PRINT`, `REM`, `LET`, `AND`, `OR`, `NOT`, `IF`, `THEN`, `RUN`, `LIST`, `NEW`, `END`, `GOTO`, `GO`, `TO`, `FOR`, `NEXT`, `STEP`, `GOSUB`, `RETURN`, `INPUT`, `GET`, `DEF`, and `FN`.

- **Recognition is by prefix, without word boundaries**, as on the C64: at any position outside a string, if the upcoming characters spell a keyword, the keyword token is produced, whatever follows. `PRINT"X"` is `Print String`; `PRINTX` is `Print Illegal(X)`; `REMARK` is a `Rem` token with the comment `ARK`.
- **Spaces inside a keyword break it.** `PR INT` is not `PRINT`; it scans as `Illegal(P) Illegal(R) Illegal(I) Illegal(N) Illegal(T)`.
- **Case-sensitive.** Keywords are recognized only in uppercase, exactly as written in their token rules. `print` and `Print` are not keywords; their letters scan as `Illegal` tokens, so `print "HI"` is a syntax error, as it is on a C64, where lowercase letters are different characters from uppercase ones.

Keywords are kept in a table, so future keywords are added by extending the table. When more than one keyword could match at a position, the longest match wins: `GOTO` is one `Goto` token and `GOSUB` one `Gosub` token, while `GO TO`, with a space, is `Go` then `To`. As on a C64, `GO` and `TO` also end names that contain them (`GOLD`, `TOTAL`), whether or not they are followed by `TO` or used with `FOR`.

## Numbers

A number literal is read the way the C64 ROM reads one:

- It starts with a digit or `.`. A lone `.` is a number: zero.
- Digits follow, with at most one `.`; a second `.` ends the number and starts another (`1.2.3` is `1.2` then `.3`).
- An `E` may follow, then an optional `+` or `-`, then any number of digits. An `E` with no digits after it means an exponent of 0 (`1E` is `1`). The `E` is not taken if a keyword starts there, because the C64 recognizes keywords before it reads numbers.
- **Spaces and tabs inside a number are ignored**, as the C64's character reader skips them: `1 2` is `12`, `1 . 5` is `1.5`, and `1 E 3` is `1000`. Whitespace after the last character of the number is not part of it.

The token's value is the literal with its whitespace removed (`1 2` gives `12`), and its position is that of its first character. The lexer does not compute the number's value; the parser converts the text.

## Names

A variable name is read the way the C64 ROM reads one (`$B08B`), within the C64's tokenizing of keywords:

- It starts with an uppercase letter `A`–`Z` at a position where no keyword begins.
- Uppercase letters and digits follow. **Spaces and tabs inside a name are skipped**, as the C64's character reader skips them: `A B` is the name `AB`.
- **A keyword ends the name**: at each position after the first letter, if a keyword begins there, the name ends before it and the keyword is read next. This is the C64's famous rule that a keyword cannot appear inside a name: `OUTLET` is the name `OUT` followed by `LET`, `PREMIUM` is the name `P` followed by a `REM` comment, and `SCORE` is the name `SC` followed by `OR` and the name `E`.
- An optional `$` or `%` follows, possibly after spaces, marking a string or integer variable. It is part of the token's value.
- Lowercase letters are not names; they remain `Illegal`, as on a C64.

The token's value is the full name as written, without spaces (`HEIGHT`, `N$`, `C%`). Which characters matter for identity is the parser's concern (see the parser design).

## Line Numbers

Whether a line is stored in the program or run at once depends on whether it starts with a line number, which the shell asks the lexer to read with `LineNumber`. The parser uses the same function for the line number after `RUN`, `GOTO`, and `THEN`. It reads a line number the way the C64 ROM does (`$A96B`):

- After any spaces and tabs, the text must start with a digit; otherwise there is no line number.
- Digits are read, **skipping spaces and tabs between them** (`1 0` is 10), until the first character that is neither. Leading zeros are allowed (`010` is 10).
- **The largest line number is 63999.** A larger number is a SYNTAX error (the ROM rejects any digit that would take the number to 64000 or more).
- The rest of the text starts after the number and any spaces and tabs after it. Nothing else is checked: `10.5 PRINT` is line 10 with the text `.5 PRINT`, and `5+5` is line 5 with the text `+5`, as on a C64.

Its value comes from the characters, not from a `Number` token, because a number token reads further than a line number does (`10.5`, `1E3`).

## Token Rules

The lexer's half of the grammar is its token rules. Each is written in EBNF, in the notation of the Go language specification, as a comment directly above the code in `lexer.go` that scans it:

```
print     = "PRINT" | "?" .
rem       = "REM" { character | `"` } .
string    = `"` { character } [ `"` ] .
name      = letter { letter | digit } [ "$" | "%" ] .   /* spaces inside are ignored; a keyword ends it */
letter    = "A" … "Z" .
let       = "LET" .
and       = "AND" .
or        = "OR" .
not       = "NOT" .
if        = "IF" .
then      = "THEN" .
run       = "RUN" .
list      = "LIST" .
new       = "NEW" .
end       = "END" .
goto      = "GOTO" .
go        = "GO" .
to        = "TO" .
for       = "FOR" .
next      = "NEXT" .
step      = "STEP" .
gosub     = "GOSUB" .
return    = "RETURN" .
input     = "INPUT" .
get       = "GET" .
def       = "DEF" .
fn        = "FN" .
number    = ( digit { digit } [ "." { digit } ] | "." { digit } )
            [ "E" [ "+" | "-" ] { digit } ] .   /* spaces inside are ignored */
digit     = "0" … "9" .
line_number = digit { digit } .   /* spaces inside are ignored; read by LineNumber */
character = /* any character except `"` and a line feed */ .
```

A line never contains a line feed, because the shell splits input into lines at line feeds, so `character` is every character the lexer can see except the double quote; a carriage return inside a line is an ordinary character. The punctuation tokens `:`, `;`, `,`, `+`, `-`, `*`, `/`, `(`, `)`, and `^` are written as literal tokens in the parser's rules; the lexer also reads `↑` as `^`. A `-` is never part of a number token except as the sign of an exponent (`1E-3`); in `1-2` it is a `Minus` between two numbers.

Uppercase rule names belong to the parser; lowercase ones are these token rules. Together, the parser's rule comments and these comments are the complete grammar of the language.

## API

```go
package lexer

// Lex returns the tokens of line. The last token is always EOL.
func Lex(line string) []token.Token

// LineNumber reads the line number at the start of s, after any spaces
// and tabs, as the C64 ROM reads one. ok is false if s does not begin
// with a digit. rest is the text after the number and the spaces and
// tabs after it. err is a SYNTAX error if the number exceeds 63999.
func LineNumber(s string) (n int, rest string, ok bool, err error)
```

`LineNumber` is the only lexer function that can return an error: whether a number is too large to be a line number is decided where the number is read, as in the ROM, for both of its callers.

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
| Spaces inside numbers | Ignored, so `1 2` is `12` | Spaces end a number | The C64's character reader skips spaces everywhere outside strings, so this is how BASIC V2 reads numbers; `PRINT 1 2` printing ` 12 ` is authentic. |
| Number value | Computed by the parser from the token text | Computed by the lexer | Tokens carry text; keeping number conversion out of the lexer keeps its job to splitting characters, and one place converts literals. |
| Exponentiation character | `^`, and `↑` as an alternative | `↑` only; `^` only; `**` | The C64's up-arrow key produces character code 94, which is `^` in ASCII, so `^` is what a C64 program's bytes contain. `↑` matches what the C64 keyboard and screen show, for users who can type it. `**` is not C64 BASIC. |
| Keywords inside names | A keyword ends the name | Names take precedence over keywords | The C64 tokenizes keywords before it ever reads a name, so `TOTAL` is `TO` plus `TAL` and is a syntax error. Reproducing this is authentic, and it falls out of checking keywords first. |
| Name value | The full name, without spaces | Only the first two characters | Keeping the full name leaves messages and a future `LIST` free to show it; the parser reduces it to the part that identifies the variable. |
| Output shape | Slice of all tokens for the line | Streaming `Next()` iterator | Lines are short; a slice is simpler to test and gives the parser unlimited lookahead. |
| Whitespace | Space and tab skipped between tokens | Space only | A tab outside a string has no meaning in BASIC V2; treating it like a space avoids surprising errors from pasted or indented scripts. |
| Line numbers | A separate function, `LineNumber`, reading characters as the ROM does | A `LineNumber` token produced by `Lex` at the start of a line | A stored line keeps its text for `LIST`, so the caller needs the raw text after the number, not tokens; and a line number stops at characters a `Number` token would continue through (`10.5`, `1E3`). |

## Open Questions & Future Decisions

### Deferred
None.

## References

- `docs/intent/parser/parser-design.md` (the parser's grammar rules)
- [The Go Programming Language Specification — Notation](https://go.dev/ref/spec#Notation)
- *Commodore 64 Programmer's Reference Guide*, chapter 2 (keyword abbreviations)
