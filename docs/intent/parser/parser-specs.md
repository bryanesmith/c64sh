# Parser Specs

Design: `parser-design.md`

## Successful parses

- [x] **PARSER-001**: When `parser.Parse` is given the tokens of a line containing no syntax error, it shall return a non-nil `*ast.Line` whose `Statements` are the line's non-empty statements in source order, and a nil error.
- [x] **PARSER-002**: When a line contains empty statements (a blank line, a line of only `:`, consecutive colons, or a leading or trailing colon), the parser shall produce no statement node for each empty statement.
- [x] **PARSER-003**: When parsing a `PRINT` statement, the parser shall produce an `*ast.PrintStmt` whose `Items` are, in source order, an `*ast.ExprItem` for each expression, an `*ast.Semicolon` for each `;`, and an `*ast.Comma` for each `,`, reading items until the next token is `Colon` or `EOL`.
- [x] **PARSER-004**: When two expressions in a `PRINT` statement follow each other with no separator (such as `PRINT "A""B"`), the parser shall produce two consecutive `*ast.ExprItem` items.
- [x] **PARSER-005**: When parsing an expression consisting of a single string literal, the parser shall produce an `*ast.StringLit` holding the `String` token's value.
- [x] **PARSER-006**: When parsing operands joined by `+` or `-`, the parser shall produce left-associative `*ast.BinaryExpr` nodes with `Op` `ast.Add` or `ast.Sub`, whatever the operands' kinds (`"A"+"B"+"C"` is `BinaryExpr(Add, BinaryExpr(Add, "A", "B"), "C")`, and `8-2-1` is `(8-2)-1`).
- [x] **PARSER-012**: When a statement begins with a `Rem` token, the parser shall produce an `*ast.RemStmt` whose `Text` is the token's value.
- [x] **PARSER-013**: When parsing a `Number` token, the parser shall produce an `*ast.NumberLit` whose `Value` is the token text converted by `strconv.ParseFloat`, after prefixing a leading `.` with `0` and giving an `E` with no exponent digits a `0` exponent (`.` is 0, `.5` is 0.5, `1E+` is 1); a literal too large for `float64` shall have an infinite `Value`.
- [x] **PARSER-014**: When parsing operands joined by `*` or `/`, the parser shall produce left-associative `*ast.BinaryExpr` nodes with `Op` `ast.Mul` or `ast.Div`, binding tighter than `+` and `-` (`2+3*4` is `2+(3*4)`, and `8/4/2` is `(8/4)/2`).
- [x] **PARSER-015**: When an operand is preceded by `-` where an operand is expected, the parser shall produce an `*ast.NegExpr` of it, binding tighter than `*` and `/` and allowed to repeat (`-2*3` is `(-2)*3`, `--5` is `NegExpr(NegExpr(5))`, and `5--5` is `5-(-5)`).
- [x] **PARSER-016**: When an operand is preceded by `+` where an operand is expected, the parser shall produce the operand itself, with no node for the `+` (`+5` is `5`, and `+"A"` is `"A"`), as the C64 ROM skips a leading `+`.
- [x] **PARSER-017**: When parsing `(` followed by an expression and `)` where an operand is expected, the parser shall produce the inner expression, with no node for the parentheses (`(2+3)*4` is `BinaryExpr(Mul, BinaryExpr(Add, 2, 3), 4)`).
- [x] **PARSER-018**: When a complete expression in a `PRINT` statement is followed by a token that can begin an expression but cannot continue one (`String`, `Number`, `Name`, `Not`, or `LParen`), the parser shall end the expression and begin a new print item (`PRINT 2(3)` has items `2` and `3`), while `+` or `-` after a complete expression shall continue it as a binary operator (`PRINT 1 -1` is `1-1`).
- [x] **PARSER-019**: When parsing operands joined by `^`, the parser shall produce left-associative `*ast.BinaryExpr` nodes with `Op` `ast.Pow`, binding tighter than a leading `-` or `+` and than every other operator (`2^3^2` is `(2^3)^2`, `-2^2` is `-(2^2)`, and `2*3^2` is `2*(3^2)`).
- [x] **PARSER-020**: When an exponent (the operand after `^`) begins with `-` or `+`, the parser shall read the rest of the exponent as a `Unary`, so that the sign applies to any following `^` chain but nothing looser (`2^-1` is `2^(-1)`, `2^-1^2` is `2^(-(1^2))`, and `2^-3*4` is `(2^(-3))*4`).
- [x] **PARSER-021**: When a statement begins with a `Name` token, or with a `Let` token followed by a `Name` token, followed by `=` and an expression, the parser shall produce an `*ast.LetStmt` assigning that expression to the variable (`A=5` and `LET A=5` produce the same node).
- [x] **PARSER-022**: When parsing a `Name` token as a variable, the parser shall produce an `*ast.VarRef` whose `Name` is the name's first character, its second character if it has one, and `$` or `%` if the name ends in `$` or `%`, and whose `Text` is the token's value (`HEIGHT` is `Name` `HE`, `NAME$` is `NA$`, `N$` is `N$`, `A1B` is `A1`, and `COUNT%` is `CO%`).
- [x] **PARSER-023**: When a `Name` token appears where an operand is expected, the parser shall produce an `*ast.VarRef` as the operand.
- [x] **PARSER-024**: If a variable's identity (`VarRef.Name`) is `TI`, `TI$`, or `ST`, or a `Name` token is followed by `(`, then the parser shall return a SYNTAX error, whether the name is assigned or used as an operand (the C64's system variables, arrays, and functions are not supported; `TI%` and `ST%` are ordinary integer variables).
- [x] **PARSER-025**: If a `LET` is not followed by a name, or an assignment's name is not followed by `=`, or `=` is not followed by an expression, then the parser shall return a SYNTAX error.
- [x] **PARSER-026**: When parsing sums (`+` and `-` expressions) joined by a comparison operator, the parser shall produce left-associative `*ast.CompareExpr` nodes, binding more loosely than every arithmetic operator (`1+1=2` is `(1+1)=2`, and `1<2<3` is `(1<2)<3`).
- [x] **PARSER-027**: When parsing a comparison operator, the parser shall read a run of one or more `Less`, `Equal`, and `Greater` tokens, in any order, adding `ast.RelLess`, `ast.RelEqual`, or `ast.RelGreater` to the operator's `Relation` for each (`<>` and `><` are less-or-greater, `<=` and `=<` less-or-equal, `>=` and `=>` greater-or-equal, and `<=>` all three); if a token repeats within one operator (`==`, `<<`), then it shall return a SYNTAX error.
- [x] **PARSER-028**: When parsing an assignment, the parser shall take the first `=` after the variable as the assignment and parse every later `=` as a comparison (`A=B=C` assigns to `A` the comparison of `B` with `C`).
- [x] **PARSER-029**: When parsing comparisons joined by `AND` or `OR`, the parser shall produce left-associative `*ast.BinaryExpr` nodes with `Op` `ast.And` or `ast.Or`, `AND` binding more tightly than `OR` and both more loosely than comparisons (`A OR B AND C` is `A OR (B AND C)`, and `1<2 AND 3<4` is `(1<2) AND (3<4)`).
- [x] **PARSER-030**: When `NOT` appears where an operand is expected, the parser shall produce an `*ast.NotExpr` of the comparison-level expression that follows it, which ends before the next `AND` or `OR` (`NOT 1=2` is `NOT (1=2)`, `NOT A AND B` is `(NOT A) AND B`, and `1+NOT 0+1` is `1+NOT (0+1)`).
- [x] **PARSER-031**: When a statement is `IF`, an expression, and `THEN`, the parser shall produce an `*ast.IfStmt` holding the expression as its condition, and shall parse the statement that follows `THEN`, if any, as the line's next statement with no `:` required (`IF A>1 THEN PRINT "X":PRINT "Y"` is `IfStmt(A>1)`, `PrintStmt("X")`, `PrintStmt("Y")`).
- [x] **PARSER-032**: If an `IF`'s expression is followed by a token other than `THEN` or `GOTO`, then the parser shall return a SYNTAX error in place of the `IF` (so `IF 1 PRINT "X"` and `IF 1 GO TO 10` are SYNTAX errors).
- [x] **PARSER-033**: When a statement is `RUN` followed by `:` or `EOL`, the parser shall produce an `*ast.RunStmt` with `HasLine` false.
- [x] **PARSER-034**: When `RUN` is followed by a `Number` token, the parser shall produce an `*ast.RunStmt` with `HasLine` true and `Line` the line number read as PARSER-039 specifies (`RUN 20` is line 20, `RUN 20.5` line 20, and `RUN .5` line 0).
- [x] **PARSER-035**: When `RUN` is followed by a token other than `:`, `EOL`, or `Number`, the parser shall produce an `*ast.RunStmt` with `HasLine` true and `Line` 0, without consuming that token (`RUN A` is `RUN 0`, as on a C64).
- [x] **PARSER-036**: When a statement is `LIST`, `NEW`, or `END` followed by `:` or `EOL`, the parser shall produce an `*ast.ListStmt`, `*ast.NewStmt`, or `*ast.EndStmt` respectively.
- [x] **PARSER-038**: When a statement is `GOTO`, or `GO` followed by `TO`, the parser shall produce an `*ast.GotoStmt` whose `Line` is the line number read as PARSER-039 specifies (`GOTO 20`, `GO TO 20`, and `GOTO 20.5` are line 20; `GOTO` alone and `GOTO A` are line 0).
- [x] **PARSER-039**: When reading the line number after `RUN`, `GOTO`, `GO TO`, or `THEN`, the parser shall consume the next token if it is a `Number` and take the value `lexer.LineNumber` reads from its text, or 0 if that text does not begin with a digit; if the next token is not a `Number`, the line number shall be 0 and the token shall not be consumed.
- [x] **PARSER-040**: When `THEN` is followed by a `Number` token whose text begins with a digit, the parser shall produce, after the `*ast.IfStmt`, an `*ast.GotoStmt` whose line number is read as PARSER-039 specifies (`IF A THEN 20` is `IfStmt(A)`, `GotoStmt(20)`).
- [x] **PARSER-041**: When an `IF`'s expression is followed by `GOTO`, the parser shall produce the `*ast.IfStmt` without consuming the `GOTO`, so that the `GOTO` is parsed as the line's next statement (`IF A GOTO 20` is `IfStmt(A)`, `GotoStmt(20)`).

