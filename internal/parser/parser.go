// Package parser turns the tokens of a line into an AST, with one parse
// function per grammar rule. Each function carries its rule in EBNF, in the
// notation of the Go language specification; lowercase names are token
// rules, documented in package lexer.
package parser

import (
	"strconv"
	"strings"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
	"github.com/bryanesmith/c64sh/internal/token"
)

// Parse parses the tokens of one line. It always returns a non-nil Line.
// If err is non-nil, it is a SYNTAX error, and the Line holds the
// statements completed before the error, followed, when the error is
// inside a PRINT statement's items, by that statement ending in a BadItem.
//
// @spec PARSER-001, PARSER-002, PARSER-003, PARSER-004, PARSER-005, PARSER-006
// @spec PARSER-007, PARSER-008, PARSER-009, PARSER-010, PARSER-011, PARSER-012, PARSER-013
// @spec PARSER-014, PARSER-015, PARSER-016, PARSER-017, PARSER-018, PARSER-019, PARSER-020
// @spec PARSER-021, PARSER-022, PARSER-023, PARSER-024, PARSER-025
func Parse(tokens []token.Token) (*ast.Line, error) {
	if n := len(tokens); n == 0 || tokens[n-1].Kind != token.EOL {
		tokens = append(tokens[:n:n], token.Token{Kind: token.EOL})
	}
	p := &parser{tokens: tokens}
	return p.parseLine()
}

type parser struct {
	tokens []token.Token // always ends with EOL
	pos    int
}

func (p *parser) peek() token.Kind {
	return p.tokens[p.pos].Kind
}

// next consumes and returns the current token. It never moves past EOL.
func (p *parser) next() token.Token {
	t := p.tokens[p.pos]
	if p.pos < len(p.tokens)-1 {
		p.pos++
	}
	return t
}

// accept consumes the current token if it is of kind k.
func (p *parser) accept(k token.Kind) bool {
	if p.peek() != k {
		return false
	}
	p.next()
	return true
}

func syntaxError() error {
	return &basicerr.Error{Kind: basicerr.Syntax}
}

// Line = Statement { ":" Statement } .
func (p *parser) parseLine() (*ast.Line, error) {
	line := &ast.Line{}
	for {
		stmt, err := p.parseStatement()
		if stmt != nil {
			line.Statements = append(line.Statements, stmt)
		}
		if err != nil {
			return line, err
		}
		if !p.accept(token.Colon) {
			break
		}
	}
	if p.peek() != token.EOL {
		return line, syntaxError()
	}
	return line, nil
}

// Statement = [ PrintStatement | RemStatement | LetStatement ] .
//
// An empty statement returns a nil Stmt.
func (p *parser) parseStatement() (ast.Stmt, error) {
	switch p.peek() {
	case token.Print:
		return p.parsePrintStatement()
	case token.Rem:
		return p.parseRemStatement()
	case token.Let, token.Name:
		stmt, err := p.parseLetStatement()
		if err != nil {
			return nil, err // no node for a statement with an error
		}
		return stmt, nil
	default:
		return nil, nil
	}
}

// LetStatement = [ let ] Variable "=" Expression .
func (p *parser) parseLetStatement() (*ast.LetStmt, error) {
	p.accept(token.Let)
	if p.peek() != token.Name {
		return nil, syntaxError()
	}
	v, err := p.parseVariable()
	if err != nil {
		return nil, err
	}
	if !p.accept(token.Equal) {
		return nil, syntaxError()
	}
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	return &ast.LetStmt{Var: v, Value: value}, nil
}

// Variable = name .
//
// The C64's system variables (TI, TI$, ST), and a name followed by "(" (an
// array element or function call), are not supported and are SYNTAX errors.
func (p *parser) parseVariable() (*ast.VarRef, error) {
	text := p.next().Value
	v := &ast.VarRef{Name: variableIdentity(text), Text: text}
	switch {
	case v.Name == "TI" || v.Name == "TI$" || v.Name == "ST":
		return nil, syntaxError()
	case p.peek() == token.LParen:
		return nil, syntaxError()
	}
	return v, nil
}

// variableIdentity returns the part of a variable name that identifies the
// variable on a C64: its first character, its second character if it has
// one, and "$" or "%" for a string or integer variable ("SCORE" is "SC",
// "NAME$" is "NA$", "COUNT%" is "CO%").
func variableIdentity(text string) string {
	suffix := ""
	if strings.HasSuffix(text, "$") || strings.HasSuffix(text, "%") {
		suffix = text[len(text)-1:]
	}
	letters := strings.TrimSuffix(text, suffix)
	if len(letters) > 2 {
		letters = letters[:2]
	}
	return letters + suffix
}

