package shell

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

// historySize is how many lines the line editor's history keeps.
const historySize = 100

// ctrlC is the byte a terminal in raw mode sends for Ctrl-C.
const ctrlC = 0x03

// lineReader reads input lines without their line terminators. At the end
// of input it returns io.EOF.
type lineReader interface {
	ReadLine() (string, error)
}

// plainReader reads lines as bytes, with no editing: from scripts, pipes,
// and any input that is not a terminal.
type plainReader struct {
	r *bufio.Reader
}

// ReadLine returns the next line without its "\n" and any "\r" before it.
// A final line without a terminator is returned; a read error other than
// io.EOF is returned without the partial line.
func (p *plainReader) ReadLine() (string, error) {
	// ReadString grows its buffer as needed, so lines have no length limit.
	line, err := p.r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	if line == "" && err == io.EOF {
		return "", io.EOF
	}
	line = strings.TrimSuffix(line, "\n")
	return strings.TrimSuffix(line, "\r"), nil
}

// editorWanted reports whether lines should be read with the line editor:
// in interactive mode, reading stdin, with stdin and stderr both terminals.
//
// @spec SHELL-EDIT-001, SHELL-EDIT-002
func editorWanted(cfg Config, stdin io.Reader, stderr io.Writer) bool {
	out, ok := stderr.(*os.File)
	return cfg.Interactive && cfg.File == "" && isTerminal(stdin) &&
		ok && term.IsTerminal(int(out.Fd()))
}

// rawModeFunc switches the terminal to raw mode and returns a function that
// restores its previous mode.
type rawModeFunc func() (restore func(), err error)

// sizeFunc returns the terminal's width and height, if known.
type sizeFunc func() (width, height int, ok bool)

// terminalRawMode and terminalSize operate on the terminal open as f.
func terminalRawMode(f *os.File) rawModeFunc {
	fd := int(f.Fd())
	return func() (func(), error) {
		state, err := term.MakeRaw(fd)
		if err != nil {
			return nil, err
		}
		return func() { term.Restore(fd, state) }, nil
	}
}

func terminalSize(f *os.File) sizeFunc {
	return func() (int, int, bool) {
		w, h, err := term.GetSize(int(f.Fd()))
		// Some pseudo-terminals report a size of 0; treat that as unknown.
		return w, h, err == nil && w > 0 && h > 0
	}
}

// editorReader reads lines with the golang.org/x/term line editor, which
// provides history (up and down arrows) and basic editing keys.
type editorReader struct {
	term  *term.Terminal
	in    *cancelFilter
	raw   rawModeFunc
	size  sizeFunc
	hist  *history
	echo  io.Writer
	style style
}

// newEditorReader returns a line editor reading keystrokes from in and
// echoing to echo. raw switches the terminal to raw mode around each line;
// size reports the terminal's size.
func newEditorReader(in io.Reader, echo io.Writer, raw rawModeFunc, size sizeFunc, historyFile string, warn io.Writer) *editorReader {
	filter := &cancelFilter{r: in}
	t := term.NewTerminal(struct {
		io.Reader
		io.Writer
	}{filter, echo}, "")
	h := &history{
		skip: func() bool { return filter.cancelled },
		file: historyFile,
		warn: warn,
	}
	h.load()
	t.History = h
	return &editorReader{term: t, in: filter, raw: raw, size: size, hist: h, echo: echo}
}

// setStyle colors typed lines with the style's input color. The prompt is
// that escape code alone, which term.Terminal counts as zero columns wide,
// so every redraw of the line is colored; the span ends after each line.
//
// @spec SHELL-STYLE-003
func (e *editorReader) setStyle(st style) {
	e.style = st
	e.term.SetPrompt(st.input)
}

// defaultHistoryFile returns the history file Main uses: $C64SH_HISTORY if
// set (empty turns the file off), otherwise ~/.c64sh_history, or none if
// the home directory is unknown.
//
// @spec SHELL-HIST-001
func defaultHistoryFile(lookupEnv func(string) (string, bool), homeDir func() (string, error)) string {
	if path, ok := lookupEnv("C64SH_HISTORY"); ok {
		return path
	}
	home, err := homeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".c64sh_history")
}

// ReadLine returns the next line entered, skipping lines discarded with
// Ctrl-C. Ctrl-D on an empty line returns io.EOF.
//
// @spec SHELL-EDIT-003, SHELL-EDIT-005, SHELL-EDIT-006, SHELL-EDIT-008, SHELL-EDIT-009
func (e *editorReader) ReadLine() (string, error) {
	for {
		line, err := e.readOnce()
		if e.in.cancelled {
			e.in.cancelled = false
			if err == nil {
				continue // the line was discarded with Ctrl-C
			}
		}
		return line, err
	}
}

