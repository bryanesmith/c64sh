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

// SetClock sets the clock that RND(0), TI, and TI$ read, and starts TI
// at 0 by it. Without one, it is the system clock, with TI starting at 0
// when the Interp is created.
func (in *Interp) SetClock(now func() time.Time)

// SetScreen sets whether program output goes to a terminal, which gets
// the screen control codes as escape codes, and whether colors are
// shown there. Without it, output is not a terminal.
func (in *Interp) SetScreen(terminal, color bool)

// ScreenState returns the escape codes that set the color and reverse
// video program output has left on in the terminal, or "" if none, so
// that a caller writing its own styled text to the terminal can put
// them back after it.
func (in *Interp) ScreenState() string

// SetConsole sets where INPUT and GET read. Without one, they find the
// end of input.
func (in *Interp) SetConsole(c Console)

// Storage holds the files that LOAD, SAVE, and VERIFY use, by name.
type Storage interface {
    // ReadFile returns a file's contents, or an error satisfying
    // errors.Is(err, fs.ErrNotExist) if there is no such file.
    ReadFile(name string) ([]byte, error)
    // WriteFile writes a file. If the file exists and replace is false,
    // it writes nothing and returns an error satisfying
    // errors.Is(err, fs.ErrExist).
    WriteFile(name string, data []byte, replace bool) error
}

// SetStorage sets where LOAD, SAVE, and VERIFY find files. Without one,
// they fail with DEVICE NOT PRESENT.
func (in *Interp) SetStorage(s Storage)

// Environment holds the environment variables ENVIRON$ reads and ENVIRON
// sets (a c64sh extension).
type Environment interface {
    Lookup(name string) (value string, ok bool)
    Set(name, value string) error
    Unset(name string) error
    List() []string // every variable as "NAME=VALUE", in any order
}

// MapEnvironment is an Environment held in a map, apart from the process's
// own environment.
type MapEnvironment map[string]string

// SetEnvironment sets the environment ENVIRON$ and ENVIRON use. Without
// one, they use an empty MapEnvironment.
func (in *Interp) SetEnvironment(e Environment)

// SetMessages sets where the C64's tape and disk messages (SAVING NAME,
// LOADING, …) are written, for statements executed in direct mode.
// Without a writer, no messages are written.
func (in *Interp) SetMessages(w io.Writer)

// StorageError is returned by Exec when storage refused or failed to read
// or write a file. It is not a BASIC error: a C64 reports these only
// through the disk drive's light and error channel.
type StorageError struct {
    File    string // the file's name in storage, such as HELLO.bas
    Err     error  // fs.ErrExist for a disk file that may not be replaced
    Replace string // for fs.ErrExist: what to write to replace the file, such as SAVE "@0:HELLO"
}

// CloseFiles closes every open data file, writing those opened for
// output to storage, and returns the first StorageError, if any.
func (in *Interp) CloseFiles() error

// Column returns the cursor column: the number of characters written
// since the last newline. It is 0 at the start of a line.
func (in *Interp) Column() int

// FreshLine ends the current output line if it is unfinished: when the
// column is not 0, it writes a newline. It returns any write error.
func (in *Interp) FreshLine() error
```

An `Interp` carries state between lines: the cursor column, which starts at 0, the variables and function definitions, which start empty, the stored program, which starts empty, and the control stack (see *Control stack*), which starts empty. It is created once per shell session, so all of them persist from line to line for the whole session or script.

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
        *ast.InputStmt, *ast.GetStmt, *ast.DefStmt,
        *ast.LoadStmt, *ast.SaveStmt, *ast.VerifyStmt,
        *ast.OpenStmt, *ast.CloseStmt, *ast.CmdStmt, *ast.OnStmt, *ast.DimStmt,
        *ast.DataStmt, *ast.ReadStmt, *ast.RestoreStmt, *ast.EnvironStmt:
        … // see Program mode
    default:
        panic(fmt.Sprintf("interp: unhandled statement %T", s))
    }
}
```

A panic here means a node type was added to `internal/ast` without interpreter support, which is a programming error, not a user error. Unit tests cover every node type so such a gap is caught before release.

## Screen control codes

On a C64, printing certain characters controls the screen instead of showing a character. The interpreter translates them in all program output (everything it writes: `PRINT`, `LIST`, `INPUT` prompts and echoes, `CMD` to the screen or printer), as `SetScreen` says:

