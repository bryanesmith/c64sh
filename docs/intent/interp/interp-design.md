---
parent: high-level-design
prefix: INTERP
---

# Interpreter

## Context and Design Philosophy

The interpreter executes an `*ast.Line` by walking it with type switches. It also holds the stored program: lines entered with a line number, which `RUN` executes in order. It writes program output to an `io.Writer` it is given and knows nothing about terminals, files, prompts, or how errors are displayed. Those belong to the shell.

Its output follows C64 BASIC V2 semantics, including the C64's 10-column print zones, adjusted for a terminal as the HLD's *C64 language, Unix I/O* tenet requires: where a C64 moves its cursor right to reach a print zone, the interpreter writes spaces.

Like the C64, the interpreter keeps track of the cursor column: the position on the current output line where the next character will appear. It is the only component that knows the column, and it is the only writer of program output, so the column is always accurate.

## API

```go
package interp

type Interp struct {
    // unexported: output writer
}

// New returns an interpreter that writes program output to out.
func New(out io.Writer) *Interp

// Exec runs the statements of line in order, in direct mode. It stops at
// the first statement that fails and returns that error; statements after
// it do not run. Output from statements before the failure has already
// been written. A RUN among the statements runs the stored program, and
// an error in it is returned with its line number.
func (in *Interp) Exec(line *ast.Line) error

// Store stores text as program line n (0 to 63999), replacing any line n,
// or deletes line n if text is empty. Either way, it clears the variables.
func (in *Interp) Store(n int, text string)

// NeverRun reports whether the stored program holds lines and no RUN,
// GOTO, or GOSUB has been executed since the Interp was created.
func (in *Interp) NeverRun() bool

// Interrupt asks the interpreter to stop, as the C64's STOP key does: the
// statement running now finishes, and Exec returns a BREAK error. It may
// be called from another goroutine.
func (in *Interp) Interrupt()

// Console is where INPUT and GET read the keyboard. stop reports whether
// the interpreter has been interrupted; a method that is waiting returns
// ErrInterrupted when it becomes true.
type Console interface {
    // ReadLine reads one line, without its line terminator. echoed reports
    // whether the console has already shown what was typed, as a terminal
    // does; if not, the interpreter writes it. At the end of input it
    // returns io.EOF.
    ReadLine(stop func() bool) (line string, echoed bool, err error)
    // ReadKey returns the next key as a one-character string, or "" if no
    // key is waiting. At the end of input it returns io.EOF.
    ReadKey(stop func() bool) (key string, err error)
}

// ErrInterrupted is returned by a Console that stopped waiting because
// the interpreter was interrupted.
var ErrInterrupted = errors.New("interrupted")

// ErrEndOfInput is returned by Exec when INPUT or GET found the end of
// the console's input. It is not a BASIC error.
var ErrEndOfInput = errors.New("end of input")

// SetConsole sets where INPUT and GET read. Without one, they find the
// end of input.
func (in *Interp) SetConsole(c Console)

// Column returns the cursor column: the number of characters written
// since the last newline. It is 0 at the start of a line.
func (in *Interp) Column() int

// FreshLine ends the current output line if it is unfinished: when the
// column is not 0, it writes a newline. It returns any write error.
func (in *Interp) FreshLine() error
```

An `Interp` carries state between lines: the cursor column, which starts at 0, the variables, which start empty, the stored program, which starts empty, and the control stack (see *Control stack*), which starts empty. It is created once per shell session, so all of them persist from line to line for the whole session or script.

## Cursor Column

The column counts the characters written since the last newline, across statements and lines: after `PRINT "AB";`, the next `PRINT` starts at column 2, whether it is on the same input line or a later one. Writing a newline sets it to 0.

- A character is a Unicode code point; each byte that is not valid UTF-8 counts as one, as in the 255-character string limit.
- Every character counts as one column, including characters a terminal displays differently, such as a tab or carriage return inside a string, or a double-width East Asian character. A C64 has none of these, so there is no C64 behavior to follow; the count stays simple and predictable, and alignment after such characters may look uneven in a terminal.
- The column is unbounded. A C64 wraps at 40 columns, but because 40 is a multiple of 10, print zones fall in the same places whether or not a line wraps, so no screen width needs to be modeled.
- The column is updated from what was actually written; if a write fails, the column is not advanced for it.

## Dispatch

