package interp

import (
	"errors"
	"io"
	"io/fs"
	"strconv"
	"strings"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
	"github.com/bryanesmith/c64sh/internal/lexer"
)

// Storage holds the files that LOAD, SAVE, and VERIFY use, by name.
type Storage interface {
	// ReadFile returns a file's contents, or an error satisfying
	// errors.Is(err, fs.ErrNotExist) if there is no such file.
	ReadFile(name string) ([]byte, error)
	// WriteFile writes a file. If the file exists and replace is false,
	// it writes nothing and returns an error satisfying
	// errors.Is(err, fs.ErrExist).
	WriteFile(name string, data []byte, replace bool) error
}

// StorageError is returned by Exec when storage refused or failed to read
// or write a file. It is not a BASIC error: a C64 reports these only
// through the disk drive's light and error channel.
type StorageError struct {
	File    string // the file's name in storage, such as HELLO.bas
	Err     error  // fs.ErrExist for a disk file that may not be replaced
	Replace string // for fs.ErrExist: what to write to replace the file, such as SAVE "@0:HELLO"
}

func (e *StorageError) Error() string { return e.File + ": " + e.Err.Error() }
func (e *StorageError) Unwrap() error { return e.Err }

// SetStorage sets where LOAD, SAVE, and VERIFY find files. Without one,
// they fail with DEVICE NOT PRESENT.
func (in *Interp) SetStorage(s Storage) { in.storage = s }

// SetMessages sets where the C64's tape and disk messages (SAVING NAME,
// LOADING, …) are written, for statements executed in direct mode.
// Without a writer, no messages are written.
func (in *Interp) SetMessages(w io.Writer) { in.messages = w }

// programHeader is the first line of a saved program.
const programHeader = "#!/usr/bin/env c64sh\n"

// target is the file a LOAD, SAVE, or VERIFY names.
type target struct {
	name    string // as the program gave it, for messages
	tape    bool   // device 1; otherwise a disk drive
	replace bool   // SAVE may replace an existing file
	base    string // the name without a disk prefix
}

// file returns the name of the file in storage: the base name, with
// ".bas" added if it has no extension.
func (t target) file() string {
	if strings.Contains(t.base, ".") {
		return t.base
	}
	return t.base + ".bas"
}

// target evaluates the arguments of LOAD, SAVE, or VERIFY and checks the
// device and name, as the ROM and Kernal do ($E1D4).
//
// @spec INTERP-096, INTERP-097, INTERP-098, INTERP-099
func (in *Interp) target(a ast.FileArgs) (target, error) {
	var t target
	if a.Name != nil {
		v, err := in.eval(a.Name)
		if err != nil {
			return t, err
		}
		if v.isNum {
			return t, &basicerr.Error{Kind: basicerr.TypeMismatch}
		}
		t.name = v.str
	}
	device := 1
	for i, e := range []ast.Expr{a.Device, a.Secondary} {
		if e == nil {
			continue
		}
		n, err := in.evalByte(e)
		if err != nil {
			return t, err
		}
		if i == 0 {
			device = n
		}
	}
	switch {
	case device == 0 || (device >= 3 && device <= 5):
		return t, &basicerr.Error{Kind: basicerr.IllegalDeviceNumber}
	case device != 1 && (device < 8 || device > 11), in.storage == nil:
		return t, &basicerr.Error{Kind: basicerr.DeviceNotPresent}
	case t.name == "":
		return t, &basicerr.Error{Kind: basicerr.MissingFileName}
	}
	t.tape, t.replace, t.base = device == 1, device == 1, t.name
	if !t.tape {
		for _, prefix := range []string{"@0:", "@:", "0:"} {
			if base, ok := strings.CutPrefix(t.name, prefix); ok {
				t.base, t.replace = base, prefix[0] == '@'
				break
			}
		}
	}
	return t, nil
}

// evalByte evaluates e as a number from 0 to 255, rounded down ($B79E).
func (in *Interp) evalByte(e ast.Expr) (int, error) {
	v, err := in.eval(e)
	if err != nil {
		return 0, err
	}
	n, err := toInt16(v)
	if err == nil && (n < 0 || n > 255) {
		err = &basicerr.Error{Kind: basicerr.IllegalQuantity}
	}
	return int(n), err
}