| Code | On a C64 | Terminal | Not a terminal | Column |
|---|---|---|---|---|
| 13, 141, and the newline ending a `PRINT` | Return: next line, reverse off | `\n`, after `ESC[27m` if reverse is on | `\n` | 0 |
| 29 | Cursor right | a space | a space | +1 |
| 157 | Cursor left | `ESC[D` | left out | -1 (not below 0) on a terminal |
| 17, 145 | Cursor down, up | `ESC[B`, `ESC[A` | left out | unchanged |
| 19 | Home | `ESC[H` | left out | 0 on a terminal |
| 147 | Clear the screen | `ESC[2J ESC[H` | left out | 0 on a terminal |
| 18, 146 | Reverse on, off | `ESC[7m`, `ESC[27m` | left out | unchanged |
| 144, 5, 28, 159, 156, 30, 31, 158, 129, 149, 150, 151, 152, 153, 154, 155 | The 16 colors, black to light gray | the color, as 24-bit `ESC[38;2;R;G;Bm` | left out | unchanged |

Every other character is written as it is. The colors are the C64's own (the measured "Pepto" palette): black, white, red, cyan, purple, green, blue, yellow, orange, brown, light red, dark gray, gray, light green, light blue, light gray, in the order of the codes above. When colors are off, the color codes are left out even on a terminal; the other codes are still translated. As on a C64, a color stays in effect until another is printed; the interpreter does not reset it (see the shell design).

The interpreter remembers whether reverse video is on (after `18`, until `146` or a Return), so that it writes `ESC[27m` at a Return only when it changes something; ordinary lines carry no escape codes. It also remembers the last color it wrote to the terminal; `ScreenState` returns that color's code, followed by `ESC[7m` if reverse is on, for the shell to restore after its own styled text.

The cursor-right code becomes a space everywhere, as a comma's move to a print zone does: in a terminal, output only appears after the cursor, so a space looks the same.

## PRINT

`PRINT` processes its items left to right:

| Item | Output |
|---|---|
| `ExprItem` | A string value as it is; a number value in C64 number format (see *Numbers*) followed by one space. |
| `Semicolon` | Nothing. It only separates items. |
| `Comma` | Spaces up to the start of the next print zone: `10 - (column % 10)` spaces, where `column` is the cursor column at that point. This is never 0: at the start of a zone (column 0, 10, 20, …), a comma moves a full 10 columns, as the C64 ROM does. |
| `TabItem` | Spaces up to column `X`: `X - column` spaces if the cursor is left of column `X`, otherwise nothing (`$AAF8`). |
| `SpcItem` | `X` spaces. |
| `BadItem` | Nothing; execution of the `PRINT` fails with the item's error (see below). The parser produces a `BadItem` where a syntax error occurred inside a `PRINT`. |

For `TabItem` and `SpcItem`, `X` is rounded down and must be from 0 to 255 (else `ILLEGAL QUANTITY`; a string is `TYPE MISMATCH`). Like a comma, `TAB` counts from the screen's cursor column, even when writing to a storage file.

After the last item, a newline (`\n`) is written **unless the last item is `;`, `,`, `TAB(`, or `SPC(`** (the ROM continues past `TAB(` and `SPC(` as it does past `;`, `$AB13`). This is the C64 rule, and it is how a BASIC program prints several things on one line:

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

### ON

`ON X GOTO …` and `ON X GOSUB …` evaluate `X` as a number from 0 to 255, rounded down (a string is `TYPE MISMATCH`; out of range is `ILLEGAL QUANTITY`), as the ROM reads it (`$A94B`). If `X` is between 1 and the number of line numbers in the list, the statement acts as `GOTO` or `GOSUB` with the `X`th line number; a `GOSUB` entry holds the position after the `ON` statement, so `RETURN` continues after the whole statement, as on a C64. If `X` is 0 or larger than the list, execution continues with the next statement.

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

## User-defined functions

Executing `DEF FN` follows the ROM (`$B3B3`): if the function's name is a string name, fail with `TYPE MISMATCH`; in direct mode, fail with `ILLEGAL DIRECT`; if the parameter is a string variable, fail with `TYPE MISMATCH`. Otherwise record the definition under the name's identity, replacing any earlier one. Function names are separate from variable names: `FN A` and the variable `A` are unrelated. Definitions are part of the variables, as on a C64, so whatever clears the variables (`RUN`, `NEW`, storing a line) clears them too.

Evaluating `FN NAME(ARG)` follows the ROM (`$B3F4`), in this order:

1. If the name is a string name, fail with `TYPE MISMATCH`.
2. Evaluate the argument; if it is a string, fail with `TYPE MISMATCH`.
3. If no function of that name is defined, fail with `UNDEF'D FUNCTION`.
4. If 9 calls are already in progress, fail with `OUT OF MEMORY` (see below).
5. Save the parameter variable's value and assign it the argument.
6. Evaluate the body: if the definition has a `BodyErr`, fail with it; if the value is a string, fail with `TYPE MISMATCH`.
7. Restore the parameter variable's saved value, and return the body's value.