Each node category is handled by one type switch whose `default` case panics with the unhandled node's type:

```go
func (in *Interp) execStmt(s ast.Stmt) error {
    switch s := s.(type) {
    case *ast.PrintStmt:
        return in.execPrint(s)
    case *ast.RemStmt:
        return nil
    case *ast.LetStmt:
        return in.execLet(s)
    case *ast.IfStmt:
        return in.execIf(s) // may end the line early
    case *ast.BadStmt:
        return s.Err
    case *ast.RunStmt, *ast.GotoStmt, *ast.ListStmt, *ast.NewStmt, *ast.EndStmt,
        *ast.ForStmt, *ast.NextStmt, *ast.GosubStmt, *ast.ReturnStmt,
        *ast.InputStmt, *ast.GetStmt:
        … // see Program mode
    default:
        panic(fmt.Sprintf("interp: unhandled statement %T", s))
    }
}
```

A panic here means a node type was added to `internal/ast` without interpreter support, which is a programming error, not a user error. Unit tests cover every node type so such a gap is caught before release.

## PRINT

`PRINT` processes its items left to right:

| Item | Output |
|---|---|
| `ExprItem` | A string value as it is; a number value in C64 number format (see *Numbers*) followed by one space. |
| `Semicolon` | Nothing. It only separates items. |
| `Comma` | Spaces up to the start of the next print zone: `10 - (column % 10)` spaces, where `column` is the cursor column at that point. This is never 0: at the start of a zone (column 0, 10, 20, …), a comma moves a full 10 columns, as the C64 ROM does. |
| `BadItem` | Nothing; execution of the `PRINT` fails with the item's error (see below). The parser produces a `BadItem` where a syntax error occurred inside a `PRINT`. |

After the last item, a newline (`\n`) is written **unless the last item is `;` or `,`**. This is the C64 rule, and it is how a BASIC program prints several things on one line:

| Input | Output |
|---|---|
| `PRINT` | `\n` |
| `PRINT "A"` | `A\n` |
| `PRINT "A";"B"` | `AB\n` |
| `PRINT "A""B"` | `AB\n` |
| `PRINT "A","B"` | `A` + 9 spaces + `B\n` (`B` at column 10) |
| `PRINT "A";` | `A` |
| `PRINT "A",` | `A` + 9 spaces (no newline; the column is 10) |
| `PRINT ,"A"` | 10 spaces + `A\n` (from column 0, a full zone) |
| `PRINT "0123456789","X"` | `0123456789` + 10 spaces + `X\n` (`X` at column 20) |
| `PRINT "A","B","C"` | `A` + 9 spaces + `B` + 9 spaces + `C\n` (columns 0, 10, 20) |
| `PRINT 45` | ` 45 \n` |
| `PRINT "5*9=";45` | `5*9= 45 \n` |
| `PRINT 2,3` | ` 2 ` + 7 spaces + ` 3 \n` (` 3 ` at column 10) |

The output for one `PRINT` is collected and written in a single call to the writer. If an item fails, either because evaluating its expression fails or because it is a `BadItem`, the output of the items before it is written (a C64 prints each item as it is evaluated), and then the error is returned. For `PRINT "A";X` where `X` fails, `A` is written, with no newline.

## IF

Executing an `IfStmt` evaluates its condition. A number is true when it is not 0. A string is true when it is not empty: the C64 ROM's `IF` (`$A928`) tests the byte where string evaluation leaves the string's length (`$B4D5`). When the condition is true, execution continues with the next statement on the line; when it is false, **the rest of the line is skipped** without error: no later statement runs, and a `BadStmt` or `BadItem` among them is never reached. An error while evaluating the condition is returned like any other.

## BadStmt

Executing a `BadStmt` returns its error: the syntax error the parser found at that point, reported now that execution has reached it.

## Program mode

### Storing lines

`Store` keeps each line's number, its text exactly as given (the text after the line number, which the shell has read with `lexer.LineNumber`), and the text lexed and parsed into an `*ast.Line`. Lines are kept in ascending order of line number. Storing a line with the number of an existing line replaces it, and storing empty text deletes the line, or does nothing if there is no such line.

The text is parsed when the line is stored, so running a line many times does not parse it again. A syntax error in it is part of the parsed tree (a `BadStmt` or `BadItem`), so it is reported only when execution reaches it, as on a C64, which checks a line's syntax only as it runs it.

