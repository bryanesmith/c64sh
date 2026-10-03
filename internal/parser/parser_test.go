package parser

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
	"github.com/bryanesmith/c64sh/internal/token"
)

// Token helpers. The parser ignores Pos, so tests leave it zero.
var (
	pr    = token.Token{Kind: token.Print, Value: "PRINT"}
	colon = token.Token{Kind: token.Colon, Value: ":"}
	semi  = token.Token{Kind: token.Semicolon, Value: ";"}
	comma = token.Token{Kind: token.Comma, Value: ","}
	plus  = token.Token{Kind: token.Plus, Value: "+"}
	minus = token.Token{Kind: token.Minus, Value: "-"}
	star  = token.Token{Kind: token.Star, Value: "*"}
	slash = token.Token{Kind: token.Slash, Value: "/"}
	lp    = token.Token{Kind: token.LParen, Value: "("}
	rp    = token.Token{Kind: token.RParen, Value: ")"}
	caret = token.Token{Kind: token.Caret, Value: "^"}
	let   = token.Token{Kind: token.Let, Value: "LET"}
	eq    = token.Token{Kind: token.Equal, Value: "="}
	lt    = token.Token{Kind: token.Less, Value: "<"}
	gt    = token.Token{Kind: token.Greater, Value: ">"}
	and   = token.Token{Kind: token.And, Value: "AND"}
	or    = token.Token{Kind: token.Or, Value: "OR"}
	not   = token.Token{Kind: token.Not, Value: "NOT"}
	iff   = token.Token{Kind: token.If, Value: "IF"}
	then  = token.Token{Kind: token.Then, Value: "THEN"}
	run   = token.Token{Kind: token.Run, Value: "RUN"}
	list  = token.Token{Kind: token.List, Value: "LIST"}
	nw    = token.Token{Kind: token.New, Value: "NEW"}
	end   = token.Token{Kind: token.End, Value: "END"}
	gotok = token.Token{Kind: token.Goto, Value: "GOTO"}
	gok   = token.Token{Kind: token.Go, Value: "GO"}
	tok   = token.Token{Kind: token.To, Value: "TO"}
	fork  = token.Token{Kind: token.For, Value: "FOR"}
	nextk = token.Token{Kind: token.Next, Value: "NEXT"}
	stepk = token.Token{Kind: token.Step, Value: "STEP"}
	gosub = token.Token{Kind: token.Gosub, Value: "GOSUB"}
	retk  = token.Token{Kind: token.Return, Value: "RETURN"}
	input = token.Token{Kind: token.Input, Value: "INPUT"}
	getk  = token.Token{Kind: token.Get, Value: "GET"}
	def   = token.Token{Kind: token.Def, Value: "DEF"}
	fn    = token.Token{Kind: token.Fn, Value: "FN"}
	load  = token.Token{Kind: token.Load, Value: "LOAD"}
	save  = token.Token{Kind: token.Save, Value: "SAVE"}
	verif = token.Token{Kind: token.Verify, Value: "VERIFY"}
	prf   = token.Token{Kind: token.PrintFile, Value: "PRINT#"}
	inf   = token.Token{Kind: token.InputFile, Value: "INPUT#"}
	openk = token.Token{Kind: token.Open, Value: "OPEN"}
	close = token.Token{Kind: token.Close, Value: "CLOSE"}
	cmdk  = token.Token{Kind: token.Cmd, Value: "CMD"}
	hash  = token.Token{Kind: token.Hash, Value: "#"}
	onk   = token.Token{Kind: token.On, Value: "ON"}
	dim   = token.Token{Kind: token.Dim, Value: "DIM"}
	eol   = token.Token{Kind: token.EOL}
)

func str(v string) token.Token    { return token.Token{Kind: token.String, Value: v} }
func ill(v string) token.Token    { return token.Token{Kind: token.Illegal, Value: v} }
func rem(v string) token.Token    { return token.Token{Kind: token.Rem, Value: v} }
func number(v string) token.Token { return token.Token{Kind: token.Number, Value: v} }
func name(v string) token.Token   { return token.Token{Kind: token.Name, Value: v} }

// toks returns ts followed by EOL.
func toks(ts ...token.Token) []token.Token { return append(ts, eol) }

// isSyntax reports whether err is a SYNTAX BASIC error.
func isSyntax(err error) bool {
	var be *basicerr.Error
	return errors.As(err, &be) && be.Kind == basicerr.Syntax
}

// dump renders a Line compactly: statements joined by " : ", PRINT items in
// brackets, string literals quoted, numbers as #value, BinaryExpr Add as
// (l+r), BadItem as BAD(SYNTAX),
// RemStmt as REM(text).
func dump(l *ast.Line) string {
	var stmts []string
	for _, s := range l.Statements {
		stmts = append(stmts, dumpStmt(s))
	}
	return strings.Join(stmts, " : ")
}

func dumpStmt(s ast.Stmt) string {
	switch s := s.(type) {
	case *ast.PrintStmt:
		prefix := "PRINT["
		if s.File != nil {
			prefix = "PRINT#" + dumpExpr(s.File) + "["
		}
		return prefix + dumpItems(s.Items) + "]"
	case *ast.CmdStmt:
		return "CMD " + dumpExpr(s.File) + "[" + dumpItems(s.Items) + "]"
	case *ast.OpenStmt:
		out := "OPEN"
		for _, e := range []ast.Expr{s.File, s.Device, s.Secondary, s.Name} {
			if e != nil {
				out += " " + dumpExpr(e)
			}
		}
		return out
	case *ast.CloseStmt:
		return "CLOSE " + dumpExpr(s.File)
	case *ast.OnStmt:
		kw := "GOTO"
		if s.Gosub {
			kw = "GOSUB"
		}
		return fmt.Sprintf("ON %s %s %v", dumpExpr(s.Index), kw, s.Lines)
	case *ast.RemStmt:
		return "REM(" + strconv.Quote(s.Text) + ")"
	case *ast.LetStmt:
		return "LET " + dumpExpr(s.Var) + "=" + dumpExpr(s.Value)
	case *ast.IfStmt:
		return "IF " + dumpExpr(s.Cond)
	case *ast.RunStmt:
		if s.HasLine {
			return fmt.Sprintf("RUN %d", s.Line)
		}
		return "RUN"
	case *ast.GotoStmt:
		return fmt.Sprintf("GOTO %d", s.Line)
	case *ast.ForStmt:
		out := "FOR " + dumpExpr(s.Var) + "=" + dumpExpr(s.From) + " TO " + dumpExpr(s.To)
		if s.Step != nil {
			out += " STEP " + dumpExpr(s.Step)
		}
		return out
	case *ast.NextStmt:
		var vars []string
		for _, v := range s.Vars {
			vars = append(vars, dumpExpr(v))
		}
		return strings.TrimSpace("NEXT " + strings.Join(vars, ","))
	case *ast.GosubStmt:
		return fmt.Sprintf("GOSUB %d", s.Line)
	case *ast.ReturnStmt:
		return "RETURN"
	case *ast.InputStmt:
		out := "INPUT "
		if s.File != nil {
			out = "INPUT#" + dumpExpr(s.File) + " "
		}
		if s.HasPrompt {
			out += strconv.Quote(s.Prompt) + ";"
		}
		return out + dumpVars(s.Vars)
	case *ast.GetStmt:
		if s.File != nil {
			return "GET#" + dumpExpr(s.File) + " " + dumpVars(s.Vars)
		}
		return "GET " + dumpVars(s.Vars)
	case *ast.DefStmt:
		out := "DEF FN " + dumpExpr(s.Name) + "(" + dumpExpr(s.Param) + ")="
		if s.BodyErr != nil {
			return out + "<" + s.BodyErr.Error() + ">"
		}
		return out + dumpExpr(s.Body)
	case *ast.LoadStmt:
		return "LOAD" + dumpFileArgs(s.FileArgs)
	case *ast.SaveStmt:
		return "SAVE" + dumpFileArgs(s.FileArgs)
	case *ast.VerifyStmt:
		return "VERIFY" + dumpFileArgs(s.FileArgs)
	case *ast.DimStmt:
		return "DIM " + dumpVars(s.Arrays)
	case *ast.ListStmt:
		return "LIST"
	case *ast.NewStmt:
		return "NEW"
	case *ast.EndStmt:
		return "END"
	case *ast.BadStmt:
		if isSyntax(s.Err) {
			return "BADSTMT"
		}
		return fmt.Sprintf("BADSTMT(%v)", s.Err)
	default:
		return fmt.Sprintf("<stmt %T>", s)
	}
}