The body sees every variable's current value, so `DEF FN F(X)=X*K` uses whatever `K` holds at the call, and it can call other functions. If an error stops the call, the parameter keeps the argument, as on a C64, which restores it only after the body has been evaluated.

**Calls in progress.** A call keeps its saved parameter and the evaluator's state on the C64's stack while its body is evaluated, about 20 bytes, and the expression evaluator fails with `OUT OF MEMORY` when the stack runs out (`$ADAC`). c64sh does not model the stack space expressions use, but it limits calls in progress to 9, roughly what the C64's stack holds, so that a function that calls itself (`DEF FN A(X)=FN A(X)`) fails with `OUT OF MEMORY` as on a C64, instead of recursing without end.

## Program files

`LOAD`, `SAVE`, and `VERIFY` move the whole program between memory and the `Storage` the shell sets (see the shell design).

### Arguments

The arguments are evaluated in order: the name must be a string (else `TYPE MISMATCH`), and the device and secondary address numbers from 0 to 255, rounded down (else `ILLEGAL QUANTITY`), as the ROM reads them (`$E1D4`, `$B79E`). The device defaults to 1 (tape); the secondary address is ignored. The device then decides what happens, as on a C64:

| Device | Meaning |
|---|---|
| 1 | Tape: storage |
| 8 to 11 | Disk drives: storage |
| 0, 3, 4, 5 | Keyboard, screen, printers: `ILLEGAL DEVICE NUMBER` |
| any other | `DEVICE NOT PRESENT` |

An empty or missing name is `MISSING FILE NAME`. (A C64 lets tape use an empty name, meaning the next file on the tape; storage has no "next file".)

### File names

For a disk drive, a leading `@0:` or `@:` means "replace the file if it exists", and a leading `0:` (drive 0) is dropped; the rest is the name. For tape, the name is used as given. The file in storage is the name with `.bas` added if the name has no extension (no `.` in it): `HELLO` is `HELLO.bas`, and `HELLO.TXT` is `HELLO.TXT`. Letters keep their case.

### SAVE

`SAVE` writes the program as text: the line `#!/usr/bin/env c64sh`, then each line, in order, as its number, a space, and its text exactly as stored, each ending in `\n`. Saving an empty program writes just the first line. On tape, an existing file is replaced. On a disk drive, it is replaced only with `@0:`; otherwise nothing is written and `Exec` returns a `StorageError` with `fs.ErrExist`, the 1541's `63, FILE EXISTS`, which a C64 shows only by blinking the drive's light, and `Replace` set to `SAVE "@0:NAME"`, `NAME` being the name without any `0:`. Any other storage failure is also a `StorageError`.

### LOAD

