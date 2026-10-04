package tutorial_test

import (
	"os"
	"strings"
	"testing"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/lexer"
	"github.com/bryanesmith/c64sh/internal/parser"
)

// @spec TUTORIAL-005
func TestIdiomsParse(t *testing.T) {
	data, err := os.ReadFile("../../docs/idioms.md")
	if err != nil {
		t.Fatal(err)
	}
	blocks := basicBlocks(string(data))
	if len(blocks) == 0 {
		t.Fatal("docs/idioms.md has no basic blocks")
	}
	for _, b := range blocks {
		for _, line := range strings.Split(strings.TrimRight(b, "\n"), "\n") {
			text := line
			if _, rest, ok, err := lexer.LineNumber(line); ok {
				if err != nil {
					t.Errorf("idioms: %q: bad line number", line)
					continue
				}
				text = rest
			}
			tree, err := parser.Parse(lexer.Lex(text))
			if err == nil {
				for _, s := range tree.Statements {
					if d, ok := s.(*ast.DefStmt); ok && d.BodyErr != nil {
						err = d.BodyErr
					}
				}
			}
			if err != nil {
				t.Errorf("idioms: %q is not valid c64sh BASIC: %v", line, err)
			}
		}
	}
}
