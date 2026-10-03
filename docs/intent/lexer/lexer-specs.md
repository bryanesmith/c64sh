# Lexer Specs

Design: `lexer-design.md`

## Token stream

- [x] **LEXER-001**: When `lexer.Lex` is given a line, it shall return that line's tokens in source order, ending with exactly one `EOL` token.
- [x] **LEXER-002**: The lexer shall set each token's `Pos` to the byte offset of the token's first character in the line, and the `EOL` token's `Pos` to the length of the line in bytes.
- [x] **LEXER-003**: The lexer shall skip spaces and tabs that occur outside a string literal, producing no token for them.

## String literals

- [x] **LEXER-004**: When the lexer encounters `"` outside a string literal, it shall produce a `String` token whose value is every byte after that quote up to, but not including, the next `"`, and shall consume the closing `"`.
- [x] **LEXER-005**: If a string literal has no closing `"` before the end of the line, then the lexer shall produce a `String` token whose value is every byte after the opening quote to the end of the line, including trailing spaces.
- [x] **LEXER-006**: The lexer shall keep the bytes inside a string literal exactly as they appear in the line, including spaces, tabs, `:`, `;`, `,`, `+`, `?`, letter case, non-ASCII characters, and bytes that are not valid UTF-8.

## Keywords

- [x] **LEXER-007**: When the characters at the current position outside a string literal are the uppercase letters `PRINT`, the lexer shall produce a `Print` token, whatever character follows (so `PRINT"X"` is `Print String` and `PRINTX` is `Print Name(X)`).
- [x] **LEXER-008**: When the lexer encounters `?` outside a string literal, it shall produce a `Print` token.
- [x] **LEXER-009**: The lexer shall recognize keywords only in uppercase; lowercase letters outside a string literal (such as those of `print` or `Print`) shall each produce an `Illegal` token (so `Print` is `Name(P)` followed by four `Illegal` tokens).
- [x] **LEXER-010**: If a keyword's letters are separated by whitespace outside a string literal (such as `PR INT`), then the lexer shall not produce the keyword token, and shall read the letters as a name (so `PR INT` is `Name(PRINT)`).
- [x] **LEXER-031**: When the lexer encounters `FOR`, `NEXT`, or `STEP` in uppercase outside a string literal, it shall produce a `For`, `Next`, or `Step` token respectively.
- [x] **LEXER-032**: When the lexer encounters `GOSUB` or `RETURN` in uppercase outside a string literal, it shall produce a `Gosub` or `Return` token respectively (`GOSUB` being one token, the longest match, rather than `GO` followed by a name).
- [x] **LEXER-033**: When the lexer encounters `INPUT` or `GET` in uppercase outside a string literal, it shall produce an `Input` or `Get` token respectively.
- [x] **LEXER-034**: When the lexer encounters `DEF` or `FN` in uppercase outside a string literal, it shall produce a `Def` or `Fn` token respectively.
- [x] **LEXER-035**: When the lexer encounters `LOAD`, `SAVE`, or `VERIFY` in uppercase outside a string literal, it shall produce a `Load`, `Save`, or `Verify` token respectively.

## Punctuation and illegal input

- [x] **LEXER-011**: When the lexer encounters `:`, `;`, `,`, `+`, `-`, `*`, `/`, `(`, or `)` outside a string literal and outside a number literal's exponent, it shall produce a `Colon`, `Semicolon`, `Comma`, `Plus`, `Minus`, `Star`, `Slash`, `LParen`, or `RParen` token respectively (so `1-2` is `Number Minus Number`, while `1E-2` is one `Number`).
- [x] **LEXER-012**: If the lexer encounters, outside a string literal, a valid UTF-8 character that no other scanning rule accepts (including lowercase letters, and whitespace other than space and tab), then it shall produce an `Illegal` token whose value is that one character.
- [x] **LEXER-013**: If the lexer encounters, outside a string literal, a byte that is not valid UTF-8, then it shall produce an `Illegal` token whose value is that one byte.

## Comments

- [x] **LEXER-014**: When the characters at the current position outside a string literal are the uppercase letters `REM`, the lexer shall produce a `Rem` token whose value is every byte after `REM` to the end of the line, exactly as written (including a leading space, double quotes, colons, keywords, and bytes that are not valid UTF-8), followed by the `EOL` token (so `REM A:PRINT "X"` produces a `Rem` token with value ` A:PRINT "X"`, and `REMARK` produces a `Rem` token with value `ARK`).

