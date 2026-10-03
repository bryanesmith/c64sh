package interp

import (
	"errors"
	"io/fs"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// maxFiles is the number of logical files the C64's Kernal keeps open at
// once ($F35B).
const maxFiles = 10

// ioFile is an open logical file.
type ioFile struct {
	input, output bool
	keyboard      bool            // input from the console (device 0)
	screen        bool            // output to the program output (devices 3, 4, 5)
	file          string          // the file in storage; "" for a device
	data          string          // input: the file's contents, read at OPEN
	pos           int             // input: where reading continues in data
	out           strings.Builder // output: written to storage at CLOSE
}

// execOpen opens a logical file, following the Kernal ($F34A).
//
// @spec INTERP-108, INTERP-109, INTERP-110
func (in *Interp) execOpen(s *ast.OpenStmt) error {
	nums := []int{0, 1, 0} // file number, device, secondary address
	for i, e := range []ast.Expr{s.File, s.Device, s.Secondary} {
		if e == nil {
			continue
		}
		n, err := in.evalByte(e)
		if err != nil {
			return err
		}
		nums[i] = n
	}
	lfn, device, secondary := nums[0], nums[1], nums[2]
	name := ""
	if s.Name != nil {
		v, err := in.eval(s.Name)
		if err != nil {
			return err
		}
		if v.isNum {
			return &basicerr.Error{Kind: basicerr.TypeMismatch}
		}
		name = v.str
	}
	switch {
	case lfn == 0:
		return &basicerr.Error{Kind: basicerr.NotInputFile}
	case in.files[lfn] != nil:
		return &basicerr.Error{Kind: basicerr.FileOpen}
	case len(in.files) >= maxFiles:
		return &basicerr.Error{Kind: basicerr.TooManyFiles}
	}
	var f *ioFile
	switch {
	case device == 0:
		f = &ioFile{input: true, keyboard: true}
	case device >= 3 && device <= 5:
		f = &ioFile{output: true, screen: true}
	case device == 1 || (device >= 8 && device <= 11):
		var err error
		if f, err = in.openStorage(device, secondary, name); err != nil {
			return err
		}
	default:
		return &basicerr.Error{Kind: basicerr.DeviceNotPresent}
	}
	in.files[lfn] = f
	in.status = 0
	return nil
}

// openStorage opens a tape or disk file: read whole now, or collecting
// output to write at CLOSE.
func (in *Interp) openStorage(device, secondary int, name string) (*ioFile, error) {
	disk := device >= 8
	if in.storage == nil || (disk && secondary == 15) {
		return nil, &basicerr.Error{Kind: basicerr.DeviceNotPresent}
	}
	if name == "" {
		return nil, &basicerr.Error{Kind: basicerr.MissingFileName}
	}
	base, replace, mode := name, !disk, "R"
	if disk {
		for _, prefix := range []string{"@0:", "@:", "0:"} {
			if rest, ok := strings.CutPrefix(base, prefix); ok {
				base, replace = rest, prefix[0] == '@'
				break
			}
		}
		parts := strings.Split(base, ",")
		base = parts[0]
		switch {
		case secondary == 1:
			mode = "W"
		case secondary >= 2:
			for _, p := range parts[1:] {
				if p == "W" || p == "A" {
					mode = p
				}
			}
		}
	} else if secondary != 0 {
		mode = "W"
	}
	f := &ioFile{file: base}
	if mode == "W" {
		f.output = true
		if !replace {
			if _, err := in.storage.ReadFile(base); err == nil {
				return nil, &StorageError{File: base, Err: &fs.PathError{Op: "open", Path: base, Err: fs.ErrExist}, Replace: `"@0:` + base + `,S,W"`}
			}
		}
		return f, nil
	}
	data, err := in.storage.ReadFile(base)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, &basicerr.Error{Kind: basicerr.FileNotFound}
	case err != nil:
		return nil, &StorageError{File: base, Err: err}
	case mode == "A":
		f.output = true
		f.out.Write(data)
	default:
		f.input, f.data = true, string(data)
	}
	return f, nil
}

// execClose closes a logical file, writing its output to storage.
//
// @spec INTERP-111
func (in *Interp) execClose(s *ast.CloseStmt) error {
	lfn, err := in.evalByte(s.File)
	if err != nil {
		return err
	}
	return in.close(lfn)
}

// close closes file lfn, if it is open.
func (in *Interp) close(lfn int) error {
	f := in.files[lfn]
	if f == nil {
		return nil
	}
	delete(in.files, lfn)
	if in.cmd == lfn {
		in.cmd = 0
	}
	if f.output && f.file != "" {
		if err := in.storage.WriteFile(f.file, []byte(f.out.String()), true); err != nil {
			return &StorageError{File: f.file, Err: err}
		}
	}
	return nil
}

// CloseFiles closes every open data file, writing those opened for
// output to storage, and returns the first StorageError, if any.
//
// @spec INTERP-112
func (in *Interp) CloseFiles() error {
	return in.closeAll()
}

