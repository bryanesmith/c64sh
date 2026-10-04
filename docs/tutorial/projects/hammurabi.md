# Project: Hammurabi

[Tutorial](../index.md)

*Hammurabi* is one of the oldest computer games: written in 1968, and made famous by David Ahl's *BASIC Computer Games* (1978). You rule an ancient city for ten years. Each year you buy or sell land, decide how much grain your people eat, and how much land to plant; then the harvest, the rats, and the plague decide the rest. It is short, and it shows how much a game can do with a few variables, `INPUT`, and `RND`.

## The state of the city

```basic
100 P=100:G=2800:A=1000:Y=3:E=200:B=5:Z=1
```

The whole game is a handful of numbers: `P` people, `G` grain in bushels, `A` acres, `Y` the last harvest per acre, `E` grain eaten by rats, `B` people born or arrived, and `Z` the year. Each year reports them, and asks for decisions.

## Asking, and asking again

```basic
400 INPUT "HOW MANY BUSHELS WILL YOU FEED THEM";F
410 IF F<0 THEN 400
420 IF F>G THEN GOSUB 800:GOTO 400
430 G=G-F
```

Every answer is checked before it is used, and a bad answer asks again with `GOTO`. The complaint is a subroutine, because several questions share it:

```basic
800 PRINT "HAMMURABI, THINK AGAIN. YOU HAVE ONLY";G;"BUSHELS.":RETURN
```

## Fate

```basic
90 DEF FN R(X)=INT(RND(1)*X)+1
510 Y=FN R(5):G=G+S*Y
520 E=0:IF RND(1)<.4 THEN E=INT(G*RND(1)/4):G=G-E
560 Q=RND(1)<.15:IF Q THEN P=INT(P/2)
```

`FN R(X)` is a random whole number from 1 to `X`, the most useful one-line function in BASIC. The harvest yields 1 to 5 bushels an acre; in four years out of ten, rats eat up to a quarter of the store; and line 560 stores a comparison in `Q`: -1 (true) for a plague, which strikes three years in twenty, and which the next report mentions.

## Starvation

```basic
530 D=P-INT(F/20):IF D<0 THEN D=0
540 IF D>.45*P THEN 950
```

Each person needs 20 bushels; starve more than 45% of the people, and the reign ends early.

## Try it

The sample session below feeds everyone and plants as much as it can, selling land when grain runs short. Can you do better, ending with more than ten acres for each person?

## The complete program

```basic
10 REM HAMMURABI
20 PRINT TAB(14);"HAMMURABI"
30 PRINT "RULE THE ANCIENT CITY OF SUMER FOR TEN"
40 PRINT "YEARS. BUY AND SELL LAND, FEED YOUR"
50 PRINT "PEOPLE, AND PLANT GRAIN FOR NEXT YEAR."
55 PRINT "EACH PERSON EATS 20 BUSHELS A YEAR AND CAN"
56 PRINT "FARM 10 ACRES; SEED IS HALF A BUSHEL AN ACRE."
60 PRINT:PRINT "PRESS ANY KEY.":REM SEED RND FROM THE KEY PRESS
70 GET K$:IF K$="" THEN 70
80 X=RND(-TI)
90 DEF FN R(X)=INT(RND(1)*X)+1
100 P=100:G=2800:A=1000:Y=3:E=200:B=5:Z=1
200 REM REPORT ON THE YEAR
210 PRINT:PRINT "HAMMURABI, I BEG TO REPORT THAT IN YEAR";Z
220 PRINT D;"PEOPLE STARVED, AND";B;"CAME TO THE CITY."
230 IF Q THEN PRINT "A HORRIBLE PLAGUE KILLED HALF THE PEOPLE."
240 PRINT "THE POPULATION IS NOW";P
250 PRINT "THE CITY OWNS";A;"ACRES."
260 PRINT "YOU HARVESTED";Y;"BUSHELS PER ACRE."
270 PRINT "RATS ATE";E;"BUSHELS."
280 PRINT "YOU NOW HAVE";G;"BUSHELS IN STORE."
290 IF Z>10 THEN 900
300 REM THE KING'S DECISIONS
310 L=FN R(10)+16:PRINT:PRINT "LAND COSTS";L;"BUSHELS PER ACRE."
320 INPUT "HOW MANY ACRES WILL YOU BUY";AB
330 IF AB<0 THEN 320
340 IF AB*L>G THEN GOSUB 800:GOTO 320
350 IF AB>0 THEN A=A+AB:G=G-AB*L:GOTO 400
360 INPUT "HOW MANY ACRES WILL YOU SELL";SL
370 IF SL<0 THEN 360
380 IF SL>A THEN GOSUB 810:GOTO 360
390 A=A-SL:G=G+SL*L
400 INPUT "HOW MANY BUSHELS WILL YOU FEED THEM";F
410 IF F<0 THEN 400
420 IF F>G THEN GOSUB 800:GOTO 400
430 G=G-F
440 INPUT "HOW MANY ACRES WILL YOU PLANT";S
450 IF S<0 THEN 440
460 IF S>A THEN GOSUB 810:GOTO 440
470 IF INT(S/2)>G THEN GOSUB 800:GOTO 440
480 IF S>10*P THEN PRINT "BUT YOU HAVE ONLY";P;"PEOPLE TO FARM IT.":GOTO 440
490 G=G-INT(S/2)
500 REM THE YEAR PASSES
510 Y=FN R(5):G=G+S*Y
520 E=0:IF RND(1)<.4 THEN E=INT(G*RND(1)/4):G=G-E
530 D=P-INT(F/20):IF D<0 THEN D=0
540 IF D>.45*P THEN 950
550 T=T+D:B=FN R(10)+INT(A/100):P=P-D+B
560 Q=RND(1)<.15:IF Q THEN P=INT(P/2)
570 Z=Z+1:GOTO 200
800 PRINT "HAMMURABI, THINK AGAIN. YOU HAVE ONLY";G;"BUSHELS.":RETURN
810 PRINT "HAMMURABI, THINK AGAIN. YOU OWN ONLY";A;"ACRES.":RETURN
900 REM THE END OF TEN YEARS
910 PRINT:PRINT "IN TEN YEARS,";T;"PEOPLE STARVED."
920 PRINT "YOU LEAVE";INT(A/P);"ACRES FOR EACH PERSON."
930 IF A/P>=10 AND T<50 THEN PRINT "A GLORIOUS REIGN! THE PEOPLE LOVE YOU.":END
940 PRINT "THE PEOPLE WILL NOT MISS YOU.":END
950 PRINT:PRINT "YOU STARVED";D;"PEOPLE IN ONE YEAR!"
960 PRINT "YOU HAVE BEEN THROWN OUT OF OFFICE.":END
```