## Numbers

- [x] **LEXER-015**: When the lexer encounters a digit or `.` outside a string literal, it shall produce a `Number` token for the number literal starting there (digits with at most one `.`, optionally followed by `E`, an optional `+` or `-`, and digits), positioned at its first character, whose value is the literal's characters with spaces and tabs removed (so `.` alone, `1E`, and `1E+` are complete literals).
- [x] **LEXER-016**: While scanning a number literal, the lexer shall skip spaces and tabs between its characters (`1 2` produces one `Number` token with value `12`, and `1 . 5 E 3` one with value `1.5E3`), and shall not include whitespace after the literal's last character in the token.
- [x] **LEXER-017**: When a `.` follows a number literal that already contains a `.`, the lexer shall end the literal before it, so that `.` starts the next token (`1.2.3` produces `Number(1.2)` and `Number(.3)`).
- [D] **LEXER-018**: When a keyword begins at an `E` that would otherwise continue a number literal, the lexer shall end the literal before the `E`, so that the keyword is recognized (the C64 reads keywords before numbers).
- [x] **LEXER-019**: When the lexer encounters `^` or `↑` (U+2191) outside a string literal, it shall produce a `Caret` token whose value is that character.
- [x] **LEXER-020**: When the lexer encounters, outside a string literal, an uppercase letter at a position where no keyword begins, it shall produce a `Name` token positioned at that letter, whose value is the letter followed by any uppercase letters and digits after it, skipping spaces and tabs between them, followed by a `$` or `%` if one comes next (possibly after spaces or tabs), all without the spaces and tabs (`HEIGHT` is `Name(HEIGHT)`, `A B` is `Name(AB)`, `N $` is `Name(N$)`, and `C%` is `Name(C%)`).
- [x] **LEXER-021**: While reading a name, when a keyword begins at the next character, the lexer shall end the name before it (`OUTLET` is `Name(OUT)` followed by `Let`, `PREMIUM` is `Name(P)` followed by a `Rem` token, and `SCORE` is `Name(SC)`, `Or`, `Name(E)`).
- [x] **LEXER-022**: When the lexer encounters `LET` in uppercase outside a string literal, it shall produce a `Let` token, and when it encounters `=` outside a string literal, an `Equal` token.
- [x] **LEXER-023**: When the lexer encounters `<` or `>` outside a string literal, it shall produce a `Less` or `Greater` token respectively, one token per character (so `<>` is `Less Greater`).
- [x] **LEXER-024**: When the lexer encounters `AND`, `OR`, or `NOT` in uppercase outside a string literal, it shall produce an `And`, `Or`, or `Not` token respectively.
- [x] **LEXER-025**: When the lexer encounters `IF` or `THEN` in uppercase outside a string literal, it shall produce an `If` or `Then` token respectively.
- [x] **LEXER-026**: When the lexer encounters `RUN`, `LIST`, `NEW`, or `END` in uppercase outside a string literal, it shall produce a `Run`, `List`, `New`, or `End` token respectively.
- [x] **LEXER-030**: When the lexer encounters `GOTO`, `GO`, or `TO` in uppercase outside a string literal, it shall produce a `Goto`, `Go`, or `To` token respectively, `GOTO` being one `Goto` token (the longest match) and `GO TO` a `Go` token followed by a `To` token.

## Line numbers

- [x] **LEXER-027**: When `lexer.LineNumber` is given text that begins, after any spaces and tabs, with a digit, it shall return `ok` true, `n` the value of the digits read from there, skipping spaces and tabs between them and stopping at the first other character, and `rest` the text after the last digit and any spaces and tabs following it (`10 PRINT` gives 10 and `PRINT`, `1 0X` gives 10 and `X`, `10.5` gives 10 and `.5`, and `010` gives 10 and the empty string).
- [x] **LEXER-028**: When `lexer.LineNumber` is given text that does not begin, after any spaces and tabs, with a digit, it shall return `ok` false and a nil error.
- [x] **LEXER-029**: If the digits read by `lexer.LineNumber` have a value above 63999, then it shall return `ok` true and a SYNTAX error.