func dumpItem(it ast.PrintItem) string {
	switch it := it.(type) {
	case *ast.ExprItem:
		return dumpExpr(it.Expr)
	case *ast.Semicolon:
		return ";"
	case *ast.Comma:
		return ","
	case *ast.TabItem:
		return "TAB(" + dumpExpr(it.X) + ")"
	case *ast.SpcItem:
		return "SPC(" + dumpExpr(it.X) + ")"
	case *ast.BadItem:
		if isSyntax(it.Err) {
			return "BAD(SYNTAX)"
		}
		return fmt.Sprintf("BAD(%v)", it.Err)
	default:
		return fmt.Sprintf("<item %T>", it)
	}
}

func dumpExpr(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.StringLit:
		return strconv.Quote(e.Value)
	case *ast.NumberLit:
		return "#" + strconv.FormatFloat(e.Value, 'g', -1, 64)
	case *ast.BinaryExpr:
		op, ok := map[ast.Op]string{ast.Add: "+", ast.Sub: "-", ast.Mul: "*", ast.Div: "/", ast.Pow: "^", ast.And: " AND ", ast.Or: " OR "}[e.Op]
		if !ok {
			return fmt.Sprintf("<op %d>", e.Op)
		}
		return "(" + dumpExpr(e.Left) + op + dumpExpr(e.Right) + ")"
	case *ast.NegExpr:
		return "(-" + dumpExpr(e.X) + ")"
	case *ast.NotExpr:
		return "(NOT " + dumpExpr(e.X) + ")"
	case *ast.VarRef:
		out := "$" + e.Name + "[" + e.Text + "]"
		if len(e.Subs) > 0 {
			var subs []string
			for _, x := range e.Subs {
				subs = append(subs, dumpExpr(x))
			}
			out += "(" + strings.Join(subs, ",") + ")"
		}
		return out
	case *ast.CallExpr:
		var args []string
		for _, a := range e.Args {
			args = append(args, dumpExpr(a))
		}
		return e.Name + "(" + strings.Join(args, ",") + ")"
	case *ast.FnExpr:
		return "FN " + dumpExpr(e.Name) + "(" + dumpExpr(e.Arg) + ")"
	case *ast.CompareExpr:
		rel := ""
		if e.Rel&ast.RelLess != 0 {
			rel += "<"
		}
		if e.Rel&ast.RelEqual != 0 {
			rel += "="
		}
		if e.Rel&ast.RelGreater != 0 {
			rel += ">"
		}
		return "(" + dumpExpr(e.Left) + " " + rel + " " + dumpExpr(e.Right) + ")"
	default:
		return fmt.Sprintf("<expr %T>", e)
	}
}

type parseCase struct {
	name    string
	tokens  []token.Token
	want    string // dump of the returned Line
	wantErr bool   // whether a SYNTAX error is expected
}

func runParseCases(t *testing.T, cases []parseCase) {
	t.Helper()
	for _, c := range cases {
		line, err := Parse(c.tokens)
		if line == nil {
			t.Errorf("%s: Parse returned a nil Line", c.name)
			continue
		}
		if got := dump(line); got != c.want {
			t.Errorf("%s: Parse(%v)\n got: %s\nwant: %s", c.name, c.tokens, got, c.want)
		}
		switch {
		case c.wantErr && !isSyntax(err):
			t.Errorf("%s: error = %v, want a SYNTAX error", c.name, err)
		case !c.wantErr && err != nil:
			t.Errorf("%s: unexpected error %v", c.name, err)
		}
	}
}

// @spec PARSER-001
func TestStatementsInSourceOrder(t *testing.T) {
	runParseCases(t, []parseCase{
		{"one statement", toks(pr, str("A")), `PRINT["A"]`, false},
		{"two statements", toks(pr, str("A"), colon, pr, str("B")), `PRINT["A"] : PRINT["B"]`, false},
	})
}

// @spec PARSER-002
func TestEmptyStatementsProduceNoNode(t *testing.T) {
	runParseCases(t, []parseCase{
		{"blank line", toks(), ``, false},
		{"only a colon", toks(colon), ``, false},
		{"consecutive colons", toks(pr, str("A"), colon, colon, pr, str("B")), `PRINT["A"] : PRINT["B"]`, false},
		{"leading and trailing colons", toks(colon, pr, str("A"), colon), `PRINT["A"]`, false},
	})
}

// @spec PARSER-003
func TestPrintItemsInSourceOrder(t *testing.T) {
	runParseCases(t, []parseCase{
		{"PRINT alone", toks(pr), `PRINT[]`, false},
		{"mixed items", toks(pr, str("A"), semi, str("B"), comma), `PRINT["A" ; "B" ,]`, false},
		{"leading comma", toks(pr, comma, str("A")), `PRINT[, "A"]`, false},
		{"separators only", toks(pr, semi, semi, comma), `PRINT[; ; ,]`, false},
		{"items stop at colon", toks(pr, str("A"), semi, colon, pr), `PRINT["A" ;] : PRINT[]`, false},
	})
}

// @spec PARSER-004
func TestAdjacentExpressionsAreSeparateItems(t *testing.T) {
	runParseCases(t, []parseCase{
		{"two strings, no separator", toks(pr, str("A"), str("B")), `PRINT["A" "B"]`, false},
	})
}

// @spec PARSER-005
func TestSingleStringLiteral(t *testing.T) {
	runParseCases(t, []parseCase{
		{"string value kept", toks(pr, str("HELLO, WORLD")), `PRINT["HELLO, WORLD"]`, false},
		{"empty string", toks(pr, str("")), `PRINT[""]`, false},
	})
}

// @spec PARSER-006
func TestConcatIsLeftAssociative(t *testing.T) {
	runParseCases(t, []parseCase{
		{"two strings", toks(pr, str("A"), plus, str("B")), `PRINT[("A"+"B")]`, false},
		{"three strings", toks(pr, str("A"), plus, str("B"), plus, str("C")), `PRINT[(("A"+"B")+"C")]`, false},
		{"concat then separator", toks(pr, str("A"), plus, str("B"), semi, str("C")), `PRINT[("A"+"B") ; "C"]`, false},
		{"numbers", toks(pr, number("1"), plus, number("2"), plus, number("3")), `PRINT[((#1+#2)+#3)]`, false},
		{"mixed kinds", toks(pr, str("A"), plus, number("1")), `PRINT[("A"+#1)]`, false},
	})
}

// @spec PARSER-012
func TestRemStatement(t *testing.T) {
	runParseCases(t, []parseCase{
		{"comment alone", toks(rem(" HELLO")), `REM(" HELLO")`, false},
		{"empty comment", toks(rem("")), `REM("")`, false},
		{"comment after a statement", toks(pr, str("A"), colon, rem(" X")), `PRINT["A"] : REM(" X")`, false},
		{"comment text kept exactly", toks(rem(` A:PRINT "X"`)), `REM(" A:PRINT \"X\"")`, false},
	})
}

// @spec PARSER-007, PARSER-011
func TestStatementStartingWithWrongTokenIsSyntaxError(t *testing.T) {
	runParseCases(t, []parseCase{
		{"string", toks(str("A")), `BADSTMT`, true},
		{"illegal", toks(ill("@")), `BADSTMT`, true},
		{"semicolon", toks(semi), `BADSTMT`, true},
		{"comma", toks(comma), `BADSTMT`, true},
		{"plus", toks(plus), `BADSTMT`, true},
		{"after a good statement", toks(pr, str("A"), colon, ill("X")), `PRINT["A"] : BADSTMT`, true},
	})
}

