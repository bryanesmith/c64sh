---
parent: high-level-design
prefix: SNAPSHOT
---

# Examples and Snapshot Tests

## Context and Design Philosophy

`examples/` is a set of numbered BASIC scripts that show every language feature in many forms. They are documentation first: a user learning c64sh reads them to see how lines are read, what each form prints, and where the C64's rules produce surprising results. Each script is also executable, so running it shows the same thing live.

Snapshot tests make the examples trustworthy. Each script is run through the shell and its complete result is compared with a recorded snapshot. Any change in behavior, intended or not, shows up as a failing test and, once accepted, as a reviewable diff of a plain text file. The snapshots are never written without a deliberate update command.

## Example Scripts

### Naming

Each example is a file directly in `examples/` named `NNN-words.bas`:

- `NNN` is a three-digit number. Numbers start at `001` and run sequentially with no gaps or duplicates.
- `words` is one or more lowercase ASCII letter-or-digit words joined by single hyphens, describing the topic (`hello-world`, `concat-strings`).

`examples/` holds only example scripts: no other files and no subdirectories. Hidden files (names beginning with `.`, such as the `.DS_Store` files macOS creates) are ignored. Numbering is the reading order, from the simplest feature to the most involved.

### Contents

```
#!/usr/bin/env c64sh
REM Different ways to join strings
PRINT "HELLO ";"WORLD":REM HELLO WORLD
…
```

- **Line 1** is exactly `#!/usr/bin/env c64sh`, so the file runs as a script.
- **Line 2** is a `REM` comment with text explaining what the file shows.
- **Later lines** exercise the topic in many forms. `REM` lines introduce groups of related forms.
- **Every line containing a `PRINT` ends with a comment showing what it prints**, written `:REM …` because `PRINT` ends only at `:` or the end of the line. Where an end-of-line comment is impossible (after an unclosed string, which runs to the end of the line) or awkward, the comment goes on its own `REM` line immediately before. Spaces are written exactly as printed and a missing newline as `(no newline)`; an example about print zones may print a ruler line of column digits so the zones can be checked by eye. A line demonstrating a mistake still ends with a comment, saying what goes wrong.
- **Errors may be shown.** A script stops at its first error, so lines after it do not run; the snapshot records exactly what happens.
- The file is **executable** (owner execute permission set), so `./examples/001-hello-world.bas` runs.

## Snapshot Tests

### Location and Running

The tests live in `test/snapshot/` (package `snapshot_test`). One test function runs a subtest per example, named after the example's file name without `.bas`.

Each example is run in-process with `shell.Main([]string{path}, stdin, stdout, stderr)`, exactly as `c64sh FILE` runs it. This makes the snapshot record what a user sees when running the file, including the skipped `#!` line and script-mode behavior.

**Stdin** is empty, unless the example has an input file: `test/snapshot/testdata/` holding a file named after the example with `.input` in place of `.bas` (`019-keyboard-input.input`). Its contents are then the example's stdin, which is what `INPUT` and `GET` read, as they would from `c64sh FILE < answers`. An example that reads input says in its comments which answers the snapshot uses, and its `PRINT` comments describe the output for those answers. An input file whose example no longer exists fails the tests, like a stale snapshot; unlike snapshots, input files are written by hand, so `make update-snapshots` neither creates nor deletes them.

### Snapshot Format

A snapshot records all three observable results of a run, in this order:

```
exit status: 0
--- stdout ---
HELLO WORLD
HELLO	WORLD
--- stderr ---
```

- The first line is `exit status: ` and the status number.
- `--- stdout ---` is followed by the stdout bytes exactly as written, including tabs.
- `--- stderr ---` is followed by the stderr bytes exactly as written.
- If a stream's output is non-empty and does not end with a newline, its header instead reads `--- stdout (no newline at end) ---` (or `stderr`), and a newline is added after the output so the next header starts on its own line. This keeps every snapshot unambiguous: an output of `A` and an output of `A` followed by a newline produce different snapshots.

### Storage

Each example's snapshot is a plain file in `test/snapshot/testdata/` named after the example, with `.snap` in place of `.bas` (`001-hello-world.bas` → `001-hello-world.snap`). Its contents are exactly the formatted result above, byte for byte, including tab characters. The test code reads, compares, and writes these files itself, using only the standard library.

### Checking and Updating