## Syntax errors

- [x] **PARSER-007**: If a statement begins with a token other than `Print`, `Rem`, `Let`, `If`, `Run`, `Goto`, `Go`, `List`, `New`, `End`, `Name`, `Colon`, or `EOL`, then the parser shall return a SYNTAX error (the `basicerr.Syntax` kind, displayed as `?SYNTAX  ERROR`).
- [x] **PARSER-008**: If, within a `PRINT` statement, the parser encounters a token that cannot begin a print item (any token other than `String`, `Number`, `Name`, `Not`, `Fn`, `Minus`, `Plus`, `LParen`, `Semicolon`, or `Comma`, such as `Illegal`, `Star`, `Caret`, `And`, `Or`, `Equal`, `Less`, `Greater`, `RParen`, `Print`, or `Rem`) where a print item is expected, an operator with no operand after it, or a `(` without a matching `)`, then it shall return a SYNTAX error (so `PRINT "A" REM NOTE` and `PRINT (1+2` are SYNTAX errors, while `PRINT "A":REM NOTE` is valid).
- [x] **PARSER-009**: When the parser returns a SYNTAX error, it shall also return a non-nil `*ast.Line` holding the statements completed before the statement in which the error occurred, followed by the error itself at that point (PARSER-010 or PARSER-011), and it shall not parse any tokens after the error.
- [x] **PARSER-010**: If a SYNTAX error occurs within a `PRINT` statement's items, then the parser shall include that statement in the returned `*ast.Line` as its last statement, with `Items` holding the items completed before the error followed by an `*ast.BadItem` carrying the SYNTAX error (so `PRINT "A";"B"+@` has items `ExprItem("A")`, `Semicolon`, `BadItem`).
- [x] **PARSER-011**: If a SYNTAX error occurs other than within a `PRINT` statement's items (for example at the first token of a statement, in an assignment, or in an `IF`), then the parser shall put an `*ast.BadStmt` holding the error in place of the failing statement, as the line's last statement.
- [x] **PARSER-037**: If `LIST`, `NEW`, or `END` is followed by a token other than `:` or `EOL`, `GO` is not followed by `TO`, or a line number read as PARSER-039 specifies is above 63999, then the parser shall return a SYNTAX error, with an `*ast.BadStmt` in place of the statement (so `LIST 10`, `END 1`, and `GO 10` do nothing but report the error).

