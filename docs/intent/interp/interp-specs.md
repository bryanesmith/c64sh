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

- [x] **INTERP-004**: When executing a `PRINT` statement, the interpreter shall write, for each item in order: the string value of an `ExprItem`'s expression; nothing for a `Semicolon`; spaces up to the next print zone for a `Comma` (INTERP-015).
- [x] **INTERP-005**: When executing a `PRINT` statement whose last item is neither `Semicolon` nor `Comma`, including a `PRINT` with no items, the interpreter shall write a newline (`\n`) after the items.
- [x] **INTERP-006**: When executing a `PRINT` statement whose last item is `Semicolon` or `Comma`, the interpreter shall not write a newline after the items.
- [x] **INTERP-007**: When a `PRINT` statement executes without error, the interpreter shall write its entire output in a single call to the output writer.
- [x] **INTERP-008**: If an item of a `PRINT` statement fails, because evaluating its expression fails or because it is an `ast.BadItem` (a syntax error recorded by the parser), then the interpreter shall write the output of the items before it in a single call, write no newline, and return the item's error.

## Expressions

- [x] **INTERP-009**: When evaluating a `StringLit`, the interpreter shall return its value unchanged.
- [x] **INTERP-010**: When evaluating a `Concat`, the interpreter shall return the left operand's value followed by the right operand's value.
- [x] **INTERP-011**: If a `Concat` result would be longer than 255 characters, counted as Unicode code points with each byte that is not valid UTF-8 counting as one, then the interpreter shall fail with a STRING TOO LONG error (the `basicerr.StringTooLong` kind).
- [x] **INTERP-012**: The interpreter shall evaluate a `StringLit` of any length without error when it is not part of a `Concat`.

## Output errors

- [x] **INTERP-013**: If writing to the output writer fails, then `Interp.Exec` shall return the writer's error unchanged and execute no further statements.
