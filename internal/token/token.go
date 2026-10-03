// Package token defines the tokens the lexer produces from a line of
// C64 BASIC input.
package token

import "fmt"

// Kind identifies the type of a token.
type Kind int

const (
	EOL       Kind = iota // end of line; always the last token
	Illegal               // a character no rule accepts
	Print                 // PRINT or ?
	Rem                   // REM and the rest of the line
	Number                // 12, 3.5, .5, 1E3
	String                // "…"
	Colon                 // :
	Semicolon             // ;
	Comma                 // ,
	Plus                  // +
	Minus                 // -
	Star                  // *
	Slash                 // /
	LParen                // (
	RParen                // )
	Caret                 // ^ or ↑ (exponentiation)
	Let                   // LET
	Equal                 // =
	Name                  // a variable name, without spaces
	Less                  // <
	Greater               // >
	And                   // AND
	Or                    // OR
	Not                   // NOT
	If                    // IF
	Then                  // THEN
	Run                   // RUN
	List                  // LIST
	New                   // NEW
	End                   // END
	Goto                  // GOTO
	Go                    // GO (as in GO TO)
	To                    // TO
	For                   // FOR
	Next                  // NEXT
	Step                  // STEP
	Gosub                 // GOSUB
	Return                // RETURN
	Input                 // INPUT
	Get                   // GET
	Def                   // DEF
	Fn                    // FN
	Load                  // LOAD
	Save                  // SAVE
	Verify                // VERIFY
	PrintFile             // PRINT#
	InputFile             // INPUT#
	Open                  // OPEN
	Close                 // CLOSE
	Cmd                   // CMD
	Hash                  // #
	On                    // ON
	Function              // a built-in function: ABS, INT, …, SIN (Value: the name)
	Pi                    // π
	Tab                   // TAB(
	Spc                   // SPC(
	Dim                   // DIM
	Reserved              // a BASIC V2 keyword c64sh does not support yet (Value: the keyword)
	Data                  // DATA and its text up to ":" outside quotes (Value: the text)
	Read                  // READ
	Restore               // RESTORE
)

var kindNames = [...]string{
	EOL:       "EOL",
	Illegal:   "Illegal",
	Print:     "Print",
	Rem:       "Rem",
	Number:    "Number",
	String:    "String",
	Colon:     "Colon",
	Semicolon: "Semicolon",
	Comma:     "Comma",
	Plus:      "Plus",
	Minus:     "Minus",
	Star:      "Star",
	Slash:     "Slash",
	LParen:    "LParen",
	RParen:    "RParen",
	Caret:     "Caret",
	Let:       "Let",
	Equal:     "Equal",
	Name:      "Name",
	Less:      "Less",
	Greater:   "Greater",
	And:       "And",
	Or:        "Or",
	Not:       "Not",
	If:        "If",
	Then:      "Then",
	Run:       "Run",
	List:      "List",
	New:       "New",
	End:       "End",
	Goto:      "Goto",
	Go:        "Go",
	To:        "To",
	For:       "For",
	Next:      "Next",
	Step:      "Step",
	Gosub:     "Gosub",
	Return:    "Return",
	Input:     "Input",
	Get:       "Get",
	Def:       "Def",
	Fn:        "Fn",
	Load:      "Load",
	Save:      "Save",
	Verify:    "Verify",
	PrintFile: "PrintFile",
	InputFile: "InputFile",
	Open:      "Open",
	Close:     "Close",
	Cmd:       "Cmd",
	Hash:      "Hash",
	On:        "On",
	Function:  "Function",
	Pi:        "Pi",
	Tab:       "Tab",
	Spc:       "Spc",
	Dim:       "Dim",
	Reserved:  "Reserved",
	Data:      "Data",
	Read:      "Read",
	Restore:   "Restore",
}

func (k Kind) String() string {
	if k >= 0 && int(k) < len(kindNames) {
		return kindNames[k]
	}
	return fmt.Sprintf("Kind(%d)", int(k))
}

// Token is one token of a line.
type Token struct {
	Kind  Kind
	Value string // String: contents without quotes; Rem: the text after REM; Number: the literal without spaces; Illegal: the character; otherwise the source text
	Pos   int    // byte offset of the token's first character in the line
}
