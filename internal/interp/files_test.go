package interp

import (
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// memStorage is storage in memory.
type memStorage struct {
	files    map[string]string
	writeErr error // returned by WriteFile, if set
}

func newStorage(files ...string) *memStorage {
	m := &memStorage{files: map[string]string{}}
	for i := 0; i+1 < len(files); i += 2 {
		m.files[files[i]] = files[i+1]
	}
	return m
}

func (m *memStorage) ReadFile(name string) ([]byte, error) {
	data, ok := m.files[name]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return []byte(data), nil
}

func (m *memStorage) WriteFile(name string, data []byte, replace bool) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	if _, ok := m.files[name]; ok && !replace {
		return &fs.PathError{Op: "open", Path: name, Err: fs.ErrExist}
	}
	m.files[name] = string(data)
	return nil
}

type fileCase struct {
	name     string
	storage  *memStorage
	lines    []string
	want     string // output
	wantMsgs string // messages; "-" for no messages writer
	wantErr  error
}

func runFileCases(t *testing.T, cases []fileCase) {
	t.Helper()
	for _, c := range cases {
		rec := &recorder{}
		in := New(rec)
		if c.storage != nil {
			in.SetStorage(c.storage)
		}
		var msgs strings.Builder
		if c.wantMsgs != "-" {
			in.SetMessages(&msgs)
		}
		var err error
		for _, l := range c.lines {
			err = enter(in, l)
		}
		if got := rec.String(); got != c.want {
			t.Errorf("%s: output %q, want %q", c.name, got, c.want)
		}
		if c.wantMsgs != "-" && msgs.String() != c.wantMsgs {
			t.Errorf("%s: messages %q, want %q", c.name, msgs.String(), c.wantMsgs)
		}
		var se *StorageError
		if want, ok := c.wantErr.(*StorageError); ok {
			if !errors.As(err, &se) || se.File != want.File || se.Name != want.Name || !errors.Is(se.Err, want.Err) {
				t.Errorf("%s: error %v, want a StorageError for %s", c.name, err, want.File)
			}
		} else if !sameErr(err, c.wantErr) {
			t.Errorf("%s: error %#v, want %#v", c.name, err, c.wantErr)
		}
	}
}

const hello = "#!/usr/bin/env c64sh\n10 PRINT \"HELLO\"\n20 ?\"X\"\n"

// @spec INTERP-096
func TestFileArguments(t *testing.T) {
	s := newStorage()
	runFileCases(t, []fileCase{
		{"string name", s, lines(`SAVE 5`), "", "-", &basicerr.Error{Kind: basicerr.TypeMismatch}},
		{"device too large", s, lines(`SAVE "A",256`), "", "-", &basicerr.Error{Kind: basicerr.IllegalQuantity}},
		{"negative device", s, lines(`SAVE "A",-1`), "", "-", &basicerr.Error{Kind: basicerr.IllegalQuantity}},
		{"secondary too large", s, lines(`SAVE "A",8,300`), "", "-", &basicerr.Error{Kind: basicerr.IllegalQuantity}},
		{"string device", s, lines(`SAVE "A","8"`), "", "-", &basicerr.Error{Kind: basicerr.TypeMismatch}},
		{"device rounded down", s, lines(`10 REM`, `SAVE "R",8.9,1`, `LOAD "R"`, `LIST`), "\n10 REM\n", "-", nil},
		{"expressions", s, lines(`10 REM`, `N$="E":D=8`, `SAVE N$+"X",D`, `LOAD "EX"`, `LIST`), "\n10 REM\n", "-", nil},
	})
}

