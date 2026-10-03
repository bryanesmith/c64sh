# Interpreter Specs

Design: `interp-design.md`

## Execution

- [x] **INTERP-001**: When `Interp.Exec` is given a line, it shall execute the line's statements in order; if a statement fails, it shall return that statement's error and not execute later statements.
- [x] **INTERP-002**: `interp.New(out)` shall return an interpreter that writes all program output to `out` and nowhere else.
- [x] **INTERP-003**: If the interpreter is given a statement, print item, or expression node type it does not handle, then it shall panic with a message naming the node's Go type.
- [x] **INTERP-014**: When executing an `ast.RemStmt` (a `REM` comment), the interpreter shall write nothing, return no error, and continue with the next statement.

## Print zones and the cursor column

- [x] **INTERP-015**: When executing a `Comma` item in a `PRINT` statement, the interpreter shall write `10 - (c mod 10)` spaces, where `c` is the cursor column at that point, so the next item starts at the next multiple of 10 (a comma at column 0, 10, 20, … writes 10 spaces).
- [x] **INTERP-016**: The interpreter shall track the cursor column as the number of characters (Unicode code points, with each byte that is not valid UTF-8 counting as one) written to its output since the most recent newline, starting at 0 and carrying over between statements and between calls to `Exec`, and `Interp.Column` shall return it.
- [x] **INTERP-017**: When `Interp.FreshLine` is called while the cursor column is not 0, the interpreter shall write a newline (`
`) to its output and return any write error; while the column is 0, it shall write nothing and return nil.
- [x] **INTERP-018**: If a write to the output fails, then the interpreter shall advance the cursor column only by the characters actually written.

## PRINT

- [x] **INTERP-004**: When executing a `PRINT` statement, the interpreter shall write, for each item in order: for an `ExprItem`, a string value as it is, or a number value in C64 number format (INTERP-019) followed by one space; nothing for a `Semicolon`; spaces up to the next print zone for a `Comma` (INTERP-015).
- [x] **INTERP-005**: When executing a `PRINT` statement whose last item is neither `Semicolon` nor `Comma`, including a `PRINT` with no items, the interpreter shall write a newline (`\n`) after the items.
- [x] **INTERP-006**: When executing a `PRINT` statement whose last item is `Semicolon` or `Comma`, the interpreter shall not write a newline after the items.
- [x] **INTERP-007**: When a `PRINT` statement executes without error, the interpreter shall write its entire output in a single call to the output writer.
- [x] **INTERP-008**: If an item of a `PRINT` statement fails, because evaluating its expression fails or because it is an `ast.BadItem` (a syntax error recorded by the parser), then the interpreter shall write the output of the items before it in a single call, write no newline, and return the item's error.

## Expressions

- [x] **INTERP-009**: When evaluating a `StringLit`, the interpreter shall return its value unchanged.
- [x] **INTERP-010**: When evaluating an `ast.BinaryExpr` with `Op` `Add` whose operands are both strings, the interpreter shall return the left string followed by the right string.
- [x] **INTERP-011**: If joining two strings with `+` would produce a string longer than 255 characters, counted as Unicode code points with each byte that is not valid UTF-8 counting as one, then the interpreter shall fail with a STRING TOO LONG error (the `basicerr.StringTooLong` kind).
- [x] **INTERP-012**: The interpreter shall evaluate a `StringLit` of any length without error when it is not an operand of `+`.

## Output errors

- [x] **INTERP-013**: If writing to the output writer fails, then `Interp.Exec` shall return the writer's error unchanged and execute no further statements.

## Numbers