// @spec PARSER-008, PARSER-010
func TestSyntaxErrorInsidePrintEndsWithBadItem(t *testing.T) {
	runParseCases(t, []parseCase{
		{"illegal as first item", toks(pr, ill("X")), `PRINT[BAD(SYNTAX)]`, true},
		{"star as first item", toks(pr, star, number("2")), `PRINT[BAD(SYNTAX)]`, true},
		{"caret as first item", toks(pr, caret, number("2")), `PRINT[BAD(SYNTAX)]`, true},
		{"dangling caret", toks(pr, number("2"), caret), `PRINT[BAD(SYNTAX)]`, true},
		{"less as first item", toks(pr, lt, number("2")), `PRINT[BAD(SYNTAX)]`, true},
		{"equal as first item", toks(pr, eq, number("2")), `PRINT[BAD(SYNTAX)]`, true},
		{"dangling comparison", toks(pr, number("1"), lt), `PRINT[BAD(SYNTAX)]`, true},
		{"AND as first item", toks(pr, and, number("2")), `PRINT[BAD(SYNTAX)]`, true},
		{"dangling OR", toks(pr, number("1"), or), `PRINT[BAD(SYNTAX)]`, true},
		{"NOT alone", toks(pr, not), `PRINT[BAD(SYNTAX)]`, true},
		{"close paren as first item", toks(pr, rp), `PRINT[BAD(SYNTAX)]`, true},
		{"missing close paren", toks(pr, str("A"), semi, lp, number("1"), plus, number("2")), `PRINT["A" ; BAD(SYNTAX)]`, true},
		{"empty parens", toks(pr, lp, rp), `PRINT[BAD(SYNTAX)]`, true},
		{"dangling minus", toks(pr, number("1"), minus), `PRINT[BAD(SYNTAX)]`, true},
		{"dangling star", toks(pr, number("1"), star), `PRINT[BAD(SYNTAX)]`, true},
		{"stray close paren", toks(pr, number("1"), rp), `PRINT[#1 BAD(SYNTAX)]`, true},
		{"dangling plus", toks(pr, str("A"), plus), `PRINT[BAD(SYNTAX)]`, true},
		{"plus then separator", toks(pr, str("A"), plus, semi), `PRINT[BAD(SYNTAX)]`, true},
		{"plus then illegal", toks(pr, str("A"), semi, str("B"), plus, ill("@")), `PRINT["A" ; BAD(SYNTAX)]`, true},
		{"illegal after items", toks(pr, str("HELLO"), ill("@")), `PRINT["HELLO" BAD(SYNTAX)]`, true},
		{"plus after separator", toks(pr, str("A"), semi, plus), `PRINT["A" ; BAD(SYNTAX)]`, true},
		{"REM as an item", toks(pr, str("A"), rem(" NOTE")), `PRINT["A" BAD(SYNTAX)]`, true},
		{"REM as first item", toks(pr, rem("")), `PRINT[BAD(SYNTAX)]`, true},
		{"PRINT as an item", toks(pr, pr), `PRINT[BAD(SYNTAX)]`, true},
		{"plus then separator after a number", toks(pr, number("1"), plus, semi), `PRINT[BAD(SYNTAX)]`, true},
	})
}

// @spec PARSER-009
func TestParsingStopsAtFirstError(t *testing.T) {
	runParseCases(t, []parseCase{
		{"later statements not parsed",
			toks(pr, str("A"), colon, pr, str("B"), ill("@"), colon, pr, str("C")),
			`PRINT["A"] : PRINT["B" BAD(SYNTAX)]`, true},
		{"later statement start error not parsed",
			toks(pr, str("A"), colon, ill("@"), colon, pr, str("C")),
			`PRINT["A"] : BADSTMT`, true},
	})
}

// @spec PARSER-013
func TestNumberLiteral(t *testing.T) {
	runParseCases(t, []parseCase{
		{"integer", toks(pr, number("45")), `PRINT[#45]`, false},
		{"decimal", toks(pr, number("3.14")), `PRINT[#3.14]`, false},
		{"lone point is zero", toks(pr, number(".")), `PRINT[#0]`, false},
		{"leading point", toks(pr, number(".5")), `PRINT[#0.5]`, false},
		{"trailing point", toks(pr, number("5.")), `PRINT[#5]`, false},
		{"exponent", toks(pr, number("1.5E3")), `PRINT[#1500]`, false},
		{"negative exponent", toks(pr, number("1E-3")), `PRINT[#0.001]`, false},
		{"exponent without digits", toks(pr, number("1E")), `PRINT[#1]`, false},
		{"exponent sign without digits", toks(pr, number("1E+")), `PRINT[#1]`, false},
		{"point then exponent", toks(pr, number(".E5")), `PRINT[#0]`, false},
		{"too large for float64", toks(pr, number("1E999")), `PRINT[#+Inf]`, false},
		{"number items", toks(pr, number("1"), semi, number("2"), comma, str("A"), number("3")), `PRINT[#1 ; #2 , "A" #3]`, false},
	})
}

// @spec PARSER-006
func TestAddAndSubtract(t *testing.T) {
	runParseCases(t, []parseCase{
		{"subtract", toks(pr, number("8"), minus, number("2")), `PRINT[(#8-#2)]`, false},
		{"left to right", toks(pr, number("8"), minus, number("2"), minus, number("1")), `PRINT[((#8-#2)-#1)]`, false},
		{"mixed", toks(pr, number("1"), plus, number("2"), minus, number("3")), `PRINT[((#1+#2)-#3)]`, false},
	})
}

// @spec PARSER-014
func TestMultiplyAndDivide(t *testing.T) {
	runParseCases(t, []parseCase{
		{"multiply", toks(pr, number("2"), star, number("3")), `PRINT[(#2*#3)]`, false},
		{"binds tighter than +", toks(pr, number("2"), plus, number("3"), star, number("4")), `PRINT[(#2+(#3*#4))]`, false},
		{"binds tighter than - on the left", toks(pr, number("2"), star, number("3"), minus, number("4")), `PRINT[((#2*#3)-#4)]`, false},
		{"left to right", toks(pr, number("8"), slash, number("4"), slash, number("2")), `PRINT[((#8/#4)/#2)]`, false},
		{"mixed", toks(pr, number("8"), slash, number("4"), star, number("2")), `PRINT[((#8/#4)*#2)]`, false},
	})
}

// @spec PARSER-015
func TestNegation(t *testing.T) {
	runParseCases(t, []parseCase{
		{"negative number", toks(pr, minus, number("5")), `PRINT[(-#5)]`, false},
		{"binds tighter than *", toks(pr, minus, number("2"), star, number("3")), `PRINT[((-#2)*#3)]`, false},
		{"after *", toks(pr, number("2"), star, minus, number("3")), `PRINT[(#2*(-#3))]`, false},
		{"repeated", toks(pr, minus, minus, number("5")), `PRINT[(-(-#5))]`, false},
		{"after binary minus", toks(pr, number("5"), minus, minus, number("5")), `PRINT[(#5-(-#5))]`, false},
		{"of a string", toks(pr, minus, str("A")), `PRINT[(-"A")]`, false},
	})
}

// @spec PARSER-016
func TestLeadingPlusIsDropped(t *testing.T) {
	runParseCases(t, []parseCase{
		{"number", toks(pr, plus, number("5")), `PRINT[#5]`, false},
		{"string", toks(pr, plus, str("A")), `PRINT["A"]`, false},
		{"after an operator", toks(pr, number("2"), star, plus, number("3")), `PRINT[(#2*#3)]`, false},
		{"mixed signs", toks(pr, minus, plus, minus, number("5")), `PRINT[(-(-#5))]`, false},
	})
}

// @spec PARSER-017
func TestParentheses(t *testing.T) {
	runParseCases(t, []parseCase{
		{"grouping", toks(pr, lp, number("2"), plus, number("3"), rp, star, number("4")), `PRINT[((#2+#3)*#4)]`, false},
		{"nested", toks(pr, lp, lp, number("1"), rp, rp), `PRINT[#1]`, false},
		{"negated group", toks(pr, minus, lp, number("1"), plus, number("2"), rp), `PRINT[(-(#1+#2))]`, false},
		{"string in parens", toks(pr, lp, str("A"), plus, str("B"), rp), `PRINT[("A"+"B")]`, false},
	})
}

// @spec PARSER-018
func TestExpressionEndsBeforeNextItem(t *testing.T) {
	runParseCases(t, []parseCase{
		{"paren after number", toks(pr, number("2"), lp, number("3"), rp), `PRINT[#2 #3]`, false},
		{"string after expression", toks(pr, number("1"), plus, number("1"), str("A")), `PRINT[(#1+#1) "A"]`, false},
		{"minus continues", toks(pr, number("1"), minus, number("1")), `PRINT[(#1-#1)]`, false},
		{"minus after string continues", toks(pr, str("A"), minus, number("1")), `PRINT[("A"-#1)]`, false},
	})
}