// message writes the C64's tape and disk messages, one per line, in
// direct mode when a messages writer is set.
//
// @spec INTERP-107
func (in *Interp) message(lines ...string) {
	if in.messages == nil || in.cur.line != directLine {
		return
	}
	in.FreshLine()
	for _, l := range lines {
		io.WriteString(in.messages, l+"\n")
	}
}

// execSave writes the program to its file as text.
//
// @spec INTERP-100, INTERP-101
func (in *Interp) execSave(s *ast.SaveStmt) error {
	t, err := in.target(s.FileArgs)
	if err != nil {
		return err
	}
	if t.tape {
		in.message("PRESS RECORD & PLAY ON TAPE", "OK")
	}
	in.message("SAVING " + t.name)
	var b strings.Builder
	b.WriteString(programHeader)
	for _, l := range in.program {
		b.WriteString(strconv.Itoa(l.number) + " " + l.text + "\n")
	}
	if err := in.storage.WriteFile(t.file(), []byte(b.String()), t.replace); err != nil {
		se := &StorageError{File: t.file(), Err: err}
		if errors.Is(err, fs.ErrExist) {
			se.Replace = `SAVE "@0:` + t.base + `"`
		}
		return se
	}
	return nil
}

// errNotProgram is returned by readProgram for a file that is not a
// program.
var errNotProgram = errors.New("not a program")

// readProgram finds and reads a program file, with the messages a C64
// shows while searching, and returns its lines as stored.
//
// @spec INTERP-102, INTERP-103
func (in *Interp) readProgram(t target, action string) ([]progLine, error) {
	if t.tape {
		in.message("PRESS PLAY ON TAPE", "OK")
	}
	in.message("SEARCHING FOR " + t.name)
	names := []string{t.base}
	if !strings.Contains(t.base, ".") {
		names = append(names, t.file())
	}
	var data []byte
	var err error
	for _, name := range names {
		data, err = in.storage.ReadFile(name)
		if err == nil {
			break
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, &StorageError{File: name, Err: err}
		}
	}
	if err != nil {
		return nil, &basicerr.Error{Kind: basicerr.FileNotFound}
	}
	if t.tape {
		in.message("FOUND " + t.name)
	}
	in.message(action)
	var prog []progLine
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if (i == 0 && strings.HasPrefix(line, "#!")) || strings.Trim(line, " \t") == "" {
			continue
		}
		n, rest, ok, err := lexer.LineNumber(line)
		if !ok || err != nil {
			return nil, errNotProgram
		}
		prog = storeLine(prog, n, rest)
	}
	return prog, nil
}

// execLoad replaces the program with a file's. Typed directly, it clears
// the variables and ends the line; in a running program, it keeps them and
// runs the new program from its start, as a C64 chains programs ($E1AB).
//
// @spec INTERP-104, INTERP-105
func (in *Interp) execLoad(s *ast.LoadStmt) error {
	t, err := in.target(s.FileArgs)
	if err != nil {
		return err
	}
	prog, err := in.readProgram(t, "LOADING")
	if err == errNotProgram {
		return &basicerr.Error{Kind: basicerr.Load}
	}
	if err != nil {
		return err
	}
	in.program = prog
	if in.cur.line == directLine {
		in.clr()
		return errEnd
	}
	in.stack = nil
	in.restore()
	return &jump{}
}

// execVerify compares the program with a file's.
//
// @spec INTERP-106
func (in *Interp) execVerify(s *ast.VerifyStmt) error {
	t, err := in.target(s.FileArgs)
	if err != nil {
		return err
	}
	prog, err := in.readProgram(t, "VERIFYING")
	if err != nil && err != errNotProgram {
		return err
	}
	same := err == nil && len(prog) == len(in.program)
	for i := 0; same && i < len(prog); i++ {
		same = prog[i].number == in.program[i].number && prog[i].text == in.program[i].text
	}
	if !same {
		return &basicerr.Error{Kind: basicerr.Verify}
	}
	in.message("OK")
	return nil
}
