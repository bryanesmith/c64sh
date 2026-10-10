package shell

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/bryanesmith/c64sh/internal/interp"
	"github.com/bryanesmith/c64sh/internal/lexer"
	"github.com/bryanesmith/c64sh/internal/parser"
)

// @spec SHELL-SYS-001
func TestSysRunsChildProcess(t *testing.T) {
	cases := []struct {
		name, line, out, errOut string
	}{
		{"output and status", `SYS "sh","-c","echo HI; exit 3":PRINT ST`, "HI\n 3 \n", ""},
		{"arguments unchanged, no shell", `SYS "printf","[%s]\n","A B","*","$HOME"`, "[A B]\n[*]\n[$HOME]\n", ""},
		{"a signal", `SYS "sh","-c","kill -TERM $$":PRINT ST`, " 143 \n", ""},
		{"a path", `SYS "/bin/sh","-c","exit 0":PRINT ST`, " 0 \n", ""},
		{"not found", `SYS "no-such-c64sh-program"`, "", "?FILE NOT FOUND  ERROR\n"},
	}
	for _, c := range cases {
		var stdout, stderr bytes.Buffer
		Run(Config{}, strings.NewReader(c.line+"\n"), &stdout, &stderr)
		if stdout.String() != c.out || stderr.String() != c.errOut {
			t.Errorf("%s: stdout %q, stderr %q; want %q, %q", c.name, stdout.String(), stderr.String(), c.out, c.errOut)
		}
	}
}

// @spec SHELL-SYS-002
func TestSysLeavesProgramMode(t *testing.T) {
	var log []string
	var out bytes.Buffer
	s := &session{stderr: &out, interp: interp.New(&out)}
	s.programMode = func() (func(), error) {
		log = append(log, "program mode")
		return func() { log = append(log, "restore") }, nil
	}
	s.interp.SetSystem(childSystem{s: s, stdout: &out, stderr: &out})
	tree, _ := parser.Parse(lexer.Lex(`SYS "true":PRINT "DONE"`))
	if err := s.exec(tree); err != nil {
		t.Fatal(err)
	}
	if want := []string{"program mode", "restore", "program mode", "restore"}; !slices.Equal(log, want) {
		t.Errorf("terminal modes %q, want %q (restored for the program, then program mode again)", log, want)
	}
}
