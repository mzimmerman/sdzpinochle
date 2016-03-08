package main

import (
	"fmt"
	sdz "github.com/mzimmerman/sdzpinochle"
	ai "github.com/mzimmerman/sdzpinochle/ai"
	"sort"
)

const (
	winningScore   int  = 120
	numberOfTricks int  = 12
	debugLog       bool = false
)

type NamedBid struct {
	Name string
	Bid  ai.BiddingStrategy
}

type NamedPlay struct {
	Name string
	Play ai.PlayingStrategy
}

func main() {
	sdz.Init()
	bidding_strategies := []NamedBid{
		NamedBid{"NeverBid", ai.NeverBid},
		NamedBid{"MostMeld", ai.ChooseSuitWithMostMeld},
	}
	playing_strategies := []NamedPlay{
		NamedPlay{"PlayHighest", ai.PlayHighest},
		NamedPlay{"PlayLowest", ai.PlayLowest},
		NamedPlay{"PlayRandom", ai.PlayRandom},
	}
	players := make([]ai.Player, 0)
	for _, b := range bidding_strategies {
		for _, p := range playing_strategies {
			players = append(players, ai.Player{fmt.Sprintf("%v:%v", b.Name, p.Name), b.Bid, p.Play})
		}
	}
	player1 := players[0]
	for _, player2 := range players[1:] {
		win0 := 0
		win1 := 0
		fmt.Printf("%v vs %v\n", player1.Name, player2.Name)
		for x := 0; x < 10000; x++ {
			winningPartnership, match := playMatch(player1, player2)
			if winningPartnership == 0 {
				win0++
			} else {
				win1++
			}
			if debugLog {
				fmt.Printf("Partnership %v won! %v\n", winningPartnership, match)
			}
		}
		fmt.Printf("%v %v wins - %v %v wins\n", player1.Name, win0, player2.Name, win1)
		if win1 > win0 {
			// Winner stays
			player1 = player2
		}
	}
	fmt.Printf("%v is the champ!\n", player1.Name)

}

func playMatch(player1, player2 ai.Player) (int, *ai.Match) {
	var players [4]ai.Player
	for x := 0; x < 4; x++ {
		if x%2 == 0 {
			players[x] = player1
		} else {
			players[x] = player2
		}
	}
	match := ai.Match{
		Partnerships: new([2]ai.Partnership),
		Players:      players,
	}
	winner := -1
	for x := 1; winner == -1; x++ {
		bidder := playDeal(&match, x%4)
		pOne := match.Partnerships[0]
		pTwo := match.Partnerships[1]
		if pOne.MatchScore > winningScore && pTwo.MatchScore > winningScore {
			winner = bidder % 2
		} else if pOne.MatchScore > winningScore {
			winner = 0
		} else if pTwo.MatchScore > winningScore {
			winner = 1
		}
	}
	return winner, &match
}

func playDeal(match *ai.Match, dealer int) int {
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
	if debugLog {
		fmt.Println("Match before:", match)
	}
	match.Partnerships[0].SetDealScore()
	match.Partnerships[1].SetDealScore()
	if debugLog {
		fmt.Println("Match after: ", match)
	}

	return playerWithBid
}

func playHand(match *ai.Match, playerWithLead int, trump sdz.Suit) (int, sdz.Hand) {
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

func bid(match *ai.Match, dealer int) (int, int, sdz.Suit) {
	var highBid int = 20
	var highBidder int = dealer
	var trump sdz.Suit
	bids := make([]int, 0)
	_, trump = match.Players[dealer].Bid(&match.Hands[dealer], bids)
	for bidder := dealer + 1; bidder < dealer+5; bidder++ {
		index := bidder % 4
		bid, suit := match.Players[index].Bid(&match.Hands[index], bids)
		if highBid == 20 && dealer == index {
			trump = suit
		} else if bid > highBid {
			highBid, trump = bid, suit
		}
		bids = append(bids, bid)
	}
	return highBid, highBidder, trump
}
