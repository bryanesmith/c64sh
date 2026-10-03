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
- [x] **INTERP-062**: `NeverRun` shall return true when the program holds at least one line and no `ast.RunStmt`, `ast.GotoStmt`, or `ast.GosubStmt` has been executed since the interpreter was created, and false otherwise.
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
- [x] **INTERP-075**: If, when an `ast.ForStmt` is about to push its entry, the entries on the control stack take 169 bytes or more, or when an `ast.GosubStmt` is about to push its entry, they take 179 bytes or more, counting 18 bytes per `FOR` entry and 7 bytes per `GOSUB` entry, then the interpreter shall fail with an OUT OF MEMORY error.
- [x] **INTERP-076**: The interpreter shall empty the control stack whenever it clears the variables (`RUN`, `NEW`, `Store`) and when `Exec` returns a BASIC error other than BREAK, and when `Exec` returns, shall remove the first entry whose position is in its direct-mode line and every entry above it.

## Subroutines

- [x] **INTERP-077**: When executing an `ast.GosubStmt`, the interpreter shall push a `GOSUB` entry holding the position after the statement onto the control stack and continue at line `Line`, without clearing the variables, in a running program or in direct mode; if there is no such line, it shall fail with an UNDEF'D STATEMENT error.
- [x] **INTERP-078**: When executing an `ast.ReturnStmt`, the interpreter shall search the control stack from the top, passing over `FOR` entries, to the first entry that is not a `FOR` entry; if it is a `GOSUB` entry, the interpreter shall remove it and every entry above it, and continue at its position (which may be in the middle of a line or in the direct-mode line).
- [x] **INTERP-079**: If executing an `ast.ReturnStmt` finds no `GOSUB` entry (INTERP-078), then the interpreter shall fail with a RETURN WITHOUT GOSUB error.
- [x] **INTERP-080**: When searching for a `FOR` entry (INTERP-070, INTERP-073), the interpreter shall not pass a `GOSUB` entry, so that a `NEXT` inside a subroutine cannot continue a loop begun outside it.

## Keyboard input

- [x] **INTERP-081**: If an `ast.InputStmt` or `ast.GetStmt` executes in direct mode, then the interpreter shall fail with an ILLEGAL DIRECT error, an `INPUT` after writing its prompt, if any.
- [x] **INTERP-082**: When executing an `ast.InputStmt`, the interpreter shall write its prompt, if any, and `? `, read a line from the console, remove spaces at its end, and write the line if the console reports it not echoed, then a newline.
- [x] **INTERP-083**: When the line an `ast.InputStmt` reads first is empty, the interpreter shall end the statement without changing any variable.
- [x] **INTERP-084**: When reading a value for an `INPUT` variable, the interpreter shall skip spaces, then take, for a string variable, a quoted value up to its closing quote or the end of the line, or else the text up to the next `,`, `:`, or end of the line; and for a number variable, a number written as in a program (spaces inside ignored, 0 if it has no digits); and shall assign the value before reading the next.
- [x] **INTERP-085**: If, after an `INPUT` value and any spaces, the next character is not `,`, `:`, or the end of the line, then the interpreter shall write `?REDO FROM START` and a newline and execute the statement again from its prompt, keeping values already assigned.
- [x] **INTERP-086**: When an `INPUT` line has no more values (its end, or a `:`) and variables remain, the interpreter shall write `?? ` and read a new line as INTERP-082 specifies, taking the remaining values from it (an empty line giving the empty string or 0).
- [x] **INTERP-087**: When every variable of an `ast.InputStmt` has a value and text remains in the line, the interpreter shall write `?EXTRA IGNORED` and a newline.
- [x] **INTERP-088**: When executing an `ast.GetStmt`, the interpreter shall read one key from the console for each variable in turn and assign it to a string variable as it is (the empty string when no key is waiting).
- [x] **INTERP-089**: When `GET` assigns a key to a number variable, the interpreter shall assign 0 for no key, a space, `.`, `+`, `-`, or `E`, and the value of a digit; for any other key, it shall fail with a SYNTAX error that carries no line number, even in a running program.
- [x] **INTERP-090**: If the console returns `io.EOF`, or no console is set, then `INPUT` and `GET` shall stop and `Exec` shall return `ErrEndOfInput`; if the console returns `ErrInterrupted`, they shall fail with a BREAK error.

## User-defined functions

