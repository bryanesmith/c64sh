package interp

import (
	"math"
	"strings"
	"time"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// initialSeed is the RND seed of every new interpreter, so a program that
// never reseeds gets the same numbers every run, as a C64 does after it is
// switched on.
const initialSeed = 0x2545f4914f6cdd1d

// SetClock sets the clock that RND(0), TI, and TI$ read, and starts TI
// at 0 by it. Without one, it is the system clock, with TI starting at 0
// when the Interp is created.
func (in *Interp) SetClock(now func() time.Time) {
	in.clock = now
	in.tiStart, in.tiBase = now(), 0
}

// now returns the clock's time.
func (in *Interp) now() time.Time {
	if in.clock == nil {
		return time.Now()
	}
	return in.clock()
}

// call evaluates a call of a built-in function.
//
// @spec INTERP-122, INTERP-123
func (in *Interp) call(e *ast.CallExpr) (value, error) {
	if e.Name == "POS" {
		// The argument is evaluated and ignored, whatever its type ($B39E).
		// @spec INTERP-132
		if _, err := in.eval(e.Args[0]); err != nil {
			return value{}, err
		}
		if in.printing {
			return numberValue(float64(in.printColumn)), nil
		}
		return numberValue(float64(in.column)), nil
	}
	if strings.HasSuffix(e.Name, "$") || e.Name == "LEN" || e.Name == "ASC" || e.Name == "VAL" {
		return in.callString(e)
	}
	x, err := in.evalNumber(e.Args[0])
	if err != nil {
		return value{}, err
	}
	switch e.Name {
	case "ABS":
		return numberValue(math.Abs(x)), nil
	case "INT":
		return numberValue(math.Floor(x)), nil
	case "SGN":
		switch {
		case x < 0:
			return numberValue(-1), nil
		case x > 0:
			return numberValue(1), nil
		}
		return numberValue(0), nil
	case "SQR":
		if x < 0 {
			return value{}, &basicerr.Error{Kind: basicerr.IllegalQuantity}
		}
		return inRange(math.Sqrt(x))
	case "LOG":
		if x <= 0 {
			return value{}, &basicerr.Error{Kind: basicerr.IllegalQuantity}
		}
		return inRange(math.Log(x))
	case "EXP":
		return inRange(math.Exp(x))
	case "SIN":
		return inRange(math.Sin(x))
	case "COS":
		return inRange(math.Cos(x))
	case "TAN":
		// The ROM divides the sine by the cosine ($E2B4).
		if math.Cos(x) == 0 {
			return value{}, &basicerr.Error{Kind: basicerr.DivisionByZero}
		}
		return inRange(math.Tan(x))
	case "ATN":
		return inRange(math.Atan(x))
	case "RND":
		return numberValue(in.rnd(x)), nil
	}
	panic("interp: unhandled function " + e.Name)
}

// rnd returns RND(x) with the C64's rules ($E097): a negative x reseeds
// from x, 0 reseeds from the clock, and a positive x continues.
//
// @spec INTERP-124, INTERP-125
func (in *Interp) rnd(x float64) float64 {
	switch {
	case x < 0:
		in.seed = mix(math.Float64bits(x))
	case x == 0:
		in.seed = mix(uint64(in.now().UnixNano()))
	}
	in.seed += 0x9e3779b97f4a7c15
	return float64(mix(in.seed)>>11) / (1 << 53)
}

// mix scrambles the bits of z (the SplitMix64 finalizer).
func mix(z uint64) uint64 {
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}
