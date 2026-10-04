# c64sh

## LID
- Mode: Full
- Version: 1.3.0

## Examples and Snapshot Tests

Every language feature is shown in a script in `examples/`, and each script's output is recorded by a snapshot test in `test/snapshot/`. **When adding or changing a feature, add a new example or extend an existing one in the same change**, then run `make update-snapshots` and review the snapshot diff before committing.

Example scripts follow these conventions (the snapshot tests enforce them):

- Named `NNN-lowercase-words.bas`, numbered sequentially from `001` with no gaps (`001-hello-world.bas`, `002-concat-strings.bas`).
- Executable (`chmod +x`), with `#!/usr/bin/env c64sh` as the first line and a `REM` comment explaining the file as the second.
- Show the feature in many forms: they are documentation for users learning how the language is read.
- End every line containing a `PRINT` with a comment saying what it prints, with spaces exactly as printed (write a missing newline as `(no newline)`). To make print-zone columns easy to check, an example may print a ruler line such as `0123456789012345678901234567890`. The comment needs a colon before `REM`, because `PRINT` only ends at `:` or the end of the line: `PRINT "HI":REM HI`, not `PRINT "HI" REM HI` (a syntax error, as on a C64). Where an end-of-line comment is impossible, such as after an unclosed string (`PRINT "HI`, which runs to the end of the line), put the comment on its own `REM` line immediately before.
- A script that reads input (`INPUT`, `GET`) gets its stdin from `test/snapshot/testdata/NNN-lowercase-words.input`, written by hand; its comments name the answers the snapshot uses, and its `PRINT` comments describe the output for them.
- A script may show an error. Scripts stop at their first error, so lines after it do not run; the snapshot records exactly what happens.
- Hidden files such as `.DS_Store` are ignored; nothing else but example scripts belongs in `examples/`.

## Tutorial

`docs/tutorial/` teaches the language by building a text adventure, chapter by chapter, in idiomatic BASIC. **When adding or changing a feature, update the tutorial in the same change where the feature belongs**: work it into the game or a project, or show it in the chapter that uses it, as the user guide is updated. Each chapter ends with its complete program in a ```` ```basic ```` block; earlier ```` ```basic ```` blocks are excerpts of it. The tests in `test/tutorial/` run every listing with input from `test/tutorial/testdata/<page>.input` and compare it with `<page>.snap`; run `make update-snapshots` and review the diff.

## Linked-Intent Development (MANDATORY)

**Consult the `linked-intent-dev` skill for ALL code changes.** All changes flow through the arrow of intent in one direction:

```
HLD → LLDs → EARS → Tests → Code
```

- **New features and refactors**: full six-phase workflow (HLD check → LLD check/draft → EARS → intent-narrowing edge audit → tests-first → code).
- **Bug fixes**: walk the arrow like any other change — find where behavior diverged from intent and cascade from there. No short-circuit.
- **If unsure**: use the full workflow.

Stop after each phase for user review. **Docs carry current intent, written to be read cold** — write each doc as if authored fresh today, from current intent alone: no narration of how it changed, no meaning that needs the conversation that produced it, no rebuttals to questions only a past discussion raised. Rationale, considered alternatives, and constraints a fresh author would independently write stay; record rejected alternatives and why in the LLD's Decisions & Alternatives table, not as asides in body prose.

**Memory vs. intent.** Before saving durable project knowledge to agent or tool memory, test whether it is project *intent* — would a fresh agent, in any tool, next session, need it to build this system correctly? If yes, record it in the arrow (HLD / LLD / EARS / decision doc), which travels and cascades — not in private, per-tool memory, where intent escapes the arrow. Knowledge about the user or how they like to work stays in memory.

### Navigation

| What you need | Where to look |
|---|---|
| High-level design | `docs/high-level-design.md` |
| Design tree (sub-HLDs, LLDs, their specs) | `docs/intent/` — one folder per node |
| EARS specs | beside each design doc as `{node}-specs.md` in the node's folder under `docs/intent/` |
| Decision docs | `docs/decisions/` (project-level) and `docs/intent/<segment>/decisions/` |

### Terminology

- **HLD**: High-Level Design — single project-level doc at `docs/high-level-design.md`.
- **LLD**: Low-Level Design — detailed component design doc in `docs/intent/`. The design layer is a recursive tree: the root is the HLD, leaf LLDs own EARS, and a component deep enough to outgrow one doc becomes a sub-HLD (HLD-shaped, owns no EARS) with children beneath it. "HLD" and "LLD" are roles by position; depth-2 (one HLD over flat leaf LLDs) is the default.
- **EARS**: Easy Approach to Requirements Syntax — structured one-line requirements beside each design doc as `{node}-specs.md` in the node's folder under `docs/intent/`. IDs are path-concatenated — the root-to-leaf path of the owning segment plus a number — so a prefix grep gathers a subtree. Markers: `[x]` implemented, `[ ]` active gap, `[D]` deferred.
- **Arrow**: the unidirectional chain from vision to code (HLD → LLDs → EARS → Tests → Code). Strictly a DAG of intent.
- **Arrow segment**: the territory owned by one leaf LLD — the LLD itself plus the specs, tests, and code that cite its EARS IDs. The boundary is the leaf prefix. Within-segment cascade is free; across-segment cascade pauses.
- **Cascade**: propagating a change downstream through the arrow so adjacent levels stay coherent.

### Code annotations

Annotate code and tests with `@spec` comments citing EARS IDs:

```
// @spec LEXER-007, LEXER-008
```

Place the annotation at the *entry point of the behavior's implementation graph* — the topmost function or module owning the specified behavior, not every helper. When a behavior spans multiple subsystems, annotate at the entry point in each subsystem. Tests follow the same rule: annotate the test that directly exercises the spec, not every inner assertion.
