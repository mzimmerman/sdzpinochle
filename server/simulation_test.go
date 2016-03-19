package main

import (
	"fmt"
	"runtime"
	"sort"
	"time"

	sdz "github.com/mzimmerman/sdzpinochle"
)

const (
	winningScore       int  = 120
	giveUpScore        int  = -500
	numberOfTricks     int  = 12
	simulateWithServer bool = false
)

type Opponents struct {
	player1 Player
	player2 Player
}

type NamedBid struct {
	Name string
	Bid  BiddingStrategy
}

type NamedPlay struct {
	Name   string
	Play   PlayingStrategy
	HTPlay HTPlayingStrategy
}

type Result struct {
	playerOneWins int
	playerTwoWins int
}

type matchPlayer func(player1, player2 Player) uint8

func main() {
	var numberOfMatchRunners int
	var matchesToSimulate int
	var matchPlayer matchPlayer

	startTime := time.Now()
	if simulateWithServer {
		numberOfMatchRunners = 1
		matchesToSimulate = 1
		matchPlayer = playServerMatch
	} else {
		numberOfMatchRunners = runtime.NumCPU()
		matchesToSimulate = 100 * numberOfMatchRunners
		matchPlayer = playMatch
	}

	opponents := make(chan Opponents, numberOfMatchRunners)
	results := make(chan Result, numberOfMatchRunners)
	createMatchRunners(
		numberOfMatchRunners, matchesToSimulate,
		matchPlayer, opponents, results,
	)
	matchesSimulated := 0
	players := createPlayers()

	player1 := players[0]
	for _, player2 := range players[1:] {
		matchesSimulated += matchesToSimulate
		fmt.Printf("%v vs %v\n", player1.Name, player2.Name)

		var win1, win2 int
		for x := 0; x < numberOfMatchRunners; x++ {
			opponents <- Opponents{player1, player2}
		}
		for x := 0; x < numberOfMatchRunners; x++ {
			result := <-results
			win1 += result.playerOneWins
			win2 += result.playerTwoWins
		}

		fmt.Printf("%v %v wins - %v %v wins\n", player1.Name, win1, player2.Name, win2)
		if win2 > win1 {
			// Winner stays
			player1 = player2
		}
	}
	elapsedSeconds := time.Since(startTime).Seconds()
	fmt.Printf("%v is the champ!\n", player1.Name)
	fmt.Printf(
		"%v matches simulated in %.f2 seconds\n",
		matchesSimulated,
		elapsedSeconds,
	)
	fmt.Printf("%.2f matches simulated per second.\n", float64(matchesSimulated)/elapsedSeconds)
}

func PlayNone(hand *sdz.Hand, winningCard sdz.Card, leadSuit sdz.Suit, trump sdz.Suit) sdz.Card {
	panic("This isn't a real playing strategy")
	return sdz.AD
}

func createPlayers() []Player {
	biddingStrategies := []NamedBid{
		NamedBid{"NeverBid", NeverBid},
		NamedBid{"MostMeld", ChooseSuitWithMostMeld},
		NamedBid{fmt.Sprintf("MostMeldPlus%v", 18), MostMeldPlusX(18)},
		NamedBid{"MattBid", MattBid},
	}
	for x := 16; x <= 18; x++ {
		/*
			biddingStrategies = append(
				biddingStrategies,
				NamedBid{fmt.Sprintf("MostMeldPlus%v", x), MostMeldPlusX(uint8(x))},
			)
		*/
	}
	playingStrategies := []NamedPlay{
		NamedPlay{"PlayHighest", PlayHighest, createHTPlayingStrategy(PlayHighest)},
		NamedPlay{"PlayRandom", PlayRandom, createHTPlayingStrategy(PlayRandom)},
		NamedPlay{"PlayLowest", PlayLowest, createHTPlayingStrategy(PlayLowest)},
	}
	if simulateWithServer {
		playingStrategies = append(
			playingStrategies,
			NamedPlay{"MattPlay", PlayNone, PlayHandWithCard},
		)
	}
	players := make([]Player, 0)
	for _, b := range biddingStrategies {
		for _, p := range playingStrategies {
			players = append(players, Player{
				fmt.Sprintf("%v:%v", b.Name, p.Name), b.Bid, p.Play, p.HTPlay,
			})
		}
	}
	return players
}

