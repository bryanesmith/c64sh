package interp

import (
	"errors"
	"slices"
	"testing"

	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// fakeSystem records the programs SYS runs.
type fakeSystem struct {
	calls  [][]string
	status int
	err    error
	during func() // called while the program "runs"
}

func (f *fakeSystem) Run(name string, args []string) (int, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	if f.during != nil {
		f.during()
	}
	return f.status, f.err
}

func sysSession(t *testing.T, sys *fakeSystem, lines ...string) (string, error) {
	t.Helper()
	rec := &recorder{}
	in := New(rec)
	if sys != nil {
		in.SetSystem(sys)
		if sys.during == nil {
			sys.during = func() {}
		}
	}
	var err error
	for _, l := range lines {
		err = enter(in, l)
	}
	return rec.String(), err
}

// @spec INTERP-175
func TestSysRunsProgram(t *testing.T) {
	sys := &fakeSystem{}
	if _, err := sysSession(t, sys, `A$="A B":SYS "PROG",A$,5,-2.5,""`); err != nil {
		t.Fatal(err)
	}
	if want := [][]string{{"PROG", "A B", "5", "-2.5", ""}}; len(sys.calls) != 1 || !slices.Equal(sys.calls[0], want[0]) {
		t.Errorf("calls %q, want %q", sys.calls, want)
	}
}

// @spec INTERP-176
func TestSysWithANumber(t *testing.T) {
	sys := &fakeSystem{}
	_, err := sysSession(t, sys, "A=64738:SYS A")
	if !sameErr(err, &basicerr.Error{Kind: basicerr.Syntax}) || len(sys.calls) != 0 {
		t.Errorf("error %v, calls %q; want SYNTAX and nothing run", err, sys.calls)
	}
}

// @spec INTERP-177
func TestSysAfterwards(t *testing.T) {
	out, err := sysSession(t, &fakeSystem{status: 7}, `SYS "P":PRINT ST`)
	if err != nil || out != " 7 \n" {
		t.Errorf("status: %q, %v; want ST 7", out, err)
	}
	out, _ = sysSession(t, &fakeSystem{}, `PRINT "X";:SYS "P":PRINT POS(0)`)
	if out != "X\n 0 \n" {
		t.Errorf("lines: %q; want the unfinished line ended first and column 0 after", out)
	}
	rec := &recorder{}
	in := New(rec)
	sys := &fakeSystem{}
	sys.during = func() { in.Interrupt() } // Ctrl-C while the program runs
	in.SetSystem(sys)
	enter(in, `10 SYS "P":PRINT "AFTER"`)
	if err := enter(in, "RUN"); err != nil || rec.String() != "AFTER\n" {
		t.Errorf("interrupt during the program: %q, %v; want it discarded", rec.String(), err)
	}
}

// @spec INTERP-178
func TestSysCannotStart(t *testing.T) {
	_, err := sysSession(t, &fakeSystem{err: errors.New("not found")}, `SYS "NOPE"`)
	if !sameErr(err, &basicerr.Error{Kind: basicerr.FileNotFound}) {
		t.Errorf("error %v, want FILE NOT FOUND", err)
	}
	_, err = sysSession(t, nil, `SYS "P"`)
	if !sameErr(err, &basicerr.Error{Kind: basicerr.DeviceNotPresent}) {
		t.Errorf("no System: error %v, want DEVICE NOT PRESENT", err)
	}
}
