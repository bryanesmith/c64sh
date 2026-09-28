# Parser Specs

Design: `parser-design.md`

## Successful parses

- [x] **PARSER-001**: When `parser.Parse` is given the tokens of a line containing no syntax error, it shall return a non-nil `*ast.Line` whose `Statements` are the line's non-empty statements in source order, and a nil error.
- [x] **PARSER-002**: When a line contains empty statements (a blank line, a line of only `:`, consecutive colons, or a leading or trailing colon), the parser shall produce no statement node for each empty statement.
- [x] **PARSER-003**: When parsing a `PRINT` statement, the parser shall produce an `*ast.PrintStmt` whose `Items` are, in source order, an `*ast.ExprItem` for each expression, an `*ast.Semicolon` for each `;`, and an `*ast.Comma` for each `,`, reading items until the next token is `Colon` or `EOL`.
- [x] **PARSER-004**: When two expressions in a `PRINT` statement follow each other with no separator (such as `PRINT "A""B"`), the parser shall produce two consecutive `*ast.ExprItem` items.
- [x] **PARSER-005**: When parsing an expression consisting of a single string literal, the parser shall produce an `*ast.StringLit` holding the `String` token's value.
- [x] **PARSER-006**: When parsing string literals joined by `+`, the parser shall produce left-associative `*ast.Concat` nodes (`"A"+"B"+"C"` is `Concat(Concat("A","B"),"C")`).

## Syntax errors

- [x] **PARSER-007**: If a statement begins with a token other than `Print`, `Colon`, or `EOL`, then the parser shall return a SYNTAX error (the `basicerr.Syntax` kind, displayed as `?SYNTAX  ERROR`).
- [x] **PARSER-008**: If, within a `PRINT` statement, the parser encounters an `Illegal` or `Plus` token where a print item is expected, or a `+` that is not followed by a `String` token, then it shall return a SYNTAX error.
- [x] **PARSER-009**: When the parser returns a SYNTAX error, it shall also return a non-nil `*ast.Line` containing the statements completed before the statement in which the error occurred, and it shall not parse any tokens after the error.
- [x] **PARSER-010**: If a SYNTAX error occurs within a `PRINT` statement's items, then the parser shall include that statement in the returned `*ast.Line` as its last statement, with `Items` holding the items completed before the error followed by an `*ast.BadItem` carrying the SYNTAX error (so `PRINT "A";"B"+@` has items `ExprItem("A")`, `Semicolon`, `BadItem`).
- [x] **PARSER-011**: If a SYNTAX error occurs at the first token of a statement (a token other than `Print`, `Colon`, or `EOL`), then the parser shall produce no node for that statement.