// @spec PARSER-019
func TestPower(t *testing.T) {
	runParseCases(t, []parseCase{
		{"power", toks(pr, number("2"), caret, number("3")), `PRINT[(#2^#3)]`, false},
		{"left to right", toks(pr, number("2"), caret, number("3"), caret, number("2")), `PRINT[((#2^#3)^#2)]`, false},
		{"tighter than negation", toks(pr, minus, number("2"), caret, number("2")), `PRINT[(-(#2^#2))]`, false},
		{"tighter than *", toks(pr, number("2"), star, number("3"), caret, number("2")), `PRINT[(#2*(#3^#2))]`, false},
		{"tighter than * on the left", toks(pr, number("3"), caret, number("2"), star, number("2")), `PRINT[((#3^#2)*#2)]`, false},
		{"parenthesized base", toks(pr, lp, minus, number("2"), rp, caret, number("2")), `PRINT[((-#2)^#2)]`, false},
		{"parenthesized exponent", toks(pr, number("2"), caret, lp, number("1"), plus, number("1"), rp), `PRINT[(#2^(#1+#1))]`, false},
	})
}

// @spec PARSER-020
func TestSignedExponent(t *testing.T) {
	runParseCases(t, []parseCase{
		{"negative exponent", toks(pr, number("2"), caret, minus, number("1")), `PRINT[(#2^(-#1))]`, false},
		{"sign takes in a following power", toks(pr, number("2"), caret, minus, number("1"), caret, number("2")), `PRINT[(#2^(-(#1^#2)))]`, false},
		{"sign stops before *", toks(pr, number("2"), caret, minus, number("3"), star, number("4")), `PRINT[((#2^(-#3))*#4)]`, false},
		{"plus sign dropped", toks(pr, number("2"), caret, plus, number("3")), `PRINT[(#2^#3)]`, false},
	})
}

// @spec PARSER-021
func TestAssignment(t *testing.T) {
	runParseCases(t, []parseCase{
		{"without LET", toks(name("A"), eq, number("5")), `LET $A[A]=#5`, false},
		{"with LET", toks(let, name("A"), eq, number("5")), `LET $A[A]=#5`, false},
		{"string", toks(name("N$"), eq, str("HI")), `LET $N$[N$]="HI"`, false},
		{"expression", toks(name("B"), eq, name("A"), star, number("2"), plus, number("1")), `LET $B[B]=(($A[A]*#2)+#1)`, false},
		{"then PRINT", toks(name("A"), eq, number("1"), colon, pr, name("A")), `LET $A[A]=#1 : PRINT[$A[A]]`, false},
	})
}

// @spec PARSER-022
func TestVariableIdentity(t *testing.T) {
	runParseCases(t, []parseCase{
		{"long name", toks(pr, name("HEIGHT")), `PRINT[$HE[HEIGHT]]`, false},
		{"long string name", toks(pr, name("NAME$")), `PRINT[$NA$[NAME$]]`, false},
		{"one-letter string name", toks(pr, name("N$")), `PRINT[$N$[N$]]`, false},
		{"digit second", toks(pr, name("A1B")), `PRINT[$A1[A1B]]`, false},
		{"one letter", toks(pr, name("X")), `PRINT[$X[X]]`, false},
		{"integer name", toks(pr, name("COUNT%")), `PRINT[$CO%[COUNT%]]`, false},
		{"one-letter integer name", toks(pr, name("C%")), `PRINT[$C%[C%]]`, false},
	})
}

// @spec PARSER-023
func TestVariableOperands(t *testing.T) {
	runParseCases(t, []parseCase{
		{"in an expression", toks(pr, name("A"), plus, number("1")), `PRINT[($A[A]+#1)]`, false},
		{"negated", toks(pr, minus, name("A")), `PRINT[(-$A[A])]`, false},
		{"in parentheses", toks(pr, lp, name("A"), rp), `PRINT[$A[A]]`, false},
		{"side by side", toks(pr, number("1"), name("A")), `PRINT[#1 $A[A]]`, false},
		{"string then variable", toks(pr, str("X"), name("A")), `PRINT["X" $A[A]]`, false},
	})
}

// @spec PARSER-024
func TestUnsupportedNames(t *testing.T) {
	runParseCases(t, []parseCase{
		{"TI used", toks(pr, name("TI")), `PRINT[BAD(SYNTAX)]`, true},
		{"TIME used", toks(pr, name("TIME")), `PRINT[BAD(SYNTAX)]`, true},
		{"TI$ used", toks(pr, name("TI$")), `PRINT[BAD(SYNTAX)]`, true},
		{"ST is readable", toks(pr, name("STATUS")), `PRINT[$ST[STATUS]]`, false},
		{"TI assigned", toks(name("TI"), eq, number("1")), `BADSTMT`, true},
		{"T and I separately are fine", toks(pr, name("T"), semi, name("IT")), `PRINT[$T[T] ; $IT[IT]]`, false},
		{"TI% is ordinary", toks(pr, name("TI%"), semi, name("ST%")), `PRINT[$TI%[TI%] ; $ST%[ST%]]`, false},
	})
}

// @spec PARSER-025, PARSER-011
func TestAssignmentSyntaxErrors(t *testing.T) {
	runParseCases(t, []parseCase{
		{"LET alone", toks(let), `BADSTMT`, true},
		{"LET a number", toks(let, number("5"), eq, number("1")), `BADSTMT`, true},
		{"name alone", toks(name("A")), `BADSTMT`, true},
		{"no value", toks(name("A"), eq), `BADSTMT`, true},
		{"missing =", toks(name("A"), number("5")), `BADSTMT`, true},
		{"after a good statement", toks(pr, number("1"), colon, name("A"), eq), `PRINT[#1] : BADSTMT`, true},
		{"bad value", toks(name("A"), eq, number("1"), plus), `BADSTMT`, true},
	})
}

// @spec PARSER-026
func TestComparisons(t *testing.T) {
	runParseCases(t, []parseCase{
		{"equal", toks(pr, number("1"), eq, number("2")), `PRINT[(#1 = #2)]`, false},
		{"looser than arithmetic", toks(pr, number("1"), plus, number("1"), eq, number("2")), `PRINT[((#1+#1) = #2)]`, false},
		{"looser on the right", toks(pr, number("2"), eq, number("1"), plus, number("1")), `PRINT[(#2 = (#1+#1))]`, false},
		{"left to right", toks(pr, number("1"), lt, number("2"), lt, number("3")), `PRINT[((#1 < #2) < #3)]`, false},
		{"strings", toks(pr, str("A"), lt, str("B")), `PRINT[("A" < "B")]`, false},
		{"in parentheses", toks(pr, lp, number("1"), gt, number("2"), rp, plus, number("1")), `PRINT[((#1 > #2)+#1)]`, false},
		{"then another item", toks(pr, number("1"), eq, number("1"), semi, number("2")), `PRINT[(#1 = #1) ; #2]`, false},
	})
}

// @spec PARSER-027
func TestComparisonOperators(t *testing.T) {
	runParseCases(t, []parseCase{
		{"less", toks(pr, number("1"), lt, number("2")), `PRINT[(#1 < #2)]`, false},
		{"greater", toks(pr, number("1"), gt, number("2")), `PRINT[(#1 > #2)]`, false},
		{"not equal", toks(pr, number("1"), lt, gt, number("2")), `PRINT[(#1 <> #2)]`, false},
		{"not equal reversed", toks(pr, number("1"), gt, lt, number("2")), `PRINT[(#1 <> #2)]`, false},
		{"less or equal", toks(pr, number("1"), lt, eq, number("2")), `PRINT[(#1 <= #2)]`, false},
		{"less or equal reversed", toks(pr, number("1"), eq, lt, number("2")), `PRINT[(#1 <= #2)]`, false},
		{"greater or equal", toks(pr, number("1"), gt, eq, number("2")), `PRINT[(#1 => #2)]`, false},
		{"greater or equal reversed", toks(pr, number("1"), eq, gt, number("2")), `PRINT[(#1 => #2)]`, false},
		{"all three", toks(pr, number("1"), lt, eq, gt, number("2")), `PRINT[(#1 <=> #2)]`, false},
		{"repeated equal", toks(pr, number("1"), eq, eq, number("2")), `PRINT[BAD(SYNTAX)]`, true},
		{"repeated less", toks(pr, number("1"), lt, lt, number("2")), `PRINT[BAD(SYNTAX)]`, true},
		{"repeated after another", toks(pr, number("1"), lt, eq, lt, number("2")), `PRINT[BAD(SYNTAX)]`, true},
	})
}

