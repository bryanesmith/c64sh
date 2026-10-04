package interp

import (
	"slices"
	"strings"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// Environment holds the environment variables ENVIRON$ reads and ENVIRON
// sets (a c64sh extension).
type Environment interface {
	Lookup(name string) (value string, ok bool)
	Set(name, value string) error
	Unset(name string) error
	List() []string // every variable as "NAME=VALUE", in any order
}

// MapEnvironment is an Environment held in a map, apart from the process's
// own environment.
type MapEnvironment map[string]string

func (m MapEnvironment) Lookup(name string) (string, bool) {
	v, ok := m[name]
	return v, ok
}

func (m MapEnvironment) Set(name, value string) error {
	m[name] = value
	return nil
}

func (m MapEnvironment) Unset(name string) error {
	delete(m, name)
	return nil
}

func (m MapEnvironment) List() []string {
	var vars []string
	for name, v := range m {
		vars = append(vars, name+"="+v)
	}
	return vars
}

// SetEnvironment sets the environment ENVIRON$ and ENVIRON use. Without
// one, they use an empty MapEnvironment.
//
// @spec INTERP-155
func (in *Interp) SetEnvironment(e Environment) {
	in.env = e
}

// environ evaluates ENVIRON$: a variable's value for a string argument,
// whatever its length, or the Nth variable by name for a number.
//
// @spec INTERP-151, INTERP-152
func (in *Interp) environ(arg ast.Expr) (value, error) {
	v, err := in.eval(arg)
	if err != nil {
		return value{}, err
	}
	if !v.isNum {
		s, _ := in.env.Lookup(v.str)
		return stringValue(s), nil
	}
	n, err := toInt16(v)
	if err != nil {
		return value{}, err
	}
	if n < 1 || n > 255 {
		return value{}, &basicerr.Error{Kind: basicerr.IllegalQuantity}
	}
	vars := in.env.List()
	slices.SortFunc(vars, func(a, b string) int {
		an, _, _ := strings.Cut(a, "=")
		bn, _, _ := strings.Cut(b, "=")
		return strings.Compare(an, bn)
	})
	if int(n) > len(vars) {
		return stringValue(""), nil
	}
	return stringValue(vars[n-1]), nil
}

// execEnviron joins ENVIRON's parts, with no length limit, and sets the
// variable the text names, or removes it if the value is empty.
//
// @spec INTERP-153, INTERP-154
func (in *Interp) execEnviron(s *ast.EnvironStmt) error {
	var b strings.Builder
	for _, part := range s.Parts {
		str, err := in.evalString(part)
		if err != nil {
			return err
		}
		b.WriteString(str)
	}
	name, val, ok := strings.Cut(b.String(), "=")
	if !ok || name == "" {
		return &basicerr.Error{Kind: basicerr.IllegalQuantity}
	}
	var err error
	if val == "" {
		err = in.env.Unset(name)
	} else {
		err = in.env.Set(name, val)
	}
	if err != nil {
		return &basicerr.Error{Kind: basicerr.IllegalQuantity}
	}
	return nil
}
