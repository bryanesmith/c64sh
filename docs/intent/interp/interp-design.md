---
parent: high-level-design
prefix: INTERP
---

# Interpreter

## Context and Design Philosophy

The interpreter executes an `*ast.Line` by walking it with type switches. It writes program output to an `io.Writer` it is given and knows nothing about terminals, files, prompts, or how errors are displayed. Those belong to the shell.

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

// Exec runs the statements of line in order. It stops at the first
// statement that fails and returns that error; statements after it do
// not run. Output from statements before the failure has already been
// written.
func (in *Interp) Exec(line *ast.Line) error

// Column returns the cursor column: the number of characters written
// since the last newline. It is 0 at the start of a line.
func (in *Interp) Column() int

// FreshLine ends the current output line if it is unfinished: when the
// column is not 0, it writes a newline. It returns any write error.
func (in *Interp) FreshLine() error
```

The cursor column is the state an `Interp` carries between lines; it starts at 0. When variables and program mode are added, their state also lives in `Interp`, which is why it is a value created once per shell session rather than a free function.

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

## REM

Executing a `RemStmt` does nothing: it writes no output and returns no error, so execution continues with the next statement. A `RemStmt` is always the last statement of a line, because its comment runs to the end of the line.

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
| `NegExpr` | A number: its negation. A string: `TYPE MISMATCH`. |

Every numeric result is limited to the C64's range: `OVERFLOW` if its size exceeds `maxNumber`, and 0 if it is nonzero and its size is below `minNumber`. A negative zero prints as `0`.

### String length

On a C64, a string produced at run time can hold at most 255 characters. If joining strings with `+` would produce a longer string, evaluation fails with a `STRING TOO LONG` error; the `PRINT` containing it writes only the output of items before the failing one (see *PRINT*). Length is counted in characters (Unicode code points), not bytes; each byte that is not valid UTF-8 counts as one character. A string literal printed without being joined is not limited, as on a C64, where literals are printed directly from the program text.

## Errors

The interpreter returns BASIC errors as the error type defined in the shell design. The kinds it can produce today:

| Kind | Cause |
|---|---|
| `STRING TOO LONG` | Joining strings with `+` produces more than 255 characters. |
| `TYPE MISMATCH` | `+` with a string on one side and a number on the other; `-`, `*`, `/`, or negation with any string operand. |
| `OVERFLOW` | A number literal or arithmetic result larger in size than `maxNumber`. |
| `DIVISION BY ZERO` | `/` with a right operand of 0. |
| `SYNTAX` | A `BadItem` reached while executing `PRINT`; the error is the one the parser stored in it. |

If writing to the output fails (for example, stdout is a closed pipe), `Exec` returns that write error unchanged. It is not a BASIC error.

## Decisions & Alternatives

| Decision | Chosen | Alternatives Considered | Rationale |
|---|---|---|---|
| Number type | `float64` with C64 range checks and C64 output format | Emulating the C64's 5-byte float | See HLD *Number representation*. The range constants make overflow and underflow match the C64's limits, and formatting reproduces its output. |
| Rounding for output | Round to 9 significant digits in decimal, then choose notation | Scale by 10 in binary as the ROM does | Decimal rounding gives the same 9 digits for all but rare boundary cases, and is simple and exact with `strconv`. |
| Dispatch | Type switch with panicking `default` | Visitor pattern | See HLD *Key Design Decisions*. One pass over the tree; no `Accept`/`Visit` boilerplate. |
| Comma | Spaces to the next 10-column print zone, a full zone when already at a zone start | Tab character; cursor-right control codes | Print zones are C64 language behavior, and programs lay out columns with them. The count `10 - (column % 10)`, never 0, is what the C64 ROM's PRINT computes (`$AAE8`). Spaces are the terminal equivalent of the C64's on-screen cursor-right moves, and the characters the C64 itself sends to files and printers. |
| Cursor column owner | The interpreter, which writes all program output | The shell's output wrapper, queried by the interpreter; separate counts in both, kept in step by the shell | One owner means one count that cannot drift, and the interpreter is where a C64 program's cursor lives. `FreshLine` lets the shell start errors and `READY.` on a new line without tracking output itself. The C64's `POS()` function will read the same column. |
| Trailing `;` or `,` | Suppresses the newline | Always end with a newline | C64 BASIC V2 behavior. Scripts rely on it to build one line of output from several statements. |
| 255-character limit | Enforced on concatenation results | No limit | Tenet *C64 language, Unix I/O*: string semantics are language behavior. Enforcing it now avoids a behavior change when string functions arrive. |
| Output writes | One write per `PRINT`; on failure, one write of the items before the failing one | One write per item; write nothing on failure | Keeps output of a single statement together and reduces system calls when output is unbuffered, while still showing exactly what a C64 would have printed before the error. |
| State | `Interp` value created once per session | Stateless function | Variables and the stored program will need a home that persists across lines. |

## Open Questions & Future Decisions

### Deferred
1. Exponentiation (`^`) adds an operator with `ILLEGAL QUANTITY` for a negative base with a fractional exponent, and reuses the range checks above.

## References

- `docs/intent/parser/parser-design.md` (AST node types)
- `docs/intent/shell/shell-design.md` (error type and display)
- *Commodore 64 Programmer's Reference Guide*, `PRINT` and `STRING TOO LONG`