// readOnce reads one line with the terminal in raw mode.
//
// @spec SHELL-EDIT-007
func (e *editorReader) readOnce() (string, error) {
	restore, err := e.raw()
	if err != nil {
		return "", err
	}
	defer restore()
	if w, h, ok := e.size(); ok {
		e.term.SetSize(w, h)
	}
	line, err := e.term.ReadLine()
	if e.style.input != "" {
		io.WriteString(e.echo, e.style.end())
	}
	if errors.Is(err, term.ErrPasteIndicator) {
		err = nil // a pasted line runs like a typed one
	}
	return line, err
}

// cancelFilter reads keystrokes for the line editor and turns Ctrl-C into
// a cancelled line. term.Terminal treats Ctrl-C as end of input; instead,
// the filter replaces it with Return and records that the line ending there
// is cancelled. Bytes after the Ctrl-C are held back until the next read,
// and a line that ends before the Ctrl-C is returned on its own first, so
// that the cancellation applies only to the line it ends.
type cancelFilter struct {
	r         io.Reader
	pending   []byte // bytes held back for the next read
	err       error  // error to report once pending is empty
	cancelled bool
}

func (f *cancelFilter) Read(p []byte) (int, error) {
	var n int
	var err error
	switch {
	case len(f.pending) > 0:
		n = copy(p, f.pending)
		f.pending = f.pending[n:]
	case f.err != nil:
		err, f.err = f.err, nil
		return 0, err
	default:
		n, err = f.r.Read(p)
	}

	i := bytes.IndexByte(p[:n], ctrlC)
	if i < 0 {
		return n, err
	}
	// Hold back everything after the cut, and any error, for later reads.
	cut := i + 1
	if j := bytes.IndexAny(p[:i], "\r\n"); j >= 0 {
		cut = j + 1 // a complete line comes first
	} else {
		p[i] = '\r'
		f.cancelled = true
	}
	f.pending = append(append([]byte(nil), p[cut:n]...), f.pending...)
	if err != nil {
		f.err = err
	}
	return cut, nil
}

// history is the line editor's history of lines entered in the session. It
// keeps the most recent historySize lines and ignores blank lines and lines
// being discarded with Ctrl-C. With a history file, it starts from the
// file's lines and saves itself to the file after each line is added.
//
// @spec SHELL-EDIT-004
type history struct {
	entries []string // oldest first
	skip    func() bool
	file    string    // history file; empty: none
	warn    io.Writer // where a history-file problem is reported
	warned  bool      // whether it has been reported this session
}

func (h *history) Add(entry string) {
	if isBlank(entry) || (h.skip != nil && h.skip()) {
		return
	}
	h.entries = append(h.entries, entry)
	if len(h.entries) > historySize {
		h.entries = h.entries[len(h.entries)-historySize:]
	}
	h.save()
}

// load starts the history with the most recent non-blank lines of the
// history file. A missing file is an empty history.
//
// @spec SHELL-HIST-002, SHELL-HIST-004
func (h *history) load() {
	if h.file == "" {
		return
	}
	data, err := os.ReadFile(h.file)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err != nil {
		h.report(err)
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if !isBlank(line) {
			h.entries = append(h.entries, line)
		}
	}
	if len(h.entries) > historySize {
		h.entries = h.entries[len(h.entries)-historySize:]
	}
}

// save writes the history to a temporary file beside the history file and
// renames it into place, so the file is never left half-written.
//
// @spec SHELL-HIST-003, SHELL-HIST-004
func (h *history) save() {
	if h.file == "" {
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(h.file), ".c64sh_history-*")
	if err != nil {
		h.report(err)
		return
	}
	_, err = io.WriteString(tmp, strings.Join(h.entries, "\n")+"\n")
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Chmod(tmp.Name(), 0o600)
	}
	if err == nil {
		err = os.Rename(tmp.Name(), h.file)
	}
	if err != nil {
		os.Remove(tmp.Name())
		h.report(err)
	}
}

// report writes a history-file problem to the warning writer, once per
// session.
func (h *history) report(err error) {
	if h.warned || h.warn == nil {
		return
	}
	h.warned = true
	fmt.Fprintf(h.warn, "c64sh: history: %v\n", err)
}

// Len returns the number of entries.
func (h *history) Len() int {
	return len(h.entries)
}

// At returns an entry; 0 is the most recent.
func (h *history) At(i int) string {
	return h.entries[len(h.entries)-1-i]
}