// RemStatement = rem .
func (p *parser) parseRemStatement() (*ast.RemStmt, error) {
	return &ast.RemStmt{Text: p.next().Value}, nil
}

// PrintStatement = print { PrintItem } .
//
// If an item fails to parse, the statement is returned with the items
// before it followed by a BadItem, together with the error.
func (p *parser) parsePrintStatement() (*ast.PrintStmt, error) {
	p.next() // print
	stmt := &ast.PrintStmt{}
	for k := p.peek(); k != token.Colon && k != token.EOL; k = p.peek() {
		item, err := p.parsePrintItem()
		if err != nil {
			stmt.Items = append(stmt.Items, &ast.BadItem{Err: err})
			return stmt, err
		}
		stmt.Items = append(stmt.Items, item)
	}
	return stmt, nil
}

// PrintItem = Expression | ";" | "," .
func (p *parser) parsePrintItem() (ast.PrintItem, error) {
	switch p.peek() {
	case token.Semicolon:
		p.next()
		return &ast.Semicolon{}, nil
	case token.Comma:
		p.next()
		return &ast.Comma{}, nil
	case token.String, token.Number, token.Name, token.Minus, token.Plus, token.LParen:
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		return &ast.ExprItem{Expr: expr}, nil
	default:
		return nil, syntaxError()
	}
}

// Expression = Term { ( "+" | "-" ) Term } .
func (p *parser) parseExpression() (ast.Expr, error) {
	left, err := p.parseTerm()
	if err != nil {
		return nil, err
	}
	for {
		var op ast.Op
		switch {
		case p.accept(token.Plus):
			op = ast.Add
		case p.accept(token.Minus):
			op = ast.Sub
		default:
			return left, nil
		}
		right, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Op: op, Left: left, Right: right}
	}
}

// Term = Unary { ( "*" | "/" ) Unary } .
func (p *parser) parseTerm() (ast.Expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		var op ast.Op
		switch {
		case p.accept(token.Star):
			op = ast.Mul
		case p.accept(token.Slash):
			op = ast.Div
		default:
			return left, nil
		}
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Op: op, Left: left, Right: right}
	}
}

// Unary = "-" Unary | "+" Unary | Power .
//
// A leading "+" produces no node: the C64 ROM skips it.
func (p *parser) parseUnary() (ast.Expr, error) {
	switch {
	case p.accept(token.Minus):
		x, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.NegExpr{X: x}, nil
	case p.accept(token.Plus):
		return p.parseUnary()
	default:
		return p.parsePower()
	}
}

// Power = Operand { "^" Exponent } .
func (p *parser) parsePower() (ast.Expr, error) {
	left, err := p.parseOperand()
	if err != nil {
		return nil, err
	}
	for p.accept(token.Caret) {
		right, err := p.parseExponent()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Op: ast.Pow, Left: left, Right: right}
	}
	return left, nil
}

// Exponent = "-" Unary | "+" Unary | Operand .
//
// A signed exponent is a Unary, so the sign takes in any "^" that follows
// (2^-1^2 is 2^(-(1^2)), as on a C64); an unsigned one is a single operand,
// so "^" stays left to right.
func (p *parser) parseExponent() (ast.Expr, error) {
	if p.peek() == token.Minus || p.peek() == token.Plus {
		return p.parseUnary()
	}
	return p.parseOperand()
}

// Operand = string | number | Variable | "(" Expression ")" .
//
// Parentheses produce no node: the tree's shape records the grouping.
func (p *parser) parseOperand() (ast.Expr, error) {
	switch p.peek() {
	case token.String:
		return &ast.StringLit{Value: p.next().Value}, nil
	case token.Number:
		return &ast.NumberLit{Value: numberValue(p.next().Value)}, nil
	case token.Name:
		return p.parseVariable()
	case token.LParen:
		p.next()
		x, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if !p.accept(token.RParen) {
			return nil, syntaxError()
		}
		return x, nil
	default:
		return nil, syntaxError()
	}
}

// numberValue converts the text of a number token. It first normalizes the
// forms a C64 accepts and strconv.ParseFloat does not: a leading "." ("." is
// 0) and an "E" with no exponent digits ("1E" and "1E+" are 1). A literal too
// large for float64 converts to infinity, which the interpreter reports as
// OVERFLOW.
func numberValue(text string) float64 {
	if strings.HasPrefix(text, ".") {
		text = "0" + text
	}
	if i := strings.IndexByte(text, 'E'); i >= 0 && strings.TrimLeft(text[i+1:], "+-") == "" {
		text += "0"
	}
	v, _ := strconv.ParseFloat(text, 64) // range errors still return ±Inf or 0
	return v
}
