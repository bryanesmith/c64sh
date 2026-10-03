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
    And           // AND, bitwise
    Or            // OR, bitwise
)

// NotExpr is NOT X, the bitwise complement.
type NotExpr struct{ X Expr }

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
    Text string // the name as written, without spaces ("HEIGHT")
}

// IfStmt is IF Cond THEN. It guards the rest of its line: the statements
// after it run only when Cond is true.
type IfStmt struct{ Cond Expr }

// BadStmt marks where a statement failed to parse; it is always the last
// statement of its line.
type BadStmt struct{ Err error }

// LetStmt is an assignment: [LET] Var = Value.
type LetStmt struct {
    Var   *VarRef
    Value Expr
}

// RunStmt is RUN, or RUN n when HasLine is set: clear the variables and
// run the stored program from its first line, or from line n.
type RunStmt struct {
    Line    int
    HasLine bool
}

// GotoStmt is GOTO n, GO TO n, or the n of IF … THEN n: continue the
// program at line n.
type GotoStmt struct{ Line int }

// Program commands without arguments.
type ListStmt struct{} // LIST: print the stored program
type NewStmt struct{}  // NEW: erase the stored program and the variables
type EndStmt struct{}  // END: stop

// ForStmt is FOR Var = From TO To [STEP Step]; Step is nil when omitted.
type ForStmt struct {
    Var            *VarRef
    From, To, Step Expr
}

// NextStmt is NEXT [Var {, Var}]; Vars is empty for a bare NEXT.
type NextStmt struct{ Vars []*VarRef }

// GosubStmt is GOSUB n: call the subroutine at line n.
type GosubStmt struct{ Line int }

// ReturnStmt is RETURN: return from the latest subroutine.
type ReturnStmt struct{}

// InputStmt is INPUT ["prompt";] Var {, Var}; HasPrompt is false when
// there is no prompt.
type InputStmt struct {
    Prompt    string
    HasPrompt bool
    Vars      []*VarRef
}

// GetStmt is GET Var {, Var}.
type GetStmt struct{ Vars []*VarRef }

// DefStmt is DEF FN Name(Param) = Body. If the body is not a valid
// expression ending the statement, Body is nil and BodyErr holds the
// SYNTAX error, reported when the function is called.
type DefStmt struct {
    Name, Param *VarRef
    Body        Expr
    BodyErr     error
}

// FileArgs are the arguments of LOAD, SAVE, and VERIFY: the file name,
// the device, and the secondary address, each nil when omitted.
type FileArgs struct{ Name, Device, Secondary Expr }

// LoadStmt, SaveStmt, and VerifyStmt are LOAD, SAVE, and VERIFY.
type LoadStmt struct{ FileArgs }
type SaveStmt struct{ FileArgs }
type VerifyStmt struct{ FileArgs }

