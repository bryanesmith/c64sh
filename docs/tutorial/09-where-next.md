# 9. Where next

[Tutorial](index.md) · Previous: [Saving the game](08-saving-the-game.md)

The Lost Amulet is complete, and it is yours to grow. Some ideas, each using what you have learned:

- **More rooms**: add `DATA` lines and change the room count in line 10010. Nothing else changes, because the world lives in data.
- **A locked door**: a key object, and a check in the movement subroutine before a particular exit.
- **A lamp that runs out**: count moves with the lamp lit, or use `TI`, and darken the lamp after a while.
- **Longer nouns**: match the first four letters instead of three when two objects start alike.
- **A high-score table**: append each final score to a file with `,S,A`, and read it back in an `INPUT#` loop that stops when `ST` is 64.

## The idioms you have learned

| Idiom | Where |
|---|---|
| A short main program that calls subroutines numbered in the thousands | Chapter 1 |
| `GET K$:IF K$="" THEN …` to wait for a key | Chapter 1 |
| Facts in `DATA`, read into arrays with `FOR` loops | Chapter 2 |
| Putting an `IF` inside a loop on a line of its own | Chapter 2 |
| Building a string with `+` in a loop | Chapter 2 |
| Check, then `RETURN`, one problem per line | Chapter 3 |
| Splitting text with `LEN`, `MID$`, and `LEFT$` | Chapter 4 |
| A word table matched on the first three letters | Chapter 4 |
| `ON … GOSUB` to dispatch, and stubs for unwritten parts | Chapter 4 |
| Parallel arrays for several facts about each thing | Chapter 5 |
| Conditions packaged with `DEF FN` | Chapter 6 |
| `X=RND(-TI)` after a key press to seed randomness | Chapter 6 |
| `INT(RND(1)*N)+1` for a random whole number from 1 to `N` | Chapter 6 |
| Comparisons as numbers in arithmetic | Chapter 7 |
| Data files with `OPEN`, `PRINT#`, `INPUT#`, and `ST` | Chapter 8 |

The guide to [idiomatic BASIC](../idioms.md) collects these patterns and many more, each with the reason it exists. Two shorter [projects](index.md#projects), Hammurabi and Battleship, show the same idioms in different games. The [user guide](../user-guide.md) describes every statement in full.
