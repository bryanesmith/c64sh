package interp

import (
	"strings"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// array is an array's top subscripts and its elements, in row-major order.
type array struct {
	tops []int
	data []value
}

// memoryFree is the memory free on a C64 when it starts, which all arrays
// together may not exceed.
const memoryFree = 38911

// size returns the C64 memory an array with these tops takes, for
// elements of the identity's type ($B1D1): 5 bytes plus 2 per dimension,
// and 5, 3, or 2 bytes per number, string, or integer element.
func size(name string, tops []int) int {
	per := 5
	switch {
	case strings.HasSuffix(name, "$"):
		per = 3
	case strings.HasSuffix(name, "%"):
		per = 2
	}
	n := per
	for _, top := range tops {
		n *= top + 1
		if n > memoryFree {
			return n // large enough to fail; avoids overflow
		}
	}
	return 5 + 2*len(tops) + n
}

// create makes an array, failing with OUT OF MEMORY if all arrays would
// take more than the C64's free memory.
//
// @spec INTERP-137
func (in *Interp) create(name string, tops []int) (*array, error) {
	used := size(name, tops)
	for n, a := range in.arrays {
		used += size(n, a.tops)
	}
	if used > memoryFree {
		return nil, &basicerr.Error{Kind: basicerr.OutOfMemory}
	}
	count := 1
	for _, top := range tops {
		count *= top + 1
	}
	zero := numberValue(0)
	if strings.HasSuffix(name, "$") {
		zero = stringValue("")
	}
	a := &array{tops: tops, data: make([]value, count)}
	for i := range a.data {
		a.data[i] = zero
	}
	in.arrays[name] = a
	return a, nil
}

// subscripts evaluates subscripts or tops: whole numbers from 0 to 32767,
// rounded down ($B1B2).
func (in *Interp) subscripts(exprs []ast.Expr) ([]int, error) {
	var ns []int
	for _, e := range exprs {
		v, err := in.eval(e)
		if err != nil {
			return nil, err
		}
		n, err := toInt16(v)
		if err != nil {
			return nil, err
		}
		if n < 0 {
			return nil, &basicerr.Error{Kind: basicerr.IllegalQuantity}
		}
		ns = append(ns, int(n))
	}
	return ns, nil
}

// execDim creates the arrays it names.
//
// @spec INTERP-133
func (in *Interp) execDim(s *ast.DimStmt) error {
	for _, ref := range s.Arrays {
		if len(ref.Subs) == 0 {
			continue // a plain variable: nothing to do
		}
		tops, err := in.subscripts(ref.Subs)
		if err != nil {
			return err
		}
		if in.arrays[ref.Name] != nil {
			return &basicerr.Error{Kind: basicerr.RedimdArray}
		}
		if _, err := in.create(ref.Name, tops); err != nil {
			return err
		}
	}
	return nil
}

// element returns the array element ref names, creating its array with
// tops of 10 if it does not exist ($B1D1).
//
// @spec INTERP-134, INTERP-135
func (in *Interp) element(ref *ast.VarRef) (*value, error) {
	subs, err := in.subscripts(ref.Subs)
	if err != nil {
		return nil, err
	}
	a := in.arrays[ref.Name]
	if a == nil {
		tops := make([]int, len(subs))
		for i := range tops {
			tops[i] = 10
		}
		if a, err = in.create(ref.Name, tops); err != nil {
			return nil, err
		}
	}
	if len(subs) != len(a.tops) {
		return nil, &basicerr.Error{Kind: basicerr.BadSubscript}
	}
	i := 0
	for d, sub := range subs {
		if sub > a.tops[d] {
			return nil, &basicerr.Error{Kind: basicerr.BadSubscript}
		}
		i = i*(a.tops[d]+1) + sub
	}
	return &a.data[i], nil
}
