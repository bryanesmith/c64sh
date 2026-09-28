// Command c64sh is a shell that speaks Commodore 64 BASIC V2.
package main

import (
	"os"

	"github.com/bryanesmith/c64sh/internal/shell"
)

// @spec SHELL-CLI-001, SHELL-LINE-008
//
// Go's default SIGPIPE handling is kept, so c64sh ends silently when
// its stdout is a pipe whose reader has exited.
func main() {
	os.Exit(shell.Main(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
