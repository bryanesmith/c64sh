package interp

import (
	"testing"

	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// @spec INTERP-130
func TestTabSpc(t *testing.T) {
	runPrintCases(t, []printCase{
		{"TAB", sline(`PRINT "AB";TAB(5);"C"`), "AB   C\n"},
		{"TAB already past", sline(`PRINT "ABCDEF";TAB(3);"G"`), "ABCDEFG\n"},
		{"TAB 0", sline(`PRINT TAB(0);"X"`), "X\n"},
		{"SPC", sline(`PRINT "A";SPC(3);"B"`), "A   B\n"},
		{"SPC 0", sline(`PRINT "A";SPC(0);"B"`), "AB\n"},
		{"columns", sline(`PRINT "NAME";TAB(12);"SCORE"`), "NAME        SCORE\n"},
		{"rounded down", sline(`PRINT TAB(2.9);"X"`), "  X\n"},
		{"after numbers", sline(`PRINT 1;TAB(6);2`), " 1     2 \n"},
	})
	checkErrs(t, map[string]basicerr.Kind{
		`PRINT TAB(256)`: basicerr.IllegalQuantity,
		`PRINT SPC(-1)`:  basicerr.IllegalQuantity,
		`PRINT TAB("X")`: basicerr.TypeMismatch,
	})
	s := newStorage()
	runFileCases(t, []fileCase{
		{"file TAB counts from the screen", s, lines(`OPEN 2,8,2,"T,S,W":PRINT "XYZ";:PRINT#2,"A";TAB(5);"B":CLOSE 2:PRINT`), "XYZ\n", "-", nil},
	})
	if s.files["T"] != "A  B\n" {
		t.Errorf("T = %q, want %q", s.files["T"], "A  B\n")
	}
}

// @spec INTERP-131
func TestTabSpcEndLine(t *testing.T) {
	runPrintCases(t, []printCase{
		{"TAB last", sline(`PRINT "A";TAB(3)`), "A  "},
		{"SPC last", sline(`PRINT "A";SPC(2)`), "A  "},
	})
}

// @spec INTERP-132
func TestPos(t *testing.T) {
	runPrintCases(t, []printCase{
		{"POS", sline(`PRINT "HELLO";:PRINT POS(0)`), "HELLO 5 \n"},
		{"string argument", sline(`PRINT "AB";POS("X")`), "AB 2 \n"},
		{"start of line", sline(`PRINT POS(0)`), " 0 \n"},
	})
}
