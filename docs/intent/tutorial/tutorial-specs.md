# Tutorial Specs

Design: `tutorial-design.md`

## Pages and tests

- [x] **TUTORIAL-001**: `docs/tutorial/index.md` shall link every chapter in order, and every project, and each chapter shall link to the index and to its previous and next chapters.
- [x] **TUTORIAL-002**: For each chapter or project page with a `basic` code block, the tutorial tests shall fail, naming the line, for any line of an earlier `basic` block that is not a line of the page's last `basic` block.
- [x] **TUTORIAL-003**: For each chapter or project page with a `basic` code block, the tutorial tests shall run its last `basic` block as a program file in a new temporary current directory, through `shell.Run` with a fixed clock and stdin from `test/tutorial/testdata/<page>.input` (empty if there is none), and fail unless the exit status, stdout, and stderr match `test/tutorial/testdata/<page>.snap`; with `UPDATE_SNAPS=true` they shall write that file instead.
- [x] **TUTORIAL-004**: The tutorial tests shall fail, naming the file, for any file in `test/tutorial/testdata/` that belongs to no chapter or project page, and for any page whose name is not `NN-lowercase-words.md` or `projects/lowercase-words.md`.
- [x] **TUTORIAL-005**: The tutorial tests shall fail, naming the line, for any line of a `basic` block in `docs/idioms.md` that does not parse as c64sh BASIC (after its line number, if it has one) without a syntax error, including one kept in a `DEF FN` body.