func (in *Interp) closeAll() error {
	var first error
	lfns := make([]int, 0, len(in.files))
	for lfn := range in.files {
		lfns = append(lfns, lfn)
	}
	slices.Sort(lfns)
	for _, lfn := range lfns {
		if err := in.close(lfn); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// outputFile returns the open file numbered by e, which must accept output.
//
// @spec INTERP-113
func (in *Interp) outputFile(e ast.Expr) (*ioFile, error) {
	lfn, err := in.evalByte(e)
	if err != nil {
		return nil, err
	}
	f := in.files[lfn]
	switch {
	case f == nil:
		return nil, &basicerr.Error{Kind: basicerr.FileNotOpen}
	case !f.output:
		return nil, &basicerr.Error{Kind: basicerr.NotOutputFile}
	}
	in.status = 0
	return f, nil
}

// inputFile returns the open file numbered by e, which must accept input.
//
// @spec INTERP-118
func (in *Interp) inputFile(e ast.Expr) (*ioFile, error) {
	lfn, err := in.evalByte(e)
	if err != nil {
		return nil, err
	}
	f := in.files[lfn]
	switch {
	case f == nil:
		return nil, &basicerr.Error{Kind: basicerr.FileNotOpen}
	case !f.input:
		return nil, &basicerr.Error{Kind: basicerr.NotInputFile}
	}
	return f, nil
}

// execCmd writes its items to a file, then leaves the file as the
// destination of screen output.
//
// @spec INTERP-115
func (in *Interp) execCmd(s *ast.CmdStmt) error {
	f, err := in.outputFile(s.File)
	if err != nil {
		return err
	}
	lfn, _ := in.evalByte(s.File)
	in.cmd = lfn
	return in.printItems(s.Items, f)
}

// cmdFile returns the file CMD sends screen output to, or nil.
func (in *Interp) cmdFile() *ioFile {
	return in.files[in.cmd]
}

// endCmd returns output to the screen, as PRINT#, INPUT#, and GET# do
// when they finish ($ABB5).
func (in *Interp) endCmd() {
	in.cmd = 0
}

// emit writes text to f, or to the screen if f is nil: through the screen
// for the screen and printers, and to the output collected for a storage
// file.
func (in *Interp) emit(f *ioFile, text string) error {
	if f == nil || f.screen {
		return in.write(text)
	}
	f.out.WriteString(text)
	return nil
}

// readLine returns the next line of an input file, ending at "\n",
// "\r\n", or "\r"; ok is false at the end of the file.
func (f *ioFile) readLine() (line string, ok bool) {
	if f.pos >= len(f.data) {
		return "", false
	}
	rest := f.data[f.pos:]
	end := strings.IndexAny(rest, "\r\n")
	if end < 0 {
		f.pos = len(f.data)
		return rest, true
	}
	f.pos += end + 1
	if rest[end] == '\r' && f.pos < len(f.data) && f.data[f.pos] == '\n' {
		f.pos++
	}
	return rest[:end], true
}

// readChar returns the next character of an input file, a line end being
// CHR$(13), or "" at the end of the file.
func (f *ioFile) readChar() string {
	if f.pos >= len(f.data) {
		return ""
	}
	r, size := utf8.DecodeRuneInString(f.data[f.pos:])
	f.pos += size
	switch r {
	case '\r':
		if f.pos < len(f.data) && f.data[f.pos] == '\n' {
			f.pos++
		}
		return "\r"
	case '\n':
		return "\r"
	}
	return string(r)
}

// atEnd reports whether an input file has been read to its end.
func (f *ioFile) atEnd() bool {
	return !f.keyboard && f.pos >= len(f.data)
}

// setReadStatus sets ST after reading f: 64 at the end of a file.
func (in *Interp) setReadStatus(f *ioFile) {
	in.status = 0
	if f.atEnd() {
		in.status = 64
	}
}

// execInputFile reads values from a file as INPUT reads them, without a
// prompt or echo, skipping empty lines before the first value, taking more
// values from the following lines, and failing with FILE DATA for an
// unreadable number ($ABA5).
//
// @spec INTERP-116
func (in *Interp) execInputFile(s *ast.InputStmt) error {
	f, err := in.inputFile(s.File)
	if err != nil {
		return err
	}
	defer in.endCmd()
	defer in.setReadStatus(f)
	next := func() (string, bool, error) {
		if f.keyboard {
			line, err := in.prompt("")
			return line, err == nil, err
		}
		line, ok := f.readLine()
		return line, ok, nil
	}
	line, ok, err := next()
	for ok && err == nil && line == "" {
		line, ok, err = next()
	}
	if err != nil || !ok {
		return err
	}
	i := 0
	for n, v := range s.Vars {
		if n > 0 {
			i = skipSpaces(line, i)
			if i < len(line) && line[i] == ',' {
				i++
			} else {
				if line, ok, err = next(); err != nil || !ok {
					return err
				}
				i = 0
			}
		}
		var val value
		if strings.HasSuffix(v.Name, "$") {
			val, i, ok = inputString(line, i)
		} else {
			var num float64
			num, i, ok = inputNumber(line, i)
			if ok {
				if val, err = inRange(num); err != nil {
					return err
				}
			}
		}
		if !ok {
			return &basicerr.Error{Kind: basicerr.FileData}
		}
		if err := in.assign(v, val); err != nil {
			return err
		}
	}
	return nil
}

// readFileKey reads one character for GET# from f.
//
// @spec INTERP-117
func (in *Interp) readFileKey(f *ioFile) (string, error) {
	if !f.keyboard {
		key := f.readChar()
		in.setReadStatus(f)
		return key, nil
	}
	in.status = 0
	if in.console == nil {
		return "", ErrEndOfInput
	}
	key, err := in.console.ReadKey(in.interrupted.Load)
	if err != nil {
		return "", in.consoleError(err)
	}
	return key, nil
}