// @spec PARSER-028
func TestAssignmentOfComparison(t *testing.T) {
	runParseCases(t, []parseCase{
		{"A=B=C", toks(name("A"), eq, name("B"), eq, name("C")), `LET $A[A]=($B[B] = $C[C])`, false},
		{"LET A=1<2", toks(let, name("A"), eq, number("1"), lt, number("2")), `LET $A[A]=(#1 < #2)`, false},
	})
}

// @spec PARSER-029
func TestAndOr(t *testing.T) {
	runParseCases(t, []parseCase{
		{"AND", toks(pr, number("12"), and, number("10")), `PRINT[(#12 AND #10)]`, false},
		{"AND before OR", toks(pr, name("A"), or, name("B"), and, name("C")), `PRINT[($A[A] OR ($B[B] AND $C[C]))]`, false},
		{"AND before OR on the left", toks(pr, name("A"), and, name("B"), or, name("C")), `PRINT[(($A[A] AND $B[B]) OR $C[C])]`, false},
		{"looser than comparisons", toks(pr, number("1"), lt, number("2"), and, number("3"), lt, number("4")), `PRINT[((#1 < #2) AND (#3 < #4))]`, false},
		{"left to right", toks(pr, number("1"), or, number("2"), or, number("4")), `PRINT[((#1 OR #2) OR #4)]`, false},
		{"in parentheses", toks(pr, lp, number("1"), or, number("2"), rp, and, number("3")), `PRINT[((#1 OR #2) AND #3)]`, false},
	})
}

// @spec PARSER-030
func TestNot(t *testing.T) {
	runParseCases(t, []parseCase{
		{"NOT a number", toks(pr, not, number("0")), `PRINT[(NOT #0)]`, false},
		{"takes in a comparison", toks(pr, not, number("1"), eq, number("2")), `PRINT[(NOT (#1 = #2))]`, false},
		{"stops at AND", toks(pr, not, name("A"), and, name("B")), `PRINT[((NOT $A[A]) AND $B[B])]`, false},
		{"mid-expression", toks(pr, number("1"), plus, not, number("0"), plus, number("1")), `PRINT[(#1+(NOT (#0+#1)))]`, false},
		{"twice", toks(pr, not, not, number("0")), `PRINT[(NOT (NOT #0))]`, false},
		{"after =", toks(name("A"), eq, not, number("0")), `LET $A[A]=(NOT #0)`, false},
	})
}

// @spec PARSER-031
func TestIfStatement(t *testing.T) {
	runParseCases(t, []parseCase{
		{"IF THEN PRINT", toks(iff, name("A"), gt, number("1"), then, pr, str("X")), `IF ($A[A] > #1) : PRINT["X"]`, false},
		{"rest of line", toks(iff, number("1"), then, pr, str("X"), colon, pr, str("Y")), `IF #1 : PRINT["X"] : PRINT["Y"]`, false},
		{"assignment after THEN", toks(iff, number("1"), then, name("A"), eq, number("2")), `IF #1 : LET $A[A]=#2`, false},
		{"nothing after THEN", toks(iff, number("1"), then), `IF #1`, false},
		{"colon after THEN", toks(iff, number("1"), then, colon, pr, str("X")), `IF #1 : PRINT["X"]`, false},
		{"nested", toks(iff, name("A"), then, iff, name("B"), then, pr, str("X")), `IF $A[A] : IF $B[B] : PRINT["X"]`, false},
		{"after a statement", toks(pr, str("A"), colon, iff, number("0"), then, pr, str("B")), `PRINT["A"] : IF #0 : PRINT["B"]`, false},
		{"condition with AND", toks(iff, name("A"), and, name("B"), then, pr, str("X")), `IF ($A[A] AND $B[B]) : PRINT["X"]`, false},
		{"error after THEN is in the tree", toks(iff, number("0"), then, ill("@")), `IF #0 : BADSTMT`, true},
		{"error in PRINT after THEN", toks(iff, number("0"), then, pr, str("A"), ill("@")), `IF #0 : PRINT["A" BAD(SYNTAX)]`, true},
	})
}

// @spec PARSER-032
func TestIfSyntaxErrors(t *testing.T) {
	runParseCases(t, []parseCase{
		{"no THEN", toks(iff, number("1"), pr, str("X")), `BADSTMT`, true},
		{"GO TO", toks(iff, number("1"), gok, tok, number("100")), `BADSTMT`, true},
		{"THEN number without digits", toks(iff, number("1"), then, number(".5")), `IF #1 : BADSTMT`, true},
		{"no condition", toks(iff, then, pr, str("X")), `BADSTMT`, true},
		{"IF alone", toks(iff), `BADSTMT`, true},
	})
}

// @spec PARSER-033
func TestRun(t *testing.T) {
	runParseCases(t, []parseCase{
		{"RUN", toks(run), `RUN`, false},
		{"RUN then a statement", toks(run, colon, pr, str("X")), `RUN : PRINT["X"]`, false},
	})
}

// @spec PARSER-034
func TestRunLineNumber(t *testing.T) {
	runParseCases(t, []parseCase{
		{"RUN n", toks(run, number("20")), `RUN 20`, false},
		{"RUN 0", toks(run, number("0")), `RUN 0`, false},
		{"largest", toks(run, number("63999")), `RUN 63999`, false},
		{"fraction", toks(run, number("20.5")), `RUN 20`, false},
		{"exponent", toks(run, number("1E3")), `RUN 1`, false},
		{"no digits", toks(run, number(".5")), `RUN 0`, false},
		{"then a statement", toks(run, number("20"), colon, pr, str("X")), `RUN 20 : PRINT["X"]`, false},
	})
}

// @spec PARSER-035
func TestRunOtherArgumentIsLineZero(t *testing.T) {
	runParseCases(t, []parseCase{
		{"name", toks(run, name("A")), `RUN 0 : BADSTMT`, true},
		{"string", toks(run, str("X")), `RUN 0 : BADSTMT`, true},
		{"minus", toks(run, minus, number("5")), `RUN 0 : BADSTMT`, true},
	})
}

// @spec PARSER-036, PARSER-007
func TestListNewEnd(t *testing.T) {
	runParseCases(t, []parseCase{
		{"LIST", toks(list), `LIST`, false},
		{"NEW", toks(nw), `NEW`, false},
		{"END", toks(end), `END`, false},
		{"after a statement", toks(pr, str("A"), colon, end), `PRINT["A"] : END`, false},
		{"before a statement", toks(list, colon, pr, str("A")), `LIST : PRINT["A"]`, false},
		{"after THEN", toks(iff, number("1"), then, end), `IF #1 : END`, false},
	})
}

// @spec PARSER-037
func TestProgramCommandSyntaxErrors(t *testing.T) {
	runParseCases(t, []parseCase{
		{"LIST n", toks(list, number("10")), `BADSTMT`, true},
		{"LIST range", toks(list, number("10"), minus, number("20")), `BADSTMT`, true},
		{"NEW junk", toks(nw, name("X")), `BADSTMT`, true},
		{"END n", toks(end, number("1")), `BADSTMT`, true},
		{"after a statement", toks(pr, str("A"), colon, end, str("X")), `PRINT["A"] : BADSTMT`, true},
		{"RUN too large", toks(run, number("64000")), `BADSTMT`, true},
		{"GOTO too large", toks(gotok, number("64000")), `BADSTMT`, true},
		{"THEN line too large", toks(iff, number("1"), then, number("64000")), `IF #1 : BADSTMT`, true},
		{"GO without TO", toks(gok, number("10")), `BADSTMT`, true},
		{"GO alone", toks(gok), `BADSTMT`, true},
	})
}

// @spec PARSER-038
func TestGoto(t *testing.T) {
	runParseCases(t, []parseCase{
		{"GOTO n", toks(gotok, number("20")), `GOTO 20`, false},
		{"GO TO n", toks(gok, tok, number("20")), `GOTO 20`, false},
		{"after a statement", toks(pr, str("A"), colon, gotok, number("10")), `PRINT["A"] : GOTO 10`, false},
		{"then a statement", toks(gotok, number("10"), colon, pr, str("X")), `GOTO 10 : PRINT["X"]`, false},
	})
}

