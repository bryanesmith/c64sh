package interp

import (
	"testing"

	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// checkErrs runs each line with the C64's string limit, so positions are
// bytes, and checks its error.
func checkErrs(t *testing.T, cases map[string]basicerr.Kind) {
	t.Helper()
	for l, k := range cases {
		if _, err := execC64(sline(l)); !isKind(err, k) {
			t.Errorf("%s: error %v, want kind %d", l, err, k)
		}
	}
}

// @spec INTERP-126
func TestStringFunctionArgumentErrors(t *testing.T) {
	checkErrs(t, map[string]basicerr.Kind{
		`PRINT LEN(5)`:          basicerr.TypeMismatch,
		`PRINT LEFT$(5,1)`:      basicerr.TypeMismatch,
		`PRINT LEFT$("A","B")`:  basicerr.TypeMismatch,
		`PRINT CHR$("A")`:       basicerr.TypeMismatch,
		`PRINT ASC(65)`:         basicerr.TypeMismatch,
		`PRINT STR$("A")`:       basicerr.TypeMismatch,
		`PRINT VAL(1)`:          basicerr.TypeMismatch,
		`PRINT LEFT$("A",256)`:  basicerr.IllegalQuantity,
		`PRINT RIGHT$("A",-1)`:  basicerr.IllegalQuantity,
		`PRINT MID$("A",1,300)`: basicerr.IllegalQuantity,
		`PRINT CHR$(256)`:       basicerr.IllegalQuantity,
		`PRINT CHR$(-1)`:        basicerr.IllegalQuantity,
	})
}

// @spec INTERP-127
func TestSubstrings(t *testing.T) {
	runPrintCases(t, []printCase{
		{"LEN", sline(`PRINT LEN("HELLO");LEN("");LEN("π")`), " 5  0  1 \n"},
		{"LEFT$", sline(`PRINT LEFT$("HELLO",2);"|";LEFT$("HI",9);"|";LEFT$("HI",0);"|"`), "HE|HI||\n"},
		{"RIGHT$", sline(`PRINT RIGHT$("HELLO",3);"|";RIGHT$("HI",9)`), "LLO|HI\n"},
		{"MID$", sline(`PRINT MID$("HELLO",2,3);"|";MID$("HELLO",4);"|";MID$("HI",5);"|";MID$("HELLO",2,0);"|"`), "ELL|LO|||\n"},
		{"rounded down", sline(`PRINT LEFT$("HELLO",2.9)`), "HE\n"},
	})
	checkErrs(t, map[string]basicerr.Kind{`PRINT MID$("A",0)`: basicerr.IllegalQuantity})
}

// @spec INTERP-128
func TestCharacterCodes(t *testing.T) {
	runPrintCases(t, []printCase{
		{"CHR$", sline(`PRINT CHR$(72);CHR$(73)`), "HI\n"},
		{"ASC", sline(`PRINT ASC("A");ASC("ABC")`), " 65  65 \n"},
		{"round trip", sline(`PRINT CHR$(ASC("Q")+1)`), "R\n"},
		{"quote", sline(`PRINT CHR$(34);"X";CHR$(34)`), "\"X\"\n"},
	})
	checkErrs(t, map[string]basicerr.Kind{`PRINT ASC("")`: basicerr.IllegalQuantity})
}

// @spec INTERP-129
func TestStrVal(t *testing.T) {
	runPrintCases(t, []printCase{
		{"STR$", sline(`PRINT "[";STR$(5);"][";STR$(-2.5);"][";STR$(.5);"]"`), "[ 5][-2.5][ .5]\n"},
		{"VAL", sline(`PRINT VAL("12");VAL(" 3.5 ");VAL("-7XYZ");VAL("ABC");VAL("");VAL("1E3");VAL("1 2")`), " 12  3.5 -7  0  0  1000  12 \n"},
	})
	checkErrs(t, map[string]basicerr.Kind{`PRINT VAL("1E99")`: basicerr.Overflow})
}

// @spec INTERP-156, INTERP-126
func TestStringLimit(t *testing.T) {
	run := func(limit int, src string) (string, error) {
		rec := &recorder{}
		in := New(rec)
		if limit != 0 {
			in.SetStringLimit(limit)
		}
		err := enter(in, src)
		return rec.String(), err
	}
	long := `A$="":FOR I=1 TO 300:A$=A$+"X":NEXT:`
	// No limit by default, or with -1: long strings, sliced past 255.
	for _, limit := range []int{0, -1} {
		out, err := run(limit, long+`PRINT LEN(A$);LEN(MID$(A$,290));LEN(LEFT$(A$,280));LEN(RIGHT$(A$,1000))`)
		if err != nil || out != " 300  11  280  300 \n" {
			t.Errorf("limit %d: %q, %v", limit, out, err)
		}
		_, err = run(limit, `PRINT LEFT$("A",-1)`)
		wantKind(t, "negative position", err, basicerr.IllegalQuantity)
	}
	// The C64's limit, with positions as bytes.
	_, err := run(255, long)
	wantKind(t, "limit 255", err, basicerr.StringTooLong)
	_, err = run(255, `PRINT MID$("A",256)`)
	wantKind(t, "limit 255, position 256", err, basicerr.IllegalQuantity)
	// A limit above 255 allows positions up to it.
	if out, err := run(400, long+`PRINT LEN(MID$(A$,300))`); err != nil || out != " 1 \n" {
		t.Errorf("limit 400: %q, %v", out, err)
	}
	_, err = run(400, `PRINT MID$("A",401)`)
	wantKind(t, "limit 400, position 401", err, basicerr.IllegalQuantity)
	_, err = run(10, `A$="12345678901"`)
	wantKind(t, "limit 10", err, basicerr.StringTooLong)
}