// FnExpr is FN Name(Arg): a call of a user-defined function.
type FnExpr struct {
    Name *VarRef
    Arg  Expr
}
```

Binary operators are left-associative within a precedence level: `"A"+"B"+"C"` is `BinaryExpr(Add, BinaryExpr(Add, "A", "B"), "C")`, and `8-2-1` is `(8-2)-1`. The parser does not check operand types: whether `+` joins strings, adds numbers, or is a type mismatch is decided by the interpreter from the values, as on a C64.

### Precedence

Each precedence level is its own grammar rule, so a lower rule's operands are built by the rule below it. From loosest to tightest:

| Level | Rule | Operators |
|---|---|---|
| 1 | `Expression` | `OR`, left to right |
| 2 | `Conjunction` | `AND`, left to right |
| 3 | `Comparison` | comparisons: `=` `<>` `<` `>` `<=` `>=` and the other spellings below, left to right |
| 4 | `Sum` | `+` `-` (binary), left to right |
| 5 | `Term` | `*` `/`, left to right |
| 6 | `Unary` | a leading `-` (negation) or `+` |
| 7 | `Power` | `^`, left to right |
| 8 | `Operand` | literals, variables, `( … )`, and `NOT` (see below) |

So `A OR B AND C` is `A OR (B AND C)`, `1<2 AND 3<4` is `(1<2) AND (3<4)`, `1+1=2` is `(1+1)=2`, `1<2<3` is `(1<2)<3`, `2+3*4` is 14, `(2+3)*4` is 20, `-2*3` is `(-2)*3`, `-2^2` is `-(2^2)` = -4, and `2^3^2` is `(2^3)^2` = 64, as on a C64. A leading `+` is dropped (the C64 ROM skips it: `+5` is 5 and `+"A"` is `"A"`); a leading `-` becomes a `NegExpr`. Leading signs can repeat: `--5` is 5, and `5--5` is 10.

### NOT

`NOT` binds more loosely than comparisons and more tightly than `AND`, and, as on a C64 (`$AED0`), it is read where an operand is expected and takes in everything after it up to the next `AND`, `OR`, or the end of the expression. So `NOT 1=2` is `NOT (1=2)`, `NOT A AND B` is `(NOT A) AND B`, and in the middle of an expression `1+NOT 0+1` is `1+NOT (0+1)`. The `Operand` rule expresses this: `NOT` followed by a `Comparison`.

### Comparison operators

A comparison operator is a run of one or more of the tokens `<`, `=`, and `>`, as the C64 ROM reads it (`$ADB8`). Each symbol adds a relation to a `Relation` set: `>` adds greater, `=` equal, `<` less. **The symbols may come in any order, separated by spaces or not, and each may appear once**: `<>` and `><` both mean "not equal", `<=` and `=<` "less or equal", `>=` and `=>` "greater or equal", and `<=>` holds every relation, so it is always true. A symbol repeated within one operator (`==`, `<<`, `<=<`) is a SYNTAX error, as on a C64.

A statement that begins with a name is an assignment, so its first `=` is the assignment; any later `=` compares. `A=B=C` stores in `A` the result of comparing `B` with `C`.

### Signs in an exponent

An exponent may carry a sign: `2^-1` is .5. On a C64, a sign in an exponent is a negation with its usual precedence, just below `^`, so it takes in any `^` that follows it but nothing looser: `2^-1^2` is `2^(-(1^2))` = .5, and `2^-3*4` is `(2^-3)*4` = .5. The `Exponent` rule reproduces this: an unsigned exponent is a single operand, so `^` stays left to right, and a signed one is a `Unary`, which takes in the following `^` chain.

### Variables

A statement that begins with a name, or with `LET`, is an assignment: `A=5` and `LET A=5` are the same. A name anywhere a value is expected is a variable reference.

A `VarRef`'s `Name` is the variable's identity, as on a C64: the first character, the second character if there is one, and `$` for a string variable or `%` for an integer variable. `HEIGHT`, `HE`, and `HEX` are all `HE`, and `NAME$` and `NA$` are both `NA$`, while `N$` is a different variable. Number, integer, and string variables never share an identity: `A`, `A%`, and `A$` are three separate variables. `Text` keeps the name as written.

These forms are valid C64 BASIC that c64sh does not support yet, so they are SYNTAX errors, following the tenet *Authentic errors over helpful ones*:

- a name whose identity is `TI`, `TI$`, or `ST`, the C64's system variables (the clock and I/O status), whether used or assigned (`TI%` and `ST%` are ordinary integer variables, as on a C64, whose check for system variables includes the type);
- a name followed by `(`, which on a C64 is an array element (`A(1)`) or a function call (`CHR$(65)`).

### IF

`IF condition THEN` becomes an `IfStmt` holding the condition. The statement that follows `THEN`, and every statement after it on the line, are parsed as the line's next statements, with no `:` needed after `THEN`: `IF A>1 THEN PRINT "X":PRINT "Y"` is `IfStmt(A>1)`, `PrintStmt("X")`, `PrintStmt("Y")`. An `IfStmt` thus guards the rest of its line, which is exactly the C64's rule: when the condition is false the rest of the line is skipped, as if it were a `REM` (`$A928`). Nothing may follow `THEN`, in which case the `IF` does nothing either way, and `IF`s may follow each other (`IF A THEN IF B THEN …`).

**Jumps.** As in the ROM (`$A928`, `$A940`), the condition may be followed by `GOTO` instead of `THEN`; the `GOTO` is then not consumed, so it is parsed as the next statement: `IF X GOTO 100` is `IfStmt(X)`, `GotoStmt(100)`, exactly like `IF X THEN GOTO 100`. After `THEN`, a `Number` token whose text begins with a digit is a line number: `IF X THEN 100` is `IfStmt(X)`, `GotoStmt(100)`, its line read as for `GOTO`. A number that does not begin with a digit (`IF X THEN .5`) is not a line number and is a syntax error like any other statement that begins with a number. Anything other than `THEN` or `GOTO` after the condition (including `GO TO`, which the ROM does not check for here) is a SYNTAX error in place of the `IF`.

### Program commands

`RUN`, `GOTO`, `LIST`, `NEW`, and `END` act on the stored program (see the interpreter design). Each stops the line it is on, so nothing after it on the line ever runs.

- **`LIST`, `NEW`, and `END` take no arguments.** If anything other than `:` or the end of the line follows one, it is a SYNTAX error in place of the statement, so the command does not run, as on a C64 (`$A642`, `$A831`, `$A69C`). `LIST 10` and `LIST 10-20`, which list part of a program on a C64, are among these errors: ranges are not supported yet.
- **`RUN` alone** runs the program from its first line. **Anything else after `RUN` makes it `RUN n`**, with `n` read as for `GOTO`.
- **`GOTO` is always followed by a line number**, read from the following `Number` token's text by `lexer.LineNumber`, as the ROM reads it (`$A8A0`): `GOTO 20` and `GOTO 20.5` both go to line 20. When no `Number` follows, or its text does not begin with a digit, the line number is 0, so `GOTO` alone and `GOTO A` are `GOTO 0`, as on a C64. A line number above 63999 is a SYNTAX error in place of the statement. What follows the line number is never checked, because a jump does not return to its line.
- **`GO TO`** is the keyword `GO` followed by the keyword `TO`, then a line number as for `GOTO`, giving the same `GotoStmt` (`$A80E`). `GO` followed by anything other than `TO` is a SYNTAX error in place of the statement.

### FOR and NEXT

`FOR` is followed by a variable, `=`, the start value, `TO`, the end value, and optionally `STEP` and the step. The variable must be a plain number or string variable: an integer variable (`FOR I%=…`) is a SYNTAX error, as on a C64, whose `FOR` refuses integer counters; a string variable parses, and is a `TYPE MISMATCH` when it runs, as the ROM finds it then (`$A772`). A missing `TO`, or an expression missing where one is required, is a SYNTAX error in place of the statement.

`NEXT` is followed by nothing, or by one or more variables separated by commas. A variable list that does not end at `:` or the end of the line (`NEXT I J`, `NEXT I,`) is a SYNTAX error in place of the statement.

### GOSUB and RETURN

`GOSUB` is followed by a line number, read exactly as for `GOTO` (see *Program commands*): `GOSUB 100`, `GOSUB 100.5` (line 100), and `GOSUB` alone (line 0). `RETURN` takes no arguments; anything after it other than `:` or the end of the line is a SYNTAX error in place of the statement, as with `END` (`$A8D2`). `GO SUB`, with a space, is not `GOSUB`: it is `GO` not followed by `TO`, a SYNTAX error.

### INPUT and GET

`INPUT` is followed by an optional prompt, then one or more variables separated by commas. The prompt is a string literal followed by `;`, as the ROM requires (`$ABBF`): `INPUT "NAME";N$`. An expression is not allowed as the prompt, so `INPUT "A"+"B";X` and `INPUT P$;X` are SYNTAX errors, and so is a prompt followed by `,` instead of `;`. `GET` is followed by one or more variables separated by commas. A missing variable, or anything after the list other than `:` or the end of the line, is a SYNTAX error in place of the statement.

### DEF FN

`DEF` is followed by `FN`, the function's name, `(`, the parameter's name, `)`, `=`, and the body. The name and the parameter are variable names whose identity is computed as for variables; a name followed by `(` is allowed here, unlike a variable. An integer name or parameter (`FN A%`, `(X%)`) is a SYNTAX error, as on a C64; a string name or parameter parses, and is a `TYPE MISMATCH` when the statement runs. Anything else missing or out of place before the body is a SYNTAX error in place of the statement.

**The body is not checked when the `DEF` runs**, because a C64 skips over it to the end of the statement (`$B3DB`) and evaluates it only when the function is called, then requires the statement to end after it (`$B441`). So the parser parses the body as an expression; if that fails, or the expression is not followed by `:` or the end of the line, it records the SYNTAX error in `BodyErr`, skips to the next `:` or the end of the line, and the `DefStmt` is otherwise valid. The error is reported only when the function is called.

`FN` is read where an operand is expected, followed by the function's name and a parenthesized argument: `FN SQ(3)` is an `FnExpr`. An integer name is a SYNTAX error; a missing `(` or `)` too.

### LOAD, SAVE, VERIFY

Each takes up to three arguments, separated by commas, as on a C64 (`$E1D4`): the file name, the device, and the secondary address, each an expression, so `SAVE "GAME",8` and `LOAD N$,D` are valid. Any may be omitted from the end: `LOAD` alone, `LOAD "GAME"`. A comma not followed by an expression, or anything after the arguments other than `:` or the end of the line, is a SYNTAX error in place of the statement. The types and ranges of the arguments are checked when the statement runs.

### Items side by side

An expression ends at the first token that cannot continue it, and PRINT then reads the next item. A token that can start an expression but not continue one begins a new item: `PRINT 2(3)` prints two numbers, ` 2  3 `. A `-` or `+` after an operand always continues the expression as a binary operator, as on a C64: `PRINT 1 -1` prints ` 0 `, and `PRINT "A"-1` is `TYPE MISMATCH`.

A `NumberLit`'s value is the token text converted with `strconv.ParseFloat`, after normalizing the forms the C64 accepts and `ParseFloat` does not: a leading `.` gets a `0` before it (`.` is 0, `.5` is 0.5), and an `E` with no exponent digits gets a `0` exponent (`1E` and `1E+` are 1). A literal too large for `float64` becomes infinity; the interpreter reports it as `OVERFLOW` when evaluated, so statements before it still run.

Empty statements (from `::` or a line of only `:`) produce no node, so `Line.Statements` holds only statements that do something. A blank line parses to a `Line` with no statements.

## Parse Functions

| Grammar rule | Function | Returns |
|---|---|---|
| `Line = Statement { ":" Statement } .` | `parseLine` | `*ast.Line` |
| `Statement = [ PrintStatement \| RemStatement \| LetStatement \| IfStatement \| RunStatement \| GotoStatement \| ForStatement \| NextStatement \| GosubStatement \| ReturnStatement \| InputStatement \| GetStatement \| DefStatement \| LoadStatement \| SaveStatement \| VerifyStatement \| ListStatement \| NewStatement \| EndStatement ] .` | `parseStatement` | `ast.Stmt`, or nil for an empty statement |
| `RemStatement = rem .` | `parseRemStatement` | `*ast.RemStmt` |
| `PrintStatement = print { PrintItem } .` | `parsePrintStatement` | `*ast.PrintStmt` |
| `PrintItem = Expression \| ";" \| "," .` | `parsePrintItem` | `ast.PrintItem` |
| `Expression = Conjunction { or Conjunction } .` | `parseExpression` | `ast.Expr` |
| `Conjunction = Comparison { and Comparison } .` | `parseConjunction` | `ast.Expr` |
| `Comparison = Sum { Relation Sum } .` | `parseComparison` | `ast.Expr` |
| `Relation = ( "<" \| "=" \| ">" ) { "<" \| "=" \| ">" } .` | `parseRelation` | `ast.Relation` (each symbol at most once) |
| `Sum = Term { ( "+" \| "-" ) Term } .` | `parseSum` | `ast.Expr` |
| `Term = Unary { ( "*" \| "/" ) Unary } .` | `parseTerm` | `ast.Expr` |
| `Unary = "-" Unary \| "+" Unary \| Power .` | `parseUnary` | `ast.Expr` |
| `Power = Operand { "^" Exponent } .` | `parsePower` | `ast.Expr` |
| `Exponent = "-" Unary \| "+" Unary \| Operand .` | `parseExponent` | `ast.Expr` |
| `LetStatement = [ let ] Variable "=" Expression .` | `parseLetStatement` | `*ast.LetStmt` |
| `IfStatement = if Expression ( then [ LineNumber ] \| /* goto, parsed as the next statement */ ) .` | `parseIfStatement` | `*ast.IfStmt`, and a `*ast.GotoStmt` for `THEN n` (see *IF*) |
| `RunStatement = run [ LineNumber ] .` | `parseRunStatement` | `*ast.RunStmt` (see *Program commands*) |
| `ForStatement = for Variable "=" Expression to Expression [ step Expression ] .` | `parseForStatement` | `*ast.ForStmt` |
| `NextStatement = next [ Variable { "," Variable } ] .` | `parseNextStatement` | `*ast.NextStmt` |
| `GosubStatement = gosub LineNumber .` | `parseGosubStatement` | `*ast.GosubStmt` |
| `ReturnStatement = return .` | `parseCommand` | `*ast.ReturnStmt` |
| `InputStatement = input [ string ";" ] Variable { "," Variable } .` | `parseInputStatement` | `*ast.InputStmt` |
| `GetStatement = get Variable { "," Variable } .` | `parseGetStatement` | `*ast.GetStmt` |
| `DefStatement = def fn FunctionName "(" FunctionName ")" "=" Expression .` | `parseDefStatement` | `*ast.DefStmt` (body errors kept in `BodyErr`) |
| `FunctionName = name .` | `parseFunctionName` | `*ast.VarRef` (not an integer name; may be followed by `(`) |
| `LoadStatement = load FileArgs .` | `parseFileStatement` | `*ast.LoadStmt` |
| `SaveStatement = save FileArgs .` | `parseFileStatement` | `*ast.SaveStmt` |
| `VerifyStatement = verify FileArgs .` | `parseFileStatement` | `*ast.VerifyStmt` |
| `FileArgs = [ Expression [ "," Expression [ "," Expression ] ] ] .` | `parseFileArgs` | `ast.FileArgs` |
| `GotoStatement = ( goto \| go to ) LineNumber .` | `parseGotoStatement` | `*ast.GotoStmt` |
| `LineNumber = [ number ] .` | `parseLineNumber` | `int`: the line number, 0 if there is none (see *Program commands*) |
| `ListStatement = list .` | `parseCommand` | `*ast.ListStmt` |
| `NewStatement = new .` | `parseCommand` | `*ast.NewStmt` |
| `EndStatement = end .` | `parseCommand` | `*ast.EndStmt` (`parseCommand` parses any command without arguments) |
| `Variable = name .` | `parseVariable` | `*ast.VarRef` |
| `Operand = string \| number \| Variable \| "(" Expression ")" \| not Comparison \| fn FunctionName "(" Expression ")" .` | `parseOperand` | `ast.Expr` |

Lowercase names in these rules (`print`, `rem`, `let`, `and`, `or`, `not`, `if`, `then`, `run`, `goto`, `go`, `to`, `for`, `next`, `step`, `gosub`, `return`, `input`, `get`, `def`, `fn`, `load`, `save`, `verify`, `list`, `new`, `end`, `name`, `string`, `number`) are token rules, defined and documented in the lexer.

`parsePrintStatement` reads items until the next token is `:` or `EOL`. A statement ends only at `:` or end of line.

## Errors

The parser reports one error kind, `SYNTAX` (see the shell design for the error type and how it is printed). It is returned when:

- a statement begins with a token that cannot start a statement (for example `Illegal`, `String`, `;`), or an assignment lacks its variable, its `=`, or its value, or
- `LIST`, `NEW`, `END`, or `RETURN` is followed by anything other than `:` or `EOL`, `GO` is not followed by `TO`, or the line number after `RUN`, `GOTO`, `GOSUB`, or `THEN` exceeds 63999, or
- inside a statement, the next token is not one the rule allows (for example `Illegal`, an operator with no operand after it, a `(` without its `)`, or a `)` without its `(`), or
- after a statement, the next token is neither `:` nor `EOL`.

Parsing stops at the first error. There is no error recovery: a C64 abandons the rest of a line at the first error, so nothing after it would ever run.

The error is placed in the tree where it occurred, so that it is reported only if execution reaches it. A C64 checks a statement's syntax only as it runs it, so an error in the part of a line skipped by a false `IF` is never reported (`IF 0 THEN PRINT "A"@` does nothing). Inside a `PRINT`'s items the error is a `BadItem` (below); anywhere else, the failing statement is replaced by a `BadStmt` holding the error, as the line's last statement.

### Errors inside PRINT

A C64 executes `PRINT` one item at a time, so the items before a syntax error are printed before the error is reported: `PRINT "HELLO"@` prints `HELLO`, then `?SYNTAX  ERROR`. To reproduce this, when the error occurs inside a `PRINT` statement's items, the parser keeps that statement: its `Items` are the items completed before the error, followed by an `*ast.BadItem` holding the SYNTAX error. The interpreter prints the earlier items, reaches the `BadItem`, and fails with its error, writing no final newline.

A `Rem` token where a print item is expected is a syntax error like any other: `PRINT "A" REM NOTE` has items `ExprItem("A")`, `BadItem`, so `A` is printed and then `?SYNTAX  ERROR` is reported, as on a C64, where `PRINT` ends only at `:` or the end of the line. `PRINT "A":REM NOTE` is two statements and is valid.

An item that is itself malformed is replaced entirely by the `BadItem`. In `PRINT "A";"B"+@`, the items are `ExprItem("A")`, `Semicolon`, `BadItem`, so `A` is printed and `B` is not, matching a C64, which fails while evaluating `"B"+@` before printing it.

Any other error, such as one at the start of a statement or in an assignment, replaces the whole failing statement with a `BadStmt`.

## API

```go
package parser

// Parse parses the tokens of one line. It always returns a non-nil Line.
// If err is non-nil, it is a SYNTAX error, and the Line holds the
// statements completed before the error, followed by either that PRINT
// statement ending in a BadItem (an error inside PRINT's items) or a
// BadStmt holding the error.
func Parse(tokens []token.Token) (*ast.Line, error)
```

Examples:

| Input | Returned `Line.Statements` |
|---|---|
| `PRINT "A":@` | `PrintStmt[ExprItem("A")]`, `BadStmt` |
| `IF 0 THEN @` | `IfStmt(0)`, `BadStmt` |
| `PRINT "A":PRINT "B"@` | `PrintStmt[ExprItem("A")]`, `PrintStmt[ExprItem("B"), BadItem]` |
| `PRINT "A"+` | `PrintStmt[BadItem]` |
| `RUN 20` | `RunStmt(20)` |
| `IF A THEN 20` | `IfStmt(A)`, `GotoStmt(20)` |
| `LIST 10` | `BadStmt` |

## Decisions & Alternatives

| Decision | Chosen | Alternatives Considered | Rationale |
|---|---|---|---|
| Parsing technique | Recursive descent, one function per rule | Parser generator; Pratt parser | Each function implements, and documents, exactly one grammar rule, so the code reads as the grammar. A Pratt parser suits operator precedence and can be introduced inside `parseExpression` when math is added, without changing the rule-per-function structure. |
| Error with partial result | The statements before the error, then the error itself in the tree: a `PRINT` ending in a `BadItem`, or a `BadStmt`; the error is also returned | Return only an error; drop the statement containing the error and return the error separately | Reproduces the C64's order of events: earlier statements run, and so do the earlier items of a failing `PRINT`. Placing the error in the tree means it is reported only if execution reaches it, so a syntax error after a false `IF` goes unnoticed, as on a C64, which checks syntax only as it runs. |
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
| Line number after `RUN`, `GOTO`, `THEN` | Read with the ROM's line-number routine, 0 when no digits follow | A SYNTAX error unless a plain line number follows | The C64 reads it this way, so `GOTO 20.5` goes to line 20 and `GOTO A` to line 0; reproducing it costs nothing, since the same routine reads line numbers at the start of a line. |
| `IF … THEN n` | An `IfStmt` followed by a `GotoStmt` | A line-number field on `IfStmt` | The jump is an ordinary statement guarded by the `IF`, as in the ROM, so the interpreter needs no special case. |
| Parenthesized expressions | No node; `( … )` returns its inner expression | A `ParenExpr` node | Parentheses only group; the tree's shape already records the grouping. |
| Type checking of `+` | In the interpreter, from the operand values | In the parser, from the operand kinds | BASIC V2 types are known at run time (variables will hold either kind), and a C64 reports `?TYPE MISMATCH  ERROR` when the statement runs, after earlier statements have run. |

## Open Questions & Future Decisions

### Deferred
1. Parentheses nest without a limit; very deep nesting uses Go's stack, which grows as needed. A real C64 reports `?OUT OF MEMORY  ERROR` when nesting exhausts its stack; that limit is decided if it ever matters.

## References

- [The Go Programming Language Specification — Notation](https://go.dev/ref/spec#Notation)
- `docs/intent/lexer/lexer-design.md`
- `docs/intent/interp/interp-design.md`
