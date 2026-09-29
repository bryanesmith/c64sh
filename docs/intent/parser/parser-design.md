---
parent: high-level-design
prefix: PARSER
---

# Parser

## Context and Design Philosophy

The parser turns the lexer's tokens for one line into an abstract syntax tree (AST). It is a hand-written recursive-descent parser with one function per syntactic (uppercase) rule of `grammar/c64basic.ebnf`. Each function carries its rule as a comment directly above it, so the grammar and the code can be read side by side.

The parser owns all syntax errors. When it cannot continue, it returns the statements it parsed successfully before the problem together with a `SYNTAX` error, so the shell can run those statements first and then report the error, as a C64 does.

## AST

AST types live in `internal/ast`. Each category of node is a sealed interface: its marker method is unexported, so only `internal/ast` can add node types, and the interpreter's type switches can list every possibility.

```go
package ast

type Stmt interface{ stmt() }
type Expr interface{ expr() }
type PrintItem interface{ printItem() }

// Line is one line of input: statements in the order written.
type Line struct {
    Statements []Stmt
}

// PrintStmt is PRINT followed by its items, in order.
type PrintStmt struct {
    Items []PrintItem
}

// Print items.
type ExprItem struct{ Expr Expr } // a value to print
type Semicolon struct{}           // ;
type Comma struct{}               // ,
type BadItem struct{ Err error }  // where parsing failed; always the last item

// RemStmt is REM and its comment, exactly as written after REM.
type RemStmt struct{ Text string }

// Expressions.
type StringLit struct{ Value string }     // "…", contents without quotes
type Concat struct{ Left, Right Expr }    // Left + Right
```

`Concat` is left-associative: `"A"+"B"+"C"` is `Concat(Concat("A","B"),"C")`.

Empty statements (from `::` or a line of only `:`) produce no node, so `Line.Statements` holds only statements that do something. A blank line parses to a `Line` with no statements.

## Parse Functions

| Grammar rule | Function | Returns |
|---|---|---|
| `Line = Statement { ":" Statement } .` | `parseLine` | `*ast.Line` |
| `Statement = [ PrintStatement \| RemStatement ] .` | `parseStatement` | `ast.Stmt`, or nil for an empty statement |
| `RemStatement = rem .` | `parseRemStatement` | `*ast.RemStmt` |
| `PrintStatement = print { PrintItem } .` | `parsePrintStatement` | `*ast.PrintStmt` |
| `PrintItem = Expression \| ";" \| "," .` | `parsePrintItem` | `ast.PrintItem` |
| `Expression = string { "+" string } .` | `parseExpression` | `ast.Expr` |

These functions are registered in a table keyed by rule name, used by the grammar conformance test.

`parsePrintStatement` reads items until the next token is `:` or `EOL`. A statement ends only at `:` or end of line.

## Errors

The parser reports one error kind, `SYNTAX` (see the shell design for the error type and how it is printed). It is returned when:

- a statement begins with a token that cannot start a statement (for example `Illegal`, `String`, `;`), or
- inside a statement, the next token is not one the rule allows (for example `Illegal`, or `+` not followed by a string), or
- after a statement, the next token is neither `:` nor `EOL`.

Parsing stops at the first error. There is no error recovery: a C64 abandons the rest of a line at the first error, so nothing after it would ever run.

### Errors inside PRINT

A C64 executes `PRINT` one item at a time, so the items before a syntax error are printed before the error is reported: `PRINT "HELLO"@` prints `HELLO`, then `?SYNTAX  ERROR`. To reproduce this, when the error occurs inside a `PRINT` statement's items, the parser keeps that statement: its `Items` are the items completed before the error, followed by an `*ast.BadItem` holding the SYNTAX error. The interpreter prints the earlier items, reaches the `BadItem`, and fails with its error, writing no final newline.

A `Rem` token where a print item is expected is a syntax error like any other: `PRINT "A" REM NOTE` has items `ExprItem("A")`, `BadItem`, so `A` is printed and then `?SYNTAX  ERROR` is reported, as on a C64, where `PRINT` ends only at `:` or the end of the line. `PRINT "A":REM NOTE` is two statements and is valid.

An item that is itself malformed is replaced entirely by the `BadItem`. In `PRINT "A";"B"+@`, the items are `ExprItem("A")`, `Semicolon`, `BadItem`, so `A` is printed and `B` is not, matching a C64, which fails while evaluating `"B"+@` before printing it.

When the error is at the start of a statement (the first token cannot begin a statement), there is nothing to execute, and no node is produced for that statement.

## API

```go
package parser

// Parse parses the tokens of one line. It always returns a non-nil Line.
// If err is non-nil, it is a SYNTAX error, and the Line holds the
// statements completed before the error, followed, when the error is
// inside a PRINT statement's items, by that statement ending in a BadItem.
func Parse(tokens []token.Token) (*ast.Line, error)
```

Examples:

| Input | Returned `Line.Statements` |
|---|---|
| `PRINT "A":@` | `PrintStmt[ExprItem("A")]` |
| `PRINT "A":PRINT "B"@` | `PrintStmt[ExprItem("A")]`, `PrintStmt[ExprItem("B"), BadItem]` |
| `PRINT "A"+` | `PrintStmt[BadItem]` |

## Decisions & Alternatives

| Decision | Chosen | Alternatives Considered | Rationale |
|---|---|---|---|
| Parsing technique | Recursive descent, one function per rule | Parser generator; Pratt parser | Mirrors the grammar one to one, which the conformance test depends on. A Pratt parser suits operator precedence and can be introduced inside `parseExpression` when math is added, without changing the rule-per-function structure. |
| Error with partial result | Return the statements before the error, plus a partially parsed `PRINT` ending in a `BadItem`, together with the error | Return only an error; drop the statement containing the error; a `BadStmt` node replacing the whole failing statement | Reproduces what a C64 prints before a syntax error: earlier statements run, and so do the earlier items of a failing `PRINT`. The error travels with the partial result, as in `go/parser`. `BadItem` is limited to print items, the only place where a C64 produces output partway through a statement. |
| Syntax error inside a string-too-long expression | SYNTAX is reported; the concatenation is never evaluated | Evaluate operands up to the syntax error, as a C64 does | In `PRINT <long>+<long>+`, a C64 reports `STRING TOO LONG` before reaching the dangling `+`. Reproducing this needs expression evaluation interleaved with parsing, and the case needs an input line longer than a C64 can accept. |
| Sealed interfaces | Unexported marker methods | Exported marker methods; a single node struct with a kind field | Only `internal/ast` can add node types, so type switches elsewhere can be exhaustive and a panicking `default` reliably signals a missed case. |
| Empty statements | Dropped during parsing | `EmptyStmt` node | They have no effect, and dropping them keeps the interpreter free of a no-op case. |
| Comments | `RemStmt` node holding the comment text | Drop comments during parsing, like empty statements | Program mode will store and `LIST` lines with their comments, so the text has to survive into the AST even though executing it does nothing. |
| `+` representation | Binary `Concat` node, left-associative | Flat list of strings | A binary node generalizes to arithmetic operators when numbers are added. |

## Open Questions & Future Decisions

### Deferred
1. When numbers are added, `Expression` grows operators with C64 precedence, and `Concat` is generalized into a binary-operator node with an operator field. Type checking (`"A"+1` is `?TYPE MISMATCH  ERROR` on a C64) happens in the interpreter, since BASIC V2 types are determined at run time.

## References

- `docs/intent/grammar/grammar-design.md`
- `docs/intent/lexer/lexer-design.md`
- `docs/intent/interp/interp-design.md`
