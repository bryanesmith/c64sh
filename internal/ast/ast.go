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

// Concat is Left + Right.
type Concat struct{ Left, Right Expr }

func (*PrintStmt) stmt() {}
func (*RemStmt) stmt()   {}

func (*ExprItem) printItem()  {}
func (*Semicolon) printItem() {}
func (*Comma) printItem()     {}
func (*BadItem) printItem()   {}

func (*StringLit) expr() {}
func (*Concat) expr()    {}
