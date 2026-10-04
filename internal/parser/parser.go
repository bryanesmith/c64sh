// Package parser turns the tokens of a line into an AST, with one parse
// function per grammar rule. Each function carries its rule in EBNF, in the
// notation of the Go language specification; lowercase names are token
// rules, documented in package lexer.
package parser

import (
	"math"
	"strconv"
	"strings"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
	"github.com/bryanesmith/c64sh/internal/lexer"
	"github.com/bryanesmith/c64sh/internal/token"
)

// Parse parses the tokens of one line. It always returns a non-nil Line.
// If err is non-nil, it is a SYNTAX error, and the Line holds the
// statements completed before the error, followed by either that PRINT
// statement ending in a BadItem (an error inside PRINT's items) or a
// BadStmt holding the error.
//
// @spec PARSER-001, PARSER-002, PARSER-003, PARSER-004, PARSER-005, PARSER-006
// @spec PARSER-007, PARSER-008, PARSER-009, PARSER-010, PARSER-011, PARSER-012, PARSER-013
// @spec PARSER-014, PARSER-015, PARSER-016, PARSER-017, PARSER-018, PARSER-019, PARSER-020
// @spec PARSER-021, PARSER-022, PARSER-023, PARSER-024, PARSER-025, PARSER-026, PARSER-027
// @spec PARSER-028, PARSER-029, PARSER-030, PARSER-031, PARSER-032, PARSER-033, PARSER-034
// @spec PARSER-035, PARSER-036, PARSER-037, PARSER-038, PARSER-039, PARSER-040, PARSER-041
// @spec PARSER-042, PARSER-043, PARSER-044, PARSER-045, PARSER-046, PARSER-047
// @spec PARSER-048, PARSER-049, PARSER-050, PARSER-051, PARSER-052, PARSER-053, PARSER-054
// @spec PARSER-055, PARSER-056
// @spec PARSER-057, PARSER-058, PARSER-059, PARSER-060, PARSER-061, PARSER-062, PARSER-063, PARSER-064, PARSER-065, PARSER-066, PARSER-067, PARSER-068, PARSER-069, PARSER-070, PARSER-071, PARSER-072, PARSER-073, PARSER-074, PARSER-075, PARSER-076, PARSER-077
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
//
// The statement after an IF's THEN follows it with no ":". A syntax error
// is placed in the tree where it occurs: as a PRINT's BadItem, or as a
// BadStmt ending the line, so that it is reported only if execution
// reaches it.
func (p *parser) parseLine() (*ast.Line, error) {
	line := &ast.Line{}
	for {
		stmt, err := p.parseStatement()
		if err != nil {
			if stmt == nil {
				stmt = &ast.BadStmt{Err: err}
			}
			line.Statements = append(line.Statements, stmt)
			return line, err
		}
		if stmt != nil {
			line.Statements = append(line.Statements, stmt)
		}
		if _, isIf := stmt.(*ast.IfStmt); isIf {
			// IF … THEN n is IF … THEN GOTO n ($A940).
			if t := p.tokens[p.pos]; t.Kind == token.Number && isDigit(t.Value[0]) {
				n, err := p.parseLineNumber()
				if err != nil {
					line.Statements = append(line.Statements, &ast.BadStmt{Err: err})
					return line, err
				}
				line.Statements = append(line.Statements, &ast.GotoStmt{Line: n})
				if !p.accept(token.Colon) {
					break
				}
			}
			continue // the statement after THEN
		}
		if !p.accept(token.Colon) {
			break
		}
	}
	if p.peek() != token.EOL {
		err := syntaxError()
		line.Statements = append(line.Statements, &ast.BadStmt{Err: err})
		return line, err
	}
	return line, nil
}