- [x] **INTERP-091**: When executing an `ast.DefStmt` in a running program, the interpreter shall record the definition under the identity of its name, replacing any earlier one; if the name is a string name, it shall fail with TYPE MISMATCH, then, in direct mode, with ILLEGAL DIRECT, then, if the parameter is a string variable, with TYPE MISMATCH.
- [x] **INTERP-092**: When evaluating an `ast.FnExpr`, the interpreter shall fail with TYPE MISMATCH if the name is a string name; otherwise evaluate the argument (TYPE MISMATCH if it is a string), fail with UNDEF'D FUNCTION if no function of that name is defined, assign the argument to the parameter variable, evaluate the body (failing with its `BodyErr` if it has one, or TYPE MISMATCH if the body's value is a string), restore the parameter variable's previous value, and return the body's value.
- [x] **INTERP-093**: If a function call fails, then the interpreter shall leave the parameter variable holding the argument.
- [x] **INTERP-094**: If evaluating an `ast.FnExpr` would make 10 function calls in progress, then the interpreter shall fail with OUT OF MEMORY.
- [x] **INTERP-095**: The interpreter shall clear all function definitions whenever it clears the variables.

## Program files

- [x] **INTERP-096**: When executing a `LOAD`, `SAVE`, or `VERIFY`, the interpreter shall evaluate its name (TYPE MISMATCH unless a string), its device (1 if omitted), and its secondary address, each number rounded down and required to be 0 to 255 (else ILLEGAL QUANTITY).
- [x] **INTERP-097**: If the device of a `LOAD`, `SAVE`, or `VERIFY` is 0, 3, 4, or 5, then the interpreter shall fail with ILLEGAL DEVICE NUMBER; if it is not 1 or 8 to 11, or no storage is set, with DEVICE NOT PRESENT.
- [x] **INTERP-098**: If the name of a `LOAD`, `SAVE`, or `VERIFY` is empty or omitted, then the interpreter shall fail with MISSING FILE NAME.
- [x] **INTERP-099**: For a disk device (8 to 11), the interpreter shall remove a leading `@0:` or `@:` from the name, noting that the file may be replaced, or else a leading `0:`; and for any device, the file in storage shall be the name with `.bas` added if it contains no `.`.
- [x] **INTERP-100**: When executing a `SAVE`, the interpreter shall write to its file `#!/usr/bin/env c64sh` and a newline, then each program line in order as its number, a space, its stored text, and a newline, replacing an existing file for tape (device 1) or a name that began with `@0:` or `@:`.
- [x] **INTERP-101**: If storage refuses or fails to write the file of a `SAVE` (including an existing disk file that may not be replaced), or fails to read a file other than by its absence, then `Exec` shall return a `*StorageError` holding the file and the storage's error, with `Replace` set to `SAVE "@0:NAME"` for an existing disk file, `NAME` being the name without a leading `@0:`, `@:`, or `0:`.
- [x] **INTERP-102**: When executing a `LOAD` or `VERIFY`, the interpreter shall read the file named as given, or, if there is none and the name contains no `.`, the name with `.bas`; if neither exists, it shall fail with FILE NOT FOUND.
- [x] **INTERP-103**: When a `LOAD` reads a file, the interpreter shall accept an optional first line beginning `#!`, blank lines, and lines beginning with a line number, each stored as if typed in file order; for any other line, or a line number above 63999, it shall fail with LOAD and leave the program unchanged.
- [x] **INTERP-104**: When a `LOAD` executes in direct mode, the interpreter shall replace the program with the file's lines, clear the variables, and execute no further statements on the line.
- [x] **INTERP-105**: When a `LOAD` executes in a running program, the interpreter shall replace the program with the file's lines, keep the variables, empty the control stack, and run the new program from its first line.
- [x] **INTERP-106**: When executing a `VERIFY`, the interpreter shall fail with VERIFY if the file is not a program (INTERP-103) or its lines' numbers and texts differ from the program's, and otherwise continue.
- [x] **INTERP-107**: When a `LOAD`, `SAVE`, or `VERIFY` executes in direct mode and a messages writer is set, the interpreter shall call `FreshLine` and write to the writer, one per line, the C64's messages for the statement and device: for tape `PRESS RECORD & PLAY ON TAPE`, `OK`, `SAVING NAME` (SAVE) or `PRESS PLAY ON TAPE`, `OK`, `SEARCHING FOR NAME`, `FOUND NAME`, then `LOADING` (LOAD) or `VERIFYING` (VERIFY); for disk `SAVING NAME` or `SEARCHING FOR NAME`, then `LOADING` or `VERIFYING`; then `OK` after a VERIFY that matches; stopping after `SEARCHING FOR NAME` when the file is not found, `NAME` being the name as given.

## Data files