**Storing or deleting a line clears the variables and the control stack**, as on a C64, where the variables live in memory just after the program and the ROM clears them whenever the program changes (`$A4ED`, `$A52A`): after `A=5` and `10 PRINT A`, `PRINT A` prints ` 0 `.

### Execution position

The interpreter executes statements from a **position**: a line, either a program line or the direct-mode line `Exec` was given, and the index of a statement within it. Executing a statement normally moves to the next statement on the same line. After the last statement of a program line, execution moves to the first statement of the next program line, and after the last program line, the program ends; after the last statement of the direct-mode line, `Exec` returns. A position can name a statement in the middle of a line, which is where `NEXT` loops back to: on a C64 the loop continues just after its `FOR`, even when that is in the middle of a line or in the line typed in direct mode (`FOR I=1 TO 3:PRINT I:NEXT`).

### Control stack

`FOR` and `GOSUB` push entries onto the control stack, which `NEXT` and `RETURN` read; the C64 keeps these on the processor stack (`$A742`, `$A883`). A `FOR` entry holds the loop variable, the end value, the step, and the position just after the `FOR`; a `GOSUB` entry holds the position just after the `GOSUB`.

- **Clearing.** The stack is emptied whenever the variables are cleared (`RUN`, `NEW`, storing or deleting a line) and when an error stops execution (`$A462`). It is kept when execution ends normally or with `BREAK`, as on a C64, so a `NEXT` typed after a program ends can continue a loop the program left unfinished. When `Exec` returns, entries whose position is in its direct-mode line are removed, together with every entry above them, because that line is gone; on a C64 they would point into the input buffer, which the next line overwrites.
- **Size.** Each `FOR` entry takes 18 bytes of the C64's stack, and each `GOSUB` entry 7 (5 bytes of entry, and the 2-byte return address of the statement loop that the `GOSUB` leaves behind it, which its `RETURN` uses). Before pushing an entry, the ROM checks for room (`$A3FB`); with the stack depth of a running program, a `FOR` fails with `OUT OF MEMORY` when the entries already on the stack take 169 bytes or more, and a `GOSUB` when they take 179 bytes or more. So 10 loops can be nested and the 11th fails, and 26 subroutine calls can be nested and the 27th fails. c64sh applies the same rule in direct mode. The stack space a C64 uses while evaluating expressions is not counted.

### FOR and NEXT

Executing `FOR V=A TO B STEP S` follows the ROM (`$A742`):

1. Assign `A` to `V`, as `LET` does (so a type mismatch or other error happens here). If `V` is a string variable, fail with `TYPE MISMATCH`.
2. Remove any `FOR` entry for `V` already on the stack, with every entry above it. Only entries above the topmost non-`FOR` entry are searched (`$A38A`).
3. Check for stack room (see *Control stack*).
4. Evaluate `B`, then `S` (1 if omitted); each must be a number, or the statement fails with `TYPE MISMATCH`.
5. Push the entry, and continue with the next statement.

So the body always runs at least once: the test is made only at `NEXT`.

Executing `NEXT V` (`$AD1E`) searches the stack from the top for the `FOR` entry of `V`, or for the topmost `FOR` entry if `NEXT` has no variable. The search passes over `FOR` entries for other variables and stops at the first entry that is not a `FOR`; if it finds no match it fails with `NEXT WITHOUT FOR`. Entries above the match are removed (they are loops left unfinished inside this one). Then it adds the step to `V`, stores the result in `V`, and compares it with the end value: the loop is complete when `V` has passed the end value, that is, it is greater for a positive step, less for a negative step, or equal for a step of 0. If the loop is not complete, execution continues at the entry's position; if it is, the entry is removed and execution continues after the variable, with the next variable of `NEXT I,J` if there is one. The variable keeps its last value after the loop: after `FOR I=1 TO 3:NEXT`, `I` is 4.

Because c64sh numbers are 64-bit floating point (HLD *Number representation*), a loop with a fractional step, such as `FOR I=0 TO 1 STEP .1`, can in rare cases run a different number of times than on a C64, whose numbers round differently.

### GOSUB and RETURN

