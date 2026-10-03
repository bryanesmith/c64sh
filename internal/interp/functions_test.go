package interp

import (
	"testing"

	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// @spec INTERP-091
func TestDefFn(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"define and call", lines(`10 DEF FN SQ(X)=X*X+1`, `20 PRINT FN SQ(3)`, "RUN"), " 10 \n", nil},
		{"redefine", lines(`10 DEF FN A(X)=X:DEF FN A(X)=X*2:PRINT FN A(5)`, "RUN"), " 10 \n", nil},
		{"direct", lines(`DEF FN A(X)=X`), "", &basicerr.Error{Kind: basicerr.IllegalDirect}},
		{"string name", lines(`10 DEF FN A$(X)=X`, "RUN"), "", errIn(basicerr.TypeMismatch, 10)},
		{"string name, direct", lines(`DEF FN A$(X)=X`), "", &basicerr.Error{Kind: basicerr.TypeMismatch}},
		{"string parameter", lines(`10 DEF FN A(X$)=1`, "RUN"), "", errIn(basicerr.TypeMismatch, 10)},
		{"separate from variables", lines(`10 A=7:DEF FN A(X)=X+1:PRINT A;FN A(A)`, "RUN"), " 7  8 \n", nil},
	})
}

// @spec INTERP-092
func TestFnCalls(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"parameter is protected", lines(`10 X=100:DEF FN D(X)=X*2:PRINT FN D(4);X`, "RUN"), " 8  100 \n", nil},
		{"body sees other variables", lines(`10 K=3:DEF FN M(X)=X*K:K=5:PRINT FN M(2)`, "RUN"), " 10 \n", nil},
		{"function calls function", lines(`10 DEF FN A(X)=X+1:DEF FN B(X)=FN A(X)*2:PRINT FN B(3)`, "RUN"), " 8 \n", nil},
		{"undefined", lines(`10 PRINT FN Z(1)`, "RUN"), "", errIn(basicerr.UndefdFunction, 10)},
		{"defined later", lines(`10 PRINT FN Z(1)`, `20 DEF FN Z(X)=X`, "RUN"), "", errIn(basicerr.UndefdFunction, 10)},
		{"string argument", lines(`10 DEF FN A(X)=X:PRINT FN A("Q")`, "RUN"), "", errIn(basicerr.TypeMismatch, 10)},
		{"argument checked before definition", lines(`10 PRINT FN Z("Q")`, "RUN"), "", errIn(basicerr.TypeMismatch, 10)},
		{"string name", lines(`10 PRINT FN Z$(1)`, "RUN"), "", errIn(basicerr.TypeMismatch, 10)},
		{"string body", lines(`10 DEF FN A(X)="S":PRINT FN A(1)`, "RUN"), "", errIn(basicerr.TypeMismatch, 10)},
		{"body syntax error at call", lines(`10 DEF FN A(X)=X+:PRINT "OK"`, `20 PRINT FN A(1)`, "RUN"), "OK\n", errIn(basicerr.Syntax, 20)},
		{"called in direct mode", lines(`10 DEF FN A(X)=X*3`, "RUN", "PRINT FN A(2)"), " 6 \n", nil},
	})
}

// @spec INTERP-093
func TestFnErrorLeavesParameter(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"error in body", lines(`10 X=1:DEF FN A(X)=1/(X-5):PRINT FN A(5)`, "RUN", "PRINT X"), " 5 \n", nil},
	})
}

// @spec INTERP-094
func TestFnCallDepth(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"runaway recursion", lines(`10 DEF FN A(X)=FN A(X)`, `20 PRINT FN A(1)`, "RUN"), "", errIn(basicerr.OutOfMemory, 20)},
		{"nested arguments do not count", lines(`10 DEF FN A(X)=X+1:DEF FN B(X)=FN A(FN A(FN A(FN A(FN A(FN A(FN A(FN A(FN A(FN A(X))))))))))`, `20 PRINT FN B(0)`, "RUN"), " 10 \n", nil},
		{"nine calls in progress", lines(`10 DEF FN A(X)=FN B(X):DEF FN B(X)=FN C(X):DEF FN C(X)=FN D(X):DEF FN D(X)=FN E(X):DEF FN E(X)=FN F(X):DEF FN F(X)=FN G(X):DEF FN G(X)=FN H(X):DEF FN H(X)=FN I(X):DEF FN I(X)=X+1`, `20 PRINT FN A(0)`, "RUN"), " 1 \n", nil},
		{"ten calls in progress", lines(`10 DEF FN A(X)=FN B(X):DEF FN B(X)=FN C(X):DEF FN C(X)=FN D(X):DEF FN D(X)=FN E(X):DEF FN E(X)=FN F(X):DEF FN F(X)=FN G(X):DEF FN G(X)=FN H(X):DEF FN H(X)=FN I(X):DEF FN I(X)=FN J(X):DEF FN J(X)=X+1`, `20 PRINT FN A(0)`, "RUN"), "", errIn(basicerr.OutOfMemory, 20)},
	})
}

// @spec INTERP-095
func TestFnClearedWithVariables(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"RUN clears", lines(`10 DEF FN A(X)=X`, `20 END`, `30 PRINT FN A(1)`, "RUN", "RUN 30"), "", errIn(basicerr.UndefdFunction, 30)},
		{"storing clears", lines(`10 DEF FN A(X)=X`, "RUN", "20 REM", "PRINT FN A(1)"), "", &basicerr.Error{Kind: basicerr.UndefdFunction}},
	})
}
