# Grammar Specs

Design: `grammar-design.md`

## File

- [x] **GRAMMAR-001**: The C64 BASIC V2 grammar shall be stored in `grammar/c64basic.ebnf`, written in the notation of `golang.org/x/exp/ebnf`.
- [x] **GRAMMAR-002**: `grammar/c64basic.ebnf` shall begin with a comment block that explains each notation symbol (`=`, `.`, `|`, `[ ]`, `{ }`, `( )`, quoted literals, `…` ranges) and that uppercase rule names are parser rules and lowercase rule names are lexer rules.
- [x] **GRAMMAR-003**: Every rule in `grammar/c64basic.ebnf` shall be immediately preceded by a comment line.
- [x] **GRAMMAR-004**: `grammar/c64basic.ebnf` shall define exactly the rules `Line`, `Statement`, `PrintStatement`, `PrintItem`, `Expression`, `print`, `string`, and `character`.
- [x] **GRAMMAR-005**: `grammar/c64basic.ebnf` shall be valid for `ebnf.Verify` with start rule `Line`: every referenced rule is defined, every rule is reachable from `Line`, and lexical (lowercase) rules use only characters and character ranges.

## Loader package

- [x] **GRAMMAR-006**: The `grammar` Go package shall embed `c64basic.ebnf` and expose its contents as the string `Source` and the start rule name `"Line"` as `Start`.
- [x] **GRAMMAR-007**: When `grammar.Load` is called, it shall parse `Source` with `ebnf.Parse`, check it with `ebnf.Verify` from `Start`, and return the parsed grammar, or the error if either step fails.

## Conformance

- [x] **GRAMMAR-008**: The parser package's rule table (grammar rule name → parse function) shall have an entry for every uppercase rule in `grammar/c64basic.ebnf`, and every entry shall name an uppercase rule that exists in the grammar; a test in the parser package shall fail otherwise.
- [x] **GRAMMAR-009**: The lexer package's token table (grammar item → token kind or helper) shall have an entry for every lowercase rule in `grammar/c64basic.ebnf` and every literal token that appears in an uppercase rule, and every entry shall name such an item; a test in the lexer package shall fail otherwise.
