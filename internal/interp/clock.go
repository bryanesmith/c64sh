package interp

import (
	"fmt"
	"strings"

	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// jiffiesPerDay is when the C64's clock wraps to 0: 24 hours of jiffies.
const jiffiesPerDay = 24 * 60 * 60 * 60

// ti returns TI: whole jiffies since the clock started, counting on from
// the last TI$ setting, wrapped at 24 hours.
//
// @spec INTERP-143
func (in *Interp) ti() int {
	elapsed := int(in.now().Sub(in.tiStart).Seconds() * 60)
	return (in.tiBase + elapsed) % jiffiesPerDay
}

// tiString returns TI$: TI as HHMMSS.
//
// @spec INTERP-144
func (in *Interp) tiString() string {
	s := in.ti() / 60
	return fmt.Sprintf("%02d%02d%02d", s/3600, s/60%60, s%60)
}

// setTI sets the clock from six digits, HHMMSS, as the ROM does ($A9DA).
//
// @spec INTERP-145
func (in *Interp) setTI(s string) error {
	if len(s) != 6 || strings.Trim(s, "0123456789") != "" {
		return &basicerr.Error{Kind: basicerr.IllegalQuantity}
	}
	d := func(i int) int { return int(s[i]-'0')*10 + int(s[i+1]-'0') }
	in.tiBase = ((d(0)*60+d(2))*60 + d(4)) * 60
	in.tiStart = in.now()
	return nil
}
