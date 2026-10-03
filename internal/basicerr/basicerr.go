// Package basicerr defines the BASIC errors that c64sh reports the way a
// C64 does, such as ?SYNTAX  ERROR.
package basicerr

// Kind identifies a BASIC error.
type Kind int

const (
	Syntax              Kind = iota // SYNTAX
	StringTooLong                   // STRING TOO LONG
	TypeMismatch                    // TYPE MISMATCH
	Overflow                        // OVERFLOW
	DivisionByZero                  // DIVISION BY ZERO
	IllegalQuantity                 // ILLEGAL QUANTITY
	UndefdStatement                 // UNDEF'D STATEMENT
	Break                           // BREAK: execution stopped by Ctrl-C
	NextWithoutFor                  // NEXT WITHOUT FOR
	OutOfMemory                     // OUT OF MEMORY
	ReturnWithoutGosub              // RETURN WITHOUT GOSUB
	IllegalDirect                   // ILLEGAL DIRECT
	UndefdFunction                  // UNDEF'D FUNCTION
	FileNotFound                    // FILE NOT FOUND
	DeviceNotPresent                // DEVICE NOT PRESENT
	IllegalDeviceNumber             // ILLEGAL DEVICE NUMBER
	MissingFileName                 // MISSING FILE NAME
	Load                            // LOAD
	Verify                          // VERIFY
	FileOpen                        // FILE OPEN
	FileNotOpen                     // FILE NOT OPEN
	NotInputFile                    // NOT INPUT FILE
	NotOutputFile                   // NOT OUTPUT FILE
	TooManyFiles                    // TOO MANY FILES
	FileData                        // FILE DATA
	BadSubscript                    // BAD SUBSCRIPT
	RedimdArray                     // REDIM'D ARRAY
	OutOfData                       // OUT OF DATA
)

// names holds each kind's name as the C64 prints it.
var names = [...]string{
	Syntax:              "SYNTAX",
	StringTooLong:       "STRING TOO LONG",
	TypeMismatch:        "TYPE MISMATCH",
	Overflow:            "OVERFLOW",
	DivisionByZero:      "DIVISION BY ZERO",
	IllegalQuantity:     "ILLEGAL QUANTITY",
	UndefdStatement:     "UNDEF'D STATEMENT",
	Break:               "BREAK",
	NextWithoutFor:      "NEXT WITHOUT FOR",
	OutOfMemory:         "OUT OF MEMORY",
	ReturnWithoutGosub:  "RETURN WITHOUT GOSUB",
	IllegalDirect:       "ILLEGAL DIRECT",
	UndefdFunction:      "UNDEF'D FUNCTION",
	FileNotFound:        "FILE NOT FOUND",
	DeviceNotPresent:    "DEVICE NOT PRESENT",
	IllegalDeviceNumber: "ILLEGAL DEVICE NUMBER",
	MissingFileName:     "MISSING FILE NAME",
	Load:                "LOAD",
	Verify:              "VERIFY",
	FileOpen:            "FILE OPEN",
	FileNotOpen:         "FILE NOT OPEN",
	NotInputFile:        "NOT INPUT FILE",
	NotOutputFile:       "NOT OUTPUT FILE",
	TooManyFiles:        "TOO MANY FILES",
	FileData:            "FILE DATA",
	BadSubscript:        "BAD SUBSCRIPT",
	RedimdArray:         "REDIM'D ARRAY",
	OutOfData:           "OUT OF DATA",
}

// Error is a BASIC error.
type Error struct {
	Kind    Kind
	Line    int  // the program line where the error occurred, if HasLine
	HasLine bool // false for an error in direct mode
}

// Error returns the error's C64 name, such as "SYNTAX".
//
// @spec SHELL-ERR-003
func (e *Error) Error() string {
	return names[e.Kind]
}
