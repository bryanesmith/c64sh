---
parent: high-level-design
prefix: GRAMMAR
---

# Grammar

## Context and Design Philosophy

`grammar/c64basic.ebnf` is the written specification of the C64 BASIC V2 syntax that c64sh accepts. It is read by people first: someone learning the language or extending c64sh should be able to open it like a plain text reference and understand what is valid input without knowing EBNF in advance.

The grammar is not executed. The lexer and parser are hand-written to match it, and tests keep them aligned (see *Conformance*). When the grammar and the parser disagree, the grammar states the intent and the parser is wrong.

## File Format

The file uses the notation of `golang.org/x/exp/ebnf`:

```
Production  = name "=" [ Expression ] "." .
Expression  = Alternative { "|" Alternative } .
Alternative = Term { Term } .
Term        = name | token [ "…" token ] | Group | Option | Repetition .
Group       = "(" Expression ")" .
Option      = "[" Expression "]" .
Repetition  = "{" Expression "}" .
```

Conventions that keep it readable:

- **Notation key first.** The file opens with a comment block explaining every symbol (`=`, `.`, `|`, `[ ]`, `{ }`, `( )`, `"…"`, ranges) and the capitalized/lowercase distinction.
- **Sections by topic**, separated by comment rulers: Lines, one section per statement (PRINT, …), Expressions, Tokens.
- **Every rule has a comment** saying what it means, with one or more example inputs and, where useful, the output they produce:
  ```
  //   PRINT "A";"B"          -> AB
  ```
- **Aligned `=` signs** within a section.
- **C64 quirks are noted where they occur** — for example, the comment on `string` says the closing quote is optional, as on a C64.

## Rule Classes

`golang.org/x/exp/ebnf` distinguishes two kinds of rule by the case of the first letter of the rule name:

| Kind | Name starts with | Built from | Implemented by |
|---|---|---|---|
| Syntactic | Uppercase (`Line`, `PrintStatement`) | Other rules and literal tokens (`":"`, `";"`) | Parser — one parse function per rule |
| Lexical | Lowercase (`print`, `string`) | Characters and character ranges only | Lexer — one token kind per rule |

Literal tokens written inside syntactic rules (`":"`, `";"`, `","`, `"+"`) are also produced by the lexer, each as its own token kind.

## Start Rule

The start rule is `Line`: one line of input, as typed at the prompt or read from a script. Every rule in the file must be reachable from `Line`.

## Current Grammar

The grammar covers the direct-mode `PRINT` statement with string expressions:

| Rule | Meaning |
|---|---|
| `Line` | Statements separated by `:`. |
| `Statement` | A `PRINT` statement, or nothing (C64 BASIC allows empty statements, so `::` and a line consisting only of `:` are valid). |
| `PrintStatement` | The `print` keyword followed by any number of print items. |
| `PrintItem` | An expression, `;`, or `,`. Items may follow each other with no separator (`PRINT "A""B"`), as on a C64. |
| `Expression` | One or more string literals joined by `+`. |
| `print` | The keyword `PRINT`, or its C64 abbreviation `?`. |
| `string` | A double quote, any characters except a double quote, and an optional closing double quote. |
| `character` | Any character except a double quote or a line break. |

Keywords are case-sensitive: the uppercase text written in the grammar is exactly what is accepted, so `print` is not a keyword.

## Loader Package

The `grammar` Go package embeds the file and exposes it to tests in other packages:

| Identifier | Purpose |
|---|---|
| `Source` | The file contents, embedded with `//go:embed c64basic.ebnf`. |
| `Start` | The start rule name, `"Line"`. |
| `Load() (ebnf.Grammar, error)` | Parses `Source` with `ebnf.Parse` and checks it with `ebnf.Verify(g, Start)`. |

The package is used only by tests. The shipped `c64sh` binary does not load or depend on the grammar at runtime.

## Conformance

Three tests, each in the package that owns the implementation side, keep the grammar and the code aligned:

1. **Grammar validity** (`grammar` package) — `Load` returns no error: the file parses, every referenced rule is defined, every rule is reachable from `Line`, and lexical rules use only characters and ranges.
2. **Syntactic rules ↔ parse functions** (`parser` package) — the parser keeps a table from rule name to parse function. Every uppercase rule in the grammar has an entry, and every entry names a rule that exists.
3. **Lexical rules and literal tokens ↔ token kinds** (`lexer` package) — the lexer keeps a table from each lowercase rule name and each literal token used in a syntactic rule to its token kind. Every such grammar item has an entry, and every entry names an item that exists.

These tests catch a rule added to the grammar without code, and code for a rule removed from the grammar. They do not prove that a parse function implements its rule correctly; that is the job of the parser's and lexer's own unit tests.

## Decisions & Alternatives

| Decision | Chosen | Alternatives Considered | Rationale |
|---|---|---|---|
| Notation | `golang.org/x/exp/ebnf` | ISO 14977 EBNF; ABNF; PEG | Has a maintained Go parser and verifier, so the grammar can be checked by `go test`. Its notation is the one used in the Go language specification, which is compact and widely read. |
| Role of the grammar | Specification checked by tests; not executed | Runtime-interpreted grammar; generated parser | See HLD *Key Design Decisions*. The grammar stays free to optimize for readability because no code is generated from it. |
| Location | `grammar/` package at the repository root | `internal/parser/`; `docs/` | The grammar specifies both lexer and parser, so neither owns it. `//go:embed` cannot reference parent directories, so a package is the simplest way for other packages' tests to load it. |
| Conformance granularity | Rule names ↔ code tables, checked both ways | Parse example inputs extracted from grammar comments | Name tables are simple and catch the drift that matters most (missing or stale rules). Behavior is covered by unit and functional tests, which state expected output precisely. |
| Empty statements | Allowed (`Statement = [ PrintStatement ] .`) | Require at least one statement per line | Matches C64 BASIC V2, which accepts `::` and leading or trailing colons. |

## Open Questions & Future Decisions

### Deferred
1. When program mode is added, `Line` gains an optional line-number prefix, and a `number` lexical rule is introduced. The shape of numeric literals (`1`, `.5`, `1E3`) is decided when numbers are added.

## References

- [`golang.org/x/exp/ebnf`](https://pkg.go.dev/golang.org/x/exp/ebnf)
- [The Go Programming Language Specification — Notation](https://go.dev/ref/spec#Notation)
- `docs/intent/lexer/lexer-design.md`, `docs/intent/parser/parser-design.md`