// @spec PARSER-039
func TestJumpLineNumbers(t *testing.T) {
	runParseCases(t, []parseCase{
		{"fraction", toks(gotok, number("20.5")), `GOTO 20`, false},
		{"exponent", toks(gotok, number("1E3")), `GOTO 1`, false},
		{"no digits", toks(gotok, number(".5")), `GOTO 0`, false},
		{"GOTO alone", toks(gotok), `GOTO 0`, false},
		{"GOTO name", toks(gotok, name("A")), `GOTO 0 : BADSTMT`, true},
		{"GO TO alone", toks(gok, tok), `GOTO 0`, false},
		{"largest", toks(gotok, number("63999")), `GOTO 63999`, false},
		{"RUN fraction", toks(run, number("20.5")), `RUN 20`, false},
	})
}

// @spec PARSER-040
func TestIfThenLineNumber(t *testing.T) {
	runParseCases(t, []parseCase{
		{"THEN n", toks(iff, name("A"), then, number("20")), `IF $A[A] : GOTO 20`, false},
		{"THEN fraction", toks(iff, name("A"), then, number("20.5")), `IF $A[A] : GOTO 20`, false},
		{"THEN n then more", toks(iff, name("A"), then, number("20"), colon, pr, str("X")), `IF $A[A] : GOTO 20 : PRINT["X"]`, false},
		{"THEN GOTO n", toks(iff, name("A"), then, gotok, number("20")), `IF $A[A] : GOTO 20`, false},
	})
}

// @spec PARSER-041
func TestIfGoto(t *testing.T) {
	runParseCases(t, []parseCase{
		{"GOTO n", toks(iff, name("A"), gotok, number("20")), `IF $A[A] : GOTO 20`, false},
		{"GOTO alone", toks(iff, name("A"), gotok), `IF $A[A] : GOTO 0`, false},
	})
}

// @spec PARSER-042
func TestFor(t *testing.T) {
	runParseCases(t, []parseCase{
		{"FOR TO", toks(fork, name("I"), eq, number("1"), tok, number("3")), `FOR $I[I]=#1 TO #3`, false},
		{"FOR TO STEP", toks(fork, name("I"), eq, number("10"), tok, number("1"), stepk, minus, number("2")), `FOR $I[I]=#10 TO #1 STEP (-#2)`, false},
		{"expressions", toks(fork, name("I"), eq, name("A"), plus, number("1"), tok, name("B"), star, number("2")), `FOR $I[I]=($A[A]+#1) TO ($B[B]*#2)`, false},
		{"string variable", toks(fork, name("A$"), eq, str("X"), tok, number("1")), `FOR $A$[A$]="X" TO #1`, false},
		{"then a statement", toks(fork, name("I"), eq, number("1"), tok, number("3"), colon, pr, name("I")), `FOR $I[I]=#1 TO #3 : PRINT[$I[I]]`, false},
	})
}

// @spec PARSER-043
func TestForSyntaxErrors(t *testing.T) {
	runParseCases(t, []parseCase{
		{"integer variable", toks(fork, name("I%"), eq, number("1"), tok, number("3")), `BADSTMT`, true},
		{"no variable", toks(fork, eq, number("1"), tok, number("3")), `BADSTMT`, true},
		{"no =", toks(fork, name("I"), number("1"), tok, number("3")), `BADSTMT`, true},
		{"no start", toks(fork, name("I"), eq, tok, number("3")), `BADSTMT`, true},
		{"no TO", toks(fork, name("I"), eq, number("1"), number("3")), `BADSTMT`, true},
		{"no end", toks(fork, name("I"), eq, number("1"), tok), `BADSTMT`, true},
		{"no step", toks(fork, name("I"), eq, number("1"), tok, number("3"), stepk), `BADSTMT`, true},
		{"junk after", toks(fork, name("I"), eq, number("1"), tok, number("3"), str("X")), `BADSTMT`, true},
	})
}

// @spec PARSER-044
func TestNext(t *testing.T) {
	runParseCases(t, []parseCase{
		{"bare", toks(nextk), `NEXT`, false},
		{"one variable", toks(nextk, name("I")), `NEXT $I[I]`, false},
		{"two variables", toks(nextk, name("J"), comma, name("I")), `NEXT $J[J],$I[I]`, false},
		{"then a statement", toks(nextk, colon, pr, str("X")), `NEXT : PRINT["X"]`, false},
	})
}

// @spec PARSER-045
func TestNextSyntaxErrors(t *testing.T) {
	runParseCases(t, []parseCase{
		{"no comma", toks(nextk, name("I"), name("J")), `BADSTMT`, true},
		{"trailing comma", toks(nextk, name("I"), comma), `BADSTMT`, true},
		{"not a variable", toks(nextk, number("1")), `BADSTMT`, true},
	})
}

// @spec PARSER-046
func TestGosub(t *testing.T) {
	runParseCases(t, []parseCase{
		{"GOSUB n", toks(gosub, number("100")), `GOSUB 100`, false},
		{"fraction", toks(gosub, number("100.5")), `GOSUB 100`, false},
		{"alone", toks(gosub), `GOSUB 0`, false},
		{"after THEN", toks(iff, name("A"), then, gosub, number("100"), colon, pr, str("X")), `IF $A[A] : GOSUB 100 : PRINT["X"]`, false},
	})
}

// @spec PARSER-047
func TestReturn(t *testing.T) {
	runParseCases(t, []parseCase{
		{"RETURN", toks(retk), `RETURN`, false},
		{"then a statement", toks(retk, colon, pr, str("X")), `RETURN : PRINT["X"]`, false},
		{"junk", toks(retk, number("10")), `BADSTMT`, true},
		{"GOSUB too large", toks(gosub, number("64000")), `BADSTMT`, true},
	})
}
func dumpVars(vs []*ast.VarRef) string {
	var out []string
	for _, v := range vs {
		out = append(out, dumpExpr(v))
	}
	return strings.Join(out, ",")
}

// @spec PARSER-048
func TestInput(t *testing.T) {
	runParseCases(t, []parseCase{
		{"one variable", toks(input, name("A")), `INPUT $A[A]`, false},
		{"prompt", toks(input, str("NAME"), semi, name("N$")), `INPUT "NAME";$N$[N$]`, false},
		{"several", toks(input, name("A"), comma, name("B$"), comma, name("C%")), `INPUT $A[A],$B$[B$],$C%[C%]`, false},
		{"then a statement", toks(input, name("A"), colon, pr, name("A")), `INPUT $A[A] : PRINT[$A[A]]`, false},
	})
}

// @spec PARSER-049
func TestGet(t *testing.T) {
	runParseCases(t, []parseCase{
		{"one variable", toks(getk, name("K$")), `GET $K$[K$]`, false},
		{"several", toks(getk, name("A$"), comma, name("B")), `GET $A$[A$],$B[B]`, false},
	})
}

// @spec PARSER-050
func TestInputGetSyntaxErrors(t *testing.T) {
	runParseCases(t, []parseCase{
		{"no variable", toks(input), `BADSTMT`, true},
		{"prompt only", toks(input, str("N"), semi), `BADSTMT`, true},
		{"prompt with comma", toks(input, str("N"), comma, name("A")), `BADSTMT`, true},
		{"prompt expression", toks(input, str("A"), plus, str("B"), semi, name("X")), `BADSTMT`, true},
		{"variable prompt", toks(input, name("P$"), semi, name("X")), `BADSTMT`, true},
		{"trailing comma", toks(input, name("A"), comma), `BADSTMT`, true},
		{"GET no variable", toks(getk), `BADSTMT`, true},
		{"GET number", toks(getk, number("1")), `BADSTMT`, true},
	})
}

// @spec PARSER-051
func TestDef(t *testing.T) {
	runParseCases(t, []parseCase{
		{"DEF FN", toks(def, fn, name("SQ"), lp, name("X"), rp, eq, name("X"), star, name("X")), `DEF FN $SQ[SQ]($X[X])=($X[X]*$X[X])`, false},
		{"long names", toks(def, fn, name("AREA"), lp, name("RADIUS"), rp, eq, number("1")), `DEF FN $AR[AREA]($RA[RADIUS])=#1`, false},
		{"string name parses", toks(def, fn, name("A$"), lp, name("X"), rp, eq, number("1")), `DEF FN $A$[A$]($X[X])=#1`, false},
		{"then a statement", toks(def, fn, name("A"), lp, name("X"), rp, eq, name("X"), colon, pr, str("Y")), `DEF FN $A[A]($X[X])=$X[X] : PRINT["Y"]`, false},
	})
}

