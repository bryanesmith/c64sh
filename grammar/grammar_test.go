package grammar

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"golang.org/x/exp/ebnf"
)

func mustLoad(t *testing.T) ebnf.Grammar {
	t.Helper()
	g, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	return g
}

// @spec GRAMMAR-001, GRAMMAR-006
func TestSourceIsTheEmbeddedGrammarFile(t *testing.T) {
	data, err := os.ReadFile("c64basic.ebnf")
	if err != nil {
		t.Fatalf("reading grammar/c64basic.ebnf: %v", err)
	}
	if Source != string(data) {
		t.Errorf("Source does not match the contents of c64basic.ebnf")
	}
	if Start != "Line" {
		t.Errorf("Start = %q, want %q", Start, "Line")
	}
}

// @spec GRAMMAR-002
func TestFileBeginsWithNotationKey(t *testing.T) {
	var header []string
	for line := range strings.SplitSeq(Source, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "//") {
			break
		}
		header = append(header, trimmed)
	}
	text := strings.Join(header, "\n")
	for _, want := range []string{"|", "[", "]", "{", "}", "(", ")", `"`, "…"} {
		if !strings.Contains(text, want) {
			t.Errorf("opening comment block does not explain %q", want)
		}
	}
	lower := strings.ToLower(text)
	for _, want := range []string{"uppercase", "lowercase", "parser", "lexer"} {
		if !strings.Contains(lower, want) {
			t.Errorf("opening comment block does not mention %q", want)
		}
	}
}

// @spec GRAMMAR-003
func TestEveryRuleIsPrecededByAComment(t *testing.T) {
	ruleStart := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*=`)
	lines := strings.Split(Source, "\n")
	found := 0
	for i, line := range lines {
		m := ruleStart.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		found++
		if i == 0 || !strings.HasPrefix(strings.TrimSpace(lines[i-1]), "//") {
			t.Errorf("rule %s (line %d) is not immediately preceded by a comment line", m[1], i+1)
		}
	}
	if found == 0 {
		t.Errorf("no rules found in grammar source")
	}
}

// @spec GRAMMAR-004
func TestGrammarDefinesExactlyTheExpectedRules(t *testing.T) {
	g := mustLoad(t)
	var got []string
	for name := range g {
		got = append(got, name)
	}
	slices.Sort(got)
	want := []string{"Expression", "Line", "PrintItem", "PrintStatement", "Statement", "character", "print", "string"}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("grammar rules = %v, want %v", got, want)
	}
}

// @spec GRAMMAR-005, GRAMMAR-007
func TestLoadParsesAndVerifiesTheGrammar(t *testing.T) {
	g := mustLoad(t)
	if g[Start] == nil {
		t.Errorf("loaded grammar has no start rule %q", Start)
	}
}

// @spec GRAMMAR-007
func TestLoadReportsParseAndVerifyErrors(t *testing.T) {
	saved := Source
	t.Cleanup(func() { Source = saved })

	cases := map[string]string{
		"parse error":      "Line = ",
		"undefined rule":   "Line = Missing .",
		"unreachable rule": "Line = \"A\" .\nOther = \"B\" .",
	}
	for name, src := range cases {
		Source = src
		if _, err := Load(); err == nil {
			t.Errorf("%s: Load() returned no error for %q", name, src)
		}
	}
}
