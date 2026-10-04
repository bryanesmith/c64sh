---
parent: high-level-design
prefix: TUTORIAL
---

# Tutorial

## Context and Design Philosophy

The user guide is a reference: it says what each statement does. People learn a language better by building something with it, and a reference cannot show how experienced programmers combined the statements. The tutorial teaches BASIC V2 by building one program, a text adventure called *The Lost Amulet*, chapter by chapter, using the idioms C64 programmers used: a short main program calling subroutines numbered in the thousands, facts in `DATA` read into arrays, `ON … GOSUB` dispatch, "check, then return" validation, seeding `RND` after a key press, and so on.

The tutorial is documentation, so it must be correct: every listing in it is run by tests, and the excerpts in its prose are checked against its listings.

## Pages

The tutorial lives in `docs/tutorial/`:

- `index.md` introduces the game, explains how to work (in a file run with `c64sh adventure.bas`, or at the prompt), and links every chapter in order.
- **Chapters**, named `NN-lowercase-words.md` (two digits, numbered from `01`), each begin with links to the index and to the previous and next chapters. A chapter that adds to the program ends with the **complete program so far**, a fenced code block marked `basic`, which is the chapter's last such block. Earlier `basic` blocks in the chapter are excerpts: every line in them is a line of that chapter's complete listing. Commands typed at the prompt and transcripts use plain fenced blocks, not `basic` ones.
- **Projects**, in `docs/tutorial/projects/`, named `lowercase-words.md`, are shorter, self-contained programs linked from the index, following the same rules as chapters.

Each chapter's program is the previous chapter's with lines added, replaced, or deleted (the chapter says which to delete), so a reader can type the changes, or copy the complete listing.

## The idioms guide

`docs/idioms.md` is a companion to the tutorial: the patterns experienced BASIC programmers used, grouped by topic (program structure, keyboard input, random numbers, numbers, strings, layout, loops, data, files, time), each with a short example in a `basic` block and a brief explanation of why the pattern exists. It describes only what c64sh supports: a test parses every line of every `basic` block in it and fails on any syntax error, including one in a `DEF FN` body. Patterns that depend on a feature c64sh lacks are left out until the feature exists.

## Keeping the tutorial current

When a language feature is added or changed, the tutorial and the idioms guide are updated in the same change where the feature belongs, as the user guide is; the idioms guide gains any pattern the feature makes standard. For the tutorial: a chapter that uses the feature shows it, and a feature that the game can use naturally is worked into it (a new chapter, or an existing chapter's program) rather than left to the reference alone. A change in behavior that alters a listing's output appears in the tutorial tests, like any snapshot.

## Tests

The tutorial tests live in `test/tutorial/` (package `tutorial_test`). For each chapter and project page with at least one `basic` block:

1. Every line of each earlier `basic` block must appear as a line of the page's last `basic` block, the complete listing; otherwise the test fails, naming the line.
2. The listing is written to a file in a new temporary directory, which is the current directory while it runs, and run through `shell.Run` with that file, a fixed clock (so `RND(-TI)` and the clock give the same results every run), and stdin from `test/tutorial/testdata/<page>.input` if it exists, where `<page>` is the page's path under `docs/tutorial/` without `.md`, with `/` written as `-` (`06-darkness-and-danger`, `projects-hammurabi`).
3. The exit status, stdout, and stderr are compared with `test/tutorial/testdata/<page>.snap`. `make update-snapshots` rewrites these files, as it rewrites the examples' snapshots.

A file in `test/tutorial/testdata/` that belongs to no page fails the tests. A page name that does not follow the naming rule fails the tests.

Input files are written by hand: they play the game, so the recorded output is a sample session. Because the clock is fixed, the game's random events are the same in every run, and the input is written to suit them.

## Decisions & Alternatives

| Decision | Chosen | Alternatives Considered | Rationale |
|---|---|---|---|
| Teaching approach | One program built over several chapters, plus short projects | A collection of small unrelated examples; reference only | A growing program shows how the pieces fit, why idioms exist, and how real programs are organized; small examples already exist in `examples/`. |
| Project | A text adventure | Lunar Lander, a ledger, a quiz | It uses nearly every part of the language naturally, is fun to play at every step, and has room for future features (see the HLD). |
| Where listings live | In the chapter pages, tested by extracting the last `basic` block | Separate `.bas` files included or linked from the pages | The page is the single source: what the reader sees is what the test runs. |
| Excerpts | Checked to be lines of the chapter's listing | Unchecked | Excerpts are where prose and code drift apart; the check costs little. |

## References

- `docs/user-guide.md`, the reference for every statement
- `docs/intent/snapshot/snapshot-design.md`, whose snapshot format and update flow the tutorial tests follow