- [x] **INTERP-108**: When executing an `ast.OpenStmt`, the interpreter shall evaluate the file number, the device (1 if omitted), and the secondary address (0 if omitted) as numbers from 0 to 255 and the name as a string (empty if omitted), and shall fail with NOT INPUT FILE if the file number is 0, FILE OPEN if it is open, or TOO MANY FILES if 10 files are open.
- [x] **INTERP-109**: When opening a file, the interpreter shall open the keyboard (device 0) for input from the console; the screen (3) and printers (4, 5) for output to the program output; tape (1) for reading with secondary address 0 and writing otherwise; a disk drive (8 to 11) for reading with secondary address 0, writing with 1, and for 2 to 14 by the mode after the name's commas (`W` write, `A` append, otherwise read); and shall fail with DEVICE NOT PRESENT for a disk drive's secondary address 15, any other device, or a storage device with no storage set.
- [x] **INTERP-110**: When opening a tape or disk file, the interpreter shall fail with MISSING FILE NAME for an empty name; treat a disk name's leading `@0:`, `@:`, or `0:` as for `SAVE`; use as the file in storage the name before its first comma; fail with FILE NOT FOUND if a file opened to read or append does not exist; and return a `*StorageError` with `fs.ErrExist` and `Replace` `"@0:NAME,S,W"` for a disk file opened to write that exists, without `@0:` or `@:`.
- [x] **INTERP-111**: When executing an `ast.CloseStmt`, the interpreter shall close the file, writing the output of a file opened to write or append to storage (returning a `*StorageError` if that fails), and returning output to the screen if `CMD` was sending it there; closing a file that is not open shall do nothing.
- [x] **INTERP-112**: The interpreter shall close every open file, as `CLOSE` does, whenever it clears the variables, and when `CloseFiles` is called.
- [x] **INTERP-113**: When executing an `ast.PrintStmt` with a `File`, the interpreter shall write its items to the file as `PRINT` writes them, with `\n` as the line end, failing with FILE NOT OPEN if the file is not open, or NOT OUTPUT FILE if it is input-only.
- [x] **INTERP-114**: When `PRINT#` or `CMD` writes a comma, the interpreter shall write `10 - (C mod 10)` spaces, where `C` is the screen's cursor column; output to a storage file shall not change the cursor column.
- [x] **INTERP-115**: When executing an `ast.CmdStmt`, the interpreter shall check and write to the file as `PRINT#` does, and then send output that `PRINT` and `LIST` would write to the screen to the file, until a `PRINT#`, `INPUT#`, or `GET#` finishes, the file is closed, or `Exec` returns an error.
- [x] **INTERP-116**: When executing an `ast.InputStmt` with a `File`, the interpreter shall read values as `INPUT` does, but with no prompt and no echo, skipping empty lines before the first value, taking further values from the following lines without `?? `, failing with FILE DATA for an unreadable number, ignoring left-over values silently, and ending the statement without changing the remaining variables if the file ends; lines shall end at `\n`, `\r\n`, or `\r`.
- [x] **INTERP-117**: When executing an `ast.GetStmt` with a `File`, the interpreter shall read one character per variable as `GET` reads a key, giving the empty string at the end of the file and `CHR$(13)` for a line end, and shall fail with ILLEGAL DIRECT in direct mode.
- [x] **INTERP-118**: If the file of an `INPUT#` or `GET#` is not open, then the interpreter shall fail with FILE NOT OPEN; if it is output-only, with NOT INPUT FILE; a file opened on the keyboard shall be read from the console.
- [x] **INTERP-119**: The interpreter shall set `ST` to 64 after an `INPUT#` or `GET#` whose reading reached the end of its file, and to 0 after any other `OPEN`, `PRINT#`, `CMD`, `INPUT#`, or `GET#`; `ST` shall start at 0.

## Computed jumps

- [x] **INTERP-120**: When executing an `ast.OnStmt`, the interpreter shall evaluate its index as a number from 0 to 255, rounded down (TYPE MISMATCH for a string, ILLEGAL QUANTITY out of range), and, if the index is from 1 to the number of line numbers, execute a `GOTO` or `GOSUB` to the line number at that place in the list, with a `GOSUB` entry holding the position after the `ON` statement.
- [x] **INTERP-121**: When an `ast.OnStmt`'s index is 0 or larger than the number of line numbers, the interpreter shall continue with the next statement.

## Number functions

