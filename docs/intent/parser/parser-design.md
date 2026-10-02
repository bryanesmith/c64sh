---
parent: high-level-design
prefix: PARSER
---

# Parser

## Context and Design Philosophy

The parser turns the lexer's tokens for one line into an abstract syntax tree (AST). It is a hand-written recursive-descent parser with one function per grammar rule. Each function carries its rule, in EBNF in the notation of the Go language specification, as a comment directly above it. Together with the token rules documented in the lexer, these comments are the grammar of the language, each beside the code that implements it.

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
type NumberLit struct{ Value float64 }    // 12, 3.5, .5, 1E3

// BinaryExpr is Left Op Right.
type BinaryExpr struct {
    Op          Op
    Left, Right Expr
}

type Op int

const (
    Add Op = iota // +: adds numbers, joins strings
    Sub           // -
    Mul           // *
    Div           // /
    Pow           // ^ (exponentiation)
)

// NegExpr is -X, a leading minus sign (negation).
type NegExpr struct{ X Expr }

// CompareExpr is Left Rel Right: true (-1) when the actual relation of
// Left to Right is one of the relations in Rel.
type CompareExpr struct {
    Rel         Relation
    Left, Right Expr
}

// Relation is a set of relations, combined as a C64 combines them.
type Relation uint8

const (
    RelGreater Relation = 1 // >
    RelEqual   Relation = 2 // =
    RelLess    Relation = 4 // <
)

// VarRef is a variable, used as a value or assigned to.
type VarRef struct {
    Name string // identity: first two characters, plus "$" or "%" for a string or integer ("SC", "N$", "C%")
    Text string // the name as written, without spaces ("SCORE")
}

