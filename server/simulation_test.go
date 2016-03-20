package main

import (
	"fmt"
	"log"
	"runtime"
	"sort"
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

func (p Pairing) Name() string {
	return fmt.Sprintf("%s-%s", p.BiddingKey, p.PlayingKey)
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
	wg.Add(runtime.NumCPU())
	for x := 0; x < runtime.NumCPU(); x++ {
		go func() {
			defer wg.Done()
			for g := range gamesToPlay {
				g.NextHand()
				if g.Score[0] >= 120 || g.Score[1] >= 120 {
					finishedGames <- g
				} else {
					log.Printf("Did not finish game between %s and %s", g.Players[0].Name(), g.Players[1].Name())
				}

			}
		}()
	}
	var gamesCalculatedWG sync.WaitGroup
	gamesCalculatedWG.Add(1)
	go func() {
		defer gamesCalculatedWG.Done()
		for game := range finishedGames {
			played[game.Players[0].Name()]++
			played[game.Players[1].Name()]++
			wins[game.Players[game.WinningPartnership].Name()]++
		}
	}()
	for x := 0; x < b.N; x++ {
		for y, incumbent := range pairings {
			for z, challenger := range pairings {
				if y == z {
					continue // don't play ourselves!
				}
				game := NewGame(4)
				for x := 0; x < 4; x++ {
					if x%2 == 0 {
						game.Players[x] = CreateAI(biddingStrategies[incumbent.BiddingKey], playingStrategies[incumbent.PlayingKey], incumbent.Name())
					} else {
						game.Players[x] = CreateAI(biddingStrategies[challenger.BiddingKey], playingStrategies[challenger.PlayingKey], challenger.Name())
					}
				}
				gamesToPlay <- game
			}
		}
	}
	close(gamesToPlay)
	wg.Wait() // wait for all queued games to be finished and calculated
	close(finishedGames)
	gamesCalculatedWG.Wait()
	results := make(SortPlayersByPercent, len(pairings))
	x := 0
	for name, numPlayed := range played {
		results[x].Name = name
		results[x].Percent = float32(wins[name]) / float32(numPlayed) * 100
		results[x].Wins = wins[name]
		results[x].Played = numPlayed
		x++
	}
	sort.Sort(SortPlayersByPercent(results))
	for _, res := range results {
		log.Printf("%2.02f%% wins (%d/%d) %s", res.Percent, res.Wins, res.Played, res.Name)
	}
}

type GameResult struct {
	Name    string
	Percent float32
	Wins    int
	Played  int
}

type SortPlayersByPercent []GameResult

func (spbp SortPlayersByPercent) Len() int {
	return len(spbp)
}

func (spbp SortPlayersByPercent) Less(i, j int) bool {
	return spbp[i].Percent < spbp[j].Percent
}

func (spbp SortPlayersByPercent) Swap(i, j int) {
	spbp[i], spbp[j] = spbp[j], spbp[i]
}

func PlayNone(hand *sdz.Hand, winningCard sdz.Card, leadSuit sdz.Suit, trump sdz.Suit) sdz.Card {
	panic("This isn't a real playing strategy")
	return sdz.AD
}
