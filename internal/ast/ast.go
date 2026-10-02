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

// PrintStmt is PRINT followed by its items, in order.
type PrintStmt struct {
	Items []PrintItem
}

// ExprItem is a value to print.
type ExprItem struct{ Expr Expr }

// Semicolon is the ; print separator.
type Semicolon struct{}

// Comma is the , print separator.
type Comma struct{}

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

// VarRef is a variable, used as a value or assigned to.
type VarRef struct {
	Name string // identity: first two characters, plus "$" for a string
	Text string // the name as written, without spaces
}

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