- [x] **INTERP-019**: When formatting a number for `PRINT`, the interpreter shall write a space for zero (including negative zero) or a positive number or `-` for a negative one, followed by `0` for zero, or otherwise the number rounded to 9 significant digits: in fixed notation when the rounded size is at least 0.01 and below 1E9, with trailing zeros and a bare `.` removed and no `0` before the `.` (`.5`, `.01`, `123456789`); and otherwise in scientific notation as `d.dddddddd` with trailing zeros and a bare `.` removed, then `E`, `+` or `-`, and the exponent in at least two digits (`1E-03`, `1.23456789E+09`).
- [x] **INTERP-020**: When evaluating an `ast.NumberLit`, the interpreter shall return its `Value` as a number.
- [x] **INTERP-021**: When evaluating an `ast.BinaryExpr` with `Op` `Add` whose operands are both numbers, the interpreter shall return their sum.
- [x] **INTERP-022**: If `+` has a string operand and a number operand, in either order, then the interpreter shall fail with a TYPE MISMATCH error (the `basicerr.TypeMismatch` kind).
- [x] **INTERP-023**: If a number literal's value or the result of any arithmetic operation (`+`, `-`, `*`, `/`, `^`, or negation) is larger in size than 1.70141183E+38 (the C64's largest number, `2^127 × (1 − 2^-32)`), then the interpreter shall fail with an OVERFLOW error (the `basicerr.Overflow` kind).
- [x] **INTERP-024**: When a number literal's value or the result of any arithmetic operation is nonzero but smaller in size than 2.93873588E-39 (`2^-128`, the C64's smallest positive number), the interpreter shall use 0 instead.
- [x] **INTERP-025**: When evaluating an `ast.BinaryExpr`, the interpreter shall evaluate the left operand, then the right operand, then apply the operator, returning the first error that occurs (so `1E39+"A"` fails with OVERFLOW, not TYPE MISMATCH).
- [x] **INTERP-026**: When evaluating an `ast.BinaryExpr` with `Op` `Sub`, `Mul`, or `Div` whose operands are both numbers, the interpreter shall return their difference, product, or quotient.
- [x] **INTERP-027**: If `-`, `*`, `/`, or `^` has a string operand, or negation is applied to a string, then the interpreter shall fail with a TYPE MISMATCH error (the `basicerr.TypeMismatch` kind), checked before any division by zero (`"A"/0` is TYPE MISMATCH).
- [x] **INTERP-028**: If `/` has two number operands and the right one is 0, then the interpreter shall fail with a DIVISION BY ZERO error (the `basicerr.DivisionByZero` kind).
- [x] **INTERP-029**: When evaluating an `ast.NegExpr` whose operand is a number, the interpreter shall return the number negated.
- [x] **INTERP-030**: When evaluating an `ast.BinaryExpr` with `Op` `Pow` whose operands are both numbers, the interpreter shall return: 1 if the right operand is 0 (so `0^0` is 1); otherwise 0 if the left operand is 0 (so `0^-1` is 0); otherwise, for a negative left operand and a whole-number right operand, the left operand's size raised to the right operand, negated if the right operand is odd (`(-2)^3` is -8); otherwise the left operand raised to the right operand.
- [x] **INTERP-031**: If `^` has a negative left operand and a right operand that is not a whole number, then the interpreter shall fail with an ILLEGAL QUANTITY error (the `basicerr.IllegalQuantity` kind).
- [x] **INTERP-032**: When executing an `ast.LetStmt` whose value has the variable's type (a string for a name ending in `$`, otherwise a number), the interpreter shall store the value (for an integer variable, as INTERP-037 converts it) under the variable's identity (`VarRef.Name`), where it remains for later statements and later calls to `Exec` until assigned again.
- [x] **INTERP-033**: When evaluating an `ast.VarRef`, the interpreter shall return the value stored under its identity, or, for a variable never assigned, 0 for a number or integer variable and the empty string for a string variable.
- [x] **INTERP-034**: If an `ast.LetStmt` assigns a number to a string variable or a string to a number or integer variable, then the interpreter shall fail with a TYPE MISMATCH error and leave the variable unchanged.
- [x] **INTERP-035**: If an `ast.LetStmt` assigns a string longer than 255 characters (counted as for INTERP-011) to a string variable, then the interpreter shall fail with a STRING TOO LONG error and leave the variable unchanged.
- [x] **INTERP-036**: If evaluating an `ast.LetStmt`'s value fails, then the interpreter shall return that error and leave the variable unchanged.
- [x] **INTERP-037**: When an `ast.LetStmt` assigns a number to an integer variable (a name ending in `%`), the interpreter shall store the number rounded down to a whole number (`3.7` stores 3, and `-3.7` stores -4).
- [x] **INTERP-038**: If an `ast.LetStmt` assigns to an integer variable a number whose size is not less than 32768, other than exactly -32768, then the interpreter shall fail with an ILLEGAL QUANTITY error and leave the variable unchanged (so -32768.5 fails, while -32767.5 stores -32768 and 32767.9 stores 32767).
- [x] **INTERP-039**: When evaluating an `ast.CompareExpr` whose operands are both numbers, the interpreter shall return -1 if the left operand's relation to the right (less, equal, or greater) is in the expression's `Relation`, and 0 otherwise.
- [x] **INTERP-040**: When evaluating an `ast.CompareExpr` whose operands are both strings, the interpreter shall compare them byte by byte from the start, a string that is the start of a longer one being less (`"A"<"AB"`), and return -1 if that relation is in the expression's `Relation`, and 0 otherwise.
- [x] **INTERP-041**: If an `ast.CompareExpr` has one string operand and one number operand, then the interpreter shall fail with a TYPE MISMATCH error.
- [x] **INTERP-042**: When evaluating an `ast.BinaryExpr` with `Op` `And` or `Or` whose operands are both numbers, the interpreter shall convert each operand to a 16-bit whole number, rounding down, and return the bitwise AND or OR of the two as a number (`12 AND 10` is 8, `12 OR 10` is 14, and `-1 AND 0` is 0).
- [x] **INTERP-043**: When evaluating an `ast.NotExpr` whose operand is a number, the interpreter shall convert it to a 16-bit whole number, rounding down, and return its bitwise complement as a number (`NOT 0` is -1, `NOT -1` is 0, and `NOT 5` is -6).
- [x] **INTERP-044**: If an operand of `AND`, `OR`, or `NOT` is a number whose size is 32768 or more, other than exactly -32768, then the interpreter shall fail with an ILLEGAL QUANTITY error; if it is a string, then with a TYPE MISMATCH error.
- [x] **INTERP-045**: When executing an `ast.IfStmt` whose condition is a number other than 0, or a string that is not empty, the interpreter shall continue with the next statement on the line.
- [x] **INTERP-046**: When executing an `ast.IfStmt` whose condition is 0 or the empty string, the interpreter shall execute no further statements on the line and return no error, even if those statements include an `ast.BadStmt` or a `PRINT` with an `ast.BadItem`.
- [x] **INTERP-047**: When executing an `ast.BadStmt`, the interpreter shall return the error it holds.

