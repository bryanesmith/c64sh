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

// Statement = [ PrintStatement | RemStatement ] .
//
// An empty statement returns a nil Stmt.
func (p *parser) parseStatement() (ast.Stmt, error) {
	switch p.peek() {
	case token.Print:
		return p.parsePrintStatement()
	case token.Rem:
		return p.parseRemStatement()
	default:
		return nil, nil
	}
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
	case token.String, token.Number:
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		return &ast.ExprItem{Expr: expr}, nil
	default:
		return nil, syntaxError()
	}
}

// Expression = Operand { "+" Operand } .
func (p *parser) parseExpression() (ast.Expr, error) {
	left, err := p.parseOperand()
	if err != nil {
		return nil, err
	}
	for p.accept(token.Plus) {
		right, err := p.parseOperand()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Op: ast.Add, Left: left, Right: right}
	}
	return left, nil
}

// Operand = string | number .
func (p *parser) parseOperand() (ast.Expr, error) {
	switch p.peek() {
	case token.String:
		return &ast.StringLit{Value: p.next().Value}, nil
	case token.Number:
		return &ast.NumberLit{Value: numberValue(p.next().Value)}, nil
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