`GOSUB n` checks for stack room (see *Control stack*), pushes a `GOSUB` entry holding the position just after the `GOSUB` statement, and continues at line `n` like `GOTO` (`$A883`); if there is no line `n`, it fails with `UNDEF'D STATEMENT`. Like `GOTO`, it works in direct mode, running the program from line `n`, and keeps the variables.

`RETURN` (`$A8D2`) searches the control stack from the top, passing over every `FOR` entry, to the first entry that is not a `FOR`. If that is a `GOSUB` entry, it removes it and every entry above it (loops begun inside the subroutine and left unfinished) and continues at the entry's position: the statement after the `GOSUB`, which can be in the middle of a line, or in the line typed in direct mode. Otherwise (the stack holds only `FOR` entries, or none) it fails with `RETURN WITHOUT GOSUB`.

Because `NEXT` and `FOR` search only above the topmost non-`FOR` entry, a `NEXT` inside a subroutine cannot reach a loop begun outside it: it fails with `NEXT WITHOUT FOR`.

### RUN

`RUN` clears the variables and runs the program from its first line; `RUN n` clears the variables and runs it from line `n`, or fails with `UNDEF'D STATEMENT` if there is no line `n` (the variables are cleared either way, as the ROM clears them first, `$A87D`). A `RUN` with an empty program does nothing. `RUN` never returns to the line it is on, so nothing after it on that line runs.

Running the program executes each line's statements, in line-number order, until one of these happens:

- **The last line finishes**: the program ends.
- **`END` runs**: the program ends.
- **`LIST` or `NEW` runs**: it does its work, then the program ends.
- **A statement fails**: the program stops, and the error is returned with the number of the line it occurred in (`Line` and `HasLine` on the `basicerr.Error`), so the shell prints `?SYNTAX  ERROR IN 20`.
- **`RUN` runs**: the program starts again, with the variables cleared.
- **`GOTO n` runs**: the program continues at line `n`, or stops with `UNDEF'D STATEMENT` if there is no line `n`.
- **The interpreter is interrupted** (see *BREAK*): the program stops with a `BREAK` error.

A false `IF` ends only its own line; the program continues with the next line. `RUN` and `NEW` also empty the control stack. Program output and the cursor column carry on across lines exactly as in direct mode.

### GOTO

`GOTO n` continues at line `n` without clearing the variables: in a running program it jumps there, and in direct mode it runs the program from there, which is how a C64 continues a program while keeping its variables. If there is no line `n`, it fails with `UNDEF'D STATEMENT`, carrying the line holding the `GOTO` when the program is running. Like `RUN`, it never returns to its line. `IF … THEN n` and `IF … GOTO n` are an `IfStmt` followed by a `GotoStmt`, so they jump only when the condition is true.

### BREAK

`Interrupt` sets a flag, safe to set from another goroutine, that the interpreter checks after each statement finishes, as the C64 checks its STOP key between statements (`$A7AE`, `$A82C`). The check happens after any statement that completes normally, including a false `IF` (which ends its line) and a `RUN` or `GOTO` (before the jump), but not after one that fails or ends execution (`END`, `LIST`, `NEW`). When the flag is set, the interpreter clears it and stops with a `BREAK` error (`basicerr.Break`). In a running program the error carries the line of the statement just finished, which is the line the C64 reports: after `20 GOTO 10`, it is line 20. In direct mode it carries no line.

`Exec` clears the flag when it starts, so an interrupt that arrives after a line has finished does not stop the next one.

### LIST

`LIST` writes the whole program, in line-number order, as the C64 ROM does (`$A6C9`). Before each line it writes a newline, then the line number with no leading space, one space, and the line's text: `10 PRINT "HI"`. In the text, a `?` that the lexer reads as `PRINT` (outside strings and comments) is written as `PRINT`, because a C64 stores both as the same keyword: `10 ?"HI"` lists as `10 PRINT"HI"`. Everything else is written as it was typed, including spaces. An empty program writes nothing.

Because each line starts with a newline, the listing begins with one: a blank line after the `LIST` command in a terminal, as on a C64. After the last line `LIST` writes a newline too, because on a C64 a listing is always followed by `READY.`, whose message begins with one (`$A714`, `$A376`); so whatever follows starts on a fresh line, in a script as well. Then `LIST` stops, like `END`: in a program, the program ends; in direct mode, the rest of the line does not run (`$A714` returns to `READY.`).

### NEW, END