// @spec INTERP-097
func TestFileDevices(t *testing.T) {
	runFileCases(t, []fileCase{
		{"keyboard", newStorage(), lines(`SAVE "A",0`), "", "-", &basicerr.Error{Kind: basicerr.IllegalDeviceNumber}},
		{"screen", newStorage(), lines(`LOAD "A",3`), "", "-", &basicerr.Error{Kind: basicerr.IllegalDeviceNumber}},
		{"printer", newStorage(), lines(`VERIFY "A",4`), "", "-", &basicerr.Error{Kind: basicerr.IllegalDeviceNumber}},
		{"RS-232", newStorage(), lines(`SAVE "A",2`), "", "-", &basicerr.Error{Kind: basicerr.DeviceNotPresent}},
		{"no device 12", newStorage(), lines(`SAVE "A",12`), "", "-", &basicerr.Error{Kind: basicerr.DeviceNotPresent}},
		{"no storage", nil, lines(`SAVE "A"`), "", "-", &basicerr.Error{Kind: basicerr.DeviceNotPresent}},
		{"drive 11", newStorage("A.bas", hello), lines(`LOAD "A",11`, "LIST"), "\n10 PRINT \"HELLO\"\n20 PRINT\"X\"\n", "-", nil},
		{"in a program", newStorage(), lines(`10 SAVE "A",3`, "RUN"), "", "-", errIn(basicerr.IllegalDeviceNumber, 10)},
	})
}

// @spec INTERP-098
func TestMissingFileName(t *testing.T) {
	runFileCases(t, []fileCase{
		{"none", newStorage(), lines(`SAVE`), "", "-", &basicerr.Error{Kind: basicerr.MissingFileName}},
		{"empty", newStorage(), lines(`LOAD "",8`), "", "-", &basicerr.Error{Kind: basicerr.MissingFileName}},
		{"device checked first", newStorage(), lines(`LOAD "",3`), "", "-", &basicerr.Error{Kind: basicerr.IllegalDeviceNumber}},
	})
}

// @spec INTERP-099
func TestFileNames(t *testing.T) {
	s := newStorage()
	runFileCases(t, []fileCase{
		{"bas added", s, lines(`10 REM`, `SAVE "PROG"`), "", "-", nil},
		{"extension kept", s, lines(`10 REM`, `SAVE "PROG.TXT"`), "", "-", nil},
		{"drive prefix dropped", s, lines(`10 REM`, `SAVE "0:DISK",8`), "", "-", nil},
		{"tape keeps prefix", s, lines(`10 REM`, `SAVE "0:TAPE"`), "", "-", nil},
		{"replace prefix", s, lines(`10 REM`, `SAVE "@0:DISK",8`, `SAVE "@:DISK",8`), "", "-", nil},
	})
	for _, f := range []string{"PROG.bas", "PROG.TXT", "DISK.bas", "0:TAPE.bas"} {
		if _, ok := s.files[f]; !ok {
			t.Errorf("no file %q in %v", f, s.files)
		}
	}
}

// @spec INTERP-100
func TestSave(t *testing.T) {
	s := newStorage("OLD.bas", "old")
	runFileCases(t, []fileCase{
		{"program", s, lines(`20 ?"X"`, `10 PRINT  "HELLO"`, `SAVE "P",8`), "", "-", nil},
		{"empty program", s, lines(`SAVE "E",8`), "", "-", nil},
		{"tape replaces", s, lines(`10 REM NEW`, `SAVE "OLD"`), "", "-", nil},
		{"keeps going", s, lines(`10 REM`, `SAVE "Q",8:PRINT "AFTER"`), "AFTER\n", "-", nil},
	})
	want := map[string]string{
		"P.bas":   "#!/usr/bin/env c64sh\n10 PRINT  \"HELLO\"\n20 ?\"X\"\n",
		"E.bas":   "#!/usr/bin/env c64sh\n",
		"OLD.bas": "#!/usr/bin/env c64sh\n10 REM NEW\n",
	}
	for f, w := range want {
		if s.files[f] != w {
			t.Errorf("%s = %q, want %q", f, s.files[f], w)
		}
	}
}

