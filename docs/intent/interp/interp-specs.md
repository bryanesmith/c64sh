# Interpreter Specs

Design: `interp-design.md`

## Execution

- [x] **INTERP-001**: When `Interp.Exec` is given a line, it shall execute the line's statements in order; if a statement fails, it shall return that statement's error and not execute later statements.
- [x] **INTERP-002**: `interp.New(out)` shall return an interpreter that writes all program output to `out` and nowhere else.
- [x] **INTERP-003**: If the interpreter is given a statement, print item, or expression node type it does not handle, then it shall panic with a message naming the node's Go type.
- [x] **INTERP-014**: When executing an `ast.RemStmt` (a `REM` comment), the interpreter shall write nothing, return no error, and continue with the next statement.

## PRINT

- [x] **INTERP-004**: When executing a `PRINT` statement, the interpreter shall write, for each item in order: the string value of an `ExprItem`'s expression; nothing for a `Semicolon`; a tab character (`\t`) for a `Comma`.
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
