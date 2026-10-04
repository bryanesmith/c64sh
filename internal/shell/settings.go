package shell

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/bryanesmith/c64sh/internal/interp"
)

// settings are the shell's settings, read from environment variables (a
// c64sh extension).
type settings struct {
	noColor            bool     // NO_COLOR: no colors at all
	historyFile        string   // C64SH_HISTORY; empty: none
	historySize        int      // C64SH_HISTSIZE
	input, ready, fail string   // the styles' escape codes; empty: unstyled
	invalid            []string // reports of values not valid, defaults used instead
}

// sgr matches SGR parameters: groups of digits separated by ";".
var sgr = regexp.MustCompile(`^[0-9]+(;[0-9]+)*$`)

// lookup returns an environment variable, or false if env is nil.
func lookup(env interp.Environment, name string) (string, bool) {
	if env == nil {
		return "", false
	}
	return env.Lookup(name)
}

// readSettings reads the shell's settings from env, with home as the home
// directory ("" if unknown).
//
// @spec SHELL-HIST-001, SHELL-SET-002, SHELL-SET-003
func readSettings(env interp.Environment, home string) settings {
	var s settings
	v, _ := lookup(env, "NO_COLOR")
	s.noColor = v != ""

	if path, ok := lookup(env, "C64SH_HISTORY"); ok {
		s.historyFile = path
	} else if home != "" {
		s.historyFile = filepath.Join(home, ".c64sh_history")
	}

	s.historySize = defaultHistorySize
	if v, _ := lookup(env, "C64SH_HISTSIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 && v[0] != '+' {
			s.historySize = n
		} else {
			s.invalid = append(s.invalid, invalidValue("C64SH_HISTSIZE", v))
		}
	}

	color := func(name, def string) string {
		v, ok := lookup(env, name)
		switch {
		case !ok:
			return "\x1b[" + def + "m"
		case v == "":
			return ""
		case sgr.MatchString(v):
			return "\x1b[" + v + "m"
		}
		s.invalid = append(s.invalid, invalidValue(name, v))
		return "\x1b[" + def + "m"
	}
	s.input = color("C64SH_INPUT_COLOR", "36") // cyan
	s.ready = color("C64SH_READY_COLOR", "32") // green
	s.fail = color("C64SH_ERROR_COLOR", "31")  // red
	return s
}

// invalidValue reports a setting whose value is not valid.
//
// @spec SHELL-SET-004
func invalidValue(name, value string) string {
	return fmt.Sprintf("c64sh: %s: invalid value %q", name, value)
}

// runCommandsFile returns the run-commands file: C64SH_RC if set (empty:
// none), otherwise .c64shrc in home, or none if home is unknown.
//
// @spec SHELL-RC-001
func runCommandsFile(env interp.Environment, home string) string {
	if path, ok := lookup(env, "C64SH_RC"); ok {
		return path
	}
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".c64shrc")
}