// @spec PARSER-052
func TestDefBodyErrorsAreKept(t *testing.T) {
	runParseCases(t, []parseCase{
		{"bad body", toks(def, fn, name("A"), lp, name("X"), rp, eq, name("X"), plus), `DEF FN $A[A]($X[X])=<SYNTAX>`, false},
		{"junk after body", toks(def, fn, name("A"), lp, name("X"), rp, eq, name("X"), name("Y")), `DEF FN $A[A]($X[X])=<SYNTAX>`, false},
		{"no body", toks(def, fn, name("A"), lp, name("X"), rp, eq), `DEF FN $A[A]($X[X])=<SYNTAX>`, false},
		{"continues after colon", toks(def, fn, name("A"), lp, name("X"), rp, eq, ill("@"), colon, pr, str("Y")), `DEF FN $A[A]($X[X])=<SYNTAX> : PRINT["Y"]`, false},
	})
}

// @spec PARSER-053
func TestDefSyntaxErrors(t *testing.T) {
	runParseCases(t, []parseCase{
		{"no FN", toks(def, name("A"), lp, name("X"), rp, eq, number("1")), `BADSTMT`, true},
		{"no name", toks(def, fn, lp, name("X"), rp, eq, number("1")), `BADSTMT`, true},
		{"no (", toks(def, fn, name("A"), name("X"), rp, eq, number("1")), `BADSTMT`, true},
		{"no parameter", toks(def, fn, name("A"), lp, rp, eq, number("1")), `BADSTMT`, true},
		{"no )", toks(def, fn, name("A"), lp, name("X"), eq, number("1")), `BADSTMT`, true},
		{"no =", toks(def, fn, name("A"), lp, name("X"), rp, number("1")), `BADSTMT`, true},
		{"integer name", toks(def, fn, name("A%"), lp, name("X"), rp, eq, number("1")), `BADSTMT`, true},
		{"integer parameter", toks(def, fn, name("A"), lp, name("X%"), rp, eq, number("1")), `BADSTMT`, true},
	})
}

// @spec PARSER-054
func TestFnCall(t *testing.T) {
	runParseCases(t, []parseCase{
		{"call", toks(pr, fn, name("SQ"), lp, number("3"), rp), `PRINT[FN $SQ[SQ](#3)]`, false},
		{"in an expression", toks(pr, number("1"), plus, fn, name("A"), lp, name("B"), star, number("2"), rp), `PRINT[(#1+FN $A[A](($B[B]*#2)))]`, false},
		{"integer name", toks(pr, fn, name("A%"), lp, number("3"), rp), `PRINT[BAD(SYNTAX)]`, true},
		{"no (", toks(pr, fn, name("A"), number("3")), `PRINT[BAD(SYNTAX)]`, true},
		{"no )", toks(pr, fn, name("A"), lp, number("3")), `PRINT[BAD(SYNTAX)]`, true},
	})
}
func dumpFileArgs(a ast.FileArgs) string {
	out := ""
	for _, e := range []ast.Expr{a.Name, a.Device, a.Secondary} {
		if e != nil {
			out += " " + dumpExpr(e)
		}
	}
	return out
}

// @spec PARSER-055
func TestFileStatements(t *testing.T) {
	runParseCases(t, []parseCase{
		{"LOAD", toks(load), `LOAD`, false},
		{"name", toks(load, str("X")), `LOAD "X"`, false},
		{"device", toks(save, str("X"), comma, number("8")), `SAVE "X" #8`, false},
		{"secondary", toks(verif, str("X"), comma, number("8"), comma, number("1")), `VERIFY "X" #8 #1`, false},
		{"expressions", toks(save, name("N$"), plus, str("X"), comma, name("D")), `SAVE ($N$[N$]+"X") $D[D]`, false},
		{"then a statement", toks(save, str("X"), colon, pr, str("Y")), `SAVE "X" : PRINT["Y"]`, false},
	})
}

// @spec PARSER-056
func TestFileStatementErrors(t *testing.T) {
	runParseCases(t, []parseCase{
		{"dangling comma", toks(load, str("X"), comma), `BADSTMT`, true},
		{"four arguments", toks(load, str("X"), comma, number("8"), comma, number("1"), comma, number("2")), `BADSTMT`, true},
		{"junk", toks(save, str("X"), str("Y")), `BADSTMT`, true},
		{"comma first", toks(load, comma, number("8")), `BADSTMT`, true},
	})
}
func dumpItems(items []ast.PrintItem) string {
	var out []string
	for _, it := range items {
		out = append(out, dumpItem(it))
	}
	return strings.Join(out, " ")
}

// @spec PARSER-057
func TestOpen(t *testing.T) {
	runParseCases(t, []parseCase{
		{"file only", toks(openk, number("1")), `OPEN #1`, false},
		{"all four", toks(openk, number("2"), comma, number("8"), comma, number("2"), comma, str("F,S,W")), `OPEN #2 #8 #2 "F,S,W"`, false},
	})
}

// @spec PARSER-058
func TestClose(t *testing.T) {
	runParseCases(t, []parseCase{
		{"CLOSE", toks(close, number("2")), `CLOSE #2`, false},
		{"no file", toks(close), `BADSTMT`, true},
		{"two files", toks(close, number("2"), comma, number("3")), `BADSTMT`, true},
	})
}

// @spec PARSER-059
func TestPrintFileAndCmd(t *testing.T) {
	runParseCases(t, []parseCase{
		{"PRINT# alone", toks(prf, number("1")), `PRINT##1[]`, false},
		{"PRINT# items", toks(prf, number("1"), comma, str("A"), semi, name("B")), `PRINT##1["A" ; $B[B]]`, false},
		{"PRINT# comma then nothing", toks(prf, number("1"), comma), `PRINT##1[]`, false},
		{"CMD alone", toks(cmdk, number("4")), `CMD #4[]`, false},
		{"CMD items", toks(cmdk, number("4"), comma, str("X")), `CMD #4["X"]`, false},
	})
}

// @spec PARSER-060
func TestInputFile(t *testing.T) {
	runParseCases(t, []parseCase{
		{"INPUT#", toks(inf, number("2"), comma, name("A$"), comma, name("B")), `INPUT##2 $A$[A$],$B[B]`, false},
	})
}

// @spec PARSER-061
func TestGetFile(t *testing.T) {
	runParseCases(t, []parseCase{
		{"GET#", toks(getk, hash, number("2"), comma, name("A$")), `GET##2 $A$[A$]`, false},
	})
}

// @spec PARSER-062
func TestDataFileSyntaxErrors(t *testing.T) {
	runParseCases(t, []parseCase{
		{"OPEN alone", toks(openk), `BADSTMT`, true},
		{"OPEN five", toks(openk, number("1"), comma, number("2"), comma, number("3"), comma, str("N"), comma, number("5")), `BADSTMT`, true},
		{"PRINT# no comma", toks(prf, number("1"), str("A")), `BADSTMT`, true},
		{"PRINT# no file", toks(prf), `BADSTMT`, true},
		{"PRINT space hash", toks(pr, hash, number("1")), `PRINT[BAD(SYNTAX)]`, true},
		{"INPUT# no comma", toks(inf, number("1"), name("A")), `BADSTMT`, true},
		{"INPUT# no variable", toks(inf, number("1"), comma), `BADSTMT`, true},
		{"GET# no comma", toks(getk, hash, number("1"), name("A$")), `BADSTMT`, true},
		{"CMD no file", toks(cmdk), `BADSTMT`, true},
	})
}

// @spec PARSER-063, PARSER-024
func TestStatusVariable(t *testing.T) {
	runParseCases(t, []parseCase{
		{"read ST", toks(pr, name("ST")), `PRINT[$ST[ST]]`, false},
		{"assign ST", toks(name("ST"), eq, number("1")), `BADSTMT`, true},
		{"LET ST", toks(let, name("STATUS"), eq, number("1")), `BADSTMT`, true},
		{"TI still reserved", toks(pr, name("TI")), `PRINT[BAD(SYNTAX)]`, true},
	})
}