## Program mode

- [x] **INTERP-048**: When `Store` is called with line number `n` and non-empty text, the interpreter shall store the text and the result of lexing and parsing it as program line `n`, replacing any existing line `n`, keeping the program's lines in ascending order of line number.
- [x] **INTERP-049**: When `Store` is called with line number `n` and empty text, the interpreter shall delete program line `n` if there is one.
- [x] **INTERP-050**: When `Store` is called, the interpreter shall clear all variables, whether it stores, replaces, or deletes a line, or finds no line to delete.
- [x] **INTERP-051**: When `Store` is given text containing a syntax error, the interpreter shall store the line without reporting the error, which is reported only if execution reaches it when the line runs.
- [x] **INTERP-052**: When executing an `ast.RunStmt` with `HasLine` false, the interpreter shall clear all variables, execute the statements of each program line in ascending order of line number, starting from the first line, and end the program after the last line; with an empty program it shall do nothing further.
- [x] **INTERP-053**: When executing an `ast.RunStmt` with `HasLine` true, the interpreter shall clear all variables and run the program as in INTERP-052, starting from line `Line`.
- [x] **INTERP-054**: If an `ast.RunStmt` names a line that is not in the program, then the interpreter shall fail with an UNDEF'D STATEMENT error, after clearing the variables.
- [x] **INTERP-055**: When an `ast.IfStmt`'s condition is false in a running program, the interpreter shall skip the rest of that line only and continue with the next program line.
- [x] **INTERP-056**: If a statement fails while the program is running, then the interpreter shall stop the program and return the error with `Line` set to the number of the line containing the statement and `HasLine` true (for a failing `RUN n`, the line holding the `RUN`); an error in direct mode shall have `HasLine` false.
- [x] **INTERP-057**: When executing an `ast.RunStmt`, `ast.GotoStmt`, `ast.ListStmt`, `ast.NewStmt`, or `ast.EndStmt`, the interpreter shall execute no further statements on the line holding it, in direct mode or in a program.
- [x] **INTERP-058**: When executing an `ast.EndStmt` in a running program, or an `ast.ListStmt` or `ast.NewStmt`, the interpreter shall end the program, if one is running, and return no error.
- [x] **INTERP-059**: When executing an `ast.ListStmt`, the interpreter shall write, for each program line in ascending order, a newline, the line number without a leading space, one space, and the line's text with each `?` that the lexer reads as a `Print` token written as `PRINT` (so `10 ?"HI"` lists as `10 PRINT"HI"`), followed by a newline after the last line; for an empty program it shall write nothing.
- [x] **INTERP-060**: When executing an `ast.NewStmt`, the interpreter shall delete every program line and clear all variables.
- [x] **INTERP-061**: When executing an `ast.RunStmt` in a running program, the interpreter shall start the program again as INTERP-052 and INTERP-053 specify.
- [x] **INTERP-062**: `NeverRun` shall return true when the program holds at least one line and no `ast.RunStmt` or `ast.GotoStmt` has been executed since the interpreter was created, and false otherwise.
- [x] **INTERP-063**: When executing an `ast.GotoStmt` whose line is in the program, the interpreter shall continue the program at that line, without clearing the variables, whether the `GotoStmt` is in a running program or in direct mode.
- [x] **INTERP-064**: If an `ast.GotoStmt` names a line that is not in the program, then the interpreter shall fail with an UNDEF'D STATEMENT error, with `Line` the line holding the `GotoStmt` when the program is running.
- [x] **INTERP-065**: When `Interrupt` has been called, the interpreter shall, after the statement executing at that time finishes normally (including a false `IF`, or a `RUN` or `GOTO` before its jump), stop and return a BREAK error (`basicerr.Break`), with `Line` the line holding that statement when the program is running, and no line in direct mode.
- [x] **INTERP-066**: When `Exec` starts, the interpreter shall discard any earlier call to `Interrupt`.
- [x] **INTERP-067**: The interpreter shall allow `Interrupt` to be called from another goroutine while `Exec` runs.

