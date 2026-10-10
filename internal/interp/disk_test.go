package interp

import (
	"errors"
	"strings"
	"testing"

	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// diskSession runs lines with storage holding files (name, contents
// pairs) and returns the interpreter, its output, and the last error.
func diskSession(t *testing.T, storage *memStorage, lines ...string) (*Interp, string, error) {
	t.Helper()
	rec := &recorder{}
	in := New(rec)
	in.SetStorage(storage)
	var err error
	for _, l := range lines {
		err = enter(in, l)
	}
	return in, rec.String(), err
}

// status is a line that prints drive 8's status, one part per line.
const status = `OPEN 15,8,15:INPUT#15,E,E$,T,S:CLOSE 15:PRINT E:PRINT E$:PRINT T:PRINT S`

// statusOf returns the four parts of drive 8's status as statusLine
// prints them.
func statusOf(t *testing.T, storage *memStorage, lines ...string) string {
	t.Helper()
	_, out, err := diskSession(t, storage, append(lines, "10 "+status, "RUN")...)
	if err != nil {
		t.Fatalf("%v: %v", lines, err)
	}
	return out
}

func wantStatus(e int, msg string, track int) string {
	p := func(n int) string {
		if n < 0 {
			return "-" + itoa(-n) + " \n"
		}
		return " " + itoa(n) + " \n"
	}
	return p(e) + msg + "\n" + p(track) + p(0)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for ; n > 0; n /= 10 {
		s = string(rune('0'+n%10)) + s
	}
	return s
}

// @spec INTERP-164
func TestOpenMissingDiskFile(t *testing.T) {
	st := newStorage()
	_, out, err := diskSession(t, st, `10 OPEN 2,8,2,"NOPE":INPUT#2,A$:PRINT "[";A$;"]";ST:INPUT#2,A:PRINT A;ST:CLOSE 2`, "RUN")
	if err != nil || out != "[] 66 \n 0  66 \n" {
		t.Errorf("reading a missing file: output %q, error %v; want nothing read and ST 66", out, err)
	}
	if got := statusOf(t, st, `OPEN 2,8,2,"NOPE"`); got != wantStatus(62, "FILE NOT FOUND", 0) {
		t.Errorf("status %q, want 62 FILE NOT FOUND", got)
	}
	_, _, err = diskSession(t, st, `OPEN 2,8,2,"NOPE,S,A":PRINT#2,"X":CLOSE 2`)
	if err != nil || len(st.files) != 0 {
		t.Errorf("appending to a missing file: error %v, files %v; want no error and nothing written", err, st.files)
	}
	if _, _, err := diskSession(t, newStorage(), `OPEN 2,1,0,"NOPE"`); !sameErr(err, &basicerr.Error{Kind: basicerr.FileNotFound}) {
		t.Errorf("tape: error %v, want FILE NOT FOUND, as a C64's tape reports", err)
	}
}

// @spec INTERP-165
func TestOpenExistingDiskFileToWrite(t *testing.T) {
	st := newStorage("F", "OLD\n")
	if _, _, err := diskSession(t, st, `OPEN 2,8,2,"F,S,W":PRINT#2,"NEW":CLOSE 2`); err != nil || st.files["F"] != "OLD\n" {
		t.Errorf("error %v, file %q; want no error and the file unchanged", err, st.files["F"])
	}
	if got := statusOf(t, st, `OPEN 2,8,2,"F,S,W"`); got != wantStatus(63, "FILE EXISTS", 0) {
		t.Errorf("status %q, want 63 FILE EXISTS", got)
	}
	if got := statusOf(t, st, `OPEN 2,8,2,"@0:F,S,W":PRINT#2,"NEW":CLOSE 2`); got != wantStatus(0, "OK", 0) || st.files["F"] != "NEW\n" {
		t.Errorf("replacing: status %q, file %q; want OK and the file replaced", got, st.files["F"])
	}
}

// @spec INTERP-166, INTERP-167
func TestCommandChannelStatus(t *testing.T) {
	st := newStorage()
	_, out, err := diskSession(t, st, "10 "+status, "20 "+status, "RUN")
	if err != nil || out != wantStatus(73, "CBM DOS V2.6 1541", 0)+wantStatus(0, "OK", 0) {
		t.Errorf("output %q, error %v; want the power-on message, then OK once it has been read", out, err)
	}
	_, out, _ = diskSession(t, newStorage(), `10 OPEN 15,8,15:GET#15,A$,B$:PRINT A$;B$`, "RUN")
	if out != "73\n" {
		t.Errorf("GET#: %q, want the status's characters", out)
	}
	st = newStorage("A", "")
	if _, _, err := diskSession(t, st, `OPEN 15,8,15,"S0:A":CLOSE 15`); err != nil || len(st.files) != 0 {
		t.Errorf("a command as OPEN's name: error %v, files %v; want A scratched", err, st.files)
	}
	_, _, err = diskSession(t, newStorage(), `OPEN 15,8,15`)
	if err != nil {
		t.Errorf("OPEN 15,8,15: %v", err)
	}
}

// @spec INTERP-168, INTERP-169
func TestScratch(t *testing.T) {
	cases := []struct {
		name, cmd string
		left      string
		count     int
	}{
		{"one file", `S0:A1`, "A2 B .X", 1},
		{"without the drive number", `S:A1`, "A2 B .X", 1},
		{"a * pattern", `S0:A*`, "B .X", 2},
		{"a ? pattern", `S0:?1`, "A2 B .X", 1},
		{"several names", `S0:A1,B`, "A2 .X", 2},
		{"everything but hidden files", `S0:*`, ".X", 3},
		{"nothing matching", `S0:Z`, "A1 A2 B .X", 0},
	}
	for _, c := range cases {
		st := newStorage("A1", "", "A2", "", "B", "", ".X", "")
		got := statusOf(t, st, `OPEN 15,8,15:PRINT#15,"`+c.cmd+`":CLOSE 15`)
		var left []string
		for n := range st.files {
			left = append(left, n)
		}
		if want := wantStatus(1, "FILES SCRATCHED", c.count); got != want || !sameFiles(left, c.left) {
			t.Errorf("%s: status %q, files left %v; want %q and %s", c.name, got, left, want, c.left)
		}
	}
	st := newStorage("A", "")
	diskSession(t, st, `OPEN 15,8,15:PRINT#15,"S0:";:PRINT#15,"A":CLOSE 15`)
	if len(st.files) != 0 {
		t.Errorf("a command written in parts runs at its line end: files %v", st.files)
	}
}

func sameFiles(got []string, want string) bool {
	w := strings.Fields(want)
	if len(got) != len(w) {
		return false
	}
	for _, n := range w {
		found := false
		for _, g := range got {
			found = found || g == n
		}
		if !found {
			return false
		}
	}
	return true
}

// @spec INTERP-170
func TestRename(t *testing.T) {
	st := newStorage("OLD", "X", "TAKEN", "Y")
	if got := statusOf(t, st, `OPEN 15,8,15,"R0:NEW=OLD"`); got != wantStatus(0, "OK", 0) || st.files["NEW"] != "X" {
		t.Errorf("rename: status %q, files %v", got, st.files)
	}
	if got := statusOf(t, st, `OPEN 15,8,15,"R:NEW2=GONE"`); got != wantStatus(62, "FILE NOT FOUND", 0) {
		t.Errorf("missing old name: status %q", got)
	}
	if got := statusOf(t, st, `OPEN 15,8,15,"R0:TAKEN=NEW"`); got != wantStatus(63, "FILE EXISTS", 0) || st.files["TAKEN"] != "Y" {
		t.Errorf("existing new name: status %q, files %v", got, st.files)
	}
	if got := statusOf(t, st, `OPEN 15,8,15,"R0:N*=NEW"`); got != wantStatus(33, "SYNTAX ERROR", 0) {
		t.Errorf("pattern in a rename: status %q", got)
	}
}

// @spec INTERP-171
func TestOtherCommands(t *testing.T) {
	cases := []struct {
		cmd  string
		want string
	}{
		{"N0:DISK,01", wantStatus(26, "WRITE PROTECT ON", 0)},
		{"N:DISK", wantStatus(26, "WRITE PROTECT ON", 0)},
		{"I0", wantStatus(0, "OK", 0)},
		{"I", wantStatus(0, "OK", 0)},
		{"V0", wantStatus(0, "OK", 0)},
		{"UJ", wantStatus(73, "CBM DOS V2.6 1541", 0)},
		{"S0:", wantStatus(34, "SYNTAX ERROR", 0)},
		{"R0:NEW", wantStatus(34, "SYNTAX ERROR", 0)},
		{"X", wantStatus(31, "SYNTAX ERROR", 0)},
	}
	for _, c := range cases {
		st := newStorage("KEEP", "K")
		if got := statusOf(t, st, `OPEN 15,8,15,"`+c.cmd+`"`); got != c.want || st.files["KEEP"] != "K" {
			t.Errorf("%s: status %q, files %v; want %q and nothing changed", c.cmd, got, st.files, c.want)
		}
	}
}

// @spec INTERP-172
func TestCommandNamesFindPrograms(t *testing.T) {
	st := newStorage("P.bas", "10 END\n", "Q", "data")
	statusOf(t, st, `OPEN 15,8,15,"R0:R=P"`)
	if _, ok := st.files["R.bas"]; !ok {
		t.Errorf("rename of a program without its extension: files %v, want R.bas", st.files)
	}
	statusOf(t, st, `OPEN 15,8,15,"S0:R"`)
	statusOf(t, st, `OPEN 15,8,15,"S0:Q"`)
	if len(st.files) != 0 {
		t.Errorf("scratch by name with and without .bas: files %v, want none", st.files)
	}
}

// @spec INTERP-173
func TestDirectory(t *testing.T) {
	st := newStorage("A.bas", "0123456789", "DATA", strings.Repeat("x", 600), ".HIDDEN", "")
	in, out, err := diskSession(t, st, "X=1", `LOAD "$",8`, "LIST", "PRINT X")
	want := "\n0 \"C64SH           \" 00 2A" +
		"\n1    \"A.bas\"            PRG" +
		"\n3    \"DATA\"             SEQ" +
		"\n664 BLOCKS FREE.\n" + " 0 \n"
	if err != nil || out != want {
		t.Errorf("output %q, error %v; want %q", out, err, want)
	}
	if got := in.program[0].text; !strings.HasPrefix(got, "\x12") {
		t.Errorf("header %q, want it to begin with CHR$(18), reverse on", got)
	}
	_, out, _ = diskSession(t, st, `LOAD "$0:A*",8`, "LIST")
	if !strings.Contains(out, `"A.bas"`) || strings.Contains(out, `"DATA"`) {
		t.Errorf("pattern: %q, want only A.bas", out)
	}
	if _, _, err := diskSession(t, st, `LOAD "$",1`); !sameErr(err, &basicerr.Error{Kind: basicerr.FileNotFound}) {
		t.Errorf("tape: error %v, want FILE NOT FOUND ($ is only a directory on a disk drive)", err)
	}
}

// @spec INTERP-174
func TestLoadSaveSetDriveStatus(t *testing.T) {
	st := newStorage("P.bas", "10 END\n")
	if got := statusOf(t, st, `LOAD "NOPE",8`); got != wantStatus(62, "FILE NOT FOUND", 0) {
		t.Errorf("LOAD of a missing file: status %q", got)
	}
	_, _, err := diskSession(t, st, `SAVE "P",8`)
	var se *StorageError
	if !errors.As(err, &se) {
		t.Errorf("SAVE over a file: error %v, want a StorageError, as before", err)
	}
	if got := statusOf(t, st, `SAVE "P",8`); got != wantStatus(63, "FILE EXISTS", 0) {
		t.Errorf("SAVE over a file: status %q", got)
	}
	if got := statusOf(t, st, `LOAD "P",8`); got != wantStatus(0, "OK", 0) {
		t.Errorf("LOAD: status %q", got)
	}
}