// Statement = [ PrintStatement | RemStatement | LetStatement | IfStatement | RunStatement | GotoStatement | ForStatement | NextStatement | GosubStatement | ReturnStatement | InputStatement | GetStatement | DefStatement | LoadStatement | SaveStatement | VerifyStatement | OnStatement | DimStatement | DataStatement | ReadStatement | RestoreStatement | OpenStatement | CloseStatement | PrintFileStatement | CmdStatement | InputFileStatement | EnvironStatement | ListStatement | NewStatement | EndStatement ] .
//
// An empty statement returns a nil Stmt.
func (p *parser) parseStatement() (ast.Stmt, error) {
	switch p.peek() {
	case token.Print:
		return p.parsePrintStatement()
	case token.Rem:
		return p.parseRemStatement()
	case token.If:
		stmt, err := p.parseIfStatement()
		if err != nil {
			return nil, err
		}
		return stmt, nil
	case token.Run:
		stmt, err := p.parseRunStatement()
		if err != nil {
			return nil, err
		}
		return stmt, nil
	case token.Goto, token.Go:
		stmt, err := p.parseGotoStatement()
		if err != nil {
			return nil, err
		}
		return stmt, nil
	case token.For:
		stmt, err := p.parseForStatement()
		if err != nil {
			return nil, err
		}
		return stmt, nil
	case token.Next:
		stmt, err := p.parseNextStatement()
		if err != nil {
			return nil, err
		}
		return stmt, nil
	case token.Gosub:
		stmt, err := p.parseGosubStatement()
		if err != nil {
			return nil, err
		}
		return stmt, nil
	case token.Return:
		return p.parseCommand(&ast.ReturnStmt{})
	case token.Input:
		stmt, err := p.parseInputStatement()
		if err != nil {
			return nil, err
		}
		return stmt, nil
	case token.Load, token.Save, token.Verify:
		return p.parseFileStatement()
	case token.On:
		return p.parseOnStatement()
	case token.Dim:
		return p.parseDimStatement()
	case token.Data:
		return &ast.DataStmt{Text: p.next().Value}, nil
	case token.Read:
		p.next()
		vars, err := p.parseVariableList()
		if err != nil {
			return nil, err
		}
		return &ast.ReadStmt{Vars: vars}, nil
	case token.Restore:
		return p.parseCommand(&ast.RestoreStmt{})
	case token.Open:
		return p.parseOpenStatement()
	case token.Close:
		return p.parseCloseStatement()
	case token.PrintFile, token.Cmd:
		return p.parseFilePrint()
	case token.InputFile:
		return p.parseInputFileStatement()
	case token.Environ:
		return p.parseEnvironStatement()
	case token.Def:
		stmt, err := p.parseDefStatement()
		if err != nil {
			return nil, err
		}
		return stmt, nil
	case token.Get:
		stmt, err := p.parseGetStatement()
		if err != nil {
			return nil, err
		}
		return stmt, nil
	case token.List:
		return p.parseCommand(&ast.ListStmt{})
	case token.New:
		return p.parseCommand(&ast.NewStmt{})
	case token.End:
		return p.parseCommand(&ast.EndStmt{})
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

// RunStatement = run [ LineNumber ] .
//
// Anything after RUN other than the end of the statement makes it RUN n
// ($A871): RUN A is RUN 0, as on a C64.
func (p *parser) parseRunStatement() (*ast.RunStmt, error) {
	p.next() // RUN
	if k := p.peek(); k == token.Colon || k == token.EOL {
		return &ast.RunStmt{}, nil
	}
	n, err := p.parseLineNumber()
	if err != nil {
		return nil, err
	}
	return &ast.RunStmt{Line: n, HasLine: true}, nil
}

// ForStatement = for Variable "=" Expression to Expression [ step Expression ] .
//
// The variable cannot be an integer variable, as on a C64.
func (p *parser) parseForStatement() (*ast.ForStmt, error) {
	p.next() // FOR
	if p.peek() != token.Name {
		return nil, syntaxError()
	}
	v, err := p.parseAssignable()
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(v.Name, "%") || len(v.Subs) > 0 || !p.accept(token.Equal) {
		return nil, syntaxError()
	}
	s := &ast.ForStmt{Var: v}
	if s.From, err = p.parseExpression(); err != nil {
		return nil, err
	}
	if !p.accept(token.To) {
		return nil, syntaxError()
	}
	if s.To, err = p.parseExpression(); err != nil {
		return nil, err
	}
	if p.accept(token.Step) {
		if s.Step, err = p.parseExpression(); err != nil {
			return nil, err
		}
	}
	if k := p.peek(); k != token.Colon && k != token.EOL {
		return nil, syntaxError()
	}
	return s, nil
}

// NextStatement = next [ Variable { "," Variable } ] .
func (p *parser) parseNextStatement() (*ast.NextStmt, error) {
	p.next() // NEXT
	s := &ast.NextStmt{}
	if k := p.peek(); k == token.Colon || k == token.EOL {
		return s, nil
	}
	for {
		if p.peek() != token.Name {
			return nil, syntaxError()
		}
		v, err := p.parseVariable()
		if err != nil {
			return nil, err
		}
		if len(v.Subs) > 0 {
			return nil, syntaxError()
		}
		s.Vars = append(s.Vars, v)
		if !p.accept(token.Comma) {
			break
		}
	}
	if k := p.peek(); k != token.Colon && k != token.EOL {
		return nil, syntaxError()
	}
	return s, nil
}

// InputStatement = input [ string ";" ] Variable { "," Variable } .
//
// The prompt must be a string literal followed by ";" ($ABBF).
func (p *parser) parseInputStatement() (*ast.InputStmt, error) {
	p.next() // INPUT
	s := &ast.InputStmt{}
	if p.peek() == token.String {
		s.Prompt, s.HasPrompt = p.next().Value, true
		if !p.accept(token.Semicolon) {
			return nil, syntaxError()
		}
	}
	vars, err := p.parseVariableList()
	if err != nil {
		return nil, err
	}
	s.Vars = vars
	return s, nil
}

// GetStatement = get [ "#" Expression "," ] Variable { "," Variable } .
func (p *parser) parseGetStatement() (*ast.GetStmt, error) {
	p.next() // GET
	s := &ast.GetStmt{}
	if p.accept(token.Hash) {
		file, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if !p.accept(token.Comma) {
			return nil, syntaxError()
		}
		s.File = file
	}
	vars, err := p.parseVariableList()
	if err != nil {
		return nil, err
	}
	s.Vars = vars
	return s, nil
}

// parseVariableList parses Variable { "," Variable }, which must end the
// statement.
func (p *parser) parseVariableList() ([]*ast.VarRef, error) {
	var vars []*ast.VarRef
	for {
		if p.peek() != token.Name {
			return nil, syntaxError()
		}
		v, err := p.parseAssignable()
		if err != nil {
			return nil, err
		}
		vars = append(vars, v)
		if !p.accept(token.Comma) {
			break
		}
	}
	if k := p.peek(); k != token.Colon && k != token.EOL {
		return nil, syntaxError()
	}
	return vars, nil
}

// LoadStatement   = load FileArgs .
// SaveStatement   = save FileArgs .
// VerifyStatement = verify FileArgs .
func (p *parser) parseFileStatement() (ast.Stmt, error) {
	kind := p.next().Kind
	args, err := p.parseFileArgs()
	if err != nil {
		return nil, err
	}
	switch kind {
	case token.Load:
		return &ast.LoadStmt{FileArgs: args}, nil
	case token.Save:
		return &ast.SaveStmt{FileArgs: args}, nil
	default:
		return &ast.VerifyStmt{FileArgs: args}, nil
	}
}

// FileArgs = [ Expression [ "," Expression [ "," Expression ] ] ] .
//
// The arguments must end the statement.
func (p *parser) parseFileArgs() (ast.FileArgs, error) {
	var a ast.FileArgs
	slots := []*ast.Expr{&a.Name, &a.Device, &a.Secondary}
	if k := p.peek(); k != token.Colon && k != token.EOL {
		for i, slot := range slots {
			e, err := p.parseExpression()
			if err != nil {
				return a, err
			}
			*slot = e
			if i == len(slots)-1 || !p.accept(token.Comma) {
				break
			}
		}
	}
	if k := p.peek(); k != token.Colon && k != token.EOL {
		return a, syntaxError()
	}
	return a, nil
}

// DefStatement = def fn FunctionName "(" FunctionName ")" "=" Expression .
//
// A C64 skips the body when DEF runs and checks it only when the function
// is called ($B3DB, $B441), so an error in the body is kept in BodyErr,
// and parsing continues after the statement.
func (p *parser) parseDefStatement() (*ast.DefStmt, error) {
	p.next() // DEF
	if !p.accept(token.Fn) {
		return nil, syntaxError()
	}
	name, err := p.parseFunctionName()
	if err != nil {
		return nil, err
	}
	if !p.accept(token.LParen) {
		return nil, syntaxError()
	}
	param, err := p.parseFunctionName()
	if err != nil {
		return nil, err
	}
	if !p.accept(token.RParen) || !p.accept(token.Equal) {
		return nil, syntaxError()
	}
	s := &ast.DefStmt{Name: name, Param: param}
	body, err := p.parseExpression()
	if k := p.peek(); err == nil && (k == token.Colon || k == token.EOL) {
		s.Body = body
		return s, nil
	}
	s.BodyErr = syntaxError()
	for k := p.peek(); k != token.Colon && k != token.EOL; k = p.peek() {
		p.next()
	}
	return s, nil
}

// FunctionName = name .
//
// The name of a user-defined function or its parameter, which, unlike a
// variable, may be followed by "(", and cannot be an integer name.
func (p *parser) parseFunctionName() (*ast.VarRef, error) {
	if p.peek() != token.Name {
		return nil, syntaxError()
	}
	text := p.next().Value
	v := &ast.VarRef{Name: variableIdentity(text), Text: text}
	if strings.HasSuffix(v.Name, "%") {
		return nil, syntaxError()
	}
	return v, nil
}

// GosubStatement = gosub LineNumber .
func (p *parser) parseGosubStatement() (*ast.GosubStmt, error) {
	p.next() // GOSUB
	n, err := p.parseLineNumber()
	if err != nil {
		return nil, err
	}
	return &ast.GosubStmt{Line: n}, nil
}

// GotoStatement = ( goto | go to ) LineNumber .
//
// GO must be followed by TO ($A80E).
func (p *parser) parseGotoStatement() (*ast.GotoStmt, error) {
	if p.next().Kind == token.Go && !p.accept(token.To) {
		return nil, syntaxError()
	}
	n, err := p.parseLineNumber()
	if err != nil {
		return nil, err
	}
	return &ast.GotoStmt{Line: n}, nil
}

// LineNumber = [ number ] .
//
// A line number is read from the number's text as the ROM reads one
// ($A96B, $A8A0): GOTO 20.5 is GOTO 20, and with no digits, or no number
// at all, it is 0, so GOTO A is GOTO 0. Whatever follows it is never
// checked, because a jump does not return to its line.
func (p *parser) parseLineNumber() (int, error) {
	if p.peek() != token.Number {
		return 0, nil
	}
	n, _, _, err := lexer.LineNumber(p.next().Value)
	return n, err
}

func isDigit(c byte) bool { return '0' <= c && c <= '9' }

// ReturnStatement = return .
// ListStatement   = list .
// NewStatement    = new .
// EndStatement    = end .
//
// parseCommand parses a command that takes no arguments. Anything after
// it other than the end of the statement is a syntax error in its place,
// so the command does not run, as on a C64 ($A642, $A69C, $A831).
func (p *parser) parseCommand(stmt ast.Stmt) (ast.Stmt, error) {
	p.next()
	if k := p.peek(); k != token.Colon && k != token.EOL {
		return nil, syntaxError()
	}
	return stmt, nil
}

// EnvironStatement = environ Expression .
//
// ENVIRON is a c64sh extension, in GW-BASIC's form.
func (p *parser) parseEnvironStatement() (ast.Stmt, error) {
	p.next()
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if k := p.peek(); k != token.Colon && k != token.EOL {
		return nil, syntaxError()
	}
	return &ast.EnvironStmt{Value: value}, nil
}

// IfStatement = if Expression ( then [ LineNumber ] | /* goto, parsed as the next statement */ ) .
//
// The IfStmt guards the rest of the line; the statement after THEN, or
// the GotoStmt for THEN's line number, is parsed by parseLine as the
// line's next statement.
func (p *parser) parseIfStatement() (*ast.IfStmt, error) {
	p.next() // if
	cond, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	// IF … GOTO n leaves the GOTO to be parsed as the next statement, as
	// the ROM leaves it to be executed ($A92E).
	if p.peek() != token.Goto && !p.accept(token.Then) {
		return nil, syntaxError()
	}
	return &ast.IfStmt{Cond: cond}, nil
}

// LetStatement = [ let ] Variable "=" Expression .
func (p *parser) parseLetStatement() (*ast.LetStmt, error) {
	p.accept(token.Let)
	if p.peek() != token.Name {
		return nil, syntaxError()
	}
	v, err := p.parseAssignable()
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

// Variable = name [ "(" Expression { "," Expression } ")" ] .
//
// A name followed by "(" is an array element.
func (p *parser) parseVariable() (*ast.VarRef, error) {
	text := p.next().Value
	v := &ast.VarRef{Name: variableIdentity(text), Text: text}
	if p.accept(token.LParen) {
		for {
			sub, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			v.Subs = append(v.Subs, sub)
			if !p.accept(token.Comma) {
				break
			}
		}
		if !p.accept(token.RParen) {
			return nil, syntaxError()
		}
	}
	return v, nil
}

// DimStatement = dim Variable { "," Variable } .
func (p *parser) parseDimStatement() (ast.Stmt, error) {
	p.next() // DIM
	s := &ast.DimStmt{}
	for {
		if p.peek() != token.Name {
			return nil, syntaxError()
		}
		v, err := p.parseVariable()
		if err != nil {
			return nil, err
		}
		s.Arrays = append(s.Arrays, v)
		if !p.accept(token.Comma) {
			break
		}
	}
	if k := p.peek(); k != token.Colon && k != token.EOL {
		return nil, syntaxError()
	}
	return s, nil
}

// parseAssignable parses a variable that is assigned to, which cannot be
// ST or TI, the read-only I/O status and clock.
func (p *parser) parseAssignable() (*ast.VarRef, error) {
	v, err := p.parseVariable()
	if err == nil && (v.Name == "ST" || v.Name == "TI") {
		return nil, syntaxError()
	}
	return v, err
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
	var err error
	stmt.Items, err = p.parsePrintItems()
	return stmt, err
}

// parsePrintItems parses print items up to the end of the statement. If an
// item fails, the items end with a BadItem holding the error.
func (p *parser) parsePrintItems() ([]ast.PrintItem, error) {
	var items []ast.PrintItem
	for k := p.peek(); k != token.Colon && k != token.EOL; k = p.peek() {
		item, err := p.parsePrintItem()
		if err != nil {
			return append(items, &ast.BadItem{Err: err}), err
		}
		items = append(items, item)
	}
	return items, nil
}

// PrintFileStatement = printfile Expression [ "," { PrintItem } ] .
// CmdStatement       = cmd Expression [ "," { PrintItem } ] .
//
// Without a comma after the file number, the statement must end ($AA86).
func (p *parser) parseFilePrint() (ast.Stmt, error) {
	kind := p.next().Kind
	file, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	var items []ast.PrintItem
	if p.accept(token.Comma) {
		items, err = p.parsePrintItems()
	} else if k := p.peek(); k != token.Colon && k != token.EOL {
		return nil, syntaxError()
	}
	if kind == token.Cmd {
		return &ast.CmdStmt{File: file, Items: items}, err
	}
	return &ast.PrintStmt{File: file, Items: items}, err
}

// OnStatement = on Expression ( goto | gosub ) number { "," number } .
//
// Only the GOTO and GOSUB keywords are accepted ($A94F).
func (p *parser) parseOnStatement() (ast.Stmt, error) {
	p.next() // ON
	index, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	s := &ast.OnStmt{Index: index}
	switch p.next().Kind {
	case token.Goto:
	case token.Gosub:
		s.Gosub = true
	default:
		return nil, syntaxError()
	}
	for {
		if p.peek() != token.Number {
			return nil, syntaxError()
		}
		n, err := p.parseLineNumber()
		if err != nil {
			return nil, err
		}
		s.Lines = append(s.Lines, n)
		if !p.accept(token.Comma) {
			break
		}
	}
	if k := p.peek(); k != token.Colon && k != token.EOL {
		return nil, syntaxError()
	}
	return s, nil
}

// OpenStatement = open Expression [ "," Expression [ "," Expression [ "," Expression ] ] ] .
func (p *parser) parseOpenStatement() (ast.Stmt, error) {
	p.next() // OPEN
	s := &ast.OpenStmt{}
	for i, slot := range []*ast.Expr{&s.File, &s.Device, &s.Secondary, &s.Name} {
		e, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		*slot = e
		if i == 3 || !p.accept(token.Comma) {
			break
		}
	}
	if k := p.peek(); k != token.Colon && k != token.EOL {
		return nil, syntaxError()
	}
	return s, nil
}

// CloseStatement = close Expression .
func (p *parser) parseCloseStatement() (ast.Stmt, error) {
	p.next() // CLOSE
	file, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if k := p.peek(); k != token.Colon && k != token.EOL {
		return nil, syntaxError()
	}
	return &ast.CloseStmt{File: file}, nil
}

// InputFileStatement = inputfile Expression "," Variable { "," Variable } .
func (p *parser) parseInputFileStatement() (ast.Stmt, error) {
	p.next() // INPUT#
	file, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if !p.accept(token.Comma) {
		return nil, syntaxError()
	}
	vars, err := p.parseVariableList()
	if err != nil {
		return nil, err
	}
	return &ast.InputStmt{File: file, Vars: vars}, nil
}

// PrintItem = Expression | ";" | "," | tab Expression ")" | spc Expression ")" .
func (p *parser) parsePrintItem() (ast.PrintItem, error) {
	switch p.peek() {
	case token.Tab, token.Spc:
		kind := p.next().Kind
		x, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if !p.accept(token.RParen) {
			return nil, syntaxError()
		}
		if kind == token.Tab {
			return &ast.TabItem{X: x}, nil
		}
		return &ast.SpcItem{X: x}, nil
	case token.Semicolon:
		p.next()
		return &ast.Semicolon{}, nil
	case token.Comma:
		p.next()
		return &ast.Comma{}, nil
	case token.String, token.Number, token.Name, token.Not, token.Fn, token.Function, token.Pi, token.Minus, token.Plus, token.LParen:
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		return &ast.ExprItem{Expr: expr}, nil
	default:
		return nil, syntaxError()
	}
}

// Expression = Conjunction { or Conjunction } .
func (p *parser) parseExpression() (ast.Expr, error) {
	left, err := p.parseConjunction()
	if err != nil {
		return nil, err
	}
	for p.accept(token.Or) {
		right, err := p.parseConjunction()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Op: ast.Or, Left: left, Right: right}
	}
	return left, nil
}

// Conjunction = Comparison { and Comparison } .
func (p *parser) parseConjunction() (ast.Expr, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}
	for p.accept(token.And) {
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Op: ast.And, Left: left, Right: right}
	}
	return left, nil
}