// @spec PARSER-064
func TestOn(t *testing.T) {
	runParseCases(t, []parseCase{
		{"GOTO", toks(onk, name("X"), gotok, number("100"), comma, number("200")), `ON $X[X] GOTO [100 200]`, false},
		{"GOSUB", toks(onk, name("X"), plus, number("1"), gosub, number("10")), `ON ($X[X]+#1) GOSUB [10]`, false},
		{"fraction", toks(onk, number("1"), gotok, number("10.5")), `ON #1 GOTO [10]`, false},
		{"then a statement", toks(onk, name("X"), gotok, number("10"), colon, pr, str("Y")), `ON $X[X] GOTO [10] : PRINT["Y"]`, false},
	})
}

// @spec PARSER-065
func TestOnSyntaxErrors(t *testing.T) {
	runParseCases(t, []parseCase{
		{"no GOTO", toks(onk, name("X"), number("10")), `BADSTMT`, true},
		{"GO TO", toks(onk, name("X"), gok, tok, number("10")), `BADSTMT`, true},
		{"THEN", toks(onk, name("X"), then, number("10")), `BADSTMT`, true},
		{"no line", toks(onk, name("X"), gotok), `BADSTMT`, true},
		{"empty element", toks(onk, name("X"), gotok, number("10"), comma, comma, number("30")), `BADSTMT`, true},
		{"name element", toks(onk, name("X"), gotok, name("A")), `BADSTMT`, true},
		{"too large", toks(onk, name("X"), gotok, number("64000")), `BADSTMT`, true},
		{"junk", toks(onk, name("X"), gotok, number("10"), str("Y")), `BADSTMT`, true},
	})
}
func fun(name string) token.Token { return token.Token{Kind: token.Function, Value: name} }

// @spec PARSER-066
func TestCall(t *testing.T) {
	runParseCases(t, []parseCase{
		{"call", toks(pr, fun("SIN"), lp, name("X"), rp), `PRINT[SIN($X[X])]`, false},
		{"in an expression", toks(pr, number("1"), plus, fun("INT"), lp, number("2.5"), rp, star, number("2")), `PRINT[(#1+(INT(#2.5)*#2))]`, false},
		{"power binds tighter", toks(pr, fun("ABS"), lp, number("2"), rp, caret, number("2")), `PRINT[(ABS(#2)^#2)]`, false},
		{"nested", toks(pr, fun("SQR"), lp, fun("ABS"), lp, minus, number("4"), rp, rp), `PRINT[SQR(ABS((-#4)))]`, false},
		{"no parentheses", toks(pr, fun("SIN"), number("1")), `PRINT[BAD(SYNTAX)]`, true},
		{"two arguments", toks(pr, fun("SIN"), lp, number("1"), comma, number("2"), rp), `PRINT[BAD(SYNTAX)]`, true},
		{"no )", toks(pr, fun("SIN"), lp, number("1")), `PRINT[BAD(SYNTAX)]`, true},
		{"no argument", toks(pr, fun("RND"), lp, rp), `PRINT[BAD(SYNTAX)]`, true},
	})
}

// @spec PARSER-067
func TestPi(t *testing.T) {
	runParseCases(t, []parseCase{
		{"pi", toks(pr, token.Token{Kind: token.Pi, Value: "π"}), `PRINT[#3.141592653589793]`, false},
	})
}

// @spec PARSER-068
func TestStringFunctionArity(t *testing.T) {
	runParseCases(t, []parseCase{
		{"LEN", toks(pr, fun("LEN"), lp, name("A$"), rp), `PRINT[LEN($A$[A$])]`, false},
		{"LEFT$", toks(pr, fun("LEFT$"), lp, name("A$"), comma, number("2"), rp), `PRINT[LEFT$($A$[A$],#2)]`, false},
		{"MID$ 2", toks(pr, fun("MID$"), lp, name("A$"), comma, number("2"), rp), `PRINT[MID$($A$[A$],#2)]`, false},
		{"MID$ 3", toks(pr, fun("MID$"), lp, name("A$"), comma, number("2"), comma, number("1"), rp), `PRINT[MID$($A$[A$],#2,#1)]`, false},
		{"MID$ 1", toks(pr, fun("MID$"), lp, name("A$"), rp), `PRINT[BAD(SYNTAX)]`, true},
		{"LEFT$ 1", toks(pr, fun("LEFT$"), lp, name("A$"), rp), `PRINT[BAD(SYNTAX)]`, true},
		{"CHR$ 2", toks(pr, fun("CHR$"), lp, number("65"), comma, number("1"), rp), `PRINT[BAD(SYNTAX)]`, true},
	})
}

// @spec PARSER-069
func TestTabSpc(t *testing.T) {
	tab := token.Token{Kind: token.Tab, Value: "TAB("}
	spc := token.Token{Kind: token.Spc, Value: "SPC("}
	runParseCases(t, []parseCase{
		{"TAB", toks(pr, tab, number("5"), rp, str("X")), `PRINT[TAB(#5) "X"]`, false},
		{"SPC", toks(pr, str("A"), spc, name("N"), plus, number("1"), rp, str("B")), `PRINT["A" SPC(($N[N]+#1)) "B"]`, false},
		{"PRINT#", toks(prf, number("1"), comma, tab, number("3"), rp), `PRINT##1[TAB(#3)]`, false},
		{"no )", toks(pr, tab, number("5")), `PRINT[BAD(SYNTAX)]`, true},
		{"outside PRINT", toks(name("A"), eq, tab, number("5"), rp), `BADSTMT`, true},
	})
}

// @spec PARSER-070
func TestPos(t *testing.T) {
	runParseCases(t, []parseCase{
		{"POS", toks(pr, fun("POS"), lp, number("0"), rp), `PRINT[POS(#0)]`, false},
		{"two arguments", toks(pr, fun("POS"), lp, number("0"), comma, number("1"), rp), `PRINT[BAD(SYNTAX)]`, true},
	})
}

// @spec PARSER-071
func TestArrayElements(t *testing.T) {
	runParseCases(t, []parseCase{
		{"used", toks(pr, name("A"), lp, number("1"), rp), `PRINT[$A[A](#1)]`, false},
		{"two subscripts", toks(pr, name("B$"), lp, name("I"), comma, name("J"), plus, number("1"), rp), `PRINT[$B$[B$]($I[I],($J[J]+#1))]`, false},
		{"assigned", toks(name("A"), lp, number("1"), rp, eq, number("2")), `LET $A[A](#1)=#2`, false},
		{"INPUT", toks(input, name("A"), lp, name("I"), rp), `INPUT $A[A]($I[I])`, false},
		{"no )", toks(pr, name("A"), lp, number("1")), `PRINT[BAD(SYNTAX)]`, true},
		{"empty", toks(pr, name("A"), lp, rp), `PRINT[BAD(SYNTAX)]`, true},
	})
}

// @spec PARSER-072
func TestArrayLoopVariable(t *testing.T) {
	runParseCases(t, []parseCase{
		{"FOR", toks(fork, name("A"), lp, number("1"), rp, eq, number("1"), tok, number("2")), `BADSTMT`, true},
		{"NEXT", toks(nextk, name("A"), lp, number("1"), rp), `BADSTMT`, true},
	})
}

// @spec PARSER-073
func TestDim(t *testing.T) {
	runParseCases(t, []parseCase{
		{"one", toks(dim, name("A"), lp, number("10"), rp), `DIM $A[A](#10)`, false},
		{"several", toks(dim, name("A"), lp, number("10"), rp, comma, name("B$"), lp, number("3"), comma, number("4"), rp), `DIM $A[A](#10),$B$[B$](#3,#4)`, false},
		{"plain variable", toks(dim, name("X")), `DIM $X[X]`, false},
		{"nothing", toks(dim), `BADSTMT`, true},
		{"junk", toks(dim, name("A"), lp, number("1"), rp, number("2")), `BADSTMT`, true},
	})
}

// @spec PARSER-074
func TestReservedKeywords(t *testing.T) {
	res := func(v string) token.Token { return token.Token{Kind: token.Reserved, Value: v} }
	runParseCases(t, []parseCase{
		{"statement", toks(res("POKE"), number("1"), comma, number("2")), `BADSTMT`, true},
		{"operand", toks(pr, res("PEEK"), lp, number("1"), rp), `PRINT[BAD(SYNTAX)]`, true},
		{"assignment", toks(name("A"), eq, res("FRE"), lp, number("0"), rp), `BADSTMT`, true},
	})
}