func createMatchRunners(
	numberOfMatchRunners, matchesToSimulate int,
	matchPlayer matchPlayer, opponents chan Opponents, results chan Result) {
	for x := 0; x < numberOfMatchRunners; x++ {
		go simulateMatches(
			matchesToSimulate/numberOfMatchRunners, matchPlayer,
			opponents, results)
	}
}

func simulateMatches(
	matchesToSimulate int, matchPlayer matchPlayer,
	opponents chan Opponents, results chan Result) (int, int) {
	for {
		opponents := <-opponents
		win1 := 0
		win2 := 0
		for x := 0; x < matchesToSimulate; x++ {
			winningPartnership := matchPlayer(opponents.player1, opponents.player2)
			if winningPartnership == 0 {
				win1++
			} else {
				win2++
			}
			if debugLog && x%10 == 0 {
				fmt.Printf("Current standings: %v - %v\n", win1, win2)
			}
		}
		results <- Result{win1, win2}
	}
}

func handFromHT(ht *HandTracker) *sdz.Hand {
	hand := make(sdz.Hand, 0)
	for x := 1; x < 25; x++ {
		if ht.Cards[ht.Trick.Next][x] > 0 {
			hand = append(hand, sdz.Card(x))
		}
	}
	return &hand
}

func createHTPlayingStrategy(ps PlayingStrategy) HTPlayingStrategy {
	return func(ht *HandTracker, t sdz.Suit) sdz.Card {
		return ps(handFromHT(ht), ht.Trick.WinningCard(), ht.Trick.LeadSuit(), t)
	}
}

func playServerMatch(player1, player2 Player) uint8 {
	game := NewGame(4)
	game.Dealer = 0

	deck := sdz.CreateDeck()
	deck.Shuffle()
	hands := deck.Deal()
	for x := uint8(0); x < 4; x++ {
		ai := CreateAI()
		if x%2 == 0 {
			BiddingStrategy = player1.Bid
			PlayingStrategy = player1.HTPlay
		} else {
			BiddingStrategy = player2.Bid
			PlayingStrategy = player2.HTPlay
		}
		game.Players[x] = ai
		game.Players[x].SetHand(game, hands[x], 0, x)
	}
	game.Meld = make([]uint8, len(game.Players)/2)
	game.CountMeld = make([]bool, len(game.Players)/2)
	game.Counters = make([]uint8, len(game.Players)/2)
	game.HighBid = 20
	game.HighPlayer = game.Dealer
	game.State = StateBid
	game.Next = game.Dealer
	//oright = game.Players[0].(*AI).HT
	//Log(oright.Owner, "Start of game hands")
	//oright.Debug()
	game.Inc() // so dealer's not the first to bid

	game.ProcessAction(game.Players[game.Next].Tell(game, sdz.CreateBid(0, game.Next)))
	return game.WinningPartnership
}

func playMatch(player1, player2 Player) uint8 {
	var players [4]Player
	for x := 0; x < 4; x++ {
		if x%2 == 0 {
			players[x] = player1
		} else {
			players[x] = player2
		}
	}
	match := Match{
		Partnerships: new([2]Partnership),
		Players:      players,
	}
	winner := -1
	for x := 1; winner == -1; x++ {
		bidder := playDeal(&match, x%4)
		pOne := match.Partnerships[0]
		pTwo := match.Partnerships[1]
		if pOne.MatchScore >= winningScore && pTwo.MatchScore >= winningScore {
			winner = bidder % 2
		} else if pOne.MatchScore >= winningScore || pTwo.MatchScore <= giveUpScore {
			winner = 0
		} else if pTwo.MatchScore >= winningScore || pTwo.MatchScore <= giveUpScore {
			winner = 1
		}
	}
	if debugLog {
		fmt.Printf("Partnership %v won! %v\n", winner, match)
	}
	return uint8(winner)
}

