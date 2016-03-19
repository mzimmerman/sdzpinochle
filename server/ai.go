package main

import (
	"fmt"
	"math/rand"

	sdz "github.com/mzimmerman/sdzpinochle"
)

type PlayingStrategy func(h *sdz.Hand, c sdz.Card, l sdz.Suit, t sdz.Suit) sdz.Card
type BiddingStrategy func(h *sdz.Hand, bids []uint8) (uint8, sdz.Suit, sdz.Hand)
type HTPlayingStrategy func(ht *HandTracker, t sdz.Suit) sdz.Card

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

func MostMeldPlusX(x uint8) BiddingStrategy {
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

func init() {
	hts := make(HTStack, 0, 1000)
	htstack = &hts
}

var htstack *HTStack

func (ht *HandTracker) reset(owner uint8) {
	ht.Owner = owner
	for x := 0; x < len(ht.PlayedCards); x++ {
		for y := uint8(0); y < 4; y++ {
			if y == ht.Owner {
				ht.PlayedCards[x] = None
				ht.Cards[y][x] = None
			} else {
				ht.Cards[y][x] = Unknown
			}
		}
	}
	ht.PlayCount = 0
	ht.Trick = new(Trick)
	ht.Trick.reset()
}

type HTStack []*HandTracker

func (hts *HTStack) Push(ht *HandTracker) {
	*hts = append(*hts, ht)
}

func (hts *HTStack) Pop() (ht *HandTracker, err error) {
	//x, a = a[len(a)-1], a[:len(a)-1]
	l := len(*hts) - 1
	if l < 0 {
		//memstats := new(runtime.MemStats)
		//runtime.ReadMemStats(memstats)
		//Log(4, "MemStats = %#v", memstats)
		ht = new(HandTracker)
		ht.Trick = new(Trick)
		return
	}
	ht, *hts = (*hts)[l], (*hts)[:l]
	return
}

type HandTracker struct {
	Cards [4]CardMap
	// 0 = know nothing = Unknown
	// 3 = does not have any of this card = None
	// 1 = has this card
	// 2 = has two of these cards
	PlayedCards CardMap
	Owner       uint8 // the playerid of the "owning" player
	Trick       *Trick
	PlayCount   uint8
}

func (ht *HandTracker) sum(cardIndex sdz.Card) (sum uint8) {
	sum = ht.PlayedCards[cardIndex]
	for x := 0; x < len(ht.Cards); x++ {
		if ht.Cards[x][cardIndex] != Unknown {
			sum += ht.Cards[x][cardIndex]
		}
	}
	if sum > 2 {
		sdz.Log(ht.Owner, "Summing card %s, sum = %d", cardIndex, sum)
		ht.Debug()
		panic("sumthing is wrong, get it?!?!")
	}
	return
}

func (oldht *HandTracker) Copy() (newht *HandTracker, err error) {
	newht, err = getHT(oldht.Owner)
	for x := uint8(0); x < uint8(len(oldht.Cards)); x++ {
		newht.Cards[x] = oldht.Cards[x]
	}
	newht.PlayedCards = oldht.PlayedCards
	newht.PlayCount = oldht.PlayCount
	*newht.Trick = *oldht.Trick
	return
}

func (ht *HandTracker) Debug() {
	sdz.Log(ht.Owner, "ht.PlayedCards = %v", ht.PlayedCards)
	for x := 0; x < 4; x++ {
		sdz.Log(ht.Owner, "Player%d - %s", x, ht.Cards[x])
	}
	sdz.Log(ht.Owner, "PlayCount = %d, Next=%d", ht.PlayCount, ht.Trick.Next)
	panic("don't call debug")
}

func (ht *HandTracker) PlayCard(card sdz.Card, trump sdz.Suit) {
	//ht.Debug()
	playerid := ht.Trick.Next
	//Log(ht.Owner, "In ht.PlayCard for %d-%s on player %d", playerid, card, ht.Owner)
	val := ht.Cards[playerid][card]
	if val == None {
		fmt.Println("\n\n\nCard", card)
		sdz.Log(ht.Owner, "Player %d does not have card %s, panicking", playerid, card)
		ht.Debug()
		panic("panic")
	}
	ht.PlayedCards.inc(card)
	ht.Cards[playerid].dec(card)
	if ht.sum(card) > 2 {
		panic("Cannot play this card, something is wrong")
	}
	//if card == KS && playerid == 0 {
	//	Log(ht.Owner, "Decremented %s for player %d from %d to %d", card, playerid, val, ht.Cards[playerid][card])
	//}
	if val == 1 && ht.PlayedCards[card] == 1 && playerid != ht.Owner {
		// Other player could have only shown one in meld, but has two - now we don't know who has the last one
		ht.Cards[playerid][card] = Unknown
		//if oright == ht && card == JD && ht.Owner == 0 && playerid == 1 {
		//	Log(ht.Owner, "htcardset - deleted card %s for player %d, setting to Unknown", card, playerid)
		//	ht.Debug()
		//}
		//} else if oright == ht && card == JD && ht.Owner == 0 && playerid == 1 {
		//	Log(ht.Owner, "Not setting %s to unknown, val=%d, played=%d, playerid=%d", card, val, ht.PlayedCards[card], playerid)
		//	ht.Debug()
	}
	//if oright == ht && playerid == 1 && card == JD {
	//	Log(ht.Owner, "Before Calculate")
	//	ht.Debug()
	//}
	ht.calculateCard(card)
	//if oright == ht && playerid == 1 && card == JD {
	//	Log(ht.Owner, "After Calculate")
	//	ht.Debug()
	//}
	ht.Trick.PlayCard(card, trump)
	switch {
	case ht.Trick.LeadSuit() == sdz.NASuit || trump == sdz.NASuit:
		// do nothing, start of the trick, everything is legal
	case card.Suit() != ht.Trick.LeadSuit() && card.Suit() != trump: // couldn't follow suit, couldn't lay trump
		ht.noSuit(playerid, trump)
		//if oright == ht && playerid == 1 {
		//	Log(ht.Owner, "Setting all %s to None for playerid=%d", trump, playerid)
		//}
		fallthrough
	case card.Suit() != ht.Trick.LeadSuit(): // couldn't follow suit
		ht.noSuit(playerid, ht.Trick.LeadSuit())
		//if oright == ht && playerid == 1 {
		//	Log(ht.Owner, "Setting all %s to None for playerid=%d", trick.LeadSuit(), playerid)
		//}
	}
	if playerid != ht.Trick.WinningPlayer { // did not win
		for _, f := range sdz.Faces {
			tempCard := sdz.CreateCard(card.Suit(), f)
			if tempCard.Beats(ht.Trick.WinningCard(), trump) {
				//if oright == ht && playerid == 1 {
				//Log(ht.Owner, "Setting %s to None for playerid=%d because it could have won", tempCard, playerid)
				//}
				ht.Cards[playerid][tempCard] = None
				ht.calculateCard(tempCard)
			} else {
				break
			}
		}
	}
	ht.PlayCount++
	//Log(ht.Owner, "Player %d played card %s, PlayCount=%d", playerid, card, ht.PlayCount)
}

type CardMap [25]uint8

func (cm *CardMap) inc(x sdz.Card) {
	if cm[x] == Unknown {
		cm[x] = 1
	} else {
		cm[x]++
	}
}

func (cm *CardMap) dec(x sdz.Card) {
	//Log(4, "Before dec, %s = %d", x, cm[x])
	if cm[x] == 0 {
		//Log(4, "Attempting to decrement %s from %d", card(x), cm[x])
		panic("Cannot decrement past 0")
	}
	if cm[x] != Unknown {
		cm[x]--
	}
	//Log(4, "After dec, %s = %d", x, cm[x])
}

func (cm CardMap) String() string {
	output := "CardMap={"
	for x := sdz.AS; int8(x) <= sdz.AllCards; x++ {
		if cm[x] == Unknown {
			continue
		}
		if cm[x] == None {
			output += fmt.Sprintf("%s:%d ", x, 0)
		} else {
			output += fmt.Sprintf("%s:%d ", x, cm[x])
		}
	}
	return output + "}"
}
