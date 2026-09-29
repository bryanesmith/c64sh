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
	eol   = token.Token{Kind: token.EOL}
)

func str(v string) token.Token    { return token.Token{Kind: token.String, Value: v} }
func ill(v string) token.Token    { return token.Token{Kind: token.Illegal, Value: v} }
func rem(v string) token.Token    { return token.Token{Kind: token.Rem, Value: v} }
func number(v string) token.Token { return token.Token{Kind: token.Number, Value: v} }

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
		var items []string
		for _, it := range s.Items {
			items = append(items, dumpItem(it))
		}
		return "PRINT[" + strings.Join(items, " ") + "]"
	case *ast.RemStmt:
		return "REM(" + strconv.Quote(s.Text) + ")"
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
		op, ok := map[ast.Op]string{ast.Add: "+", ast.Sub: "-", ast.Mul: "*", ast.Div: "/", ast.Pow: "^"}[e.Op]
		if !ok {
			return fmt.Sprintf("<op %d>", e.Op)
		}
		return "(" + dumpExpr(e.Left) + op + dumpExpr(e.Right) + ")"
	case *ast.NegExpr:
		return "(-" + dumpExpr(e.X) + ")"
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
		{"string", toks(str("A")), ``, true},
		{"illegal", toks(ill("@")), ``, true},
		{"semicolon", toks(semi), ``, true},
		{"comma", toks(comma), ``, true},
		{"plus", toks(plus), ``, true},
		{"after a good statement", toks(pr, str("A"), colon, ill("X")), `PRINT["A"]`, true},
	})
}

// @spec PARSER-008, PARSER-010
func TestSyntaxErrorInsidePrintEndsWithBadItem(t *testing.T) {
	runParseCases(t, []parseCase{
		{"illegal as first item", toks(pr, ill("X")), `PRINT[BAD(SYNTAX)]`, true},
		{"star as first item", toks(pr, star, number("2")), `PRINT[BAD(SYNTAX)]`, true},
		{"caret as first item", toks(pr, caret, number("2")), `PRINT[BAD(SYNTAX)]`, true},
		{"dangling caret", toks(pr, number("2"), caret), `PRINT[BAD(SYNTAX)]`, true},
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
			`PRINT["A"]`, true},
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
