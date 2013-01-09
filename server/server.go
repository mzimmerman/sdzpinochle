// sdzpinochle-client project main.go
package main

import (
	//	"encoding/json"
	//	"fmt"
	sdz "github.com/mzimmerman/sdzpinochle"
	"sort"
)

type AI struct {
	hand *sdz.Hand
	c    chan *sdz.Action
}

func (ai AI) Tell(action *sdz.Action) {
	ai.c <- action
}
func (a AI) Listen() *sdz.Action {
	return &sdz.Action{}
}
func (a AI) Hand() *sdz.Hand {
	return &sdz.Hand{}
}
func (ai AI) SetHand(h *sdz.Hand) {
	ai.hand = h
}

type Human struct {
	hand *sdz.Hand
	c    chan *sdz.Action
}

func createAI() (p sdz.Player) {
	a := AI{}
	c := make(chan *sdz.Action)
	a.c = c
	return a
}

func main() {
	game := createGame()
	game.Players = make([]sdz.Player, 4)
	// connect players
	for x := 0; x < 4; x++ {
		game.Players[x] = createAI()
	}
	for {
		// shuffle & deal
		game.Deck.Shuffle()
		hands := game.Deck.Deal()
		for x := 0; x < 4; x++ {
			game.Dealer = (game.Dealer + 1) % 4
			sort.Sort(hands[x])
			game.Players[game.Dealer].SetHand(&hands[x])
		}
		// ask players to bid
		// receive bids
		for {
			//			// play the hand
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