func playDeal(match *Match, dealer int) int {
	deck := sdz.CreateDeck()
	deck.Shuffle()
	hands := deck.Deal()

	match.Partnerships[0].DealScore = 0
	match.Partnerships[1].DealScore = 0

	for x, hand := range hands {
		match.Hands[x] = hand
		sort.Sort(hand)
		if debugLog {
			fmt.Println(hand)
			for _, suit := range sdz.Suits {
				meld, _ := hand.Meld(suit)
				fmt.Printf("%v: %v ", suit, meld)
			}
		}
	}
	bid, playerWithBid, trump := bid(match, dealer)
	playerWithLead := playerWithBid
	match.Partnerships[playerWithLead%2].Bid = bid
	match.Partnerships[(playerWithLead+1)%2].Bid = 0
	match.SetMeld(trump)
	if debugLog {
		fmt.Println("Trump:", trump)
		fmt.Println("bids", match.Partnerships[0].Bid, match.Partnerships[1].Bid)
		fmt.Println("deal scores", match.Partnerships[0].DealScore, match.Partnerships[1].DealScore)
	}
	for x := 0; x < numberOfTricks; x++ {
		playerWithLead, trick := playHand(match, playerWithLead, trump)
		if debugLog {
			fmt.Println("lead", playerWithLead, trick, "counters:", trick.Counters())
			fmt.Println("deal scores before:", match.Partnerships[0].DealScore, match.Partnerships[1].DealScore)
		}
		match.Partnerships[playerWithLead%2].DealScore += trick.Counters()
		match.Partnerships[playerWithLead%2].HasTakenTrick = true
		// Last trick
		if x+1 == numberOfTricks {
			match.Partnerships[playerWithLead%2].DealScore++
		}
		if debugLog {
			fmt.Println("deal scores after:", match.Partnerships[0].DealScore, match.Partnerships[1].DealScore)
		}
	}
	match.Partnerships[0].SetDealScore()
	match.Partnerships[1].SetDealScore()
	if debugLog {
		fmt.Println("Match after: ", match)
	}

	return playerWithBid
}

func playHand(match *Match, playerWithLead int, trump sdz.Suit) (int, sdz.Hand) {
	winningCard := sdz.NACard
	winningPlayer := playerWithLead
	leadSuit := sdz.NASuit
	trick := make([]sdz.Card, 0)

	for x := playerWithLead; x < playerWithLead+4; x++ {
		currPlayer := x % 4
		currHand := &match.Hands[currPlayer]
		if debugLog {
			fmt.Println(currHand)
		}
		card := match.Players[currPlayer].Play(currHand, winningCard, leadSuit, trump)
		currHand.Remove(card)
		trick = append(trick, card)
		if winningCard == sdz.NACard {
			winningCard = card
			leadSuit = card.Suit()
		} else if card.Beats(winningCard, trump) {
			winningCard = card
			winningPlayer = currPlayer
		}
	}
	if debugLog {
		fmt.Println("Trick: ", trick)
		fmt.Println("Match: ", match)
	}
	return winningPlayer, trick
}

func bid(match *Match, dealer int) (uint8, int, sdz.Suit) {
	var highBid uint8 = 20
	var highBidder int = dealer
	var trump sdz.Suit
	bids := make([]uint8, 0)
	_, trump, _ = match.Players[dealer].Bid(&match.Hands[dealer], bids)
	for bidder := dealer + 1; bidder < dealer+5; bidder++ {
		index := bidder % 4
		bid, suit, _ := match.Players[index].Bid(&match.Hands[index], bids)
		if highBid == 20 && dealer == index {
			trump = suit
		} else if bid > highBid {
			highBid, trump = bid, suit
		}
		bids = append(bids, bid)
	}
	return highBid, highBidder, trump
}