## Loops

- [x] **PARSER-042**: When a statement is `FOR`, a variable, `=`, an expression, `TO`, and an expression, optionally followed by `STEP` and an expression, the parser shall produce an `*ast.ForStmt` holding the variable and the three expressions, with `Step` nil when `STEP` is omitted.
- [x] **PARSER-043**: If a `FOR` statement's variable is an integer variable, or it lacks its variable, `=`, start value, `TO`, end value, or (after `STEP`) step, then the parser shall return a SYNTAX error with an `*ast.BadStmt` in place of the statement.
- [x] **PARSER-044**: When a statement is `NEXT` followed by `:` or `EOL`, the parser shall produce an `*ast.NextStmt` with no variables; when it is `NEXT` followed by variables separated by commas, an `*ast.NextStmt` holding them in order.
- [x] **PARSER-045**: If the variables after `NEXT` are not separated by commas, or a comma is not followed by a variable, then the parser shall return a SYNTAX error with an `*ast.BadStmt` in place of the statement.

## Subroutines

- [x] **PARSER-046**: When a statement is `GOSUB`, the parser shall produce an `*ast.GosubStmt` whose `Line` is the line number read as PARSER-039 specifies.
- [x] **PARSER-047**: When a statement is `RETURN` followed by `:` or `EOL`, the parser shall produce an `*ast.ReturnStmt`; if `RETURN` is followed by any other token, or the line number after `GOSUB` is above 63999, the parser shall return a SYNTAX error with an `*ast.BadStmt` in place of the statement.

