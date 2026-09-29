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
