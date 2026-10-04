# Examples and Snapshot Tests Specs

Design: `snapshot-design.md`

## Running examples

- [x] **SNAPSHOT-001**: The snapshot tests in `test/snapshot/` shall run every example script (every file directly in `examples/features/` whose name matches `NNN-words.bas`, SNAPSHOT-002), as `c64sh FILE` runs it, and fail unless the example's exit status, stdout, and stderr exactly match its recorded snapshot, `test/snapshot/testdata/<example name>.snap`.
- [x] **SNAPSHOT-008**: When `test/snapshot/testdata/<example name>.input` exists, the snapshot tests shall run the example with that file's contents as its stdin; otherwise with an empty stdin.
- [x] **SNAPSHOT-009**: The snapshot tests shall fail, naming the file, for any `.input` file in `test/snapshot/testdata/` whose example does not exist, and `make update-snapshots` shall neither create nor delete `.input` files.
- [x] **SNAPSHOT-010**: The snapshot tests shall run each example with a new, empty temporary directory as the current directory, restoring the previous one afterwards.
- [x] **SNAPSHOT-011**: The snapshot tests shall run each example through `shell.Run` with `File` set to the example and `Clock` set to a fixed time.
- [x] **SNAPSHOT-012**: If `examples/features/` does not exist or holds no example scripts, then the snapshot tests shall run no examples, and report no failure for that reason.
- [x] **SNAPSHOT-013**: The snapshot tests shall fail, naming the snapshot, for any `.snap` file in `test/snapshot/testdata/` whose example does not exist, and `make update-snapshots` shall delete it.
- [x] **SNAPSHOT-014**: The snapshot tests shall neither run nor check `examples/features/README.md`, nor any entry of `examples/` outside `examples/features/`.

## Example conventions

- [x] **SNAPSHOT-002**: The snapshot tests shall fail, naming the file, for any non-hidden entry in `examples/features/` that is a directory, or that is not named exactly `README.md` and whose name does not match `NNN-words.bas` (a three-digit number, a hyphen, and one or more lowercase ASCII letter-or-digit words joined by single hyphens); entries whose names begin with `.` shall be ignored.
- [x] **SNAPSHOT-003**: The snapshot tests shall fail unless the example numbers in `examples/features/` are exactly `001` through the number of examples, with no gaps or duplicates.
- [x] **SNAPSHOT-004**: The snapshot tests shall fail, naming the file, for any example whose first line is not exactly `#!/usr/bin/env c64sh`.
- [x] **SNAPSHOT-005**: The snapshot tests shall fail, naming the file, for any example whose second line is not `REM` followed by text containing at least one non-space character.
- [x] **SNAPSHOT-006**: The snapshot tests shall fail, naming the file, for any example without owner execute permission.
- [x] **SNAPSHOT-007**: The snapshot tests shall fail, naming the file and line number, for any line of an example after the first whose tokens (as produced by `lexer.Lex`) include a `Print` token, unless the line's last token before `EOL` is a `Rem` token or the line immediately before it is a comment line (a line whose only token before `EOL` is a `Rem` token).