## Keyboard input

- [x] **PARSER-048**: When a statement is `INPUT`, optionally followed by a string literal and `;`, then one or more variables separated by commas, the parser shall produce an `*ast.InputStmt` holding the prompt (with `HasPrompt` true) if there is one, and the variables in order.
- [x] **PARSER-049**: When a statement is `GET` followed by one or more variables separated by commas, the parser shall produce an `*ast.GetStmt` holding the variables in order.
- [x] **PARSER-050**: If an `INPUT` prompt is not a string literal followed by `;`, or an `INPUT` or `GET` lacks a variable, or its variable list is not separated by commas or not followed by `:` or `EOL`, then the parser shall return a SYNTAX error with an `*ast.BadStmt` in place of the statement.

## User-defined functions

- [x] **PARSER-051**: When a statement is `DEF`, `FN`, a name, `(`, a name, `)`, `=`, and an expression followed by `:` or `EOL`, the parser shall produce an `*ast.DefStmt` holding the function's name, the parameter, and the expression as `Body`, with `BodyErr` nil.
- [x] **PARSER-052**: If the body of a `DEF FN` is not a valid expression followed by `:` or `EOL`, then the parser shall produce the `*ast.DefStmt` with `Body` nil and `BodyErr` a SYNTAX error, continue after the next `:` (or at `EOL`), and return no error for the statement.
- [x] **PARSER-053**: If a `DEF` lacks `FN`, the name, `(`, the parameter, `)`, or `=`, or its name or parameter is an integer name, then the parser shall return a SYNTAX error with an `*ast.BadStmt` in place of the statement.
- [x] **PARSER-054**: When `FN` appears where an operand is expected, followed by a name, `(`, an expression, and `)`, the parser shall produce an `*ast.FnExpr`; if the name is an integer name or a part is missing, the parser shall return a SYNTAX error.

## Program files

- [x] **PARSER-055**: When a statement is `LOAD`, `SAVE`, or `VERIFY` followed by up to three expressions separated by commas and then `:` or `EOL`, the parser shall produce an `*ast.LoadStmt`, `*ast.SaveStmt`, or `*ast.VerifyStmt` whose `Name`, `Device`, and `Secondary` hold the expressions in order, nil for those omitted.
- [x] **PARSER-056**: If a comma after `LOAD`, `SAVE`, or `VERIFY` or one of its arguments is not followed by an expression, or the arguments are followed by anything other than `:` or `EOL`, then the parser shall return a SYNTAX error with an `*ast.BadStmt` in place of the statement.