`NEW` erases the program and clears the variables, then stops like `END`. `END` stops: it ends a running program, and in direct mode it ends the line. Neither writes anything.

### NeverRun

`NeverRun` lets the shell run a script's program when the script never ran it itself (see the shell design). It is true when the program holds at least one line and no `RUN`, `GOTO`, or `GOSUB` has been executed, successfully or not, since the `Interp` was created.

## Keyboard input

`INPUT` and `GET` read from the `Console` the shell sets (see the shell design). Both work only in a running program: in direct mode they fail with `ILLEGAL DIRECT` (`$B3A6`), `INPUT` after writing its prompt, as the ROM writes the prompt first (`$ABBF`).

### INPUT

`INPUT` follows the ROM (`$ABBF`, `$AC0D`):

1. Write the prompt, if there is one, then `? `.
2. Read a line with `ReadLine`. Spaces at the end of the line are removed, as the C64's screen editor removes them. Then end the output line: write the line if the console did not echo it, then a newline, so the column is 0.
3. If the line is empty, the statement ends; the variables keep their values.
4. Read a value for each variable in turn, from the current place in the line, and assign it at once:
   - Spaces before the value are skipped.
   - **A string variable** takes a quoted value up to the closing quote (or the end of the line), or else the text up to the next `,` or `:` or the end of the line, keeping spaces inside it.
   - **A number variable** takes a number written as in a program: an optional sign, digits with at most one `.`, an optional exponent, with spaces inside ignored. No digits at all gives 0. A number too large is `OVERFLOW`; for an integer variable, out of range is `ILLEGAL QUANTITY`.
   - After the value, after any spaces, the next character must be `,`, `:`, or the end of the line. Otherwise, write `?REDO FROM START` and a newline, and start the statement again from step 1. Values already assigned keep their new values.
   - A `,` is skipped before the next value. When the line has no more values (its end, or a `:`), and variables remain, write `?? ` and read a new line as in step 2, which supplies the next values. An empty line here gives the empty string or 0.
5. When every variable has a value, if text remains in the line (starting with `,` or `:`), write `?EXTRA IGNORED` and a newline.

`?REDO FROM START` and `?EXTRA IGNORED` are not errors: they are program output, written on the screen as a C64 writes them, and execution continues.

### GET

`GET` reads one key for each of its variables with `ReadKey`, in turn (`$AB7B`):

- **A string variable** takes the key, or the empty string if no key is waiting.
- **A number variable** takes 0 if no key is waiting or the key is a space, `.`, `+`, `-`, or `E`, and the digit's value if it is a digit. Any other key is a `SYNTAX` error, reported **without a line number** even in a running program, as on a C64, whose `GET` marks this error as if in direct mode (`$AB53`).

### Input errors

If the console returns `io.EOF`, the statement stops and `Exec` returns `ErrEndOfInput`, which is not a BASIC error. If it returns `ErrInterrupted`, the statement stops with a `BREAK` error, carrying the line in a running program. On a C64 the STOP key does nothing while `INPUT` waits; c64sh lets Ctrl-C stop a program waiting for input, the usual meaning of Ctrl-C in a terminal.

## REM

Executing a `RemStmt` does nothing: it writes no output and returns no error, so execution continues with the next statement. A `RemStmt` is always the last statement of a line, because its comment runs to the end of the line.

## Variables

Variables are kept in a map from a `VarRef`'s `Name` (its identity, such as `SC` or `N$`) to a value.

- **Assignment** (`LetStmt`) evaluates the value, checks its type against the variable's, and stores it. A string variable (a name ending in `$`) takes only strings, and number and integer variables only numbers; the wrong kind is `TYPE MISMATCH`.
- **An integer variable** (a name ending in `%`) stores the value rounded down to a whole number, as the C64 ROM's conversion does (`$BC9B`): `C%=3.7` stores 3 and `C%=-3.7` stores -4. A value whose size is 32768 or more, other than exactly -32768, is `ILLEGAL QUANTITY` (`$B1BF`); the range is checked before rounding, so -32768.5 is out of range while -32767.5 stores -32768. Integer variables are otherwise ordinary numbers in expressions. A string longer than 255 characters is `STRING TOO LONG`, the limit on any string a C64 stores. On any error the variable keeps its old value.
- **A reference** (`VarRef`) evaluates to the stored value, or, for a variable never assigned, to 0 or the empty string, as on a C64.