## Loops

- [x] **INTERP-068**: When executing an `ast.ForStmt`, the interpreter shall assign the start value to the variable as an assignment does, then evaluate the end value and the step (1 when `Step` is nil) once, push a `FOR` entry holding the variable, the end value, the step, and the position after the statement onto the control stack, and continue with the next statement.
- [x] **INTERP-069**: If an `ast.ForStmt`'s variable is a string variable, or its end value or step is a string, then the interpreter shall fail with a TYPE MISMATCH error (for a string variable, after assigning the start value).
- [x] **INTERP-070**: When executing an `ast.ForStmt` whose variable already has a `FOR` entry above the topmost non-`FOR` entry of the control stack, the interpreter shall remove that entry and every entry above it before pushing the new one.
- [x] **INTERP-071**: When executing an `ast.NextStmt` for a variable with a matching `FOR` entry, the interpreter shall add the entry's step to the variable, store the sum in the variable, and, unless the sum is greater than the end value for a positive step, less than it for a negative step, or equal to it for a step of 0, continue execution at the entry's position.
- [x] **INTERP-072**: When an `ast.NextStmt` finds its loop complete (INTERP-071), the interpreter shall remove the `FOR` entry and continue with the next variable of the `NEXT`, if any, as INTERP-071 specifies, or else with the next statement.
- [x] **INTERP-073**: When executing an `ast.NextStmt`, the interpreter shall search the control stack from the top for the `FOR` entry of its variable, or for the topmost `FOR` entry if it has no variable, passing over `FOR` entries for other variables and stopping at the first entry that is not a `FOR`, and shall remove every entry above the one found; if none is found, it shall fail with a NEXT WITHOUT FOR error.
- [x] **INTERP-074**: The interpreter shall continue a loop at the statement after its `FOR`, whether that statement is in the middle of a line or in the direct-mode line given to `Exec` (so `FOR I=1 TO 3:PRINT I:NEXT` typed directly prints 1, 2, and 3).
- [x] **INTERP-075**: If, when an `ast.ForStmt` is about to push its entry, the entries on the control stack take 169 bytes or more, counting 18 bytes per `FOR` entry, then the interpreter shall fail with an OUT OF MEMORY error.
- [x] **INTERP-076**: The interpreter shall empty the control stack whenever it clears the variables (`RUN`, `NEW`, `Store`) and when `Exec` returns a BASIC error other than BREAK, and when `Exec` returns, shall remove the first entry whose position is in its direct-mode line and every entry above it.
