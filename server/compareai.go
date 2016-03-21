package main

import (
	"fmt"
	"log"
	"runtime"
	"sort"
	"sync"
	"time"
)

type AIStrategy struct {
	BiddingKey string
	PlayingKey string
}

func (p AIStrategy) Name() string {
	return fmt.Sprintf("%s-%s", p.BiddingKey, p.PlayingKey)
}

func compareAI(duration time.Duration) {
	end := time.Now().Add(duration)
	log.Printf("Starting AI comparison, will end no earlier than %s", end.Format(time.ANSIC))
	wins := make(map[string]int)
	played := make(map[string]int)
	gamesToPlay := make(chan *Game)   // channel of created games to play
	finishedGames := make(chan *Game) // channel of finished games to compute scores
	aistrategies := make([]AIStrategy, 0)
	for bsKey := range biddingStrategies {
		for psKey := range playingStrategies {
			aistrategies = append(aistrategies, AIStrategy{
				BiddingKey: bsKey,
				PlayingKey: psKey,
			})
			log.Printf("Comparing %s", aistrategies[len(aistrategies)-1].Name())
		}
	}
	var wg sync.WaitGroup
	wg.Add(runtime.NumCPU())
	for x := 0; x < runtime.NumCPU(); x++ {
		go func() {
			defer wg.Done()
			for g := range gamesToPlay {
				g.NextHand()
				finishedGames <- g
			}
		}()
	}
	var gamesCalculatedWG sync.WaitGroup
	gamesCalculatedWG.Add(1)
	go func() {
		defer gamesCalculatedWG.Done()
		interval := time.Tick(time.Minute)
		for {
			select {
			case <-interval:
				printResults(wins, played)
			case game, done := <-finishedGames:
				if !done {
					printResults(wins, played)
					return
				}
				played[game.Players[0].Name()]++
				played[game.Players[1].Name()]++
				wins[game.Players[game.WinningPartnership].Name()]++
			}
		}
	}()
CreateGames:
	for {
		for y, incumbent := range aistrategies {
			for z, challenger := range aistrategies {
				if y == z {
					continue // don't play ourselves!
				}
				if time.Since(end) > 0 {
					break CreateGames
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
}

func printResults(wins, played map[string]int) {
	results := make(SortPlayersByPercent, len(played))
	x := 0
	for name, numPlayed := range played {
		results[x].Name = name
		results[x].Percent = float32(wins[name]) / float32(numPlayed) * 100
		results[x].Wins = wins[name]
		results[x].Played = numPlayed
		x++
	}
	sort.Sort(SortPlayersByPercent(results))
	log.Printf("---------------------------")
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