## Values

Every expression evaluates to a value that is either a **string** or a **number**. Numbers are Go `float64`s kept within the C64's range:

| Constant | Value | Meaning |
|---|---|---|
| `maxNumber` | 1.70141183E+38 (`2^127 × (1 − 2^-32)`) | Largest C64 number; anything larger is `OVERFLOW` |
| `minNumber` | 2.93873588E-39 (`2^-128`) | Smallest positive C64 number; anything smaller in size becomes 0 |

## Numbers

A number is printed as the C64 ROM formats it (`$BDDD`):

1. A **sign character**: a space for zero and positive numbers, `-` for negative ones.
2. The number rounded to **9 significant digits**.
3. **Fixed notation** when the rounded value is at least 0.01 and below 1E9: the digits with trailing zeros removed, and the `.` removed if nothing follows it. There is **no leading zero** before the `.`: `.5`, not `0.5`.
4. **Scientific notation** otherwise: the digits as `d.dddddddd` with trailing zeros (and a bare `.`) removed, then `E`, the exponent's sign (`+` or `-`), and the exponent as at least two digits.
5. Zero is printed as `0`.

| Value | Printed by `PRINT` |
|---|---|
| 45 | ` 45 ` |
| 3.14 | ` 3.14 ` |
| 0.5 | ` .5 ` |
| 1/3 | ` .333333333 ` |
| 0.01 | ` .01 ` |
| 0.001 | ` 1E-03 ` |
| 123456789 | ` 123456789 ` |
| 999999999.6 | ` 1E+09 ` (rounds up to 1E9) |
| 1234567890 | ` 1.23456789E+09 ` |
| 1E38 | ` 1E+38 ` |

The decision between fixed and scientific notation uses the value after rounding to 9 digits, as the ROM does.

## Expressions

A `BinaryExpr` evaluates its left operand, then its right operand, then applies its operator; the first error met is the one returned. So `1E39+"A"` is `OVERFLOW` (from the left operand), not `TYPE MISMATCH`, as on a C64, which reads the number before it reaches the `+`.

| Node | Value |
|---|---|
| `StringLit` | Its `Value`, a string. |
| `NumberLit` | Its `Value`, a number: `OVERFLOW` if its size exceeds `maxNumber`, 0 if its size is below `minNumber`. |
| `BinaryExpr` `Add` | Two strings: the left followed by the right (see *String length*). Two numbers: their sum. A string and a number, in either order: `TYPE MISMATCH`. |
| `BinaryExpr` `Sub`, `Mul` | Two numbers: the difference or product. Any string operand: `TYPE MISMATCH`. |
| `BinaryExpr` `Div` | Two numbers: the quotient, or `DIVISION BY ZERO` if the right operand is 0. Any string operand: `TYPE MISMATCH`, checked before the divisor. |
| `BinaryExpr` `Pow` | Two numbers: the left raised to the power of the right, following the C64 ROM's power routine (`$BF7B`): anything to the power 0 is 1 (including `0^0`); 0 to any other power is 0 (including a negative power, so `0^-1` is 0, not a division by zero); a negative number to a whole-number power is computed from its size, then negated if the power is odd (`(-2)^3` is -8); a negative number to a fractional power is `ILLEGAL QUANTITY`. Any string operand: `TYPE MISMATCH`. |
| `NegExpr` | A number: its negation. A string: `TYPE MISMATCH`. |
| `VarRef` | The variable's value; 0 or `""` if never assigned. |
| `BinaryExpr` `And`, `Or` | Two numbers, each converted to a 16-bit whole number as for an integer variable (rounded down; `ILLEGAL QUANTITY` if its size is 32768 or more, other than -32768); the result is their bitwise AND or OR, as a number. Any string operand: `TYPE MISMATCH`. |
| `NotExpr` | A number, converted to a 16-bit whole number as for `And`; the result is its bitwise complement (`NOT 0` is -1, `NOT -1` is 0, `NOT 5` is -6). A string: `TYPE MISMATCH`. |

Because true is -1 (every bit set) and false is 0, these bitwise operators also act as logical ones on comparison results: `1<2 AND 3<4` is -1, and `NOT (1=2)` is -1.

| Node | Value |
|---|---|
| `CompareExpr` | Two numbers, or two strings: -1 if the actual relation of the left operand to the right (less, equal, or greater) is in `Rel`, otherwise 0. A string and a number: `TYPE MISMATCH`. |