// LetStmt is an assignment: [LET] Var = Value.
type LetStmt struct {
    Var   *VarRef
    Value Expr
}
```

Binary operators are left-associative within a precedence level: `"A"+"B"+"C"` is `BinaryExpr(Add, BinaryExpr(Add, "A", "B"), "C")`, and `8-2-1` is `(8-2)-1`. The parser does not check operand types: whether `+` joins strings, adds numbers, or is a type mismatch is decided by the interpreter from the values, as on a C64.

### Precedence

Each precedence level is its own grammar rule, so a lower rule's operands are built by the rule below it. From loosest to tightest:

| Level | Rule | Operators |
|---|---|---|
| 1 | `Expression` | comparisons: `=` `<>` `<` `>` `<=` `>=` and the other spellings below, left to right |
| 2 | `Sum` | `+` `-` (binary), left to right |
| 3 | `Term` | `*` `/`, left to right |
| 4 | `Unary` | a leading `-` (negation) or `+` |
| 5 | `Power` | `^`, left to right |
| 6 | `Operand` | literals and `( … )` |

So `1+1=2` is `(1+1)=2`, `1<2<3` is `(1<2)<3`, `2+3*4` is 14, `(2+3)*4` is 20, `-2*3` is `(-2)*3`, `-2^2` is `-(2^2)` = -4, and `2^3^2` is `(2^3)^2` = 64, as on a C64. A leading `+` is dropped (the C64 ROM skips it: `+5` is 5 and `+"A"` is `"A"`); a leading `-` becomes a `NegExpr`. Leading signs can repeat: `--5` is 5, and `5--5` is 10.

### Comparison operators

A comparison operator is a run of one or more of the tokens `<`, `=`, and `>`, as the C64 ROM reads it (`$ADB8`). Each symbol adds a relation to a `Relation` set: `>` adds greater, `=` equal, `<` less. **The symbols may come in any order, separated by spaces or not, and each may appear once**: `<>` and `><` both mean "not equal", `<=` and `=<` "less or equal", `>=` and `=>` "greater or equal", and `<=>` holds every relation, so it is always true. A symbol repeated within one operator (`==`, `<<`, `<=<`) is a SYNTAX error, as on a C64.

A statement that begins with a name is an assignment, so its first `=` is the assignment; any later `=` compares. `A=B=C` stores in `A` the result of comparing `B` with `C`.

### Signs in an exponent

An exponent may carry a sign: `2^-1` is .5. On a C64, a sign in an exponent is a negation with its usual precedence, just below `^`, so it takes in any `^` that follows it but nothing looser: `2^-1^2` is `2^(-(1^2))` = .5, and `2^-3*4` is `(2^-3)*4` = .5. The `Exponent` rule reproduces this: an unsigned exponent is a single operand, so `^` stays left to right, and a signed one is a `Unary`, which takes in the following `^` chain.

### Variables

A statement that begins with a name, or with `LET`, is an assignment: `A=5` and `LET A=5` are the same. A name anywhere a value is expected is a variable reference.

A `VarRef`'s `Name` is the variable's identity, as on a C64: the first character, the second character if there is one, and `$` for a string variable or `%` for an integer variable. `SCORE`, `SC`, and `SCX` are all `SC`, and `NAME$` and `NA$` are both `NA$`, while `N$` is a different variable. Number, integer, and string variables never share an identity: `A`, `A%`, and `A$` are three separate variables. `Text` keeps the name as written.

These forms are valid C64 BASIC that c64sh does not support yet, so they are SYNTAX errors, following the tenet *Authentic errors over helpful ones*:

- a name whose identity is `TI`, `TI$`, or `ST`, the C64's system variables (the clock and I/O status), whether used or assigned (`TI%` and `ST%` are ordinary integer variables, as on a C64, whose check for system variables includes the type);
- a name followed by `(`, which on a C64 is an array element (`A(1)`) or a function call (`CHR$(65)`).

### Items side by side

An expression ends at the first token that cannot continue it, and PRINT then reads the next item. A token that can start an expression but not continue one begins a new item: `PRINT 2(3)` prints two numbers, ` 2  3 `. A `-` or `+` after an operand always continues the expression as a binary operator, as on a C64: `PRINT 1 -1` prints ` 0 `, and `PRINT "A"-1` is `TYPE MISMATCH`.

A `NumberLit`'s value is the token text converted with `strconv.ParseFloat`, after normalizing the forms the C64 accepts and `ParseFloat` does not: a leading `.` gets a `0` before it (`.` is 0, `.5` is 0.5), and an `E` with no exponent digits gets a `0` exponent (`1E` and `1E+` are 1). A literal too large for `float64` becomes infinity; the interpreter reports it as `OVERFLOW` when evaluated, so statements before it still run.

Empty statements (from `::` or a line of only `:`) produce no node, so `Line.Statements` holds only statements that do something. A blank line parses to a `Line` with no statements.

## Parse Functions

| Grammar rule | Function | Returns |
|---|---|---|
| `Line = Statement { ":" Statement } .` | `parseLine` | `*ast.Line` |
| `Statement = [ PrintStatement \| RemStatement \| LetStatement ] .` | `parseStatement` | `ast.Stmt`, or nil for an empty statement |
| `RemStatement = rem .` | `parseRemStatement` | `*ast.RemStmt` |
| `PrintStatement = print { PrintItem } .` | `parsePrintStatement` | `*ast.PrintStmt` |
| `PrintItem = Expression \| ";" \| "," .` | `parsePrintItem` | `ast.PrintItem` |
| `Expression = Sum { Relation Sum } .` | `parseExpression` | `ast.Expr` |
| `Relation = ( "<" \| "=" \| ">" ) { "<" \| "=" \| ">" } .` | `parseRelation` | `ast.Relation` (each symbol at most once) |
| `Sum = Term { ( "+" \| "-" ) Term } .` | `parseSum` | `ast.Expr` |
| `Term = Unary { ( "*" \| "/" ) Unary } .` | `parseTerm` | `ast.Expr` |
| `Unary = "-" Unary \| "+" Unary \| Power .` | `parseUnary` | `ast.Expr` |
| `Power = Operand { "^" Exponent } .` | `parsePower` | `ast.Expr` |
| `Exponent = "-" Unary \| "+" Unary \| Operand .` | `parseExponent` | `ast.Expr` |
| `LetStatement = [ let ] Variable "=" Expression .` | `parseLetStatement` | `*ast.LetStmt` |
| `Variable = name .` | `parseVariable` | `*ast.VarRef` |
| `Operand = string \| number \| Variable \| "(" Expression ")" .` | `parseOperand` | `ast.Expr` |

Lowercase names in these rules (`print`, `rem`, `let`, `name`, `string`, `number`) are token rules, defined and documented in the lexer.

`parsePrintStatement` reads items until the next token is `:` or `EOL`. A statement ends only at `:` or end of line.

## Errors

The parser reports one error kind, `SYNTAX` (see the shell design for the error type and how it is printed). It is returned when:

- a statement begins with a token that cannot start a statement (for example `Illegal`, `String`, `;`), or an assignment lacks its variable, its `=`, or its value, or
- inside a statement, the next token is not one the rule allows (for example `Illegal`, an operator with no operand after it, a `(` without its `)`, or a `)` without its `(`), or
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
| Parsing technique | Recursive descent, one function per rule | Parser generator; Pratt parser | Each function implements, and documents, exactly one grammar rule, so the code reads as the grammar. A Pratt parser suits operator precedence and can be introduced inside `parseExpression` when math is added, without changing the rule-per-function structure. |
| Error with partial result | Return the statements before the error, plus a partially parsed `PRINT` ending in a `BadItem`, together with the error | Return only an error; drop the statement containing the error; a `BadStmt` node replacing the whole failing statement | Reproduces what a C64 prints before a syntax error: earlier statements run, and so do the earlier items of a failing `PRINT`. The error travels with the partial result, as in `go/parser`. `BadItem` is limited to print items, the only place where a C64 produces output partway through a statement. |
| Syntax error inside a string-too-long expression | SYNTAX is reported; the concatenation is never evaluated | Evaluate operands up to the syntax error, as a C64 does | In `PRINT <long>+<long>+`, a C64 reports `STRING TOO LONG` before reaching the dangling `+`. Reproducing this needs expression evaluation interleaved with parsing, and the case needs an input line longer than a C64 can accept. |
| Sealed interfaces | Unexported marker methods | Exported marker methods; a single node struct with a kind field | Only `internal/ast` can add node types, so type switches elsewhere can be exhaustive and a panicking `default` reliably signals a missed case. |
| Empty statements | Dropped during parsing | `EmptyStmt` node | They have no effect, and dropping them keeps the interpreter free of a no-op case. |
| Comments | `RemStmt` node holding the comment text | Drop comments during parsing, like empty statements | Program mode will store and `LIST` lines with their comments, so the text has to survive into the AST even though executing it does nothing. |
| `+` representation | `BinaryExpr` with an `Op`, left-associative | A separate node per operator; a flat list of operands | One node type covers every binary operator, so `-`, `*`, `/`, and `^` add `Op` values rather than node types. |
| Precedence parsing | One grammar rule, and function, per precedence level | Precedence climbing or a Pratt parser in a single function | Keeps one function per grammar rule, each carrying its rule as a comment, so the code still reads as the grammar. C64 BASIC has few levels, so the extra functions are few. |
| Leading `+` | Dropped by the parser | A unary-plus node | The C64 ROM ignores a leading `+`, whatever follows, so there is no behavior for a node to carry. |
| `^` associativity | Left to right: `2^3^2` is 64 | Right to left (512), as in mathematics and many languages | C64 BASIC V2 evaluates `^` left to right, like its other operators. |
| Variable identity | Computed by the parser into `VarRef.Name` | Computed by the interpreter at each use | One place decides identity; the interpreter only stores and looks up. |
| Unsupported names (`TI`, `ST`, arrays, functions) | SYNTAX error | Treat as ordinary variables | On a C64 these read the clock, the I/O status, an array element, or a function result; silently treating them as plain variables would print wrong answers. |
| Comparison representation | `CompareExpr` with a `Relation` bit set | One `Op` per operator (`<`, `<=`, …) | A set of relations reproduces the C64's own rule for combining `<`, `=`, and `>` directly, including the unusual spellings, with one evaluation rule. |
| Parenthesized expressions | No node; `( … )` returns its inner expression | A `ParenExpr` node | Parentheses only group; the tree's shape already records the grouping. |
| Type checking of `+` | In the interpreter, from the operand values | In the parser, from the operand kinds | BASIC V2 types are known at run time (variables will hold either kind), and a C64 reports `?TYPE MISMATCH  ERROR` when the statement runs, after earlier statements have run. |

## Open Questions & Future Decisions

### Deferred
1. Parentheses nest without a limit; very deep nesting uses Go's stack, which grows as needed. A real C64 reports `?OUT OF MEMORY  ERROR` when nesting exhausts its stack; that limit is decided if it ever matters.

## References

- [The Go Programming Language Specification — Notation](https://go.dev/ref/spec#Notation)
- `docs/intent/lexer/lexer-design.md`
- `docs/intent/interp/interp-design.md`
