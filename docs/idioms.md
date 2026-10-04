# Idiomatic BASIC

The patterns experienced Commodore 64 programmers used, and why. BASIC V2 is a small language with few built-in conveniences, so programmers developed standard ways of doing common things; knowing them makes BASIC programs shorter, clearer, and easier to read for anyone who learned the language back then.

Everything here works in c64sh. The [user guide](user-guide.md) describes each statement in full, and the [tutorial](tutorial/index.md) uses most of these patterns in a complete game.

- [Program structure](#program-structure)
- [Keyboard input](#keyboard-input)
- [Random numbers](#random-numbers)
- [Numbers](#numbers)
- [Strings](#strings)
- [Output and layout](#output-and-layout)
- [Colors and the screen](#colors-and-the-screen)
- [Loops](#loops)
- [Data and arrays](#data-and-arrays)
- [Files](#files)
- [Time](#time)
- [The environment](#the-environment) (c64sh extension)

## Program structure

### Number lines in tens

```basic
10 PRINT "FIRST"
20 PRINT "THIRD"
15 PRINT "SECOND"
```

**Why:** lines run in number order, not the order they were typed, and BASIC V2 has no command to renumber a program. Leaving gaps lets you insert lines later (15 between 10 and 20) without retyping everything.

### A short main program, subroutines by the thousand

```basic
10 GOSUB 9000:REM SET UP
20 GOSUB 1000:REM PLAY A ROUND
30 GOTO 20
1000 REM PLAY A ROUND
1990 RETURN
9000 REM SET UP
9990 RETURN
```

**Why:** BASIC V2 has no named procedures, only line numbers. Keeping the main loop at the top and giving each job its own block of thousands (1000s for one task, 2000s for another) makes a long program navigable, and a `REM` on each block's first line names it. Put `END` (or a loop) before the first subroutine, so the program cannot run into it.

### Guard clauses: check, then RETURN

```basic
3000 REM TAKE AN OBJECT
3010 IF N$="" THEN PRINT "TAKE WHAT?":RETURN
3020 IF O=0 THEN PRINT "I DON'T SEE THAT.":RETURN
3030 IF W>9 THEN PRINT "YOU CAN'T CARRY MORE.":RETURN
3040 PRINT "TAKEN.":RETURN
```

**Why:** BASIC V2's `IF` has no `ELSE` and no blocks, so nesting conditions is awkward. Ruling out one problem per line, leaving the subroutine as soon as something is wrong, keeps every check simple, and only valid commands reach the end.

### An IF inside a loop goes on its own line

```basic
10 FOR I=1 TO 10
20 IF I/2=INT(I/2) THEN PRINT I;
30 NEXT
```

**Why:** when an `IF` is false, BASIC skips **the rest of its line**. Written as `FOR I=1 TO 10:IF I/2=INT(I/2) THEN PRINT I;:NEXT`, the first odd number would skip the `NEXT` too, ending the loop. Splitting the loop over lines keeps the `NEXT` safe.

### ON … GOSUB instead of a chain of IFs

```basic
100 ON C GOSUB 1000,2000,3000
110 IF C<1 OR C>3 THEN PRINT "CHOOSE 1, 2, OR 3."
```

**Why:** BASIC V2 has no `SELECT CASE`. `ON C GOSUB` calls the `C`th subroutine in its list in one line, and simply does nothing if `C` is 0 or larger than the list (a negative `C` is an error), so the line after it can catch bad choices.

### A flag that ends the main loop

```basic
100 GOSUB 2000:REM ONE TURN; SETS DN WHEN THE GAME IS OVER
110 IF DN=0 THEN 100
120 PRINT "GAME OVER"
```

**Why:** a subroutine cannot jump out to a different part of the program cleanly (a `GOTO` out of a `GOSUB` leaves its return address behind, and `?OUT OF MEMORY  ERROR` follows eventually). Instead it sets a variable, and the main loop checks it.

## Keyboard input

### Wait for any key

```basic
10 PRINT "PRESS ANY KEY."
20 GET K$:IF K$="" THEN 20
```

**Why:** `GET` never waits: it gives the empty string when no key has been pressed. So a program that must wait loops back to the same line until a key arrives. `INPUT` would need Return, and would print a `?`.

### A one-key answer, checked

```basic
100 PRINT "PLAY AGAIN (Y/N)?"
110 GET K$:IF K$<>"Y" AND K$<>"N" THEN 110
120 IF K$="Y" THEN RUN
```

**Why:** the same loop also rejects every key except the ones you want, so the program never has to handle a wrong answer. (`RUN` restarts the whole program, with variables cleared.)

### A menu chosen by a key

```basic
100 PRINT "1 NEW GAME  2 LOAD  3 QUIT"
110 GET K$:IF K$<"1" OR K$>"3" THEN 110
120 ON VAL(K$) GOSUB 1000,2000,3000
```

**Why:** strings compare by character code, so `K$<"1" OR K$>"3"` keeps only the keys `1` to `3`; `VAL` turns the key into a number for `ON … GOSUB`.

### Ask again until the answer makes sense

```basic
100 INPUT "HOW MANY PLAYERS (1-4)";P
110 IF P<1 OR P>4 OR P<>INT(P) THEN PRINT "1 TO 4, PLEASE.":GOTO 100
```

**Why:** `INPUT` accepts any number, so every answer is checked and the question repeated with `GOTO`. (A non-number already gets the C64's own `?REDO FROM START`.)

### Check only the first letter

```basic
100 INPUT "DO YOU WANT INSTRUCTIONS";A$
110 IF LEFT$(A$,1)="Y" THEN GOSUB 5000
```

**Why:** players type `Y`, `YES`, or `YEP`; comparing the first letter accepts them all. The same trick, comparing the first three letters, lets a word list accept `LOOK`, `LOO`, and `LOOKING`.

## Random numbers

### Seed with RND(-TI), then use RND(1)

```basic
10 PRINT "PRESS ANY KEY TO START."
20 GET K$:IF K$="" THEN 20
30 X=RND(-TI)
40 PRINT INT(RND(1)*6)+1
```

**Why:** `RND` does not make truly random numbers; it computes each number from the last one (the *seed*), so after the computer starts, the sequence is the same every time. A **negative** argument sets a new seed from that number, which is why `RND(-TI)` comes once, at the start. A **positive** argument (any positive number; 1 by convention) gives the next number in the sequence, which is what you want every time after. `TI` is the clock, in sixtieths of a second since the computer started, so it is different every run, and seeding *after waiting for a key* makes it depend on the player's timing, which is never the same twice. (`X=` just gives the unwanted first number somewhere to go.) In c64sh, `RND(0)` also reseeds from the clock.

### A whole number from 1 to N

```basic
10 DEF FN R(X)=INT(RND(1)*X)+1
20 PRINT "DIE:";FN R(6),"CARD:";FN R(52)
```

**Why:** `RND(1)` is at least 0 and less than 1; times `X` it is at least 0 and less than `X`; `INT` rounds down to 0 … `X`-1; adding 1 gives 1 … `X`. Wrapping it in `DEF FN` makes every random choice in the program short and readable.

### Something that happens one time in N

```basic
100 IF RND(1)<.25 THEN PRINT "A STORM BLOWS IN!"
```

**Why:** `RND(1)` is evenly spread from 0 to 1, so it is below .25 a quarter of the time. This is the quickest way to give an event a probability.

### Shuffle an array

```basic
100 FOR I=N TO 2 STEP -1
110 J=INT(RND(1)*I)+1
120 T=D(I):D(I)=D(J):D(J)=T
130 NEXT
```

**Why:** swapping each element with a random one at or before it (the Fisher-Yates shuffle) gives every order an equal chance, and needs no second array, which mattered with 38911 bytes of memory. `T` holds one value during the swap, because BASIC has no swap statement.

### The same "random" game every time, for testing

```basic
10 X=RND(-1)
```

**Why:** seeding with a fixed negative number gives the same sequence every run, which makes a bug that depends on luck repeatable. Change it back to `RND(-TI)` when the program works.

## Numbers

### Round to the nearest whole number, or to two places

```basic
10 X=2.71828
20 PRINT INT(X+.5),INT(X*100+.5)/100
```

**Why:** `INT` always rounds *down*, so adding .5 first rounds to the nearest. Scaling by 100 before, and dividing after, rounds to two decimal places, as for money.

### Remainder without MOD

```basic
10 A=17:B=5
20 Q=INT(A/B):R=A-Q*B
30 PRINT Q;R
```

**Why:** BASIC V2 has no `MOD` operator. The remainder is what is left after taking away the whole number of `B`s. `IF X/2=INT(X/2)` is the usual test for an even number.

### Comparisons are numbers

```basic
10 X=7
20 S=10*(X>5)
30 C=C-(X=7)
40 PRINT S;C
```

**Why:** a comparison is -1 when true and 0 when false, so it can be used in arithmetic. `10*(X>5)` is -10 or 0; subtracting a comparison (`C-(X=7)`) adds 1 when it is true. This replaces small `IF`s, especially inside `DEF FN`, which can hold only an expression.

### Flip a flag with NOT

```basic
10 F=0
20 F=NOT F:PRINT F
30 F=NOT F:PRINT F
```

**Why:** with true as -1 (every bit set) and false as 0, `NOT` turns one into the other, so `F=NOT F` toggles a flag, such as a light switch or whose turn it is.

### Integer variables for big arrays

```basic
10 DIM M%(100,100)
```

**Why:** an integer array element takes 2 bytes, a number 5, so `M%` fits where a number array would run out of memory. Integer variables are not faster for arithmetic on a C64 (the ROM converts them to floating point anyway), so plain variables are the norm everywhere else.

## Strings

### Build a string piece by piece

```basic
10 L$="":FOR I=1 TO 10:L$=L$+"-":NEXT
20 PRINT L$
```

**Why:** BASIC V2 has no function to repeat a character, so a loop adds one at a time with `+`.

### Go through a string's characters

```basic
10 A$="HELLO"
20 FOR I=1 TO LEN(A$)
30 PRINT MID$(A$,I,1);"-";
40 NEXT:PRINT
```

**Why:** `MID$(A$,I,1)` is the `I`th character; there is no other way to index a string. Counting `I` down instead (`FOR I=LEN(A$) TO 1 STEP -1`) reverses it.

### Look things up by position in a string

```basic
10 D=3
20 PRINT MID$("NSEW",D,1)
30 PRINT MID$("SUNMONTUEWEDTHUFRISAT",D*3-2,3)
```

**Why:** a string of fixed-width entries is the shortest lookup table there is: no `DATA`, no array. Each entry is `D*W-(W-1)` characters in, where `W` is its width.

### Split at a space

```basic
10 C$="TAKE LAMP":P=0
20 FOR I=1 TO LEN(C$)
30 IF P=0 AND MID$(C$,I,1)=" " THEN P=I
40 NEXT
50 IF P THEN V$=LEFT$(C$,P-1):N$=MID$(C$,P+1)
```

**Why:** BASIC V2 has no function to search a string (no `INSTR`), so a loop finds the space. `P=0 AND …` keeps the first space found.

### A number without its leading space

```basic
10 N=42
20 PRINT "SCORE:"+MID$(STR$(N),2)
```

**Why:** `STR$` formats a number as `PRINT` does, with a space (or `-`) in front; `MID$(…,2)` drops it, for a non-negative number, when joining numbers into text.

### Right-align numbers in a column

```basic
10 N=7
20 PRINT RIGHT$("      "+STR$(N),6)
```

**Why:** padding with spaces in front and then keeping the last 6 characters makes every number the same width, so columns line up on the right, as for prices and scores.

### A quote mark inside a string

```basic
10 PRINT "SHE SAID ";CHR$(34);"HELLO";CHR$(34)
```

**Why:** a string cannot contain `"`, which would end it, so `CHR$(34)`, the quote's character code, puts one in.

## Output and layout

### Stay on the line with ;

```basic
10 PRINT "LOADING";
20 FOR I=1 TO 3:PRINT ".";:NEXT
30 PRINT
```

**Why:** a `PRINT` ending in `;` leaves the cursor where it is, so the next `PRINT` continues the line; a bare `PRINT` ends it (or prints a blank line).

### Columns with TAB, zones with commas

```basic
10 PRINT "NAME";TAB(14);"SCORE"
20 PRINT "ALICE";TAB(14);120
30 PRINT "A","B","C"
```

**Why:** `TAB(N)` moves to column `N`, wherever the line has got to, so different-length names still line up. A comma moves to the next 10-column zone: quick, when the data fits.

### Center a title

```basic
10 T$="THE LOST AMULET"
20 PRINT TAB((40-LEN(T$))/2);T$
```

**Why:** a C64 screen is 40 columns wide, so starting at half the leftover space centers the text. (`TAB` rounds the column down.)

### Draw a bar

```basic
10 FOR I=1 TO 3
20 PRINT I;SPC(I*3);"#"
30 NEXT
```

**Why:** `SPC(N)` moves right `N` columns, so a computed amount draws simple bar charts and indents without building strings.

## Colors and the screen

### Clear the screen first

```basic
10 PRINT CHR$(147)
```

**Why:** the C64 has no `CLS` statement; printing character 147 clears the screen and puts the cursor at the top left. Nearly every C64 program starts this way.

### Keep control codes in variables

```basic
10 RE$=CHR$(28):YE$=CHR$(158):LB$=CHR$(154)
20 PRINT YE$;"WARNING: ";RE$;"LOW FUEL";LB$
```

**Why:** colors are set by printing characters, and `CHR$(28)` says nothing about red. Named variables make the `PRINT` statements readable, and set up once, at the start, they cost nothing later.

### Set the color back

```basic
10 PRINT CHR$(28);"GAME OVER";CHR$(154)
```

**Why:** a color stays in effect until another is printed, so a program that changes it sets it back to the usual light blue (`CHR$(154)`) afterwards, or every later line is red too.

### Reverse video for headings and menus

```basic
10 PRINT CHR$(18);" MAIN MENU ";CHR$(146)
```

**Why:** reverse video (`CHR$(18)` on, `CHR$(146)` off) makes a bar that stands out without needing any graphics characters.

## Loops

### Count down with a negative STEP

```basic
10 FOR I=10 TO 1 STEP -1:PRINT I;:NEXT:PRINT
```

**Why:** `STEP` can be negative, and fractional (`STEP .5`). Without `STEP` a `FOR` counts up by 1, and runs its body once even if the start is already past the end.

### Leave a FOR loop early by finishing it

```basic
10 FOR I=1 TO 100
20 IF I*I>50 THEN F=I:I=100
30 NEXT
40 PRINT F
```

**Why:** setting the counter to its end value makes the next `NEXT` finish the loop normally. Jumping out with `GOTO` also works (a later `FOR` with the same variable replaces the old loop), but repeatedly jumping out of loops that never finish, especially inside subroutines, leaves loops on the C64's stack and leads to `?OUT OF MEMORY  ERROR`.

### Close nested loops together

```basic
10 FOR R=1 TO 3:FOR C=1 TO 3:PRINT R*C;:NEXT C:PRINT:NEXT R
```

**Why:** naming the variable on each `NEXT` documents which loop ends where; `NEXT C,R` closes both at once when nothing comes between them.

### Loop until something happens

```basic
100 GOSUB 2000:REM ONE MOVE
110 IF H>0 THEN 100
```

**Why:** BASIC V2 has no `WHILE` or `REPEAT`; an `IF … THEN` back to the start of the loop is the standard loop with a condition.

## Data and arrays

### Keep facts in DATA, with a count first

```basic
10 READ N:DIM C$(N)
20 FOR I=1 TO N:READ C$(I):NEXT
30 DATA 3,RED,GREEN,BLUE
```

**Why:** putting the facts in `DATA` lines separates them from the code: adding a color means editing data, not logic. Reading the count first sizes the array exactly.

### Or end the list with a marker

```basic
10 READ W$:IF W$="END" THEN 40
20 PRINT W$
30 GOTO 10
40 PRINT "DONE"
50 DATA APPLE,PEAR,PLUM,END
```

**Why:** with a marker value, adding items never means updating a count. `RESTORE` goes back to the first item to read the list again.

### DIM everything at the start

```basic
10 DIM N$(20),S(20),B(8,8)
```

**Why:** an array used without `DIM` gets 10 as its top subscript automatically, and a later `DIM` of it is `?REDIM'D ARRAY  ERROR`. Dimensioning everything once, at the start, avoids both surprises and shows the program's data at a glance.

### Parallel arrays for records

```basic
10 DIM N$(3),S(3)
20 N$(1)="ALICE":S(1)=120
30 N$(2)="BOB":S(2)=95
```

**Why:** BASIC V2 has no records or structures, so several facts about each thing go in several arrays that share a subscript: `N$(I)` and `S(I)` are player `I`'s name and score.

### Short names, no keywords inside

```basic
10 PN$="ALICE":HS=120
```

**Why:** only the first two characters of a name count (`HEIGHT` and `HEAVY` are the same variable, `HE`), and a C64 finds keywords anywhere in a name: `TOTAL` contains `TO`, and `SCORE` contains `OR`, so both are syntax errors. Short, deliberate names avoid both traps; a `REM` can say what they mean.

## Files

### Write a file, replacing any old one

```basic
10 OPEN 2,8,2,"@0:SCORES,S,W"
20 PRINT#2,"ALICE";",";120
30 CLOSE 2
```

**Why:** `8` is the first disk drive, `,S,W` a sequential file for writing, and `@0:` replaces the file if it exists, because a C64 disk drive otherwise refuses to (as c64sh does). Write values separated by `;",";` so they are separated by commas in the file: `PRINT#2,A,B` would put a 10-column zone of spaces between them. Always `CLOSE`: the file is complete only then.

### Read a file to its end with ST

```basic
10 OPEN 2,8,2,"SCORES"
20 INPUT#2,N$,S:PRINT N$,S
30 IF ST=0 THEN 20
40 CLOSE 2
```

**Why:** `ST` is the status of the last read, and becomes 64 when reading reaches the end of the file; checking it after each read stops the loop at the right time. (A C64 does not stop a program reading past the end; the values it gets are meaningless.)

### Add to the end of a file

```basic
10 OPEN 2,8,2,"LOG,S,A"
20 PRINT#2,"ANOTHER LINE"
30 CLOSE 2
```

**Why:** `,S,A` appends, keeping what is there: the usual way to keep a log or a high-score list. The file must already exist (otherwise it is a `?FILE NOT FOUND  ERROR`), so a program creates it once with `,S,W`.

### Print a listing on the printer

```basic
OPEN 4,4:CMD 4:LIST
PRINT#4:CLOSE 4
```

**Why:** `CMD 4` sends everything that would go to the screen to the printer (device 4), including `LIST`; `PRINT#4` ends the redirection, and `CLOSE` finishes the file. In c64sh the printer is the program's output.

### Save your program before running it

```basic
SAVE "@0:GAME",8
```

**Why:** a crash or a typo like `NEW` loses unsaved work. `@0:` replaces the previous copy on disk.

## Time

### Time something

```basic
10 T=TI
20 FOR I=1 TO 1000:NEXT
30 PRINT "TOOK";(TI-T)/60;"SECONDS"
```

**Why:** `TI` counts sixtieths of a second, so the difference of two readings, divided by 60, is the time in seconds. `TI$="000000"` resets the clock instead, when you want `TI$` to show elapsed time as `HHMMSS`.

### Wait a moment

```basic
10 T=TI
20 IF TI-T<60 THEN 20
30 PRINT "ONE SECOND LATER"
```

**Why:** BASIC V2 has no `SLEEP`; looping until the clock has moved on waits for a set time, the same on every computer. (An empty `FOR` loop also waits, but for different times on different machines.)

## The environment

*c64sh extension: `ENVIRON$` and `ENVIRON` are not part of C64 BASIC V2, so these patterns do not work on a real C64.*

### Read a setting with a default

```basic
10 E$=ENVIRON$("EDITOR"):IF E$="" THEN E$="VI"
20 PRINT "EDITING WITH ";E$
```

**Why:** an unset variable reads as the empty string, never an error, so testing for `""` straight after reading it is how a program falls back to its own default.

### Extend PATH

```basic
ENVIRON "PATH="+ENVIRON$("PATH")+":/opt/bin"
```

**Why:** `PATH` is a list of directories separated by `:`, searched in order, so adding to the end makes a directory a last resort, and `ENVIRON "PATH=/opt/bin:"+ENVIRON$("PATH")` puts it first. `PATH` is usually longer than a C64 string's 255 characters; c64sh's strings are unlimited by default, so this works unless `C64SH_STRING_LIMIT` is set.

### List every variable

```basic
10 I=1
20 E$=ENVIRON$(I):IF E$="" THEN END
30 PRINT E$:I=I+1:GOTO 20
```

**Why:** `ENVIRON$(N)` gives the `N`th variable as `NAME=VALUE`, in order of name, and the empty string past the last, so counting up until it comes back empty visits them all.