- [x] **INTERP-122**: When evaluating an `ast.CallExpr` of a number function, the interpreter shall fail with TYPE MISMATCH if its argument is a string, and otherwise return for `ABS` the argument's size, `INT` the argument rounded down, `SGN` -1, 0, or 1 by its sign, and `SQR`, `LOG`, `EXP`, `SIN`, `COS`, `TAN`, and `ATN` the square root, natural logarithm, exponential, and trigonometric functions in radians, limited to the C64's range.
- [x] **INTERP-123**: If `SQR`'s argument is negative or `LOG`'s is 0 or less, then the interpreter shall fail with ILLEGAL QUANTITY; if `TAN`'s cosine is 0, with DIVISION BY ZERO.
- [x] **INTERP-124**: When evaluating `RND(X)`, the interpreter shall return a number at least 0 and less than 1: for a negative `X`, after replacing the seed with one determined by `X` alone; for an `X` of 0, after replacing the seed with one determined by the clock; and for a positive `X`, after advancing the seed.
- [x] **INTERP-125**: The interpreter shall start the `RND` seed at the same value in every interpreter, and shall read the clock set with `SetClock`, or the system clock if none is set.

## String functions

- [x] **INTERP-126**: When evaluating a string function, the interpreter shall fail with TYPE MISMATCH for an argument of the wrong type, and with ILLEGAL QUANTITY for a position, length, or character code that, rounded down, is below 0 or above 255.
- [x] **INTERP-127**: The interpreter shall return for `LEN(S$)` the number of characters of `S$`; for `LEFT$(S$,N)` and `RIGHT$(S$,N)` its first or last `N` characters, or all of it if shorter; and for `MID$(S$,P,N)` the `N` characters (all, if `N` is omitted) from position `P`, counting from 1, or the empty string if `P` is past the end; `MID$` with `P` of 0 shall fail with ILLEGAL QUANTITY.
- [x] **INTERP-128**: The interpreter shall return for `CHR$(N)` the one-character string whose code is `N`, and for `ASC(S$)` the code of the first character of `S$`, failing with ILLEGAL QUANTITY for the empty string.
- [x] **INTERP-129**: The interpreter shall return for `STR$(X)` `X` formatted as `PRINT` formats a number, without the trailing space, and for `VAL(S$)` the number at the start of `S$`, read as `INPUT` reads a number, or 0 if there is none, failing with OVERFLOW if it is too large.

## Print formatting

- [x] **INTERP-130**: When executing a `TabItem`, the interpreter shall write `X - C` spaces if the cursor column `C` (the screen's, for a storage file) is less than `X`, and nothing otherwise; for an `SpcItem`, `X` spaces; `X` being rounded down and required to be from 0 to 255 (ILLEGAL QUANTITY otherwise, TYPE MISMATCH for a string).
- [x] **INTERP-131**: When the last item of a `PRINT` is a `TabItem` or `SpcItem`, the interpreter shall not write the final newline.
- [x] **INTERP-132**: When evaluating `POS(X)`, the interpreter shall evaluate `X`, of either type, and return the cursor column.

## Arrays

- [x] **INTERP-133**: When executing an `ast.DimStmt`, the interpreter shall create each array with subscripts, with the given tops, rounded down, and elements of 0 or the empty string; it shall fail with REDIM'D ARRAY if the array exists, and with ILLEGAL QUANTITY for a top below 0 or beyond -32768 to 32767; a variable without subscripts shall do nothing.
- [x] **INTERP-134**: When evaluating or assigning an array element, the interpreter shall evaluate its subscripts in order, rounded down, failing with ILLEGAL QUANTITY for one below 0 or beyond -32768 to 32767, and shall create a missing array with that many dimensions, each with top 10.
- [x] **INTERP-135**: If an array element has a different number of subscripts than its array has dimensions, or a subscript greater than its dimension's top, then the interpreter shall fail with BAD SUBSCRIPT.
- [x] **INTERP-136**: The interpreter shall keep arrays separate from plain variables of the same identity, and shall assign to an element by the rules for a plain variable of its type.
- [x] **INTERP-137**: If creating an array would make all arrays take more than 38911 bytes, counting 5 bytes plus 2 per dimension for each array and 5, 3, or 2 bytes per element for numbers, strings, or integers, then the interpreter shall fail with OUT OF MEMORY.
- [x] **INTERP-138**: The interpreter shall clear all arrays whenever it clears the variables.

## DATA statements

- [x] **INTERP-139**: When executing an `ast.ReadStmt`, the interpreter shall assign each variable, in order, the next item of the program's `DATA` statements in program order, read as `INPUT` reads a value (quoted or unquoted strings; numbers, an empty item being 0), items being separated by commas, and move the data pointer past it.
- [x] **INTERP-140**: If no `DATA` item remains, then `READ` shall fail with OUT OF DATA.
- [x] **INTERP-141**: If a `DATA` item cannot be read into its variable, or is followed by anything but a comma or the end of its text, then `READ` shall fail with a SYNTAX error carrying the number of the line holding the `DATA` statement.
- [x] **INTERP-142**: The interpreter shall move the data pointer to the first item when executing an `ast.RestoreStmt`, when it clears the variables, and when a `LOAD` chains; executing an `ast.DataStmt` shall do nothing.
