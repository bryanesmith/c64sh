package interp

import (
	"testing"

	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// @spec INTERP-139
func TestRead(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"numbers and strings", lines(`10 READ A,B$,C:PRINT A;B$;C`, `20 DATA 1,HELLO,2.5`, "RUN"), " 1 HELLO 2.5 \n", nil},
		{"across statements and lines", lines(`10 DATA 1,2:DATA 3`, `20 FOR I=1 TO 4:READ X:PRINT X;:NEXT`, `30 DATA 4`, "RUN"), " 1  2  3  4 ", nil},
		{"quoted", lines(`10 READ A$,B$:PRINT A$;"|";B$`, `20 DATA "A,B:C", " SPACED "`, "RUN"), "A,B:C| SPACED \n", nil},
		{"spaces", lines(`10 READ A$,B$:PRINT "[";A$;"][";B$;"]"`, `20 DATA  X , Y`, "RUN"), "[X ][Y]\n", nil},
		{"empty items", lines(`10 READ A,B$,C:PRINT A;"[";B$;"]";C`, `20 DATA ,,7`, "RUN"), " 0 [] 7 \n", nil},
		{"trailing comma", lines(`10 READ A,B:PRINT A;B`, `20 DATA 5,`, "RUN"), " 5  0 \n", nil},
		{"keywords are text", lines(`10 READ A$:PRINT A$`, `20 DATA PRINT`, "RUN"), "PRINT\n", nil},
		{"into an array", lines(`10 DIM N$(2):FOR I=0 TO 2:READ N$(I):NEXT:PRINT N$(2);N$(0)`, `20 DATA RED,GREEN,BLUE`, "RUN"), "BLUERED\n", nil},
		{"direct mode", lines(`10 DATA 42`, "READ X:PRINT X"), " 42 \n", nil},
		{"DATA does nothing", lines(`10 DATA 1:PRINT "OK"`, "RUN"), "OK\n", nil},
	})
}

// @spec INTERP-140
func TestOutOfData(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"none", lines(`10 READ A`, "RUN"), "", errIn(basicerr.OutOfData, 10)},
		{"used up", lines(`10 READ A,B`, `20 DATA 1`, "RUN"), "", errIn(basicerr.OutOfData, 10)},
		{"direct DATA not read", lines("DATA 1:READ A"), "", &basicerr.Error{Kind: basicerr.OutOfData}},
	})
}

// @spec INTERP-141
func TestReadBadItem(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"text into a number", lines(`10 READ A`, `90 DATA ABC`, "RUN"), "", errIn(basicerr.Syntax, 90)},
		{"after quotes", lines(`10 READ A$`, `90 DATA "AB"C`, "RUN"), "", errIn(basicerr.Syntax, 90)},
		{"overflow", lines(`10 READ A`, `90 DATA 1E99`, "RUN"), "", errIn(basicerr.Overflow, 10)},
	})
}

// @spec INTERP-142
func TestRestore(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"RESTORE", lines(`10 READ A,B:RESTORE:READ C:PRINT A;B;C`, `20 DATA 1,2`, "RUN"), " 1  2  1 \n", nil},
		{"RUN restores", lines(`10 READ A:PRINT A`, `20 DATA 7`, "RUN", "RUN"), " 7 \n 7 \n", nil},
	})
}
