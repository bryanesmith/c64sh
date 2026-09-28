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

- [x] **LEXER-007**: When the characters at the current position outside a string literal are the uppercase letters `PRINT`, the lexer shall produce a `Print` token, whatever character follows (so `PRINT"X"` is `Print String` and `PRINTX` is `Print Illegal`).
- [x] **LEXER-008**: When the lexer encounters `?` outside a string literal, it shall produce a `Print` token.
- [x] **LEXER-009**: The lexer shall recognize keywords only in uppercase; letters of a lowercase or mixed-case keyword outside a string literal (such as `print` or `Print`) shall each produce an `Illegal` token.
- [x] **LEXER-010**: If a keyword's letters are separated by whitespace outside a string literal (such as `PR INT`), then the lexer shall produce an `Illegal` token for each letter instead of a keyword token.

## Punctuation and illegal input

- [x] **LEXER-011**: When the lexer encounters `:`, `;`, `,`, or `+` outside a string literal, it shall produce a `Colon`, `Semicolon`, `Comma`, or `Plus` token respectively.
- [x] **LEXER-012**: If the lexer encounters, outside a string literal, a valid UTF-8 character that no other scanning rule accepts (including digits, letters not starting a keyword, and whitespace other than space and tab), then it shall produce an `Illegal` token whose value is that one character.
- [x] **LEXER-013**: If the lexer encounters, outside a string literal, a byte that is not valid UTF-8, then it shall produce an `Illegal` token whose value is that one byte.