// Comparison = Sum { Relation Sum } .
func (p *parser) parseComparison() (ast.Expr, error) {
	left, err := p.parseSum()
	if err != nil {
		return nil, err
	}
	for isRelationToken(p.peek()) {
		rel, err := p.parseRelation()
		if err != nil {
			return nil, err
		}
		right, err := p.parseSum()
		if err != nil {
			return nil, err
		}
		left = &ast.CompareExpr{Rel: rel, Left: left, Right: right}
	}
	return left, nil
}

func isRelationToken(k token.Kind) bool {
	return k == token.Less || k == token.Equal || k == token.Greater
}

// Relation = ( "<" | "=" | ">" ) { "<" | "=" | ">" } .
//
// As the C64 ROM does ($ADB8), each symbol adds its relation to the set, in
// any order; a symbol repeated within one operator is a SYNTAX error.
func (p *parser) parseRelation() (ast.Relation, error) {
	var rel ast.Relation
	for isRelationToken(p.peek()) {
		var r ast.Relation
		switch p.next().Kind {
		case token.Less:
			r = ast.RelLess
		case token.Equal:
			r = ast.RelEqual
		default:
			r = ast.RelGreater
		}
		if rel&r != 0 {
			return 0, syntaxError()
		}
		rel |= r
	}
	return rel, nil
}

