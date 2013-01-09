// sdzpinochle-client project main.go
package main

import (
	//	"encoding/json"
	"fmt"
	sdz "github.com/mzimmerman/sdzpinochle"
	"sort"
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
	hand     sdz.Hand
	c        chan sdz.Action
	playerid int
	trump    int
}

func Log(m string, v ...interface{}) {
	fmt.Printf(m+"\n", v...)
}

func (ai AI) Close() {
	close(ai.c)
}

func (ai *AI) Go() {
	for {
		action, open := <-ai.c
		Log("Action received by player %d with hand %s - %+v", ai.playerid, ai.hand, action)
		if !open {
			return
		}
		switch action.Action {
		case sdz.Bid:
			if action.Amount == -1 {
				Log("Player %d asked to bid", ai.playerid)
				ai.c <- sdz.Action{
					Action:   sdz.Bid,
					Amount:   25,
					Playerid: ai.playerid,
				}
				Log("Player %d bid", ai.playerid)
			} else {
				// received someone else's bid value'
			}
		case sdz.PlayCard:
		case sdz.Trump:
			if action.Playerid == ai.playerid && action.Amount == -1 {
				Log("Player %d being asked to name trump on hand %s and have %d meld", ai.playerid, ai.hand, ai.hand.Meld(spades))
				if ai.hand.Meld(spades) < 5 {
					ai.c <- sdz.Action{
						Action: sdz.Throwin,
					}
				} else {
					ai.c <- sdz.Action{
						Action: sdz.Trump,
						Amount: spades, // set trump to spades
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
		Log("----- Player 0 has hand %v", game.Players[0].Hand())
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
			game.TellAll(bid)
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
			game.TellAll(response)
		} else {
			// response.Action = sdz.Trump
			game.Trump = response.Amount
		}
		game.TellAll(sdz.Action{
			Action:   sdz.Trump,
			Amount:   game.Trump,
			Playerid: game.HighBid,
		})
		for x := 0; x < 4; x++ {
			game.Dealer = (game.Dealer + 1) % 4
			game.Players[game.Dealer].Close()
		}
		return
		for {
			// play the hand
			// handle possible throwin
		}
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
