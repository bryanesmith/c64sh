# Parser Specs

Design: `parser-design.md`

## Successful parses

- [x] **PARSER-001**: When `parser.Parse` is given the tokens of a line containing no syntax error, it shall return a non-nil `*ast.Line` whose `Statements` are the line's non-empty statements in source order, and a nil error.
- [x] **PARSER-002**: When a line contains empty statements (a blank line, a line of only `:`, consecutive colons, or a leading or trailing colon), the parser shall produce no statement node for each empty statement.
- [x] **PARSER-003**: When parsing a `PRINT` statement, the parser shall produce an `*ast.PrintStmt` whose `Items` are, in source order, an `*ast.ExprItem` for each expression, an `*ast.Semicolon` for each `;`, and an `*ast.Comma` for each `,`, reading items until the next token is `Colon` or `EOL`.
- [x] **PARSER-004**: When two expressions in a `PRINT` statement follow each other with no separator (such as `PRINT "A""B"`), the parser shall produce two consecutive `*ast.ExprItem` items.
- [x] **PARSER-005**: When parsing an expression consisting of a single string literal, the parser shall produce an `*ast.StringLit` holding the `String` token's value.
- [x] **PARSER-006**: When parsing operands (string or number literals) joined by `+`, the parser shall produce left-associative `*ast.BinaryExpr` nodes with `Op` `ast.Add`, whatever the operands' kinds (`"A"+"B"+"C"` is `BinaryExpr(Add, BinaryExpr(Add, "A", "B"), "C")`).
- [x] **PARSER-012**: When a statement begins with a `Rem` token, the parser shall produce an `*ast.RemStmt` whose `Text` is the token's value.
- [x] **PARSER-013**: When parsing a `Number` token, the parser shall produce an `*ast.NumberLit` whose `Value` is the token text converted by `strconv.ParseFloat`, after prefixing a leading `.` with `0` and giving an `E` with no exponent digits a `0` exponent (`.` is 0, `.5` is 0.5, `1E+` is 1); a literal too large for `float64` shall have an infinite `Value`.

## Syntax errors

- [x] **PARSER-007**: If a statement begins with a token other than `Print`, `Rem`, `Colon`, or `EOL`, then the parser shall return a SYNTAX error (the `basicerr.Syntax` kind, displayed as `?SYNTAX  ERROR`).
- [x] **PARSER-008**: If, within a `PRINT` statement, the parser encounters a token that cannot begin a print item (any token other than `String`, `Number`, `Semicolon`, or `Comma`, such as `Illegal`, `Plus`, `Print`, or `Rem`) where a print item is expected, or a `+` that is not followed by a `String` or `Number` token, then it shall return a SYNTAX error (so `PRINT "A" REM NOTE` is a SYNTAX error, while `PRINT "A":REM NOTE` is valid).
- [x] **PARSER-009**: When the parser returns a SYNTAX error, it shall also return a non-nil `*ast.Line` containing the statements completed before the statement in which the error occurred, and it shall not parse any tokens after the error.
- [x] **PARSER-010**: If a SYNTAX error occurs within a `PRINT` statement's items, then the parser shall include that statement in the returned `*ast.Line` as its last statement, with `Items` holding the items completed before the error followed by an `*ast.BadItem` carrying the SYNTAX error (so `PRINT "A";"B"+@` has items `ExprItem("A")`, `Semicolon`, `BadItem`).
- [x] **PARSER-011**: If a SYNTAX error occurs at the first token of a statement (a token other than `Print`, `Rem`, `Colon`, or `EOL`), then the parser shall produce no node for that statement.
