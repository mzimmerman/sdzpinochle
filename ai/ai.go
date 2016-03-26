package ai

import (
	"fmt"
	sdz "github.com/mzimmerman/sdzpinochle"
	server "github.com/mzimmerman/sdzpinochle/server"
	"math/rand"
)

type PlayingStrategy func(h *sdz.Hand, c sdz.Card, l sdz.Suit, t sdz.Suit) sdz.Card

type Player struct {
	Name   string
	Bid    server.BiddingStrategy
	Play   PlayingStrategy
	HTPlay server.HTPlayingStrategy
}

type Partnership struct {
	MatchScore    int
	DealScore     int
	HasTakenTrick bool
	Bid           uint8
}

func (p *Partnership) GetDealScore() int {
	if p.HasTakenTrick && p.DealScore >= int(p.Bid) {
		return p.DealScore
	} else {
		return -1 * int(p.Bid)
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

func NeverBid(hand *sdz.Hand, bids []uint8) (uint8, sdz.Suit, sdz.Hand) {
	return 20, sdz.Hearts, *hand
}

func ChooseSuitWithMostMeld(hand *sdz.Hand, bids []uint8) (uint8, sdz.Suit, sdz.Hand) {
	var highestMeld uint8 = 0
	var trump sdz.Suit
	var showMeld sdz.Hand
	for _, suit := range sdz.Suits {
		meld, show := hand.Meld(suit)
		if meld > highestMeld {
			highestMeld = meld
			trump = suit
			showMeld = show
		}
	}
	if highestMeld < 20 {
		highestMeld = 20
	}
	return highestMeld, trump, showMeld
}

func MostMeldPlusX(x uint8) server.BiddingStrategy {
	return func(h *sdz.Hand, b []uint8) (uint8, sdz.Suit, sdz.Hand) {
		meld, suit, show := ChooseSuitWithMostMeld(h, b)
		return meld + x, suit, show
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
