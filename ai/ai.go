package ai

import (
	"fmt"
	sdz "github.com/mzimmerman/sdzpinochle"
	"math/rand"
)

type BiddingStrategy func(h *sdz.Hand, bids []int) (int, sdz.Suit)
type PlayingStrategy func(h *sdz.Hand, c sdz.Card, l sdz.Suit, t sdz.Suit) sdz.Card

type Player struct {
	Name string
	Bid  BiddingStrategy
	Play PlayingStrategy
}

type Partnership struct {
	MatchScore    int
	DealScore     int
	HasTakenTrick bool
	Bid           int
}

func (p *Partnership) GetDealScore() int {
	if p.HasTakenTrick && p.DealScore >= p.Bid {
		return p.DealScore
	} else {
		return -1 * p.Bid
	}
}

func (p *Partnership) SetDealScore() {
	p.MatchScore += p.GetDealScore()
}

type Match struct {
	Partnerships *[2]Partnership
	// Players 0 and 2 belong to partnership 0 and Players 1 and 3 belong to partnership 1
	Players [4]Player
	// Players have the hand in the same position: Player 0 has hand 0
	Hands [4]sdz.Hand
}

func (m *Match) SetMeld(trump sdz.Suit) {
	p0, _ := m.Hands[0].Meld(trump)
	p1, _ := m.Hands[1].Meld(trump)
	p2, _ := m.Hands[2].Meld(trump)
	p3, _ := m.Hands[3].Meld(trump)
	m.Partnerships[0].DealScore = int(p0 + p2)
	m.Partnerships[1].DealScore = int(p1 + p3)
}

func (m *Match) String() string {
	return fmt.Sprintf("Score: %v - %v", m.Partnerships[0].MatchScore, m.Partnerships[1].MatchScore)
}

var BiddingStrategys = []BiddingStrategy{NeverBid}
var PlayingStrategys = []PlayingStrategy{PlayRandom}

func NeverBid(hand *sdz.Hand, bids []int) (int, sdz.Suit) {
	return 0, sdz.Hearts
}

func ChooseSuitWithMostMeld(hand *sdz.Hand, bids []int) (int, sdz.Suit) {
	var highestMeld uint8 = 0
	var trump sdz.Suit
	for _, suit := range sdz.Suits {
		meld, _ := hand.Meld(suit)
		if meld > highestMeld {
			highestMeld = meld
			trump = suit
		}
	}
	return int(highestMeld), trump
}

func MostMeldPlusX(x int) BiddingStrategy {
	return func(h *sdz.Hand, b []int) (int, sdz.Suit) {
		meld, suit := ChooseSuitWithMostMeld(h, b)
		return meld + x, suit
	}
}

func PlayHighest(hand *sdz.Hand, winningCard sdz.Card, leadSuit sdz.Suit, trump sdz.Suit) sdz.Card {
	for _, face := range [6]sdz.Face{sdz.Ace, sdz.Ten, sdz.King, sdz.Queen, sdz.Jack, sdz.Nine} {
		for _, card := range *hand {
			if card.Face() == face && sdz.ValidPlay(card, winningCard, leadSuit, hand, trump) {
				return card
			}
		}
	}
	return (*hand)[0]
}

func PlayLowest(hand *sdz.Hand, winningCard sdz.Card, leadSuit sdz.Suit, trump sdz.Suit) sdz.Card {
	for _, face := range [6]sdz.Face{sdz.Nine, sdz.Jack, sdz.Queen, sdz.King, sdz.Ten, sdz.Ace} {
		for _, card := range *hand {
			if card.Face() == face && sdz.ValidPlay(card, winningCard, leadSuit, hand, trump) {
				return card
			}
		}
	}
	return (*hand)[0]
}

func PlayRandom(hand *sdz.Hand, winningCard sdz.Card, leadSuit sdz.Suit, trump sdz.Suit) sdz.Card {
	for _, v := range rand.Perm(len(*hand)) {
		if sdz.ValidPlay((*hand)[v], winningCard, leadSuit, hand, trump) {
			return (*hand)[v]
		}
	}
	return (*hand)[0]
}
