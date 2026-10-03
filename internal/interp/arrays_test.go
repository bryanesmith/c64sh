package interp

import (
	"testing"

	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// @spec INTERP-133
func TestDim(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"fill and read", lines(`DIM A(5):FOR I=0 TO 5:A(I)=I*I:NEXT:PRINT A(0);A(3);A(5)`), " 0  9  25 \n", nil},
		{"starts empty", lines(`DIM A(2),B$(2),C%(2):PRINT A(1);"[";B$(1);"]";C%(2)`), " 0 [] 0 \n", nil},
		{"two dimensions", lines(`DIM G(2,3):G(2,3)=7:G(0,1)=1:PRINT G(2,3);G(0,1);G(1,1)`), " 7  1  0 \n", nil},
		{"expression tops", lines(`N=4:DIM A(N*2):A(8)=1:PRINT A(8)`), " 1 \n", nil},
		{"rounded down", lines(`DIM A(2.9):A(2)=5:PRINT A(2)`), " 5 \n", nil},
		{"plain variable", lines(`DIM X:PRINT X`), " 0 \n", nil},
		{"twice", lines(`DIM A(5):DIM A(5)`), "", &basicerr.Error{Kind: basicerr.RedimdArray}},
		{"after use", lines(`A(1)=1:DIM A(20)`), "", &basicerr.Error{Kind: basicerr.RedimdArray}},
		{"negative top", lines(`DIM A(-1)`), "", &basicerr.Error{Kind: basicerr.IllegalQuantity}},
		{"in a program", lines(`10 DIM A(3):DIM A(3)`, "RUN"), "", errIn(basicerr.RedimdArray, 10)},
	})
}

// @spec INTERP-134
func TestArrayAutoDim(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"top 10", lines(`A(10)=3:PRINT A(10)`), " 3 \n", nil},
		{"top 10 exceeded", lines(`A(11)=3`), "", &basicerr.Error{Kind: basicerr.BadSubscript}},
		{"two dimensions", lines(`B(10,10)=1:PRINT B(10,10)`), " 1 \n", nil},
		{"read creates", lines(`PRINT Z(3):DIM Z(5)`), " 0 \n", &basicerr.Error{Kind: basicerr.RedimdArray}},
		{"subscript rounded down", lines(`A(2.7)=9:PRINT A(2)`), " 9 \n", nil},
		{"negative subscript", lines(`PRINT A(-1)`), "", &basicerr.Error{Kind: basicerr.IllegalQuantity}},
		{"string subscript", lines(`PRINT A("X")`), "", &basicerr.Error{Kind: basicerr.TypeMismatch}},
	})
}

// @spec INTERP-135
func TestBadSubscript(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"past the top", lines(`DIM A(3):A(4)=1`), "", &basicerr.Error{Kind: basicerr.BadSubscript}},
		{"too many subscripts", lines(`DIM A(3):A(1,1)=1`), "", &basicerr.Error{Kind: basicerr.BadSubscript}},
		{"too few subscripts", lines(`DIM A(3,3):PRINT A(1)`), "", &basicerr.Error{Kind: basicerr.BadSubscript}},
		{"in a program", lines(`10 DIM A(3):PRINT A(9)`, "RUN"), "", errIn(basicerr.BadSubscript, 10)},
	})
}

// @spec INTERP-136
func TestArraysSeparateAndTyped(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"separate from variables", lines(`A=5:A(1)=7:A%(1)=2:A$(1)="S":PRINT A;A(1);A%(1);A$(1)`), " 5  7  2 S\n", nil},
		{"integer rounds down", lines(`A%(0)=3.9:PRINT A%(0)`), " 3 \n", nil},
		{"type mismatch", lines(`A$(0)=1`), "", &basicerr.Error{Kind: basicerr.TypeMismatch}},
		{"loop over an array", lines(`DIM N$(2):N$(0)="A":N$(1)="B":N$(2)="C":FOR I=2 TO 0 STEP -1:PRINT N$(I);:NEXT:PRINT`), "CBA\n", nil},
	})
}

// @spec INTERP-137
func TestArrayMemory(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"fits", lines(`DIM A(7000):PRINT "OK"`), "OK\n", nil},
		{"too large", lines(`DIM A(8000)`), "", &basicerr.Error{Kind: basicerr.OutOfMemory}},
		{"integers are smaller", lines(`DIM A%(19000):PRINT "OK"`), "OK\n", nil},
		{"all arrays together", lines(`DIM A(4000):DIM B(4000)`), "", &basicerr.Error{Kind: basicerr.OutOfMemory}},
		{"huge", lines(`DIM A(32767,32767)`), "", &basicerr.Error{Kind: basicerr.OutOfMemory}},
	})
}

// @spec INTERP-138
func TestArraysCleared(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"RUN clears", lines(`10 PRINT A(1)`, "A(1)=5", "RUN"), " 0 \n", nil},
		{"DIM again after RUN", lines(`10 DIM A(3)`, "RUN", "RUN"), "", nil},
	})
}

// @spec INTERP-136
func TestInputIntoElements(t *testing.T) {
	runInputCases(t, []inputCase{
		{"INPUT", typed("4,5"), lines(`10 INPUT A(1),A(2):PRINT A(1)*A(2)`, "RUN"), "? 4,5\n 20 \n", nil},
		{"GET", pressed("K"), lines(`10 GET K$(3):PRINT K$(3)`, "RUN"), "K\n", nil},
	})
}
