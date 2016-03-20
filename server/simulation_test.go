package main

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"testing"

	sdz "github.com/mzimmerman/sdzpinochle"
)

const (
	winningScore       int  = 120
	giveUpScore        int  = -500
	numberOfTricks     int  = 12
	simulateWithServer bool = true
)

type Pairing struct {
	BiddingKey string
	PlayingKey string
}

func BenchmarkSimulation(b *testing.B) {
	wins := make(map[string]int)
	played := make(map[string]int)
	gamesToPlay := make(chan *Game)   // channel of created games to play
	finishedGames := make(chan *Game) // channel of finished games to compute scores
	pairings := make([]Pairing, 0)
	for bsKey := range biddingStrategies {
		for psKey := range playingStrategies {
			pairings = append(pairings, Pairing{
				BiddingKey: bsKey,
				PlayingKey: psKey,
			})
		}
	}
	var wg sync.WaitGroup
	wg.Add(runtime.NumCPU() + 1) // add one for finishedGames goroutine
	for x := 0; x < runtime.NumCPU(); x++ {
		go func() {
			defer wg.Done()
			for g := range gamesToPlay {
				g.NextHand()
			}
		}()
	}
	go func() {
		defer wg.Done()
		for game := range finishedGames {
			played[game.Players[0].Name()]++
			played[game.Players[1].Name()]++
			wins[game.Players[game.WinningPartnership].Name()]++
		}
	}()
	for x := 0; x < b.N; x++ {
		for _, pairing := range pairings {
			game := NewGame(4)
			for x := 0; x < 4; x++ {
				game.Players[x] = CreateAI(biddingStrategies[pairing.BiddingKey], playingStrategies[pairing.PlayingKey], fmt.Sprintf("%s-%s", pairing.BiddingKey, pairing.PlayingKey))
			}
			gamesToPlay <- game
		}
	}
	close(gamesToPlay)
	wg.Wait() // wait for all queued games to be finished and calculated
	best := ""
	for name, numPlayed := range played {
		if wins[name] > wins[best] {
			best = name
		}
		log.Printf("%s won %d out of %d", name, wins[name], numPlayed)
	}
	log.Printf("%s wins the most!", best)
}

func PlayNone(hand *sdz.Hand, winningCard sdz.Card, leadSuit sdz.Suit, trump sdz.Suit) sdz.Card {
	panic("This isn't a real playing strategy")
	return sdz.AD
}
