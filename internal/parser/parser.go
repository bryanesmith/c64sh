// Package parser turns the tokens of a line into an AST, with one parse
// function per syntactic rule of grammar/c64basic.ebnf.
package parser

import (
	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
	"github.com/bryanesmith/c64sh/internal/token"
)

// grammarRules maps each syntactic grammar rule name to its parse function.
//
// @spec GRAMMAR-008
var grammarRules = map[string]any{
	"Line":           (*parser).parseLine,
	"Statement":      (*parser).parseStatement,
	"PrintStatement": (*parser).parsePrintStatement,
	"PrintItem":      (*parser).parsePrintItem,
	"RemStatement":   (*parser).parseRemStatement,
	"Expression":     (*parser).parseExpression,
}

// Parse parses the tokens of one line. It always returns a non-nil Line.
// If err is non-nil, it is a SYNTAX error, and the Line holds the
// statements completed before the error, followed, when the error is
// inside a PRINT statement's items, by that statement ending in a BadItem.
//
// @spec PARSER-001, PARSER-002, PARSER-003, PARSER-004, PARSER-005, PARSER-006
// @spec PARSER-007, PARSER-008, PARSER-009, PARSER-010, PARSER-011, PARSER-012
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
	case token.String:
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		return &ast.ExprItem{Expr: expr}, nil
	default:
		return nil, syntaxError()
	}
}

// Expression = string { "+" string } .
func (p *parser) parseExpression() (ast.Expr, error) {
	left, err := p.parseString()
	if err != nil {
		return nil, err
	}
	for p.accept(token.Plus) {
		right, err := p.parseString()
		if err != nil {
			return nil, err
		}
		left = &ast.Concat{Left: left, Right: right}
	}
	return left, nil
}

// parseString reads a string token (the grammar's lexical rule string).
func (p *parser) parseString() (ast.Expr, error) {
	if p.peek() != token.String {
		return nil, syntaxError()
	}
	return &ast.StringLit{Value: p.next().Value}, nil
}
