package interp

import (
	"strings"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// System runs the programs SYS runs (a c64sh extension).
type System interface {
	// Run runs the named program with args and returns its exit status.
	// The name is found on the PATH unless it holds a "/". An error
	// means the program could not be started.
	Run(name string, args []string) (status int, err error)
}

// SetSystem sets what runs programs for SYS. Without one, SYS fails with
// DEVICE NOT PRESENT.
func (in *Interp) SetSystem(s System) { in.system = s }

// execSys runs a program: the first value names it, and the rest are its
// arguments, strings as they are and numbers as STR$ writes them without
// the leading space. Its exit status goes in ST, and an interrupt while it
// runs (Ctrl-C, which reaches the program too) is discarded.
//
// @spec INTERP-175, INTERP-176, INTERP-177, INTERP-178
func (in *Interp) execSys(s *ast.SysStmt) error {
	var argv []string
	for i, e := range s.Args {
		v, err := in.eval(e)
		if err != nil {
			return err
		}
		switch {
		case !v.isNum:
			argv = append(argv, v.str)
		case i == 0:
			return &basicerr.Error{Kind: basicerr.Syntax} // SYS with an address
		default:
			argv = append(argv, strings.TrimPrefix(formatNumber(v.num), " "))
		}
	}
	if in.system == nil {
		return &basicerr.Error{Kind: basicerr.DeviceNotPresent}
	}
	if err := in.FreshLine(); err != nil {
		return err
	}
	status, err := in.system.Run(argv[0], argv[1:])
	in.interrupted.Store(false)
	if err != nil {
		return &basicerr.Error{Kind: basicerr.FileNotFound}
	}
	in.status = status
	in.column = 0
	return nil
}