| Situation | Without `UPDATE_SNAPS=true` | With `UPDATE_SNAPS=true` |
|---|---|---|
| Output matches the snapshot | Pass | Pass |
| Output differs from the snapshot | Fail, naming the first differing line | Snapshot rewritten |
| Example has no snapshot | Fail, saying the snapshot is missing | Snapshot created |
| Snapshot has no example | Fail, naming the snapshot | Snapshot deleted |

`UPDATE_SNAPS=true` is set by `make update-snapshots` (see the build design). A snapshot is only ever written by an explicit update, never as a side effect of an ordinary test run.

A failure caused by a difference names the example and the first line (1-based, counting from `exit status:`) where the recorded and actual snapshots differ, and shows both versions of that line as Go-quoted strings, so tabs and other invisible characters appear as `\t` and similar escapes:

```
001-hello-world: snapshot differs at line 3:
     got: "A\tB"
    want: "A    B"
run `make update-snapshots` if the change is intended, then review the diff
```

If one version has fewer lines, the missing line is shown as `(end of snapshot)`.

### Convention Checks

The same package checks every file in `examples/` against the conventions above: the name pattern, sequential numbering from `001`, the exact `#!` first line, a `REM` second line with text, the owner execute permission, and the absence of other files or directories. A failure names the file and the rule it breaks.

The PRINT-comment convention is checked with the lexer: for every line of an example after the first, if the line's tokens include a `Print` token, the last token before `EOL` must be a `Rem` token, or the line immediately before must be a comment line (only a `Rem` token before `EOL`). Lines whose `PRINT` is not recognized as a keyword (such as a lowercase `print` shown as a mistake) are not checked.

## Current Examples

| Example | Shows |
|---|---|
| `001-hello-world.bas` | `PRINT` and `?`, with and without a space, unclosed strings |
| `002-concat-strings.bas` | Joining strings with `;`, `+`, and no separator, and combinations of them |
| `003-print-separators.bas` | `,` moving to the next 10-column print zone (with a printed ruler), the column carrying over between PRINTs, trailing `;` and `,` suppressing the newline, bare `PRINT`, several statements with `:`, empty statements |
| `004-comments.bas` | `REM` as a whole line, after `:`, without a space (`REMARK`), with colons and quotes inside, and `"REM"` inside a string |
| `006-numbers.bas` | Number literals (decimals, `E` notation, a lone `.`), C64 number printing (sign and trailing spaces, 9 digits, scientific notation), spaces inside numbers, numbers mixed with strings and in zones, `+` on numbers, the number range, and `?TYPE MISMATCH  ERROR` |
| `007-arithmetic.bas` | `+ - * /`, precedence and left-to-right evaluation, parentheses, negation and a leading `+`, division's 9 digits, underflow to zero, items side by side, and `?DIVISION BY ZERO  ERROR` |
| `008-exponents.bas` | `^` and `↑`, roots and negative exponents, `^` above negation and the other operators, left-to-right `^`, signs in exponents, the C64's special cases (`0^0`, `0^-1`, negative bases), and `?ILLEGAL QUANTITY  ERROR` |
| `009-variables.bas` | Number and string variables (and why `SCORE` cannot be a name), `=` with and without `LET`, variables in expressions and `PRINT`, unset variables, two-character names, spaces inside names, keywords inside names (`PRINTER`, `LETTER`), and `?TYPE MISMATCH  ERROR` |
| `010-integer-variables.bas` | `%` variables: rounding down (including negative values), `A`/`A%`/`A$` as separate variables, two-character names, counting, integers in calculations, the -32768..32767 range, `TI%` as an ordinary variable, and `?ILLEGAL QUANTITY  ERROR` |
| `011-comparisons.bas` | Each comparison operator on numbers and strings, -1/0 results, comparisons after arithmetic, the alternative spellings (`><`, `=<`, `=>`, spaces, `<=>`), left-to-right chains, `=` as assignment then comparison, and `?TYPE MISMATCH  ERROR` |
| `012-logic.bas` | `AND`, `OR`, `NOT` on comparisons, their order, bitwise results on whole numbers, rounding down, `NOT` taking in what follows, variables, and `?ILLEGAL QUANTITY  ERROR` |
| `013-if-then.bas` | `IF … THEN` with true and false conditions, skipping the rest of the line, numbers and strings as conditions, `AND`/`OR`/`NOT` conditions, nested `IF`s, no `ELSE`, and a syntax error that goes unnoticed when skipped but stops the script when reached |
| `014-program-mode.bas` | Storing numbered lines in any order, replacing and deleting them, `RUN` and `RUN n`, `LIST` (with `?` shown as `PRINT`), `NEW`, `END`, variables cleared by `RUN` and by storing a line, and a syntax error found only when its line runs (`?SYNTAX  ERROR IN 20`) |
| `015-numbered-scripts.bas` | The script rule: numbered lines stored and unnumbered lines run at once, and the never-run program run after the last line |
| `016-goto.bas` | `GOTO` skipping lines, a loop with `IF … THEN n`, `IF … GOTO n`, `GO TO`, `GOTO` typed directly keeping variables (unlike `RUN`), and `?UNDEF'D STATEMENT  ERROR IN 20` |
| `017-loops.bas` | `FOR … NEXT` with and without `STEP` (negative and fractional), bare `NEXT`, the variable's value after a loop, a body that runs once, end values worked out once, nested loops and `NEXT J,I`, a loop across program lines, and `?NEXT WITHOUT FOR  ERROR` |
| `018-subroutines.bas` | `GOSUB` and `RETURN`, returning mid-line, nested subroutines, passing values in variables, `GOSUB` typed directly, and `?RETURN WITHOUT GOSUB  ERROR IN 20` |
| `019-keyboard-input.bas` | `INPUT` with and without a prompt, several values, `?? `, `?REDO FROM START`, `?EXTRA IGNORED`, quoted strings, a `GET` wait loop, and `?ILLEGAL DIRECT  ERROR`; its answers come from `019-keyboard-input.input` |
| `020-user-functions.bas` | `DEF FN` and `FN`, the protected parameter, bodies using other variables and functions, two-character names separate from variables, and a body mistake reported at the call (`?SYNTAX  ERROR IN 120`) |
| `005-syntax-errors.bas` | Common mistakes explained in comments, ending in `?SYNTAX  ERROR` |

