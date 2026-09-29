---
parent: high-level-design
prefix: INTERP
---

# Interpreter

## Context and Design Philosophy

The interpreter executes an `*ast.Line` by walking it with type switches. It writes program output to an `io.Writer` it is given and knows nothing about terminals, files, prompts, or how errors are displayed. Those belong to the shell.

Its output follows C64 BASIC V2 semantics, adjusted for a terminal as the HLD's *C64 language, Unix I/O* tenet requires: the C64's 10-column print zones become a tab character.

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
```

An `Interp` carries no state between lines yet. When variables and program mode are added, their state lives in `Interp`, which is why it is a value created once per shell session rather than a free function.

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
| `ExprItem` | The expression's string value. |
| `Semicolon` | Nothing. It only separates items. |
| `Comma` | A tab character (`\t`). |
| `BadItem` | Nothing; execution of the `PRINT` fails with the item's error (see below). The parser produces a `BadItem` where a syntax error occurred inside a `PRINT`. |

After the last item, a newline (`\n`) is written **unless the last item is `;` or `,`**. This is the C64 rule, and it is how a BASIC program prints several things on one line:

| Input | Output |
|---|---|
| `PRINT` | `\n` |
| `PRINT "A"` | `A\n` |
| `PRINT "A";"B"` | `AB\n` |
| `PRINT "A""B"` | `AB\n` |
| `PRINT "A","B"` | `A\tB\n` |
| `PRINT "A";` | `A` |
| `PRINT "A",` | `A\t` |
| `PRINT ,"A"` | `\tA\n` |

The output for one `PRINT` is collected and written in a single call to the writer. If an item fails, either because evaluating its expression fails or because it is a `BadItem`, the output of the items before it is written (a C64 prints each item as it is evaluated), and then the error is returned. For `PRINT "A";X` where `X` fails, `A` is written, with no newline.

## REM

Executing a `RemStmt` does nothing: it writes no output and returns no error, so execution continues with the next statement. A `RemStmt` is always the last statement of a line, because its comment runs to the end of the line.

## Expressions

| Node | Value |
|---|---|
| `StringLit` | Its `Value`. |
| `Concat` | Left value followed by right value. |

### String length

On a C64, a string produced at run time can hold at most 255 characters. If a `Concat` would produce a longer string, evaluation fails with a `STRING TOO LONG` error; the `PRINT` containing it writes only the output of items before the failing one (see *PRINT*). Length is counted in characters (Unicode code points), not bytes; each byte that is not valid UTF-8 counts as one character. A string literal printed without concatenation is not limited, as on a C64, where literals are printed directly from the program text.

## Errors

The interpreter returns BASIC errors as the error type defined in the shell design. The kinds it can produce today:

| Kind | Cause |
|---|---|
| `STRING TOO LONG` | A concatenation result longer than 255 characters. |
| `SYNTAX` | A `BadItem` reached while executing `PRINT`; the error is the one the parser stored in it. |

If writing to the output fails (for example, stdout is a closed pipe), `Exec` returns that write error unchanged. It is not a BASIC error.

## Decisions & Alternatives

| Decision | Chosen | Alternatives Considered | Rationale |
|---|---|---|---|
| Dispatch | Type switch with panicking `default` | Visitor pattern | See HLD *Key Design Decisions*. One pass over the tree; no `Accept`/`Visit` boilerplate. |
| Comma | Tab character | C64 10-column print zones (pad with spaces to the next multiple of 10) | Tenet *C64 language, Unix I/O*. A tab is the terminal-native column separator and survives piping into tools like `cut` and `column`. |
| Trailing `;` or `,` | Suppresses the newline | Always end with a newline | C64 BASIC V2 behavior. Scripts rely on it to build one line of output from several statements. |
| 255-character limit | Enforced on concatenation results | No limit | Tenet *C64 language, Unix I/O*: string semantics are language behavior. Enforcing it now avoids a behavior change when string functions arrive. |
| Output writes | One write per `PRINT`; on failure, one write of the items before the failing one | One write per item; write nothing on failure | Keeps output of a single statement together and reduces system calls when output is unbuffered, while still showing exactly what a C64 would have printed before the error. |
| State | `Interp` value created once per session | Stateless function | Variables and the stored program will need a home that persists across lines. |

## Open Questions & Future Decisions

### Deferred
1. When numbers are added, `PRINT` of a number follows C64 formatting (a leading space or minus sign, and a trailing space), and `ExprItem` evaluation returns a typed value instead of a string.

## References

- `docs/intent/parser/parser-design.md` (AST node types)
- `docs/intent/shell/shell-design.md` (error type and display)
- *Commodore 64 Programmer's Reference Guide*, `PRINT` and `STRING TOO LONG`