// Sum = Term { ( "+" | "-" ) Term } .
func (p *parser) parseSum() (ast.Expr, error) {
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

// Operand = string | number | pi | Variable | "(" Expression ")" | not Comparison | fn FunctionName "(" Expression ")" | Call .
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
	case token.Pi:
		p.next()
		return &ast.NumberLit{Value: math.Pi}, nil
	case token.Function:
		return p.parseCall()
	case token.Fn:
		p.next()
		name, err := p.parseFunctionName()
		if err != nil {
			return nil, err
		}
		if !p.accept(token.LParen) {
			return nil, syntaxError()
		}
		arg, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if !p.accept(token.RParen) {
			return nil, syntaxError()
		}
		return &ast.FnExpr{Name: name, Arg: arg}, nil
	case token.Not:
		// NOT takes in everything up to the next AND or OR, as on a C64.
		p.next()
		x, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		return &ast.NotExpr{X: x}, nil
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

// arity is the least and most arguments each built-in function takes.
var arity = map[string][2]int{
	"ABS": {1, 1}, "INT": {1, 1}, "SGN": {1, 1}, "SQR": {1, 1}, "RND": {1, 1}, "LOG": {1, 1},
	"EXP": {1, 1}, "SIN": {1, 1}, "COS": {1, 1}, "TAN": {1, 1}, "ATN": {1, 1},
	"LEN": {1, 1}, "CHR$": {1, 1}, "ASC": {1, 1}, "STR$": {1, 1}, "VAL": {1, 1},
	"LEFT$": {2, 2}, "RIGHT$": {2, 2}, "MID$": {2, 3}, "POS": {1, 1},
	"ENVIRON$": {1, 1},
}

// Call = function "(" Expression { "," Expression } ")" .
//
// The function's number of arguments is checked here; the parentheses are
// required, as on a C64 ($AEF1).
func (p *parser) parseCall() (ast.Expr, error) {
	name := p.next().Value
	if !p.accept(token.LParen) {
		return nil, syntaxError()
	}
	call := &ast.CallExpr{Name: name}
	for {
		arg, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		call.Args = append(call.Args, arg)
		if !p.accept(token.Comma) {
			break
		}
	}
	if n := len(call.Args); !p.accept(token.RParen) || n < arity[name][0] || n > arity[name][1] {
		return nil, syntaxError()
	}
	return call, nil
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