**Comparing strings** follows the C64: characters are compared one by one from the start, by their codes, and if one string is the start of the other, the shorter is less (`"A"<"AB"`). c64sh compares the bytes of the strings. For uppercase letters, digits, and punctuation the order is the same as the C64's character codes (PETSCII); for lowercase and non-ASCII characters it may differ.

**Comparing numbers** is exact. Because c64sh's numbers are 64-bit floating point rather than the C64's 5-byte format (HLD *Number representation*), an equality test on computed values can differ from a C64's: `.1+.2=.3` depends on the last bits of each result.

Every numeric result is limited to the C64's range: `OVERFLOW` if its size exceeds `maxNumber`, and 0 if it is nonzero and its size is below `minNumber`. A negative zero prints as `0`.

### String length

On a C64, a string produced at run time can hold at most 255 characters. If joining strings with `+` would produce a longer string, evaluation fails with a `STRING TOO LONG` error; the `PRINT` containing it writes only the output of items before the failing one (see *PRINT*). Length is counted in characters (Unicode code points), not bytes; each byte that is not valid UTF-8 counts as one character. A string literal printed without being joined is not limited, as on a C64, where literals are printed directly from the program text.

## Errors

The interpreter returns BASIC errors as the error type defined in the shell design. The kinds it can produce today:

| Kind | Cause |
|---|---|
| `STRING TOO LONG` | Joining strings with `+` produces more than 255 characters, or a string longer than 255 characters is assigned to a variable. |
| `TYPE MISMATCH` | Comparing a string with a number; a string operand of `AND`, `OR`, or `NOT`; assigning a string to a number variable or a number to a string variable; `+` with a string on one side and a number on the other; `-`, `*`, `/`, `^`, or negation with any string operand. |
| `OVERFLOW` | A number literal or arithmetic result larger in size than `maxNumber`. |
| `DIVISION BY ZERO` | `/` with a right operand of 0. |
| `ILLEGAL QUANTITY` | An operand of `AND`, `OR`, or `NOT` whose size is 32768 or more (other than -32768); `^` with a negative left operand and a right operand that is not a whole number; a value whose size is 32768 or more (other than -32768) assigned to an integer variable. |
| `SYNTAX` | A `BadItem` reached while executing `PRINT`, or a `BadStmt` reached; the error is the one the parser stored in it. |
| `NEXT WITHOUT FOR` | `NEXT` with no matching `FOR` entry above the topmost non-`FOR` entry of the control stack. |
| `OUT OF MEMORY` | A `FOR` or `GOSUB` with no room left on the C64's stack (see *Control stack*). |
| `ILLEGAL DIRECT` | `INPUT` or `GET` in direct mode. |
| `RETURN WITHOUT GOSUB` | `RETURN` with no `GOSUB` entry on the control stack, other than `FOR` entries above it. |
| `UNDEF'D STATEMENT` | `RUN n` or `GOTO n` where the program has no line `n`. |
| `BREAK` | `Interrupt` was called (see *BREAK*). |

An error that occurs while the program is running carries the number of the line it occurred in; for `RUN n` and `GOTO n` that is the line holding the statement, and in direct mode there is none.

If writing to the output fails (for example, stdout is a closed pipe), `Exec` returns that write error unchanged. It is not a BASIC error.

## Decisions & Alternatives

