// sdzpinochle-client project main.go
package main

import (
	//	"encoding/json"
	"fmt"
	sdz "github.com/mzimmerman/sdzpinochle"
	//	"math/rand"
	"sort"

//	"time"
)

const (
	ace = iota
	ten
	king
	queen
	jack
	nine
)

const (
	spades = iota
	hearts
	clubs
	diamonds
)

type AI struct {
	hand      sdz.Hand
	c         chan sdz.Action
	playerid  int
	trump     int
	bidAmount int
}

func Log(m string, v ...interface{}) {
	fmt.Printf(m+"\n", v...)
}

func (ai AI) Close() {
	close(ai.c)
}

func (ai AI) powerBid(suit int) (count int) {
	suitMap := make([]int, 4)
	for _, card := range ai.hand {
		suitMap[card.Suit()]++
		if card.Suit() == suit {
			switch card.Card() {
			case ace:
				count += 3
			case ten:
				fallthrough
			case king:
				count += 2
			case queen:
				fallthrough
			case jack:
				fallthrough
			case nine:
				count += 1
			}
		} else if card.Card() == ace {
			count += 2
		}
	}
	for x := 0; x < 4; x++ {
		if x == suit {
			continue
		}
		if suitMap[x] == 0 {
			count++
		}
	}
	return
}

func (ai AI) calculateBid() (amount, trump int) {
	bids := make([]int, 4)
	for suit := 0; suit < 4; suit++ {
		bids[suit] = ai.hand.Meld(suit)
		bids[suit] = bids[suit] + ai.powerBid(suit)
		Log("Could bid %d in %d", bids[suit], suit)
		if bids[trump] < bids[suit] {
			trump = suit
		} /* else if bids[trump] == bids[suit] {
			rand.Seed(time.Now().UnixNano())
			if rand.Intn(2) == 0 { // returns one in the set of [0,2)
				trump = suit
			} // else - stay with trump as it was
		}*/
	}
	return bids[trump], trump
}

func (ai *AI) Go() {
	for {
		action, open := <-ai.c
		//		Log("Action received by player %d with hand %s - %+v", ai.playerid, ai.hand, action)
		if !open {
			return
		}
		switch action.Action {
		case sdz.Bid:
			if action.Playerid == ai.playerid {
				Log("------------------Player %d asked to bid", ai.playerid)
				ai.bidAmount, ai.trump = ai.calculateBid()
				ai.c <- sdz.Action{
					Action:   sdz.Bid,
					Amount:   ai.bidAmount,
					Playerid: ai.playerid,
				}
				Log("------------------Player %d bid %d for %d", ai.playerid, ai.bidAmount, ai.trump)
			} else {
				// received someone else's bid value'
			}
		case sdz.PlayCard:
			if action.Playerid == ai.playerid {
				action.Card = ai.hand.Play(0)
				Log("Player %d played card %s", ai.playerid, action.Card)
				ai.c <- action
			} else {
				// received someone else's play'
			}
		case sdz.Trump:
			if action.Playerid == ai.playerid && action.Amount == -1 {
				Log("Player %d being asked to name trump on hand %s and have %d meld", ai.playerid, ai.hand, ai.hand.Meld(ai.trump))
				if ai.bidAmount < 18 {
					ai.c <- sdz.Action{
						Action: sdz.Throwin,
					}
				} else {
					ai.c <- sdz.Action{
						Action: sdz.Trump,
						Amount: ai.trump, // set trump
					}
				}
			} else {
				Log("Player %d was told trump", ai.playerid)
				ai.trump = action.Amount
			}
		case sdz.Throwin:
			Log("Player %d saw that player %d threw in", ai.playerid, action.Playerid)
		case sdz.Deal: // should not happen as the server can set the Hand automagically

		}
	}
}

func (ai AI) Tell(action sdz.Action) {
	ai.c <- action
}
func (a AI) Listen() sdz.Action {
	return <-a.c
}
func (a AI) Hand() sdz.Hand {
	return a.hand
}

func (a *AI) SetHand(h sdz.Hand) {
	a.hand = h
}

type Human struct {
	hand sdz.Hand
	c    chan sdz.Action
}

func (human Human) Tell(action sdz.Action) {
	human.c <- action
	// TODO: do some network stuff here
}
func (human Human) Listen() sdz.Action {
	// TODO: do some network stuff here
	return <-human.c
}
func (human Human) Hand() sdz.Hand {
	return human.Hand()
}
func (human Human) SetHand(h sdz.Hand) {
	human.hand = h
	for x := 0; x < len(h); x++ {
		human.Tell(sdz.Action{
			Action: sdz.Deal,
			Card:   h[x],
		})
	}
}

func createAI(x int) (a *AI) {
	a = new(AI)
	a.playerid = x
	a.c = make(chan sdz.Action)
	return a
}

func main() {
	game := createGame()
	game.Players = make([]sdz.Player, 4)
	// connect players
	for x := 0; x < 4; x++ {
		game.Players[x] = createAI(x)
		go game.Players[x].Go()
	}
	for {
		// shuffle & deal
		game.Deck.Shuffle()
		hands := game.Deck.Deal()
		for x := 0; x < 4; x++ {
			game.Dealer = (game.Dealer + 1) % 4
			sort.Sort(hands[x])
			game.Players[game.Dealer].SetHand(hands[x])
			Log("Dealing player %d hand %s - Player now has %v", game.Dealer, hands[x], game.Players[game.Dealer].Hand())
		}
		// ask players to bid
		game.HighBid = 20
		game.HighPlayer = game.Dealer
		for x := 0; x < 4; x++ {
			game.Dealer = (game.Dealer + 1) % 4
			game.Players[game.Dealer].Tell(sdz.Action{
				Action:   sdz.Bid,
				Playerid: game.Dealer,
				Amount:   -1,
			})
			bid := game.Players[game.Dealer].Listen()
			bid.Playerid = game.Dealer
			game.Broadcast(bid, game.Dealer)
			if bid.Amount > game.HighBid {
				game.HighBid = bid.Amount
				game.HighPlayer = game.Dealer
			}
		}
		// ask trump
		game.Players[game.HighPlayer].Tell(sdz.Action{
			Action:   sdz.Trump,
			Amount:   -1,
			Playerid: game.HighPlayer,
		})
		response := game.Players[game.HighPlayer].Listen()
		if response.Action == sdz.Throwin {
			response.Playerid = game.HighPlayer
			game.Broadcast(response, response.Playerid)
		} else {
			// response.Action = sdz.Trump
			game.Trump = response.Amount
			game.Broadcast(sdz.Action{
				Action:   sdz.Trump,
				Amount:   game.Trump,
				Playerid: game.HighPlayer,
			}, game.HighPlayer)
		}
		next := game.HighPlayer
		for trick := 0; trick < 12; trick++ {
			for x := 0; x < 4; x++ {
				// play the hand
				// handle possible throwin
				action := sdz.Action{
					Action:   sdz.PlayCard,
					Playerid: next,
				}
				game.Players[next].Tell(action)
				action = game.Players[next].Listen()
				game.Broadcast(action, next)
				next = (next + 1) % 4
			}
			// next = winnerOfTheTrick
			// add counters to winner's points'
		}
		for x := 0; x < 4; x++ {
			game.Dealer = (game.Dealer + 1) % 4
			game.Players[game.Dealer].Close()
		}
		return
		// check the score for a winner
		game.Dealer = (game.Dealer + 1) % 4
	}
}

func createGame() (game *sdz.Game) {
	game = &sdz.Game{}
	game.Deck = sdz.CreateDeck()
	game.Dealer = 0
	return
}
