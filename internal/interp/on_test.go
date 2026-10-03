package interp

import (
	"testing"

	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// @spec INTERP-120
func TestOnJumps(t *testing.T) {
	prog := lines(`100 PRINT "A":END`, `200 PRINT "B":END`, `300 PRINT "C":END`)
	run := func(stmt string) []string { return append(append(lines(), prog...), "10 "+stmt, "RUN") }
	runSessionCases(t, []sessionCase{
		{"first", run(`ON 1 GOTO 100,200,300`), "A\n", nil},
		{"third", run(`ON 3 GOTO 100,200,300`), "C\n", nil},
		{"expression", run(`X=1:ON X+1 GOTO 100,200,300`), "B\n", nil},
		{"rounded down", run(`ON 2.9 GOTO 100,200,300`), "B\n", nil},
		{"GOSUB returns after the statement", lines(`10 ON 2 GOSUB 100,200:PRINT "BACK"`, `20 END`, `100 PRINT "A";:RETURN`, `200 PRINT "B";:RETURN`, "RUN"), "BBACK\n", nil},
		{"missing line", run(`ON 1 GOTO 999`), "", errIn(basicerr.UndefdStatement, 10)},
		{"direct mode", append(append(lines(), prog...), `ON 2 GOTO 100,200`), "B\n", nil},
		{"negative", run(`ON -1 GOTO 100`), "", errIn(basicerr.IllegalQuantity, 10)},
		{"too large", run(`ON 256 GOTO 100`), "", errIn(basicerr.IllegalQuantity, 10)},
		{"string", run(`ON "1" GOTO 100`), "", errIn(basicerr.TypeMismatch, 10)},
	})
}

// @spec INTERP-121
func TestOnFallsThrough(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"zero", lines(`10 ON 0 GOTO 100:PRINT "NEXT"`, `20 END`, `100 PRINT "NO"`, "RUN"), "NEXT\n", nil},
		{"past the list", lines(`10 ON 3 GOSUB 100,100:PRINT "NEXT"`, `20 END`, `100 PRINT "NO":RETURN`, "RUN"), "NEXT\n", nil},
	})
}