// @spec INTERP-101
func TestSaveRefused(t *testing.T) {
	s := newStorage("OLD.bas", "old")
	runFileCases(t, []fileCase{
		{"disk keeps existing file", s, lines(`10 REM`, `SAVE "OLD",8`), "", "-", &StorageError{File: "OLD.bas", Name: "OLD", Err: fs.ErrExist}},
		{"drive prefix", s, lines(`10 REM`, `SAVE "0:OLD",8`), "", "-", &StorageError{File: "OLD.bas", Name: "0:OLD", Err: fs.ErrExist}},
		{"write fails", &memStorage{files: map[string]string{}, writeErr: fs.ErrPermission}, lines(`SAVE "X"`), "", "-", &StorageError{File: "X.bas", Name: "X", Err: fs.ErrPermission}},
	})
	if s.files["OLD.bas"] != "old" {
		t.Errorf("OLD.bas was replaced: %q", s.files["OLD.bas"])
	}
}

// @spec INTERP-102
func TestLoadFindsFile(t *testing.T) {
	runFileCases(t, []fileCase{
		{"bas added", newStorage("HI.bas", "10 REM HI\n"), lines(`LOAD "HI"`, "LIST"), "\n10 REM HI\n", "-", nil},
		{"name as given first", newStorage("HI", "10 REM PLAIN\n", "HI.bas", "10 REM BAS\n"), lines(`LOAD "HI"`, "LIST"), "\n10 REM PLAIN\n", "-", nil},
		{"extension given", newStorage("HI.TXT", "10 REM T\n"), lines(`LOAD "HI.TXT"`, "LIST"), "\n10 REM T\n", "-", nil},
		{"not found", newStorage(), lines(`10 REM KEPT`, `LOAD "NONE"`, "LIST"), "\n10 REM KEPT\n", "-", nil},
		{"not found error", newStorage(), lines(`LOAD "NONE"`), "", "-", &basicerr.Error{Kind: basicerr.FileNotFound}},
		{"no bas with extension", newStorage("A.X.bas", "10 REM\n"), lines(`LOAD "A.X"`), "", "-", &basicerr.Error{Kind: basicerr.FileNotFound}},
	})
}

// @spec INTERP-103
func TestLoadFormat(t *testing.T) {
	runFileCases(t, []fileCase{
		{"shebang, blanks, order", newStorage("P.bas", "#!/usr/bin/env c64sh\n\n20 REM B\n  \n10 REM A\r\n20 REM C\n"), lines(`LOAD "P"`, "LIST"), "\n10 REM A\n20 REM C\n", "-", nil},
		{"unnumbered line", newStorage("P.bas", "10 REM\nPRINT 1\n"), lines(`5 REM KEPT`, `LOAD "P"`), "", "-", &basicerr.Error{Kind: basicerr.Load}},
		{"program unchanged", newStorage("P.bas", "10 REM\nPRINT 1\n"), lines(`5 REM KEPT`, `LOAD "P"`, "LIST"), "\n5 REM KEPT\n", "-", nil},
		{"line too large", newStorage("P.bas", "64000 REM\n"), lines(`LOAD "P"`), "", "-", &basicerr.Error{Kind: basicerr.Load}},
		{"shebang only first", newStorage("P.bas", "10 REM\n#!/usr/bin/env c64sh\n"), lines(`LOAD "P"`), "", "-", &basicerr.Error{Kind: basicerr.Load}},
		{"saved program loads back", newStorage(), lines(`10 PRINT "A";:REM X`, `20 ?"B"`, `SAVE "R"`, "NEW", `LOAD "R"`, "RUN"), "AB\n", "-", nil},
	})
}

// @spec INTERP-104
func TestLoadDirect(t *testing.T) {
	runFileCases(t, []fileCase{
		{"clears variables", newStorage("P.bas", "10 PRINT A\n"), lines("A=5", `LOAD "P"`, "PRINT A"), " 0 \n", "-", nil},
		{"ends the line", newStorage("P.bas", "10 REM\n"), lines(`LOAD "P":PRINT "X"`), "", "-", nil},
		{"replaces the program", newStorage("P.bas", "20 REM NEW\n"), lines(`10 REM OLD`, `LOAD "P"`, "LIST"), "\n20 REM NEW\n", "-", nil},
	})
}

