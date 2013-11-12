SDZPinochle
==============
A single deck pinochle engine

It works by presenting a web interface with available "tables" to sit at.  Once a table is chosen, the player has the ability to "Start" the game and the server will substitute the best known pinochle AI to play against/with.

AI
==============
A Pinochle AI is actually quite difficult due to the unknown and known amount of card(s), the alogrithm for this is roughly
**O(4*(n!)^c)** where:
- n = (number of cards left to play in the hand
- c = (number of cards with an unknown state (we don't know which player has them))

AI Playing
--------------
The AI plays the lowest card that puts it in a given position.

```
AS -> KS -> 9S -> ?
My partner (KS) is losing
I have a spade
We are going to lose the trick
I will play my lowest spade
```

```
KS -> AS -> 9S -> ?
My partner is winning
I have the KS, TS, and 9S
The AI will evaluate playing it's KS and 9S, but not the TS as that can't possibly do better than the TS
```

```
KS -> ?
Spades is trump
I'm out of spades
My partner has the AS
The other AS is unknown
Here (if possible) the lowest counter and non-counter from each other suit would be tried since each possibility could lead to the "best" position.
```

Protocol
==============
The protocol is JSON where the client sends POSTs messages to /receive and fetches messages through the Javascript AppEngine Channel API.
An alternative to the Javascript client for the AppEngine Channel API can be found at: http://schibum.blogspot.com/2011/06/using-google-appengine-channel-api-with.html
The following responses are used for the Type field:
--------------
* Message - A way to send a string of output to the client
	* Message = A string representation of what should can be shown to the client
* Hello - A way to say Hello to the client
	* Message - only used to respond to the server of what action should take (join, create, or quit)
* Game - The response to the server from a Hello message
	* Option - An integer that represents the playing mode of the human
* Deal - contains the Hand and the playerid that the client should play as responding to requests when prompted by it's playerid'
	* Playerid - An integer (0 through 3) which represents the playerid value of your player
	* Hand - a sorted Array of Cards (e.g, AS, KD, 9H, TC)
* Bid - The action that states who bid what
	* Bid - The integer amount of the bid
	* Playerid - States who is making this bid
* Trump - The action that states what trump is
	* Trump - the suit of trump (i.e, H, S, D, C)
	* Playerid - the one who named trump (and subsequently won the bid)
* Meld - Shows the hand and amount of meld for each player
	* Hand - (see Deal) but only consists of those cards that are counting toward points
	* Playerid - The player whom the meld belongs to
	* Amount  - The amount of the meld
* Play - A request from the server, or a response from the client of the card played
	* Playerid - the one who made this play (or is being requested to play)
	* PlayedCard - the card that is being played by this action (used for sharing other plays and for issuing a play by the client)
	* Lead - The suit (i.e., S, D, C, H) of the lead card
	* Trump - The suit (i.e., S, D, C, H) that is trump
	* WinningCard - The current card that is winning the hand (e.g., AD, AS, 9H, 10C)	
* Score - Comes at the end of the hand to announce the score
	* Win - boolean - Included if GameOver is true and your client has won the game
	* Score - integer array - scores, playerid % 2 is the client's team
	* GameOver - boolean - Set to true if the game is over

Playerid
-------------
Your Playerid is assigned during the Deal type.  For all other types, if the playerid received matches your playerid assigned during Deal, the server is awaiting a response from you of the same action.
Since your playerid was assigned through the Deal message as 0, the client needs to respond when prompted. Example:
```
--> {"Playerid":0,"Type":"Bid", "Bid":0}
<-- {"Playerid":0,"Type":"Bid", "Bid":0}
```

Network protocol Example
--------------
TODO: Update the start_of_game protocol to reference the new Table system
```
--> {"Message":"Do you want to join a game, create a new game, or quit? (join, create, quit)","Type":"Message"}
--> {"Type":"Hello"}
<-- {"Message":"create","Type":"Hello"}
--> {"Message":"Option 1 - Play against three AI players and start immediately","Type":"Message"}
--> {"Message":"Option 2 - Play with a human partner against two AI players","Type":"Message"}
--> {"Message":"Option 3 - Play with a human partner against one AI players and 1 Human","Type":"Message"}
--> {"Message":"Option 4 - Play with a human partner against two humans","Type":"Message"}
--> {"Message":"Option 5 - Play against a human with AI partners","Type":"Message"}
--> {"Message":"Option 6 - Go back","Type":"Message"}
--> {"Type":"Game"}
<-- {"Option":1,"Type":"Game"}
```

```
--> {"Hand":["AD","KD","KD","JD","9D","JC","TH","KH","TS","TS","JS","9S"],"Playerid":0,"Type":"Deal"}
--> {"Playerid":1,"Type":"Bid"}
--> {"Bid":25,"Playerid":2,"Type":"Bid"}
--> {"Playerid":3,"Type":"Bid"}
--> {"Playerid":0,"Type":"Bid"}
<-- {"Playerid":0,"Type":"Bid"}
--> {"Playerid":2,"Trump":"H","Type":"Trump"}
--> {"Hand":[],"Playerid":0,"Type":"Meld"}
--> {"Amount":1,"Hand":["9H"],"Playerid":1,"Type":"Meld"}
--> {"Amount":11,"Hand":["JD","JC","JH","9H","KS","QS","JS"],"Playerid":2,"Type":"Meld"}
--> {"Amount":4,"Hand":["KC","KC","QC","QC"],"Playerid":3,"Type":"Meld"}
--> {"PlayedCard":"AD","Playerid":2,"Type":"Play"}
--> {"PlayedCard":"TD","Playerid":3,"Type":"Play"}
--> {"Lead":"D","Playerid":0,"Trump":"H","Type":"Play","WinningCard":"AD"}
<-- {"PlayedCard":"KD","Playerid":0,"Type":"Play"}
--> {"PlayedCard":"QD","Playerid":1,"Type":"Play"}
--> {"Message":"Player 2 wins trick #1 with AD for 3 points","Type":"Message"}
```

*snip*

```
<-- {"PlayedCard":"KD","Playerid":0,"Type":"Play"}
--> {"PlayedCard":"TH","Playerid":1,"Type":"Play"}
--> {"PlayedCard":"KH","Playerid":2,"Type":"Play"}
--> {"PlayedCard":"JH","Playerid":3,"Type":"Play"}
--> {"Message":"Player 1 wins trick #6 with TH for 3 points","Type":"Message"}
--> {"PlayedCard":"TC","Playerid":1,"Type":"Play"}
--> {"PlayedCard":"JH","Playerid":2,"Type":"Play"}
--> {"PlayedCard":"KC","Playerid":3,"Type":"Play"}
--> {"Lead":"C","Playerid":0,"Trump":"H","Type":"Play","WinningCard":"JH"}
<-- {"PlayedCard":"JD","Playerid":0,"Type":"Play"}
```

**The server enforces only legal plays - Above, the client didn't follow suit, so it prompts it again to play a card**

```
--> {"Lead":"C","Playerid":0,"Trump":"H","Type":"Play","WinningCard":"JH"}
<-- {"PlayedCard":"TH","Playerid":0,"Type":"Play"}
--> {"Message":"Player 0 wins trick #7 with TH for 3 points","Type":"Message"}
```

*snip*

```
--> {"PlayedCard":"9H","Playerid":1,"Type":"Play"}
--> {"PlayedCard":"JS","Playerid":2,"Type":"Play"}
--> {"PlayedCard":"9S","Playerid":3,"Type":"Play"}
--> {"Lead":"H","Playerid":0,"Trump":"H","Type":"Play","WinningCard":"9H"}
<-- {"PlayedCard":"TS","Playerid":0,"Type":"Play"}
--> {"Message":"Player 1 wins trick #12 with 9H for 2 points","Type":"Message"}
--> {"Message":"Scores are now Team0 = -25 to Team1 = 19, played 1 hands","Type":"Message"}
--> {"Score":[-25,19],"Type":"Score"}
```

**Next hand is being dealt**

```
--> {"Hand":["9D","AC","TC","KC","QC","QC","JC","KH","JH","9H","9H","JS"],"Playerid":0,"Type":"Deal"}
--> {"Playerid":2,"Type":"Bid"}
--> {"Bid":29,"Playerid":3,"Type":"Bid"}
--> {"Playerid":0,"Type":"Bid"}
<-- {"Playerid":0,"Type":"Bid"}
--> {"Playerid":1,"Type":"Bid"}
--> {"Playerid":3,"Trump":"C","Type":"Trump"}
--> {"Amount":15,"Hand":["AC","TC","KC","QC","JC"],"Playerid":0,"Type":"Meld"}
--> {"Amount":1,"Hand":["9C"],"Playerid":1,"Type":"Meld"}
--> {"Hand":[],"Playerid":2,"Type":"Meld"}
--> {"Amount":11,"Hand":["AD","AC","9C","AH","AS"],"Playerid":3,"Type":"Meld"}
--> {"PlayedCard":"AD","Playerid":3,"Type":"Play"}
--> {"Lead":"D","Playerid":0,"Trump":"C","Type":"Play","WinningCard":"AD"}
<-- {"PlayedCard":"9D","Playerid":0,"Type":"Play"}
--> {"PlayedCard":"QD","Playerid":1,"Type":"Play"}
--> {"PlayedCard":"AD","Playerid":2,"Type":"Play"}
--> {"Message":"Player 3 wins trick #1 with AD for 2 points","Type":"Message"}
```

*snip*

```
<-- {"PlayedCard":"QC","Playerid":0,"Type":"Play"}
--> {"PlayedCard":"9S","Playerid":1,"Type":"Play"}
--> {"PlayedCard":"9S","Playerid":2,"Type":"Play"}
--> {"PlayedCard":"QS","Playerid":3,"Type":"Play"}
--> {"Message":"Player 0 wins trick #12 with QC for 1 points","Type":"Message"}
--> {"Message":"Scores are now Team0 = 3 to Team1 = -10, played 2 hands","Type":"Message"}
--> {"Score":[3,-10],"Type":"Score"}
--> {"Hand":["TD","9D","AC","9C","9C","AH","KH","QH","9H","QS","9S","9S"],"Playerid":0,"Type":"Deal"}
--> {"Bid":22,"Playerid":3,"Type":"Bid"}
--> {"Playerid":0,"Type":"Bid"}
<-- {"Playerid":0,"Type":"Bid"}
--> {"Bid":45,"Playerid":1,"Type":"Bid"}
--> {"Playerid":2,"Type":"Bid"}
--> {"Playerid":1,"Trump":"C","Type":"Trump"}
--> {"Amount":4,"Hand":["9C","9C","KH","QH"],"Playerid":0,"Type":"Meld"}
--> {"Amount":25,"Hand":["AD","AC","TC","KC","QC","JC","AH","AS"],"Playerid":1,"Type":"Meld"}
--> {"Amount":6,"Hand":["JD","KS","QS"],"Playerid":2,"Type":"Meld"}
--> {"Amount":4,"Hand":["KD","KD","QD","QD"],"Playerid":3,"Type":"Meld"}
```

*snip*

```
--> {"PlayedCard":"9D","Playerid":3,"Type":"Play"}
--> {"Lead":"D","Playerid":0,"Trump":"C","Type":"Play","WinningCard":"9D"}
<-- {"PlayedCard":"KH","Playerid":0,"Type":"Play"}
--> {"PlayedCard":"JS","Playerid":1,"Type":"Play"}
--> {"PlayedCard":"QS","Playerid":2,"Type":"Play"}
```

TODO: Further update the end_of_game network protocol

```
--> {"Message":"Player 3 wins trick #12 with 9D for 2 points","Type":"Message"}
--> {"Message":"Scores are now Team0 = 24 to Team1 = -55, played 3 hands","Type":"Message"}
--> {"Message":"Team0 wins with a score of 24!","Type":"Message"}
--> {"GameOver":true,"Score":[24,-55],"Type":"Score","Win":true}
--> {"Message":"Do you want to join a game, create a new game, or quit? (join, create, quit)","Type":"Message"}
--> {"Type":"Hello"}
<-- {"Message":"quit","Type":"Hello"}
--> {"Message":"Ok, bye bye!","Type":"Message"}
```