## Decisions & Alternatives

| Decision | Chosen | Alternatives Considered | Rationale |
|---|---|---|---|
| Where expected output lives | Separate snapshot file per example | Expected output parsed from end-of-line `:REM` comments; hand-maintained expected files | Keeps comments free-form for readers and not every line needs one. Snapshots capture tabs, missing newlines, stderr, and exit status exactly, and are regenerated by a command instead of by hand. |
| Snapshot storage | Plain golden files, read, compared, and written by the test code | `github.com/gkampitakis/go-snaps`; `github.com/sebdah/goldie/v2` | The snapshot must hold the exact output. go-snaps renders standalone snapshots through `kr/pretty`, which runs text through `text/tabwriter` and turns each tab into alignment spaces, so a tab-versus-spaces change would pass unnoticed; it also brings about eight test-only dependencies. goldie has had little recent activity. The needed logic (compare, write on update, report the first differing line) is a few dozen lines of standard library code. |
| Missing snapshot outside an update | Fail | Create it automatically on first run | A snapshot is a claim that the output is correct; it should only be recorded when someone deliberately runs the update and reviews the result. |
| Snapshot contents | Exit status, stdout, and stderr in one file | stdout only | Examples may end in an error to demonstrate it; the error text and exit status are part of what the reader should see. |
| Obsolete snapshots | Failing check; deleted by the update run | Leave them for manual cleanup | A snapshot without an example documents behavior nothing exercises, and renumbering examples would otherwise leave stale files behind. |
| Failure report | First differing line, Go-quoted | A full diff | Pinpoints the change and makes tabs visible; `make update-snapshots` followed by `git diff` shows the complete change. |
| Input for examples that read it | An optional `.input` file beside the snapshot, used as stdin | Answers inside the example file; no examples of `INPUT` and `GET` | The example file must stay a plain BASIC script that a user can run and answer. Keeping the answers next to the snapshot keeps `examples/` to example scripts only, while the snapshot still records a real run. |
| How examples run | In-process `shell.Main` with the file argument | Execute each file through its `#!` line with a built binary | In-process runs are fast and need no build; `#!` execution is already covered by the shell's functional tests. |
| PRINT comments | Required on every line with a `PRINT`, and checked | Optional, where useful | Showing the output beside each form is what makes the examples teach; checking it keeps new examples consistent. |
| Example numbering | Sequential, no gaps, three digits | Free-form names; numbering with gaps for insertion | A fixed reading order from simple to involved suits documentation. Inserting an example renumbers those after it, which is a rename in version control. |

## Open Questions & Future Decisions

### Deferred
1. If the number of examples approaches 999, the number width grows; not expected soon.

## References

- HLD *Tests*, *Build and installation*
- `docs/intent/shell/shell-design.md` (script mode)