// @spec INTERP-105
func TestLoadChains(t *testing.T) {
	runFileCases(t, []fileCase{
		{"runs the new program, keeping variables", newStorage("PART2.bas", "10 PRINT \"PART 2:\";A\n"), lines(`10 A=42:LOAD "PART2"`, `20 PRINT "NOT REACHED"`, "RUN"), "PART 2: 42 \n", "-", nil},
		{"empties the stack", newStorage("P.bas", "10 NEXT\n"), lines(`10 FOR I=1 TO 2:LOAD "P"`, "RUN"), "", "-", errIn(basicerr.NextWithoutFor, 10)},
		{"no messages in a program", newStorage("P.bas", "10 REM\n"), lines(`10 LOAD "P",8`, "RUN"), "", "", nil},
	})
}

// @spec INTERP-106
func TestVerify(t *testing.T) {
	runFileCases(t, []fileCase{
		{"matches", newStorage(), lines(`10 PRINT "A"`, `SAVE "V"`, `VERIFY "V":PRINT "SAME"`), "SAME\n", "-", nil},
		{"differs", newStorage(), lines(`10 PRINT "A"`, `SAVE "V"`, `10 PRINT "B"`, `VERIFY "V"`), "", "-", &basicerr.Error{Kind: basicerr.Verify}},
		{"not a program", newStorage("V.bas", "PRINT 1\n"), lines(`VERIFY "V"`), "", "-", &basicerr.Error{Kind: basicerr.Verify}},
		{"not found", newStorage(), lines(`VERIFY "V"`), "", "-", &basicerr.Error{Kind: basicerr.FileNotFound}},
		{"in a program", newStorage("V.bas", "10 VERIFY \"V\":PRINT \"OK\"\n"), lines(`LOAD "V"`, "RUN"), "OK\n", "-", nil},
	})
}

// @spec INTERP-107
func TestFileMessages(t *testing.T) {
	runFileCases(t, []fileCase{
		{"disk save", newStorage(), lines(`10 REM`, `SAVE "HI",8`), "", "SAVING HI\n", nil},
		{"tape save", newStorage(), lines(`10 REM`, `SAVE "HI"`), "", "PRESS RECORD & PLAY ON TAPE\nOK\nSAVING HI\n", nil},
		{"disk load", newStorage("HI.bas", "10 REM\n"), lines(`LOAD "HI",8`), "", "SEARCHING FOR HI\nLOADING\n", nil},
		{"tape load", newStorage("HI.bas", "10 REM\n"), lines(`LOAD "HI"`), "", "PRESS PLAY ON TAPE\nOK\nSEARCHING FOR HI\nFOUND HI\nLOADING\n", nil},
		{"disk verify", newStorage(), lines(`10 REM`, `SAVE "HI",8`, `VERIFY "HI",8`), "", "SAVING HI\nSEARCHING FOR HI\nVERIFYING\nOK\n", nil},
		{"tape verify fails", newStorage("HI.bas", "10 REM X\n"), lines(`10 REM`, `VERIFY "HI"`), "", "PRESS PLAY ON TAPE\nOK\nSEARCHING FOR HI\nFOUND HI\nVERIFYING\n", &basicerr.Error{Kind: basicerr.Verify}},
		{"not found", newStorage(), lines(`LOAD "NO",8`), "", "SEARCHING FOR NO\n", &basicerr.Error{Kind: basicerr.FileNotFound}},
		{"name as given", newStorage(), lines(`10 REM`, `SAVE "@0:HI",8`), "", "SAVING @0:HI\n", nil},
		{"after unfinished output", newStorage(), lines(`10 REM`, `PRINT "A";:SAVE "HI",8`), "A\n", "SAVING HI\n", nil},
	})
}