`LOAD` reads the file: the name as given first, and if there is no such file and the name has no extension, the name with `.bas`. If neither exists, it fails with `FILE NOT FOUND`. The file must be a program: an optional first line beginning `#!`, blank lines, and lines beginning with a line number, each stored as if typed (so a later line replaces an earlier one with the same number). Anything else, or a line number above 63999, is `LOAD` (the C64's `?LOAD  ERROR`), and the program is unchanged. Then, as on a C64:

- **In direct mode**, the program is replaced and the variables are cleared, and the rest of the line does not run.
- **In a running program**, the program is replaced, the variables are kept, the control stack is emptied, and the new program runs from its first line (`$E1AB`): this is how C64 programs chain to the next part of a program too large for memory.

### VERIFY

`VERIFY` reads the file as `LOAD` does and compares its lines (numbers and text) with the program's. If they differ, or the file is not a program, it fails with `VERIFY`; if they match, it continues.

### Messages

When a writer is set with `SetMessages` and the statement runs in direct mode, the C64's messages are written to it, each on its own line, after `FreshLine`; a running program writes none, as on a C64, which turns them off for `RUN` (`$A871`). `NAME` is the name as given.

| Statement | Tape | Disk |
|---|---|---|
| `SAVE` | `PRESS RECORD & PLAY ON TAPE`, `OK`, `SAVING NAME` | `SAVING NAME` |
| `LOAD` | `PRESS PLAY ON TAPE`, `OK`, `SEARCHING FOR NAME`, `FOUND NAME`, `LOADING` | `SEARCHING FOR NAME`, `LOADING` |
| `VERIFY` | as `LOAD`, with `VERIFYING` for `LOADING`, then `OK` if it matches | `SEARCHING FOR NAME`, `VERIFYING`, then `OK` if it matches |

When the file is not found, the messages stop after `SEARCHING FOR NAME`. A tape never needs a key pressed: c64sh writes `PRESS PLAY ON TAPE` and carries on.

## Data files

`OPEN`, `CLOSE`, `PRINT#`, `INPUT#`, `GET#`, and `CMD` work with **logical files**, numbered 1 to 255, which the interpreter keeps in a table, as the C64's Kernal does.

### OPEN

`OPEN F, D, S, "NAME"` evaluates the file number, device (1 if omitted), and secondary address (0 if omitted) as numbers from 0 to 255, and the name as a string (empty if omitted), then follows the Kernal (`$F34A`):

1. If `F` is 0, fail with `NOT INPUT FILE` (the Kernal's error for it). If `F` is open, fail with `FILE OPEN`. If 10 files are open, fail with `TOO MANY FILES`.
2. Open by device:

| Device | File |
|---|---|
| 0, the keyboard | Input only, read from the console as `INPUT` and `GET` read it. |
| 3, the screen; 4 and 5, printers | Output only, written to the program output (stdout), as `PRINT` writes. |
| 1, tape | A file in storage: read when the secondary address is 0, written (replacing any file) when it is 1 or 2. |
| 8 to 11, disk drives | A file in storage, by the secondary address: 0 reads, 1 writes; 2 to 14 take the mode from the name, `NAME,S,W` (write) or `NAME,P,W`, `NAME,S,A` (append), or `NAME,S,R` and plain `NAME` (read); 15, the drive's command channel, is `DEVICE NOT PRESENT` (not supported yet). |
| any other | `DEVICE NOT PRESENT`, as is any storage device when no storage is set. |

For a storage file, the name is required (`MISSING FILE NAME`), and for a disk drive a leading `@0:`, `@:`, or `0:` means what it means for `SAVE`. The file in storage is the name before the first comma, as given: data files get no `.bas`. A file opened to read, or to append, must exist (`FILE NOT FOUND`; a C64's tape reports this, and c64sh reports it for disk files too, rather than through the drive's error channel). A disk file opened to write that exists, without `@0:`, is a `StorageError` with `fs.ErrExist` and `Replace` set to `"@0:NAME,S,W"`. A file opened to read is read whole when opened; one opened to write or append collects its output, which is written to storage when the file is closed.

### CLOSE

`CLOSE F` closes file `F`, writing an output file to storage (a `StorageError` if that fails); closing a file that is not open does nothing, as on a C64. If `CMD` was sending output to `F`, output returns to the screen. Clearing the variables (`RUN`, `NEW`, storing a line, `LOAD` in direct mode) closes every file the same way, as the C64's `CLR` closes them (`$A660`), without reporting a storage failure, since nothing is there to report it to; `CloseFiles`, which the shell calls when it ends, closes them too and returns the first failure.

### PRINT# and CMD

`PRINT# F, items` writes the items to file `F` exactly as `PRINT` writes them to the screen, ending with a newline unless the items end with `;` or `,` (a lone `PRINT#F` writes just a newline). The file must be open (`FILE NOT OPEN`) and not input-only (`NOT OUTPUT FILE`). Line ends are written as `\n`, the Unix convention, where a C64 writes a carriage return.

A comma in `PRINT#` moves to the next print zone **of the screen's cursor column**, as on a C64, whose `PRINT` asks the screen for the cursor position even when writing to a file (`$AAE8`): since file output does not move the screen's cursor, a comma usually writes 10 spaces.

`CMD F, items` checks the file as `PRINT#` does, writes the items in the same way, and then leaves the file as the destination of all output that would go to the screen through `PRINT` and `LIST`, until `PRINT#`, `INPUT#`, or `GET#` runs (each returns output to the screen when it finishes, as the ROM does, `$ABB5`), the file is closed, or an error stops execution. `CMD F` with no items writes a newline, as on a C64. Output to the screen, printer, or `CMD` file through the screen keeps the cursor column; output to a storage file does not change it.

### INPUT# and GET#

`INPUT# F, vars` reads values from file `F` as `INPUT` reads them, with these differences, as in the ROM (`$ABA5`): there is no prompt and nothing is echoed; empty lines before the first value are skipped; when a line runs out and variables remain, the next line supplies them without `?? `; a number that cannot be read is `FILE DATA` instead of `?REDO FROM START`; values left over are ignored without `?EXTRA IGNORED`. If the file ends before a value is read, the statement ends, leaving the remaining variables unchanged. A line ends at `\n`, `\r\n`, or `\r`. `INPUT#` works in direct mode.

`GET# F, vars` reads one character per variable as `GET` reads a key: the empty string when the file has none left; a line end is `CHR$(13)`. Like `GET`, it is `ILLEGAL DIRECT` in direct mode.

For both, the file must be open (`FILE NOT OPEN`) and not output-only (`NOT INPUT FILE`). From the keyboard (device 0), they read the console, as `INPUT` (without its prompt) and `GET` do.

### ST

`ST` is the status of the last `OPEN`, `PRINT#`, `CMD`, `INPUT#`, or `GET#`: 64 after a read that reached the end of a file (its last character was read, or there was nothing left to read), and 0 otherwise. It starts at 0.

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

## Number functions

A `CallExpr` evaluates its argument, which must be a number (else `TYPE MISMATCH`), and returns, as the C64 ROM computes them:

| Function | Value |
|---|---|
| `ABS(X)` | The size of `X`. |
| `INT(X)` | `X` rounded down to a whole number (`INT(-2.5)` is -3), `$BCCC`. |
| `SGN(X)` | -1, 0, or 1 as `X` is negative, zero, or positive. |
| `SQR(X)` | The square root; `ILLEGAL QUANTITY` for a negative `X`. |
| `LOG(X)` | The natural logarithm; `ILLEGAL QUANTITY` for an `X` of 0 or less (`$B9EA`). |
| `EXP(X)` | e to the power `X`; `OVERFLOW` if larger than the C64's largest number. |
| `SIN(X)`, `COS(X)`, `TAN(X)`, `ATN(X)` | In radians; `TAN` of an angle whose cosine is exactly 0 is `DIVISION BY ZERO`, as the ROM divides the sine by the cosine (`$E2B4`). |
| `RND(X)` | A pseudo-random number at least 0 and less than 1 (see below). |
| `POS(X)` | The cursor column (see *Cursor Column*), counting the output of the `PRINT` items before it on the screen, which a C64 has already printed; `X`, of either type, is ignored, as the ROM ignores it (`$B39E`). |

Results are limited to the C64's range like any other number (see *Values*).

**`RND`** keeps a seed, as the C64 does (`$E097`). A negative `X` replaces the seed with one computed from `X`, so `RND(-7)` always gives the same number and starts the same sequence. A positive `X` (whatever its size) advances the seed and gives the next number. An `X` of 0 replaces the seed with one computed from the clock (see `SetClock`), the counterpart of the C64 reading its timers, and gives a number from it. The seed starts at the same value in every interpreter, so a program that never reseeds gets the same numbers every run, as a C64 does after power-on. The generator is c64sh's own (a 64-bit mixing function), so its numbers differ from a C64's; programs that relied on a C64's exact sequence will see different values.

## String functions

String functions evaluate their arguments in order; each must have its type (a string or a number, below, else `TYPE MISMATCH`), and each number marked *byte* is rounded down and must be from 0 to 255 (else `ILLEGAL QUANTITY`), as the ROM reads them (`$B79E`). Characters are Unicode characters, counted as for the 255-character limit.

| Function | Value |
|---|---|
| `LEN(S$)` | The number of characters in `S$`. |
| `LEFT$(S$, N)` | The first `N` (byte) characters, or all of `S$` if it is shorter. |
| `RIGHT$(S$, N)` | The last `N` (byte) characters, or all of `S$`. |
| `MID$(S$, P [, N])` | `N` (byte; all if omitted) characters from position `P` (byte, counting from 1); `P` of 0 is `ILLEGAL QUANTITY`; past the end, the empty string. |
| `CHR$(N)` | The character with code `N` (byte): the Unicode character `N`, which for 32 to 126 is the same as the C64's. |
| `ASC(S$)` | The code of the first character (its Unicode code point); `ILLEGAL QUANTITY` for the empty string (`$B78B`). |
| `STR$(X)` | `X` formatted as `PRINT` formats it, without the space after: `STR$(5)` is `" 5"`. |
| `VAL(S$)` | The number `S$` starts with, read as `INPUT` reads a number (spaces skipped, stopping at the first character that cannot continue it); 0 if there is none; `OVERFLOW` if too large (`$B7AD`). |

## The environment

*c64sh extension*, with the names GW-BASIC gives these words. The interpreter reads and changes environment variables through its `Environment`: the shell gives it the process's own environment, so programs c64sh starts later see the changes; tests and other callers give it a `MapEnvironment` or nothing, so they never depend on, or change, the real one.

| Form | Effect |
|---|---|
| `ENVIRON$(S$)` | The value of the variable named `S$`, or the empty string if it is not set. |
| `ENVIRON$(N)` | The `N`th (byte, from 1) variable as `NAME=VALUE`, counting in order of name, or the empty string past the last; `N` of 0 is `ILLEGAL QUANTITY`. A loop up to the first empty string lists them all. |
| `ENVIRON S$ [; S$ …]` | Joins the strings into one text and splits it at its first `=` into a name and a value: sets the variable, or removes it if the value is empty. A number among them is `TYPE MISMATCH`; no `=`, or nothing before it, is `ILLEGAL QUANTITY`, as is a name or value the environment refuses. |

Environment values are often longer than a BASIC string's 255 characters (`PATH` is). `ENVIRON$` returns the whole value, so `PRINT ENVIRON$("PATH")` shows it, and the string functions read it; storing it in a variable, or joining it with `+`, is `STRING TOO LONG` as for any other string, so a value is never silently cut short. `ENVIRON`'s parts are joined without that limit, so `ENVIRON "PATH=";ENVIRON$("PATH");":/opt/bin"` extends a long `PATH`.

## The clock

`TI` counts **jiffies**, sixtieths of a second, as the C64's clock does: it starts at 0 when the `Interp` is created (or a clock is set with `SetClock`), the counterpart of a C64's power-on, and counts the clock's time since, rounded down to whole jiffies. `TI$` is the same clock as six digits, hours, minutes, and seconds: `TI` of 216000 is `"010000"`. Both wrap to 0 after 24 hours (5184000 jiffies).

Assigning to `TI$` sets the clock, as the ROM does (`$A9DA`): the value must be six characters, all digits (else `ILLEGAL QUANTITY`), read as `HHMMSS`, and the clock counts on from that many jiffies (hours, minutes, and seconds are not range-checked, as in the ROM, beyond wrapping at 24 hours). `TI` cannot be assigned (see the parser design).

## Variables

Variables are kept in a map from a `VarRef`'s `Name` (its identity, such as `SC` or `N$`) to a value.

- **Assignment** (`LetStmt`) evaluates the value, checks its type against the variable's, and stores it. A string variable (a name ending in `$`) takes only strings, and number and integer variables only numbers; the wrong kind is `TYPE MISMATCH`.
- **An integer variable** (a name ending in `%`) stores the value rounded down to a whole number, as the C64 ROM's conversion does (`$BC9B`): `C%=3.7` stores 3 and `C%=-3.7` stores -4. A value whose size is 32768 or more, other than exactly -32768, is `ILLEGAL QUANTITY` (`$B1BF`); the range is checked before rounding, so -32768.5 is out of range while -32767.5 stores -32768. Integer variables are otherwise ordinary numbers in expressions. A string longer than 255 characters is `STRING TOO LONG`, the limit on any string a C64 stores. On any error the variable keeps its old value.
- **A reference** (`VarRef`) evaluates to the stored value, or, for a variable never assigned, to 0 or the empty string, as on a C64.

## Arrays

Arrays are kept apart from plain variables, by identity: `A`, `A(…)`, `A%(…)`, and `A$(…)` are four separate things. Each array has a number of dimensions and a top subscript for each, and its elements start as 0 or the empty string.

- **`DIM`** creates each array it names with the given tops, evaluated in order, each rounded down to a whole number from -32768 to 32767 (`ILLEGAL QUANTITY` otherwise, and for a negative top). If the array exists, it fails with `REDIM'D ARRAY`. A name without subscripts does nothing.
- **An element** evaluates its subscripts in order, each rounded down like a top (`ILLEGAL QUANTITY` for a negative or out-of-range subscript). If the array does not exist, it is created with as many dimensions as there are subscripts, each with top 10, as the ROM does (`$B1D1`). If the number of subscripts differs from the array's dimensions, or a subscript exceeds its top, it fails with `BAD SUBSCRIPT`. Reading gives the element's value; assigning follows the rules of a plain variable of the same type (an integer element rounds down, and so on).
- **Memory.** An array takes, as on a C64, 5 bytes plus 2 per dimension, plus 5 bytes per element for numbers, 3 for strings, and 2 for integers. Creating an array (with `DIM` or by use) that would make all arrays together take more than 38911 bytes, the memory free on a C64 when it starts, fails with `OUT OF MEMORY`. The program, the variables, and the strings themselves are not counted, so a C64 runs out of memory sooner.
- Arrays are cleared with the variables (`RUN`, `NEW`, storing a line).

## DATA and READ

The interpreter keeps a **data pointer**: a place among the items of the program's `DATA` statements, in program order (line by line, statement by statement). It starts at the first item, and `RESTORE`, clearing the variables (`RUN`, `NEW`, storing a line), and a `LOAD` that chains move it back there (`$A81D`, `$A677`). Executing a `DATA` statement does nothing; `DATA` in the line typed in direct mode is never read.

`READ` assigns each of its variables the next item, as the ROM does (`$AC06`):

- Items are separated by commas. Each is read as `INPUT` reads a value: for a string variable, a quoted string, or the text up to the next comma, with spaces before it skipped and spaces inside or after it kept; for a number variable, a number written as in a program, or 0 for an empty item. After the item, any spaces, the next character must be a comma or the end of the `DATA` text; a comma at the end of the text leaves one more, empty, item.
- When a `DATA` statement's items are used up, reading continues with the next `DATA` statement, on the same line or a later one. If there is none, `READ` fails with `OUT OF DATA`.
- An item that cannot be read (a number variable given text that is not a number, or anything after a quoted string other than a comma) is `SYNTAX`, reported with the number of the **`DATA` line**, as the ROM reports it (`$AB57`); a number too large is `OVERFLOW`.
- `READ` works in direct mode too, reading the program's `DATA`.

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
| `OUT OF MEMORY` | A `FOR` or `GOSUB` with no room left on the C64's stack (see *Control stack*), a 10th function call in progress, or arrays taking more than 38911 bytes. |
| `ILLEGAL DIRECT` | `INPUT`, `GET`, or `DEF` in direct mode. |
| `FILE NOT FOUND` | `LOAD`, `VERIFY`, or an `OPEN` to read or append, with no such file. |
| `FILE OPEN` | `OPEN` of a file number already open. |
| `FILE NOT OPEN` | `PRINT#`, `CMD`, `INPUT#`, or `GET#` with a file number not open. |
| `NOT INPUT FILE` | `INPUT#` or `GET#` with an output-only file; `OPEN` with file number 0. |
| `NOT OUTPUT FILE` | `PRINT#` or `CMD` with an input-only file. |
| `TOO MANY FILES` | `OPEN` with 10 files already open. |
| `FILE DATA` | `INPUT#` reading something other than a number into a number variable. |
| `DEVICE NOT PRESENT` | `LOAD`, `SAVE`, `VERIFY`, or `OPEN` with a device that is not storage, a keyboard, a screen, or a printer, or with no storage set; `OPEN` of a disk drive's command channel. |
| `ILLEGAL DEVICE NUMBER` | `LOAD`, `SAVE`, or `VERIFY` with the keyboard (0), the screen (3), or a printer (4, 5). |
| `MISSING FILE NAME` | `LOAD`, `SAVE`, or `VERIFY` with an empty or missing name. |
| `LOAD` | `LOAD` of a file that is not a program. |
| `VERIFY` | `VERIFY` of a file that differs from the program. |
| `UNDEF'D FUNCTION` | `FN` calling a function that has not been defined. |
| `BAD SUBSCRIPT` | An element with the wrong number of subscripts, or a subscript past the top. |
| `REDIM'D ARRAY` | `DIM` of an array that exists, including one created by use. |
| `OUT OF DATA` | `READ` with no `DATA` items left. |
| `RETURN WITHOUT GOSUB` | `RETURN` with no `GOSUB` entry on the control stack, other than `FOR` entries above it. |
| `UNDEF'D STATEMENT` | `RUN n` or `GOTO n` where the program has no line `n`. |
| `BREAK` | `Interrupt` was called (see *BREAK*). |

An error that occurs while the program is running carries the number of the line it occurred in; for `RUN n` and `GOTO n` that is the line holding the statement, and in direct mode there is none.

If writing to the output fails (for example, stdout is a closed pipe), `Exec` returns that write error unchanged. It is not a BASIC error.

## Decisions & Alternatives

| Decision | Chosen | Alternatives Considered | Rationale |
|---|---|---|---|
| Environment access | Through an `Environment`, the process's own from the shell | Read and set the process environment directly | Tests and snapshots stay deterministic and never change the test process's environment. |
| Values over 255 characters | `ENVIRON$` returns all of it; storing or `+` is `STRING TOO LONG` | Truncate to 255; `STRING TOO LONG` from `ENVIRON$` itself | Truncating would silently corrupt a value written back (a cut-short `PATH`); an error from `ENVIRON$` would make long values impossible even to print. |
| Building long values | `ENVIRON` joins `;`-separated parts, without the string limit | GW-BASIC's single string only; raise the string limit | Keeps GW-BASIC's form working unchanged while making `PATH` editable; the 255-character limit stays a C64 rule everywhere else. |
| `ENVIRON$(N)` order | By name | The process's order, as GW-BASIC lists its table | The process's order is arbitrary; by name, a listing is stable and easy to read. |
| Number type | `float64` with C64 range checks and C64 output format | Emulating the C64's 5-byte float | See HLD *Number representation*. The range constants make overflow and underflow match the C64's limits, and formatting reproduces its output. |
| Integer variable conversion | Round down (toward minus infinity) | Truncate toward zero; round to nearest | The C64 ROM converts by shifting the two's-complement mantissa, which rounds down, so `-3.7` becomes -4. |
| Rounding for output | Round to 9 significant digits in decimal, then choose notation | Scale by 10 in binary as the ROM does | Decimal rounding gives the same 9 digits for all but rare boundary cases, and is simple and exact with `strconv`. |
| Dispatch | Type switch with panicking `default` | Visitor pattern | See HLD *Key Design Decisions*. One pass over the tree; no `Accept`/`Visit` boilerplate. |
| Comma | Spaces to the next 10-column print zone, a full zone when already at a zone start | Tab character; cursor-right control codes | Print zones are C64 language behavior, and programs lay out columns with them. The count `10 - (column % 10)`, never 0, is what the C64 ROM's PRINT computes (`$AAE8`). Spaces are the terminal equivalent of the C64's on-screen cursor-right moves, and the characters the C64 itself sends to files and printers. |
| Screen codes | Translated in the interpreter's single writer, which also keeps the column | Translate in the shell's output stream | The column must follow the codes' meaning (clear and home return it to 0, colors take no column), and only the interpreter keeps the column. |
| Cursor column owner | The interpreter, which writes all program output | The shell's output wrapper, queried by the interpreter; separate counts in both, kept in step by the shell | One owner means one count that cannot drift, and the interpreter is where a C64 program's cursor lives. `FreshLine` lets the shell start errors and `READY.` on a new line without tracking output itself. The C64's `POS()` function will read the same column. |
| Trailing `;` or `,` | Suppresses the newline | Always end with a newline | C64 BASIC V2 behavior. Scripts rely on it to build one line of output from several statements. |
| 255-character limit | Enforced on concatenation results | No limit | Tenet *C64 language, Unix I/O*: string semantics are language behavior. Enforcing it now avoids a behavior change when string functions arrive. |
| Output writes | One write per `PRINT`; on failure, one write of the items before the failing one | One write per item; write nothing on failure | Keeps output of a single statement together and reduces system calls when output is unbuffered, while still showing exactly what a C64 would have printed before the error. |
| State | `Interp` value created once per session | Stateless function | Variables and the stored program need a home that persists across lines. |
| Execution model | A position (line, statement index) and a control stack of entries holding positions | Run line by line, with loops handled by re-running whole lines; a tree-walking loop construct built by the parser | A C64 resumes a loop just after its `FOR`, which can be mid-line or in the direct-mode line, and lets `NEXT` and `FOR` be anywhere, unmatched in the text. Only positions reproduce that; a parsed loop construct would reject valid programs such as one `FOR` with two `NEXT`s. |
| Stack limit | The ROM's byte budget: 18 bytes per `FOR` and 7 per `GOSUB`, `OUT OF MEMORY` from 169 bytes in use for a `FOR` and 179 for a `GOSUB` | No limit; a fixed count of loops | It is the C64's own rule, derived from the stack check at `$A3FB`, and it stops runaway programs from growing memory without end. |
| `RND` algorithm | c64sh's own generator, with the C64's rules for negative, zero, and positive arguments | Reproduce the C64's sequence | The ROM's generator works on the C64's 5-byte floating-point format; reproducing its sequence would need an exact emulation of that arithmetic (see HLD *Number representation*). The rules are what programs depend on: reseed with a negative number for a repeatable sequence, or with the clock for a fresh one. |
| Function call depth | A fixed limit of 9 calls in progress | Count the stack bytes of every expression; no limit | Modeling the evaluator's stack use precisely would touch every expression for little gain; with no limit, a recursive function would exhaust the Go stack. A fixed limit near the C64's reproduces its error for runaway recursion. |
| Data file buffering | Read a file whole at `OPEN`; collect output and write it at `CLOSE` | Stream through an open file handle | Whole-file reads and writes reuse the `Storage` interface of `LOAD` and `SAVE`, keep tests in memory, and suit C64-sized files. A C64 also completes a file only when it is closed. |
| Storage | A `Storage` interface set by the shell, with whole-file reads and writes | `os` calls in the interpreter; an `fs.FS` | Tests use storage in memory and never touch the filesystem (HLD *Non-Goals*). `fs.FS` cannot write, and the disk-overwrite rule needs a write that refuses to replace. |
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
