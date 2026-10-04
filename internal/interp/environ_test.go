package interp

import (
	"errors"
	"strings"
	"testing"

	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// envRun runs lines in an interpreter with env, and returns its output and
// the first error.
func envRun(t *testing.T, env Environment, lines ...string) (string, error) {
	t.Helper()
	rec := &recorder{}
	in := New(rec)
	if env != nil {
		in.SetEnvironment(env)
	}
	for _, l := range lines {
		if err := enter(in, l); err != nil {
			return rec.String(), err
		}
	}
	return rec.String(), nil
}

func wantKind(t *testing.T, name string, err error, kind basicerr.Kind) {
	t.Helper()
	be, ok := errors.AsType[*basicerr.Error](err)
	if !ok || be.Kind != kind {
		t.Errorf("%s: error %v, want %v", name, err, kind)
	}
}

// @spec INTERP-151
func TestEnvironByName(t *testing.T) {
	long := strings.Repeat("/X", 200) // 400 characters
	env := MapEnvironment{"HOME": "/home/c64", "PATH": long}
	out, err := envRun(t, env, `PRINT ENVIRON$("HOME")`, `PRINT "[";ENVIRON$("NOPE");"]"`, `PRINT LEN(ENVIRON$("PATH"))`, `PRINT ENVIRON$("PATH")`)
	if err != nil {
		t.Fatal(err)
	}
	if want := "/home/c64\n[]\n 400 \n" + long + "\n"; out != want {
		t.Errorf("output %q; want %q", out, want)
	}
	_, err = envRun(t, env, `P$=ENVIRON$("PATH")`)
	wantKind(t, "storing a long value", err, basicerr.StringTooLong)
	_, err = envRun(t, env, `PRINT ENVIRON$("PATH")+":"`)
	wantKind(t, "joining a long value with +", err, basicerr.StringTooLong)
}

// @spec INTERP-152
func TestEnvironByNumber(t *testing.T) {
	env := MapEnvironment{"B": "2", "A": "1", "C": ""}
	out, err := envRun(t, env, `FOR I=1 TO 4:PRINT "[";ENVIRON$(I);"]":NEXT`, `PRINT ENVIRON$(1.9)`)
	if err != nil {
		t.Fatal(err)
	}
	if want := "[A=1]\n[B=2]\n[C=]\n[]\nA=1\n"; out != want {
		t.Errorf("output %q; want %q", out, want)
	}
	for _, src := range []string{`PRINT ENVIRON$(0)`, `PRINT ENVIRON$(256)`, `PRINT ENVIRON$(-1)`} {
		_, err := envRun(t, env, src)
		wantKind(t, src, err, basicerr.IllegalQuantity)
	}
}

// @spec INTERP-153
func TestEnvironSets(t *testing.T) {
	env := MapEnvironment{"OLD": "1", "PATH": strings.Repeat("/X", 200)}
	_, err := envRun(t, env, `ENVIRON "GREETING=HELLO=WORLD"`, `ENVIRON "OLD="`, `N$="NAME":ENVIRON N$;"=";"C";"64"`, `ENVIRON "PATH=";ENVIRON$("PATH");":/OPT"`)
	if err != nil {
		t.Fatal(err)
	}
	if env["GREETING"] != "HELLO=WORLD" || env["NAME"] != "C64" {
		t.Errorf("env %v: want GREETING=HELLO=WORLD and NAME=C64", env)
	}
	if _, ok := env["OLD"]; ok {
		t.Error("ENVIRON \"OLD=\" left OLD set")
	}
	if want := strings.Repeat("/X", 200) + ":/OPT"; env["PATH"] != want {
		t.Errorf("PATH is %d characters; want %d", len(env["PATH"]), len(want))
	}
	_, err = envRun(t, env, `ENVIRON "X=";1`)
	wantKind(t, "a number part", err, basicerr.TypeMismatch)
}

// refusing is an environment that refuses every change.
type refusing struct{ MapEnvironment }

func (refusing) Set(string, string) error { return errors.New("refused") }

// @spec INTERP-154
func TestEnvironErrors(t *testing.T) {
	env := MapEnvironment{"A": "1"}
	for _, src := range []string{`ENVIRON "A"`, `ENVIRON "=1"`, `ENVIRON ""`} {
		_, err := envRun(t, env, src)
		wantKind(t, src, err, basicerr.IllegalQuantity)
	}
	if len(env) != 1 || env["A"] != "1" {
		t.Errorf("env %v: want it unchanged", env)
	}
	_, err := envRun(t, refusing{MapEnvironment{}}, `ENVIRON "B=2"`)
	wantKind(t, "refused", err, basicerr.IllegalQuantity)
}

// @spec INTERP-155
func TestEnvironDefault(t *testing.T) {
	out, err := envRun(t, nil, `ENVIRON "A=1"`, `PRINT ENVIRON$("A");ENVIRON$(1)`, `PRINT "[";ENVIRON$("HOME");"]"`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "1A=1\n[]\n" {
		t.Errorf("output %q: want the interpreter's own empty environment", out)
	}
}
