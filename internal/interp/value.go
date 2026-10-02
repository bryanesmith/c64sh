package interp

import (
	"cmp"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// The range of C64 numbers. A C64 number is a 32-bit mantissa between 0.5
// and 1 times 2^-127 through 2^127, so maxNumber is 2^127 × (1 − 2^-32), the
// largest value it holds, and minNumber is 0.5 × 2^-127 = 2^-128, the
// smallest positive one.
const (
	maxNumber = 1<<127 - 1<<95
	minNumber = 1.0 / (1 << 128)
)

// value is the result of evaluating an expression: a string or a number.
type value struct {
	isNum bool
	num   float64
	str   string
}

func stringValue(s string) value  { return value{str: s} }
func numberValue(n float64) value { return value{isNum: true, num: n} }

// inRange returns n limited to the C64's range: OVERFLOW if its size is
// above maxNumber, and 0 if its size is below minNumber.
//
// @spec INTERP-023, INTERP-024
func inRange(n float64) (value, error) {
	size := math.Abs(n)
	if size > maxNumber {
		return value{}, &basicerr.Error{Kind: basicerr.Overflow}
	}
	if size < minNumber {
		return numberValue(0), nil
	}
	return numberValue(n), nil
}

// formatNumber returns n as the C64 prints it, without the trailing space
// PRINT adds: a sign character (" " or "-"), then n rounded to 9
// significant digits, in fixed notation from 0.01 up to 1E9 and in
// scientific notation otherwise.
//
// @spec INTERP-019
func formatNumber(n float64) string {
	if n == 0 {
		return " 0"
	}
	sign := " "
	if n < 0 {
		sign = "-"
		n = -n
	}
	// 9 significant digits, rounded: d.ddddddddE±xx.
	mantissa, expText, _ := strings.Cut(strconv.FormatFloat(n, 'e', 8, 64), "e")
	exp, _ := strconv.Atoi(expText)
	digits := strings.TrimRight(strings.Replace(mantissa, ".", "", 1), "0")

	switch {
	case exp < -2 || exp >= 9:
		m := digits[:1]
		if len(digits) > 1 {
			m += "." + digits[1:]
		}
		expSign := "+"
		if exp < 0 {
			expSign, exp = "-", -exp
		}
		return fmt.Sprintf("%s%sE%s%02d", sign, m, expSign, exp)
	case exp < 0:
		return sign + "." + strings.Repeat("0", -exp-1) + digits
	case len(digits) <= exp+1:
		return sign + digits + strings.Repeat("0", exp+1-len(digits))
	default:
		return sign + digits[:exp+1] + "." + digits[exp+1:]
	}
}

// power returns x^y the way the C64 ROM's power routine ($BF7B) does:
// anything to the power 0 is 1; 0 to any other power is 0, even a
// negative one; a negative number needs a whole-number power, and the
// result is negated when that power is odd.
//
// @spec INTERP-030, INTERP-031
func power(x, y float64) (value, error) {
	switch {
	case y == 0:
		return numberValue(1), nil
	case x == 0:
		return numberValue(0), nil
	case x < 0:
		if y != math.Trunc(y) {
			return value{}, &basicerr.Error{Kind: basicerr.IllegalQuantity}
		}
		p := math.Pow(-x, y)
		if math.Mod(y, 2) != 0 {
			p = -p
		}
		return inRange(p)
	default:
		return inRange(math.Pow(x, y))
	}
}

// compare returns -1 (true) if the relation of l to r is in rel, and 0
// (false) otherwise. Strings compare byte by byte, a prefix being less.
func compare(rel ast.Relation, l, r value) (value, error) {
	if l.isNum != r.isNum {
		return value{}, &basicerr.Error{Kind: basicerr.TypeMismatch}
	}
	var c int
	if l.isNum {
		c = cmp.Compare(l.num, r.num)
	} else {
		c = strings.Compare(l.str, r.str)
	}
	actual := ast.RelEqual
	switch {
	case c < 0:
		actual = ast.RelLess
	case c > 0:
		actual = ast.RelGreater
	}
	if rel&actual != 0 {
		return numberValue(-1), nil
	}
	return numberValue(0), nil
}
