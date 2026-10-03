package interp

import (
	"io/fs"
	"testing"

	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// @spec INTERP-108
func TestOpenChecks(t *testing.T) {
	runFileCases(t, []fileCase{
		{"file 0", newStorage(), lines(`OPEN 0,3`), "", "-", &basicerr.Error{Kind: basicerr.NotInputFile}},
		{"already open", newStorage(), lines(`OPEN 1,3:OPEN 1,4`), "", "-", &basicerr.Error{Kind: basicerr.FileOpen}},
		{"ten files", newStorage(), lines(`FOR I=1 TO 10:OPEN I,3:NEXT:PRINT "OK"`), "OK\n", "-", nil},
		{"eleven files", newStorage(), lines(`FOR I=1 TO 11:OPEN I,3:NEXT`), "", "-", &basicerr.Error{Kind: basicerr.TooManyFiles}},
		{"file number range", newStorage(), lines(`OPEN 256,3`), "", "-", &basicerr.Error{Kind: basicerr.IllegalQuantity}},
		{"string name required", newStorage(), lines(`OPEN 1,8,2,5`), "", "-", &basicerr.Error{Kind: basicerr.TypeMismatch}},
	})
}

// @spec INTERP-109
func TestOpenDevices(t *testing.T) {
	runFileCases(t, []fileCase{
		{"screen", newStorage(), lines(`OPEN 1,3:PRINT#1,"HI"`), "HI\n", "-", nil},
		{"printer", newStorage(), lines(`OPEN 4,4:PRINT#4,"PAPER"`), "PAPER\n", "-", nil},
		{"screen is output only", newStorage(), lines(`10 OPEN 1,3:INPUT#1,A$`, "RUN"), "", "-", errIn(basicerr.NotInputFile, 10)},
		{"keyboard is input only", newStorage(), lines(`OPEN 1,0:PRINT#1,"X"`), "", "-", &basicerr.Error{Kind: basicerr.NotOutputFile}},
		{"RS-232", newStorage(), lines(`OPEN 1,2`), "", "-", &basicerr.Error{Kind: basicerr.DeviceNotPresent}},
		{"no device 7", newStorage(), lines(`OPEN 1,7`), "", "-", &basicerr.Error{Kind: basicerr.DeviceNotPresent}},
		{"command channel", newStorage(), lines(`OPEN 15,8,15`), "", "-", &basicerr.Error{Kind: basicerr.DeviceNotPresent}},
		{"no storage", nil, lines(`OPEN 1,8,2,"X,S,W"`), "", "-", &basicerr.Error{Kind: basicerr.DeviceNotPresent}},
		{"tape write then read", newStorage(), lines(`OPEN 1,1,1,"T":PRINT#1,"ON TAPE":CLOSE 1`, `10 OPEN 1,1,0,"T":INPUT#1,A$:PRINT A$`, "RUN"), "ON TAPE\n", "-", nil},
		{"disk modes", newStorage(), lines(`OPEN 2,8,2,"D,S,W":PRINT#2,"ONE":CLOSE 2`, `OPEN 2,8,2,"D,S,A":PRINT#2,"TWO":CLOSE 2`, `10 OPEN 2,8,2,"D,S,R":INPUT#2,A$,B$:PRINT A$;B$`, "RUN"), "ONETWO\n", "-", nil},
		{"disk secondary 1 writes", newStorage(), lines(`OPEN 2,8,1,"S1":PRINT#2,"X":CLOSE 2`, `10 OPEN 2,8,0,"S1":INPUT#2,A$:PRINT A$`, "RUN"), "X\n", "-", nil},
		{"read mode by default", newStorage("PLAIN", "Q\n"), lines(`10 OPEN 2,8,2,"PLAIN":INPUT#2,A$:PRINT A$`, "RUN"), "Q\n", "-", nil},
	})
}

// @spec INTERP-110
func TestOpenFileNames(t *testing.T) {
	s := newStorage("OLD", "x\n")
	runFileCases(t, []fileCase{
		{"name required", s, lines(`OPEN 2,8,2,""`), "", "-", &basicerr.Error{Kind: basicerr.MissingFileName}},
		{"tape name required", s, lines(`OPEN 1,1,1`), "", "-", &basicerr.Error{Kind: basicerr.MissingFileName}},
		{"no bas for data", s, lines(`OPEN 2,8,2,"DATA,S,W":CLOSE 2`), "", "-", nil},
		{"missing to read", s, lines(`OPEN 2,8,2,"NONE,S,R"`), "", "-", &basicerr.Error{Kind: basicerr.FileNotFound}},
		{"missing to append", s, lines(`OPEN 2,8,2,"NONE,S,A"`), "", "-", &basicerr.Error{Kind: basicerr.FileNotFound}},
		{"disk keeps existing file", s, lines(`OPEN 2,8,2,"OLD,S,W"`), "", "-", &StorageError{File: "OLD", Err: fs.ErrExist, Replace: `"@0:OLD,S,W"`}},
		{"replace with @0:", s, lines(`OPEN 2,8,2,"@0:OLD,S,W":PRINT#2,"NEW":CLOSE 2`), "", "-", nil},
		{"drive prefix", s, lines(`OPEN 2,8,2,"0:ZERO,S,W":CLOSE 2`), "", "-", nil},
		{"tape replaces", s, lines(`OPEN 1,1,1,"OLD":PRINT#1,"TAPE":CLOSE 1`), "", "-", nil},
	})
	want := map[string]string{"DATA": "", "OLD": "TAPE\n", "ZERO": ""}
	for f, w := range want {
		if got, ok := s.files[f]; !ok || got != w {
			t.Errorf("%s = %q, %v; want %q", f, got, ok, w)
		}
	}
}

// @spec INTERP-111
func TestClose(t *testing.T) {
	s := newStorage()
	runFileCases(t, []fileCase{
		{"written at close", s, lines(`OPEN 2,8,2,"W,S,W":PRINT#2,"A";:PRINT#2,"B"`), "", "-", nil},
		{"not open", s, lines(`CLOSE 9:PRINT "OK"`), "OK\n", "-", nil},
		{"reopen after close", s, lines(`OPEN 1,3:CLOSE 1:OPEN 1,3:PRINT "OK"`), "OK\n", "-", nil},
		{"CMD ends", s, lines(`OPEN 3,8,3,"C,S,W":CMD 3:CLOSE 3:PRINT "SCREEN"`), "SCREEN\n", "-", nil},
		{"write fails", &memStorage{files: map[string]string{}, writeErr: fs.ErrPermission}, lines(`OPEN 2,8,2,"F,S,W":CLOSE 2`), "", "-", &StorageError{File: "F", Err: fs.ErrPermission}},
	})
	if _, ok := s.files["W"]; ok {
		t.Errorf("W written before CLOSE: %q", s.files["W"])
	}
	if s.files["C"] != "\n" {
		t.Errorf("C = %q, want a newline from CMD", s.files["C"])
	}
}

// @spec INTERP-112
func TestFilesClosedWithVariables(t *testing.T) {
	s := newStorage()
	rec := &recorder{}
	in := New(rec)
	in.SetStorage(s)
	for _, l := range lines(`OPEN 2,8,2,"R,S,W":PRINT#2,"BY RUN"`, "RUN", `OPEN 3,8,3,"F,S,W":PRINT#3,"BY CLOSEFILES"`) {
		if err := enter(in, l); err != nil {
			t.Fatal(err)
		}
	}
	if s.files["R"] != "BY RUN\n" {
		t.Errorf("R = %q, want it written by RUN", s.files["R"])
	}
	if err := enter(in, `PRINT#2,"X"`); !sameErr(err, &basicerr.Error{Kind: basicerr.FileNotOpen}) {
		t.Errorf("file 2 after RUN: %v, want FILE NOT OPEN", err)
	}
	if err := in.CloseFiles(); err != nil || s.files["F"] != "BY CLOSEFILES\n" {
		t.Errorf("CloseFiles: %v, F = %q", err, s.files["F"])
	}
}

// @spec INTERP-113
func TestPrintFile(t *testing.T) {
	s := newStorage()
	runFileCases(t, []fileCase{
		{"items", s, lines(`OPEN 2,8,2,"P,S,W":PRINT#2,"A";1;"B":PRINT#2,"C";:PRINT#2:PRINT#2,-5:CLOSE 2`), "", "-", nil},
		{"not open", s, lines(`PRINT#7,"X"`), "", "-", &basicerr.Error{Kind: basicerr.FileNotOpen}},
		{"input only", newStorage("R", ""), lines(`OPEN 2,8,2,"R":PRINT#2,"X"`), "", "-", &basicerr.Error{Kind: basicerr.NotOutputFile}},
	})
	if want := "A 1 B\nC\n-5 \n"; s.files["P"] != want {
		t.Errorf("P = %q, want %q", s.files["P"], want)
	}
}

// @spec INTERP-114
func TestPrintFileComma(t *testing.T) {
	s := newStorage()
	runFileCases(t, []fileCase{
		{"zones from the screen column", s, lines(`OPEN 2,8,2,"Z,S,W":PRINT#2,"A","B":PRINT "XYZ";:PRINT#2,"C","D":CLOSE 2:PRINT`), "XYZ\n", "-", nil},
	})
	if want := "A          B\nC       D\n"; s.files["Z"] != want {
		t.Errorf("Z = %q, want %q", s.files["Z"], want)
	}
}

// @spec INTERP-115
func TestCmd(t *testing.T) {
	s := newStorage()
	runFileCases(t, []fileCase{
		{"redirects PRINT and LIST", s, lines(`10 REM LISTED`, `OPEN 3,8,3,"C,S,W":CMD 3,"FIRST":PRINT "SECOND":LIST`, `PRINT "STILL"`, `PRINT#3,"LAST":PRINT "SCREEN":CLOSE 3`), "SCREEN\n", "-", nil},
		{"error ends CMD", s, lines(`OPEN 4,8,4,"E,S,W":CMD 4:PRINT 1/0`, `PRINT "SCREEN"`, `CLOSE 4`), "SCREEN\n", "-", nil},
		{"not open", s, lines(`CMD 9`), "", "-", &basicerr.Error{Kind: basicerr.FileNotOpen}},
		{"to the printer", s, lines(`10 REM`, `OPEN 4,4:CMD 4:LIST`), "\n\n10 REM\n", "-", nil},
	})
	if want := "FIRST\nSECOND\n\n10 REM LISTED\nSTILL\nLAST\n"; s.files["C"] != want {
		t.Errorf("C = %q, want %q", s.files["C"], want)
	}
	if s.files["E"] != "\n" {
		t.Errorf("E = %q, want only the newline from CMD", s.files["E"])
	}
}

// @spec INTERP-116
func TestInputFile(t *testing.T) {
	runFileCases(t, []fileCase{
		{"values", newStorage("F", "ALICE,12\n3.5\n"), lines(`10 OPEN 2,8,2,"F":INPUT#2,N$,A:INPUT#2,B:PRINT N$;A;B`, "RUN"), "ALICE 12  3.5 \n", "-", nil},
		{"line ends", newStorage("F", "A\r\nB\rC\n"), lines(`10 OPEN 2,8,2,"F":INPUT#2,X$:INPUT#2,Y$:INPUT#2,Z$:PRINT X$;Y$;Z$`, "RUN"), "ABC\n", "-", nil},
		{"empty lines skipped", newStorage("F", "\n\nX\n"), lines(`10 OPEN 2,8,2,"F":INPUT#2,A$:PRINT A$`, "RUN"), "X\n", "-", nil},
		{"next line supplies values", newStorage("F", "1\n2\n"), lines(`10 OPEN 2,8,2,"F":INPUT#2,A,B:PRINT A;B`, "RUN"), " 1  2 \n", "-", nil},
		{"extra ignored silently", newStorage("F", "1,2\n3\n"), lines(`10 OPEN 2,8,2,"F":INPUT#2,A:INPUT#2,B:PRINT A;B`, "RUN"), " 1  3 \n", "-", nil},
		{"bad number", newStorage("F", "X\n"), lines(`10 OPEN 2,8,2,"F":INPUT#2,A`, "RUN"), "", "-", errIn(basicerr.FileData, 10)},
		{"end of file", newStorage("F", "1\n"), lines(`10 B=9:OPEN 2,8,2,"F":INPUT#2,A,B:PRINT A;B`, "RUN"), " 1  9 \n", "-", nil},
		{"direct mode", newStorage("F", "HI\n"), lines(`OPEN 2,8,2,"F":INPUT#2,A$:PRINT A$`), "HI\n", "-", nil},
		{"quoted", newStorage("F", "\"A,B\",C\n"), lines(`OPEN 2,8,2,"F":INPUT#2,A$,B$:PRINT A$;"|";B$`), "A,B|C\n", "-", nil},
		{"round trip", newStorage(), lines(`OPEN 2,8,2,"RT,S,W":PRINT#2,"X";",";7:CLOSE 2`, `OPEN 2,8,2,"RT":INPUT#2,A$,B:PRINT A$;B`), "X 7 \n", "-", nil},
	})
}

// @spec INTERP-117
func TestGetFile(t *testing.T) {
	runFileCases(t, []fileCase{
		{"characters and line end", newStorage("F", "AB\r\nC"), lines(`10 OPEN 2,8,2,"F":GET#2,A$,B$,C$,D$,E$:PRINT A$;B$;C$="`+"\r"+`";D$;"[";E$;"]"`, "RUN"), "AB-1 C[]\n", "-", nil},
		{"spaced #", newStorage("F", "Q"), lines(`10 OPEN 2,8,2,"F":GET #2,A$:PRINT A$`, "RUN"), "Q\n", "-", nil},
		{"number", newStorage("F", "7"), lines(`10 OPEN 2,8,2,"F":GET#2,N:PRINT N`, "RUN"), " 7 \n", "-", nil},
		{"direct mode", newStorage("F", "Q"), lines(`OPEN 2,8,2,"F":GET#2,A$`), "", "-", &basicerr.Error{Kind: basicerr.IllegalDirect}},
	})
}

// @spec INTERP-118
func TestFileReadErrors(t *testing.T) {
	runFileCases(t, []fileCase{
		{"INPUT# not open", newStorage(), lines(`INPUT#5,A`), "", "-", &basicerr.Error{Kind: basicerr.FileNotOpen}},
		{"GET# not open", newStorage(), lines(`10 GET#5,A$`, "RUN"), "", "-", errIn(basicerr.FileNotOpen, 10)},
		{"output only", newStorage(), lines(`OPEN 2,8,2,"W,S,W":INPUT#2,A`), "", "-", &basicerr.Error{Kind: basicerr.NotInputFile}},
	})
	rec := &recorder{}
	in := New(rec)
	in.SetConsole(&fakeConsole{lines: []string{"TYPED"}, keys: []string{"K"}})
	for _, l := range lines(`10 OPEN 1,0:INPUT#1,A$:GET#1,B$:PRINT A$;B$`, "RUN") {
		if err := enter(in, l); err != nil {
			t.Fatal(err)
		}
	}
	if got := rec.String(); got != "TYPED\nTYPEDK\n" {
		t.Errorf("keyboard file: output %q, want %q", got, "TYPED\nTYPEDK\n")
	}
}

// @spec INTERP-119
func TestStatus(t *testing.T) {
	runFileCases(t, []fileCase{
		{"starts at 0", newStorage(), lines(`PRINT ST`), " 0 \n", "-", nil},
		{"end of file", newStorage("F", "1\n2\n"), lines(`10 OPEN 2,8,2,"F":INPUT#2,A:PRINT ST;:INPUT#2,B:PRINT ST;:INPUT#2,C:PRINT ST`, "RUN"), " 0  64  64 \n", "-", nil},
		{"GET# last character", newStorage("F", "AB"), lines(`10 OPEN 2,8,2,"F":GET#2,A$:PRINT ST;:GET#2,A$:PRINT ST`, "RUN"), " 0  64 \n", "-", nil},
		{"reading loop", newStorage("F", "A\nB\nC\n"), lines(`10 OPEN 2,8,2,"F"`, `20 INPUT#2,A$:PRINT A$;:IF ST=0 THEN 20`, `30 CLOSE 2:PRINT`, "RUN"), "ABC\n", "-", nil},
		{"OPEN resets", newStorage("F", "1"), lines(`10 OPEN 2,8,2,"F":INPUT#2,A:OPEN 3,3:PRINT ST`, "RUN"), " 0 \n", "-", nil},
		{"ST$ is ordinary", newStorage(), lines(`ST$="X":ST%=5:PRINT ST$;ST%`), "X 5 \n", "-", nil},
	})
}
