# Feature examples

Each script here shows one feature of c64sh's BASIC in many forms, with a comment beside every `PRINT` saying what it prints. They are documentation first: read them to see how the language is read, and run any of them directly, such as `./001-hello-world.bas`. They are numbered in reading order, from the simplest feature to the most involved.

## Adding or changing an example

Every change that adds or changes a language feature adds a script here, or extends one, in the same change.

- **Name:** `NNN-lowercase-words.bas`. A new script takes the next number; inserting one renumbers the scripts after it, so the numbers always run from `001` with no gaps.
- **First line:** `#!/usr/bin/env c64sh`, and the file is executable (`chmod +x`).
- **Second line:** a `REM` comment saying what the file shows.
- **Every line with a `PRINT`** ends with `:REM` and what it prints, spaces exactly as printed (`(no newline)` for a missing newline). Where that is impossible, such as after an unclosed string, the comment goes on its own `REM` line just before.
- **A script may end in an error** to show it; lines after the first error do not run.
- **Nothing else** belongs here besides this README.

## How they are tested

The snapshot tests in `test/snapshot/` run every script, as `c64sh FILE` would, and compare its exit status, stdout, and stderr with a recorded snapshot, `test/snapshot/testdata/NNN-lowercase-words.snap`. After adding or changing a script, run:

```sh
make update-snapshots
```

and review the snapshot diff before committing: it is exactly what the script prints. A script that reads input (`INPUT`, `GET`) gets its answers from a hand-written `test/snapshot/testdata/NNN-lowercase-words.input`. The same tests check the conventions above.

The full rules are in [`docs/intent/snapshot/snapshot-design.md`](../../docs/intent/snapshot/snapshot-design.md) and the project's [`AGENTS.md`](../../AGENTS.md).
