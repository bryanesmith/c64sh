# Tutorial: The Lost Amulet

In this tutorial you write a complete game in Commodore 64 BASIC: **The Lost Amulet**, a small text adventure set in a ruined castle. You explore rooms, pick things up, find your way in the dark, dodge a giant bat, and carry the amulet home. Along the way you learn most of the language, the way experienced BASIC programmers used it: how they laid out a program, stored a world in `DATA` statements, read commands, and saved games.

```
GATEHOUSE
A CRUMBLING GATEHOUSE. A PATH LEADS NORTH.
EXITS: N

WHAT NOW? N

COURTYARD
A WEEDY COURTYARD WITH DOORS ALL AROUND.
EXITS: N S E W

WHAT NOW? GO WEST
```

Each chapter adds to the program and ends with the complete program so far, which you can run. The [user guide](../user-guide.md) is the reference for every statement; this tutorial shows how they fit together.

## Before you start

Install c64sh (see the [user guide](../user-guide.md#installing)). You can work in either of two ways:

- **In a file** (recommended): type the program into a file called `adventure.bas` with any text editor, and run it with `c64sh adventure.bas`. A file of numbered lines is a program, and c64sh runs it.
- **At the prompt**, as on a real C64: start `c64sh`, type the numbered lines, and type `RUN`. Save your work with `SAVE "ADVENTURE"` and get it back with `LOAD "ADVENTURE"`.

Type in **uppercase**: BASIC V2 keywords are uppercase, and the game compares what you type with uppercase words. (Caps Lock helps.)

## Chapters

1. [The title screen](01-the-title-screen.md): line numbers, `PRINT`, `TAB`, `GOSUB`, and waiting for a key with `GET`
2. [A world in DATA](02-a-world-in-data.md): variables, arrays, `DATA` and `READ`, and loops
3. [Getting around](03-getting-around.md): the main loop, `INPUT`, `IF`, and `GOTO`
4. [Understanding words](04-understanding-words.md): string functions, a word table, and `ON … GOSUB`
5. [Things to carry](05-things-to-carry.md): objects, `TAKE`, `DROP`, and the inventory
6. [Darkness and danger](06-darkness-and-danger.md): logic, `DEF FN`, `RND`, and the endings
7. [Keeping score](07-keeping-score.md): counting moves, the clock, and a score formula
8. [Saving the game](08-saving-the-game.md): data files with `OPEN`, `PRINT#`, and `INPUT#`
9. [Where next](09-where-next.md): ideas for growing the game, and the idioms you have learned

## Projects

Shorter programs to type in and study once you have finished the game:

- [Hammurabi](projects/hammurabi.md): rule an ancient city for ten years; a classic of 1970s BASIC, built on `INPUT`, validation, and `RND`.
- [Battleship](projects/battleship.md): sink three hidden ships; two-dimensional arrays, `ASC` and `VAL`, and random placement.