| Decision | Chosen | Alternatives Considered | Rationale |
|---|---|---|---|
| Number type | `float64` with C64 range checks and C64 output format | Emulating the C64's 5-byte float | See HLD *Number representation*. The range constants make overflow and underflow match the C64's limits, and formatting reproduces its output. |
| Integer variable conversion | Round down (toward minus infinity) | Truncate toward zero; round to nearest | The C64 ROM converts by shifting the two's-complement mantissa, which rounds down, so `-3.7` becomes -4. |
| Rounding for output | Round to 9 significant digits in decimal, then choose notation | Scale by 10 in binary as the ROM does | Decimal rounding gives the same 9 digits for all but rare boundary cases, and is simple and exact with `strconv`. |
| Dispatch | Type switch with panicking `default` | Visitor pattern | See HLD *Key Design Decisions*. One pass over the tree; no `Accept`/`Visit` boilerplate. |
| Comma | Spaces to the next 10-column print zone, a full zone when already at a zone start | Tab character; cursor-right control codes | Print zones are C64 language behavior, and programs lay out columns with them. The count `10 - (column % 10)`, never 0, is what the C64 ROM's PRINT computes (`$AAE8`). Spaces are the terminal equivalent of the C64's on-screen cursor-right moves, and the characters the C64 itself sends to files and printers. |
| Cursor column owner | The interpreter, which writes all program output | The shell's output wrapper, queried by the interpreter; separate counts in both, kept in step by the shell | One owner means one count that cannot drift, and the interpreter is where a C64 program's cursor lives. `FreshLine` lets the shell start errors and `READY.` on a new line without tracking output itself. The C64's `POS()` function will read the same column. |
| Trailing `;` or `,` | Suppresses the newline | Always end with a newline | C64 BASIC V2 behavior. Scripts rely on it to build one line of output from several statements. |
| 255-character limit | Enforced on concatenation results | No limit | Tenet *C64 language, Unix I/O*: string semantics are language behavior. Enforcing it now avoids a behavior change when string functions arrive. |
| Output writes | One write per `PRINT`; on failure, one write of the items before the failing one | One write per item; write nothing on failure | Keeps output of a single statement together and reduces system calls when output is unbuffered, while still showing exactly what a C64 would have printed before the error. |
| State | `Interp` value created once per session | Stateless function | Variables and the stored program need a home that persists across lines. |
| Execution model | A position (line, statement index) and a control stack of entries holding positions | Run line by line, with loops handled by re-running whole lines; a tree-walking loop construct built by the parser | A C64 resumes a loop just after its `FOR`, which can be mid-line or in the direct-mode line, and lets `NEXT` and `FOR` be anywhere, unmatched in the text. Only positions reproduce that; a parsed loop construct would reject valid programs such as one `FOR` with two `NEXT`s. |
| Stack limit | The ROM's byte budget: 18 bytes per `FOR` and 7 per `GOSUB`, `OUT OF MEMORY` from 169 bytes in use for a `FOR` and 179 for a `GOSUB` | No limit; a fixed count of loops | It is the C64's own rule, derived from the stack check at `$A3FB`, and it stops runaway programs from growing memory without end. |
| Keyboard source | A `Console` interface set by the shell, with line and key reads | Reading an `io.Reader` directly | The interpreter stays free of terminals: where keys come from (a terminal, a pipe, the rest of a script), how they are echoed, and how Ctrl-C reaches a waiting read are the shell's concerns, and tests supply input as a list of lines and keys. |
| Interrupting | A flag set by `Interrupt` and checked after each statement | Cancel through a `context.Context` passed to `Exec`; the shell kills the run some other way | The C64 checks its STOP key between statements, so checking there gives the same `BREAK IN n`. A flag keeps the interpreter free of signals and goroutines; the shell decides where interrupts come from. |
| Stored line form | Text and parsed tree, parsed once when stored | Text only, parsed each time the line runs, as the C64 does; tree only | The tree runs a line many times without parsing again; syntax errors are in the tree, so they are still reported only when reached. `LIST` needs the text as typed, which the tree does not keep (spaces, `?`). |
| Ending execution | `END`, `LIST`, `NEW`, `RUN`, and `GOTO` end the line or program through internal sentinel values returned like errors | Flags on the `Interp` checked after every statement | The same path already ends a line for a false `IF`; returning a value keeps the control flow visible in each statement's code. |
| `LIST` layout | A newline before each line, as the ROM writes it, and one after the last | Only a newline after each line; leave the last line unfinished, for `FreshLine` to end | The screen matches a C64's exactly, including the blank line after `LIST`. Ending the last line stands in for the newline that begins the C64's `READY.`, which always follows a listing; without it, a script's listing would run into its next output. |

## Open Questions & Future Decisions

### Deferred
1. Powers are computed with Go's `math.Pow`. The C64 computes them as `EXP(y*LOG(x))`, whose rounding makes some results differ in the last digit (see HLD *Number representation*); reproducing that is left to an exact float emulation, if one is ever built.

## References

- `docs/intent/parser/parser-design.md` (AST node types)
- `docs/intent/shell/shell-design.md` (error type and display)
- *Commodore 64 Programmer's Reference Guide*, `PRINT` and `STRING TOO LONG`
