// Package ast defines the abstract syntax tree the parser produces for a
// line of C64 BASIC. Each node category is a sealed interface: its marker
// method is unexported, so only this package can add node types.
package ast

// Stmt is a statement.
type Stmt interface{ stmt() }

// Expr is an expression.
type Expr interface{ expr() }

// PrintItem is one item of a PRINT statement.
type PrintItem interface{ printItem() }

// Line is one line of input: statements in the order written.
type Line struct {
	Statements []Stmt
}

// PrintStmt is PRINT followed by its items, in order, or PRINT# File,
// items, when File is not nil.
type PrintStmt struct {
	File  Expr
	Items []PrintItem
}

// ExprItem is a value to print.
type ExprItem struct{ Expr Expr }

// Semicolon is the ; print separator.
type Semicolon struct{}

// Comma is the , print separator.
type Comma struct{}

// TabItem is TAB(X): move to column X.
type TabItem struct{ X Expr }

// SpcItem is SPC(X): move right X columns.
type SpcItem struct{ X Expr }

func (*TabItem) printItem() {}
func (*SpcItem) printItem() {}

// BadItem marks where a syntax error occurred inside a PRINT statement.
// It is always the statement's last item.
type BadItem struct{ Err error }

// RemStmt is REM and its comment, exactly as written after REM.
type RemStmt struct{ Text string }

// StringLit is a string literal; Value holds its contents without quotes.
type StringLit struct{ Value string }

// NumberLit is a number literal.
type NumberLit struct{ Value float64 }

// Op is a binary operator.
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

// Relation is a set of relations, combined as a C64 combines "<", "=",
// and ">".
type Relation uint8

const (
	RelGreater Relation = 1 // >
	RelEqual   Relation = 2 // =
	RelLess    Relation = 4 // <
)

// VarRef is a variable, or an array element when Subs is not empty,
// used as a value or assigned to.
type VarRef struct {
	Name string // identity: first two characters, plus "$" or "%" for a string or integer
	Text string // the name as written, without spaces
	Subs []Expr // an array element's subscripts, in order; empty for a plain variable
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

// BinaryExpr is Left Op Right.
type BinaryExpr struct {
	Op          Op
	Left, Right Expr
}

func (*PrintStmt) stmt() {}
func (*RemStmt) stmt()   {}
func (*LetStmt) stmt()   {}
func (*IfStmt) stmt()    {}
func (*BadStmt) stmt()   {}

func (*ExprItem) printItem()  {}
func (*Semicolon) printItem() {}
func (*Comma) printItem()     {}
func (*BadItem) printItem()   {}

func (*StringLit) expr()  {}
func (*NumberLit) expr()  {}
func (*BinaryExpr) expr() {}
func (*NegExpr) expr()    {}

func (*VarRef) expr()      {}
func (*CompareExpr) expr() {}
func (*NotExpr) expr()     {}

// RunStmt is RUN, or RUN n when HasLine is set: clear the variables and
// run the stored program from its first line, or from line n.
type RunStmt struct {
	Line    int
	HasLine bool
}

// GotoStmt is GOTO n, GO TO n, or the n of IF … THEN n: continue the
// program at line n.
type GotoStmt struct{ Line int }

// ListStmt is LIST: print the stored program.
type ListStmt struct{}

// NewStmt is NEW: erase the stored program and the variables.
type NewStmt struct{}

// EndStmt is END: stop.
type EndStmt struct{}

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

// InputStmt is INPUT ["prompt";] Var {, Var}, or INPUT# File, Var
// {, Var} when File is not nil; HasPrompt is false when there is no
// prompt.
type InputStmt struct {
	File      Expr
	Prompt    string
	HasPrompt bool
	Vars      []*VarRef
}

// GetStmt is GET Var {, Var}, or GET# File, Var {, Var} when File is not
// nil.
type GetStmt struct {
	File Expr
	Vars []*VarRef
}

func (*InputStmt) stmt()  {}
func (*GetStmt) stmt()    {}
func (*GosubStmt) stmt()  {}
func (*ReturnStmt) stmt() {}
func (*ForStmt) stmt()    {}
func (*NextStmt) stmt()   {}
func (*RunStmt) stmt()    {}
func (*GotoStmt) stmt()   {}
func (*ListStmt) stmt()   {}
func (*NewStmt) stmt()    {}
func (*EndStmt) stmt()    {}

// DefStmt is DEF FN Name(Param) = Body. If the body is not a valid
// expression ending the statement, Body is nil and BodyErr holds the
// SYNTAX error, reported when the function is called.
type DefStmt struct {
	Name, Param *VarRef
	Body        Expr
	BodyErr     error
}

// FnExpr is FN Name(Arg): a call of a user-defined function.
type FnExpr struct {
	Name *VarRef
	Arg  Expr
}

func (*DefStmt) stmt() {}
func (*FnExpr) expr()  {}

// FileArgs are the arguments of LOAD, SAVE, and VERIFY: the file name,
// the device, and the secondary address, each nil when omitted.
type FileArgs struct{ Name, Device, Secondary Expr }

// LoadStmt is LOAD [name [, device [, secondary]]].
type LoadStmt struct{ FileArgs }

// SaveStmt is SAVE [name [, device [, secondary]]].
type SaveStmt struct{ FileArgs }

// VerifyStmt is VERIFY [name [, device [, secondary]]].
type VerifyStmt struct{ FileArgs }

func (*LoadStmt) stmt()   {}
func (*SaveStmt) stmt()   {}
func (*VerifyStmt) stmt() {}

// OpenStmt is OPEN File [, Device [, Secondary [, Name]]]; omitted
// arguments are nil.
type OpenStmt struct{ File, Device, Secondary, Name Expr }

// CloseStmt is CLOSE File.
type CloseStmt struct{ File Expr }

// CmdStmt is CMD File [, items]: send PRINT output to the file.
type CmdStmt struct {
	File  Expr
	Items []PrintItem
}

func (*OpenStmt) stmt()  {}
func (*CloseStmt) stmt() {}
func (*CmdStmt) stmt()   {}

// OnStmt is ON Index GOTO|GOSUB Line {, Line}.
type OnStmt struct {
	Index Expr
	Gosub bool // GOSUB; otherwise GOTO
	Lines []int
}

func (*OnStmt) stmt() {}

// CallExpr is a call of a built-in function: Name(Args).
type CallExpr struct {
	Name string // "SIN", "RND", …
	Args []Expr
}

func (*CallExpr) expr() {}

// DimStmt is DIM Array {, Array}: each VarRef holds the top subscripts.
type DimStmt struct{ Arrays []*VarRef }

func (*DimStmt) stmt() {}

// DataStmt is DATA and its text, exactly as the lexer kept it.
type DataStmt struct{ Text string }

// ReadStmt is READ Var {, Var}.
type ReadStmt struct{ Vars []*VarRef }

// RestoreStmt is RESTORE.
type RestoreStmt struct{}

func (*DataStmt) stmt()    {}
func (*ReadStmt) stmt()    {}
func (*RestoreStmt) stmt() {}
