package interp

import (
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
)

// StoredFile is a file's name and size in bytes.
type StoredFile struct {
	Name string
	Size int64
}

// A disk drive's status messages, as a 1541 words them.
const (
	powerOn     = "73,CBM DOS V2.6 1541,00,00"
	statusOK    = "00, OK,00,00"
	notFound    = "62,FILE NOT FOUND,00,00"
	fileExists  = "63,FILE EXISTS,00,00"
	writeProt   = "26,WRITE PROTECT ON,00,00"
	badCommand  = "31,SYNTAX ERROR,00,00"
	badName     = "33,SYNTAX ERROR,00,00"
	noName      = "34,SYNTAX ERROR,00,00"
	blocksFree  = 664 // an empty 1541 disk
	blockBytes  = 254 // data bytes in a 1541 block
	programType = ".bas"
)

// driveStatus returns a disk drive's status, which starts as the message
// a 1541 gives after it is switched on.
func (in *Interp) driveStatus(device int) string {
	if s, ok := in.drives[device]; ok {
		return s
	}
	return powerOn
}

// setStatus sets a disk drive's status.
func (in *Interp) setStatus(device int, status string) {
	in.drives[device] = status
}

// takeStatus returns a disk drive's status for reading, which clears it,
// as a 1541 clears its status once read.
//
// @spec INTERP-167
func (in *Interp) takeStatus(device int) string {
	s := in.driveStatus(device)
	in.setStatus(device, statusOK)
	return s
}

// runCommand runs a disk command sent to a drive's command channel and
// sets the drive's status.
//
// @spec INTERP-169, INTERP-170, INTERP-171
func (in *Interp) runCommand(device int, cmd string) error {
	status, err := in.command(cmd)
	in.setStatus(device, status)
	return err
}

// command runs a disk command and returns the drive's status after it.
func (in *Interp) command(cmd string) (string, error) {
	switch {
	case cmd == "":
		return statusOK, nil
	case cmd == "UJ" || cmd == "UI" || cmd == "U:" || cmd == "U9":
		return powerOn, nil
	case cmd == "I" || cmd == "I0" || cmd == "V" || cmd == "V0":
		return statusOK, nil
	case cmd[0] == 'N':
		return writeProt, nil // a 1541 cannot format a protected disk
	case cmd[0] == 'S':
		names, ok := commandArgs(cmd)
		if !ok {
			return noName, nil
		}
		return in.scratch(strings.Split(names, ","))
	case cmd[0] == 'R':
		names, ok := commandArgs(cmd)
		newName, oldName, found := strings.Cut(names, "=")
		if !ok || !found || newName == "" || oldName == "" {
			return noName, nil
		}
		return in.rename(newName, oldName)
	}
	return badCommand, nil
}

// commandArgs returns what follows the ":" of a command such as S0:NAME,
// and whether there is something there.
func commandArgs(cmd string) (string, bool) {
	_, args, found := strings.Cut(cmd, ":")
	return args, found && args != ""
}

// scratch deletes the files matching any of the names.
func (in *Interp) scratch(names []string) (string, error) {
	files, err := in.storage.Files()
	if err != nil {
		return "", &StorageError{File: "$", Err: err}
	}
	count := 0
	for _, f := range files {
		for _, name := range names {
			if in.matches(name, f.Name, files) {
				if err := in.storage.Remove(f.Name); err != nil {
					return "", &StorageError{File: f.Name, Err: err}
				}
				count++
				break
			}
		}
	}
	return fmt.Sprintf("01, FILES SCRATCHED,%02d,00", count), nil
}

// rename renames the file oldName names to newName, adding the program
// extension to newName when oldName named a program without its
// extension.
func (in *Interp) rename(newName, oldName string) (string, error) {
	if strings.ContainsAny(newName+oldName, "*?") {
		return badName, nil
	}
	files, err := in.storage.Files()
	if err != nil {
		return "", &StorageError{File: oldName, Err: err}
	}
	resolved, ok := resolve(oldName, files)
	if !ok {
		return notFound, nil
	}
	if resolved != oldName && !strings.Contains(newName, ".") {
		newName += programType
	}
	err = in.storage.Rename(resolved, newName)
	switch {
	case errors.Is(err, fs.ErrExist):
		return fileExists, nil
	case errors.Is(err, fs.ErrNotExist):
		return notFound, nil
	case err != nil:
		return "", &StorageError{File: resolved, Err: err}
	}
	return statusOK, nil
}

// resolve returns the file a name without wildcards names: the file of
// that name, or else, for a name without an extension, the program of
// that name.
func resolve(name string, files []StoredFile) (string, bool) {
	has := func(n string) bool {
		for _, f := range files {
			if f.Name == n {
				return true
			}
		}
		return false
	}
	switch {
	case has(name):
		return name, true
	case !strings.Contains(name, ".") && has(name+programType):
		return name + programType, true
	}
	return "", false
}

// matches reports whether pattern names file: as a 1541 matches, "?"
// matching any one character and "*" the rest of the name; a pattern
// without wildcards names a file as resolve finds it.
//
// @spec INTERP-172
func (in *Interp) matches(pattern, file string, files []StoredFile) bool {
	if !strings.ContainsAny(pattern, "*?") {
		resolved, ok := resolve(pattern, files)
		return ok && resolved == file
	}
	p, f := []rune(pattern), []rune(file)
	for i, c := range p {
		switch {
		case c == '*':
			return true
		case i >= len(f):
			return false
		case c != '?' && c != f[i]:
			return false
		}
	}
	return len(p) == len(f)
}

// directoryPattern reports whether a LOAD name asks for the directory,
// "$" or "$0" with an optional ":PATTERN", and returns the pattern ("*"
// for all).
func directoryPattern(name string) (string, bool) {
	head, pattern, found := strings.Cut(name, ":")
	if head != "$" && head != "$0" {
		return "", false
	}
	if !found || pattern == "" {
		pattern = "*"
	}
	return pattern, true
}

// directory returns the listing of the files matching pattern, as the
// program a 1541 sends for LOAD "$": its lines are in the drive's order,
// numbered with block counts, which may repeat or decrease, as on a C64.
//
// @spec INTERP-173
func (in *Interp) directory(pattern string) ([]progLine, error) {
	files, err := in.storage.Files()
	if err != nil {
		return nil, &StorageError{File: "$", Err: err}
	}
	quoted := func(name string) string {
		return `"` + name + `"` + strings.Repeat(" ", max(16-len([]rune(name)), 0))
	}
	header := `"` + "C64SH" + strings.Repeat(" ", 16-len("C64SH")) + `"`
	prog := []progLine{newProgLine(0, "\x12"+header+" 00 2A")}
	for _, f := range files {
		if !in.matches(pattern, f.Name, files) {
			continue
		}
		blocks := max(int((f.Size+blockBytes-1)/blockBytes), 1)
		kind := "SEQ"
		if strings.HasSuffix(f.Name, programType) {
			kind = "PRG"
		}
		lead := strings.Repeat(" ", max(4-len(strconv.Itoa(blocks)), 0))
		prog = append(prog, newProgLine(blocks, lead+quoted(f.Name)+" "+kind))
	}
	return append(prog, newProgLine(blocksFree, "BLOCKS FREE.")), nil
}
