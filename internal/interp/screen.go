package interp

import (
	"fmt"
	"unicode/utf8"
)

// pepto is the C64's 16 colors as measured on real hardware (the "Pepto"
// palette), indexed by color number.
var pepto = [16][3]int{
	{0, 0, 0}, {255, 255, 255}, {104, 55, 43}, {112, 164, 178},
	{111, 61, 134}, {88, 141, 67}, {53, 40, 121}, {184, 199, 111},
	{111, 79, 37}, {67, 57, 0}, {154, 103, 89}, {68, 68, 68},
	{108, 108, 108}, {154, 210, 132}, {108, 94, 181}, {149, 149, 149},
}

// colorCodes maps each C64 color code to its color number.
var colorCodes = map[rune]int{
	144: 0, 5: 1, 28: 2, 159: 3, 156: 4, 30: 5, 31: 6, 158: 7,
	129: 8, 149: 9, 150: 10, 151: 11, 152: 12, 153: 13, 154: 14, 155: 15,
}

// screenCodes maps the screen control codes other than Return, cursor
// right, and the colors to their terminal escape codes.
var screenCodes = map[rune]string{
	157: "\x1b[D", 17: "\x1b[B", 145: "\x1b[A", 19: "\x1b[H",
	147: "\x1b[2J\x1b[H", 18: "\x1b[7m", 146: "\x1b[27m",
}

// SetScreen sets whether program output goes to a terminal, which gets
// the screen control codes as escape codes, and whether colors are shown
// there. Without it, output is not a terminal.
func (in *Interp) SetScreen(terminal, color bool) {
	in.terminal, in.color = terminal, color
}

// translate returns s as it should be written, with the C64's screen
// control codes translated, and the cursor column after writing it,
// starting from col.
//
// @spec INTERP-146, INTERP-147, INTERP-148, INTERP-149
func (in *Interp) translate(s string, col int) (string, int) {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		raw := s[i : i+size] // an invalid byte is written as it is
		i += size
		switch r {
		case '\n', 13, 141:
			// Every Return turns reverse off on a C64, including the one
			// ending a PRINT.
			if in.reverse {
				b, in.reverse = append(b, "\x1b[27m"...), false
			}
			b, col = append(b, '\n'), 0
		case 29:
			b, col = append(b, ' '), col+1
		default:
			if n, ok := colorCodes[r]; ok {
				if in.terminal && in.color {
					c := pepto[n]
					b = fmt.Appendf(b, "\x1b[38;2;%d;%d;%dm", c[0], c[1], c[2])
				}
				continue
			}
			esc, ok := screenCodes[r]
			if !ok {
				b, col = append(b, raw...), col+1
				continue
			}
			if !in.terminal {
				continue // left out: the column does not move
			}
			b = append(b, esc...)
			switch r {
			case 18, 146:
				in.reverse = r == 18
			case 157:
				col = max(col-1, 0)
			case 19, 147:
				col = 0
			}
		}
	}
	return string(b), col
}
