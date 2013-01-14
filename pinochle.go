// pinochle.go
package sdzpinochle

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	debug       = false
	ace         = "A"
	ten         = "T"
	king        = "K"
	queen       = "Q"
	jack        = "J"
	nine        = "9"
	spades      = "S"
	hearts      = "H"
	clubs       = "C"
	diamonds    = "D"
	acearound   = 10
	kingaround  = 8
	queenaround = 6
	jackaround  = 4
)

type Card string // two chars Face + String
type Suit string // one char
type Face string // one char

func Faces() [6]Face {
	return [6]Face{ace, ten, king, queen, jack, nine}
}

func Suits() [4]Suit {
	return [4]Suit{spades, hearts, clubs, diamonds}
}

type Deck [48]Card
type Hand []Card

func CreateCard(suit Suit, face Face) Card {
	return Card(string(face) + string(suit))
}

func (c Card) Suit() Suit {
	return Suit(c[1])
}

func (c Card) Face() Face {
	return Face(c[0])
}

func (c Card) isTrump(trump Suit) bool {
	return c.Suit() == trump
}

func (d *Deck) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func (d *Deck) Shuffle() {
	//	http://en.wikipedia.org/wiki/Fisher%E2%80%93Yates_shuffle#The_modern_algorithm
	rand.Seed(time.Now().UnixNano())
	for i := len(d) - 1; i >= 1; i-- {
		if j := rand.Intn(i); i != j {
			d.Swap(i, j)
		}
	}
}

func (h Hand) String() {
	cards := ""
	for x := 0; x < len(h); x++ {
		cards += string(h[x]) + " "
	}
}

func (h Hand) Len() int {
	return len(h)
}

func (h Hand) Less(i, j int) bool {
	if h[i].Suit() == h[j].Suit() {
		return h[i].Face().Less(h[j].Face())
	}
	return h[i].Suit().Less(h[j].Suit())
}

func (a Face) Less(b Face) bool {
	switch {
	case b == ace:
		return false
	case a == ace:
		return true
	case b == ten:
		return false
	case a == ten:
		return true
	case b == king:
		return false
	case a == king:
		return true
	case b == queen:
		return false
	case a == queen:
		return true
	case b == jack:
		return false
	case a == jack:
		return true
	case b == nine:
		return false
	}
	return true
}

func (a Suit) Less(b Suit) bool { // only for sorting the suits for display in the hand
	switch {
	case a == spades:
		return false
	case b == spades:
		return true
	case a == hearts:
		return false
	case b == hearts:
		return true
	case a == clubs:
		return false
	case b == clubs:
		return true
	case a == diamonds:
		return false
	}
	return true
}

func (h Hand) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (d Deck) Deal() (hands []Hand) {
	hands = make([]Hand, 4)
	for x := 0; x < 4; x++ {
		hands[x] = make([]Card, 12)
	}
	for y := 0; y < 12; y++ {
		for x := 0; x < 4; x++ {
			hands[x][y] = d[y*4+x]
		}
	}
	return
}

func CreateDeck() (deck Deck) {
	index := 0
	for _, face := range Faces() {
		for _, suit := range Suits() {
			for z := 0; z < 2; z++ {
				deck[index] = Card(string(face) + string(suit))
				index++
			}
		}
	}
	return
}

func CreateBid(bid, playerid int) *BidAction {
	bida := BidAction{bid: bid}
	bida.SetPlayer(playerid)
	return &bida
}

type BidAction struct {
	ActionImpl
	bid int
}

func (b BidAction) Value() interface{} {
	return b.bid
}

func CreatePlay(card Card, playerid int) *PlayAction {
	play := PlayAction{card: card}
	play.SetPlayer(playerid)
	return &play
}

type PlayAction struct {
	ActionImpl
	card Card
}

func (b PlayAction) Value() interface{} {
	return b.card
}

func CreateTrump(trump Suit, playerid int) *TrumpAction {
	x := TrumpAction{trump: trump}
	x.SetPlayer(playerid)
	return &x
}

type TrumpAction struct {
	ActionImpl
	trump Suit
}

func (x TrumpAction) Value() interface{} {
	return x.trump
}

func CreateThrowin(playerid int) *ThrowinAction {
	x := ThrowinAction{}
	x.SetPlayer(playerid)
	return &x
}

type ThrowinAction struct {
	ActionImpl
}

func (x ThrowinAction) Value() interface{} {
	return nil
}

func CreateDeal(hand Hand, playerid int) *DealAction {
	x := DealAction{hand: hand}
	x.SetPlayer(playerid)
	return &x
}

type DealAction struct {
	ActionImpl
	hand Hand
}

func (x DealAction) Value() interface{} {
	return x.hand
}

func (a *ActionImpl) SetPlayer(playerid int) {
	a.playerid = playerid
}

type ActionImpl struct {
	playerid int
}

func (action ActionImpl) Playerid() int {
	return action.playerid
}

type Action interface {
	Playerid() int
	Value() interface{}
}

type Player interface {
	Tell(Action)
	Listen() (Action, bool)
	Hand() Hand
	SetHand(Hand, int)
	Go()
	Close()
	Playerid() int
}

type PlayerImpl struct {
	Id int
}

func (p PlayerImpl) Playerid() int {
	return p.Id
}

func (p PlayerImpl) IsPartner(player int) bool {
	switch p.Playerid() {
	case player - 2:
		fallthrough
	case player + 2:
		return true
	}
	return false
}

type Game struct {
	Deck       Deck
	Players    []Player
	Dealer     int
	Score1     int
	Score2     int
	HighBid    int
	HighPlayer int
	Trump      Suit
}

func (g Game) Broadcast(a Action, p int) {
	for x, player := range g.Players {
		if p != x {
			player.Tell(a)
		}
	}
}

func (h *Hand) Play(x int) (card Card) {
	card = (*h)[x]
	*h = append((*h)[:x], (*h)[x+1:]...)
	return
}

func (h Hand) Count() (cards map[Card]int) {
	cards = make(map[Card]int)
	for _, face := range Faces() {
		for _, suit := range Suits() {
			cards[CreateCard(suit, face)] = 0
		}
	}
	for x := 0; x < len(h); x++ {
		cards[h[x]]++
	}
	return
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (h Hand) Meld(trump Suit) (meld int, show map[Card]int) {
	// hand does not have to be sorted
	count := h.Count()
	if debug {
		fmt.Printf("Count is %v\n", count)
	}
	show = make(map[Card]int)
	around := make(map[Face]int)
	for _, value := range Faces() {
		around[value] = 2
	}
	//	fmt.Printf("AroundBefore = %v\n", around)
	for _, suit := range Suits() { // look through each suit
		switch { // straights & marriages
		case trump == suit:
			if debug {
				fmt.Printf("Scoring %d nine(s) in trump %s\n", count[CreateCard(suit, nine)], trump)
			}
			meld += count[CreateCard(suit, nine)] // 9s in trump
			show[CreateCard(suit, nine)] = count[CreateCard(suit, nine)]
			switch {
			// double straight
			case count[CreateCard(suit, ace)] == 2 && count[CreateCard(suit, ten)] == 2 && count[CreateCard(suit, king)] == 2 && count[CreateCard(suit, queen)] == 2 && count[CreateCard(suit, jack)] == 2:
				meld += 150
				for _, face := range Faces() {
					show[CreateCard(suit, face)] = 2
				}
				if debug {
					fmt.Println("DoubleStraight")
				}
			// single straight
			case count[CreateCard(suit, ace)] >= 1 && count[CreateCard(suit, ten)] >= 1 && count[CreateCard(suit, king)] >= 1 && count[CreateCard(suit, queen)] >= 1 && count[CreateCard(suit, jack)] >= 1:
				for _, face := range []Face{ace, ten, king, queen, jack} {
					show[CreateCard(suit, face)] = max(show[CreateCard(suit, face)], 1)
				}
				if count[CreateCard(suit, king)] == 2 && count[CreateCard(suit, queen)] == 2 {
					show[CreateCard(suit, king)] = 2
					show[CreateCard(suit, queen)] = 2
					meld += 19
					if debug {
						fmt.Println("SingleStraightWithExtraMarriage")
					}
				} else {
					if debug {
						fmt.Println("SingleStraight")
					}
					meld += 15
				}
			case count[CreateCard(suit, king)] == 2 && count[CreateCard(suit, queen)] == 2:
				meld += 8
				show[CreateCard(suit, king)] = 2
				show[CreateCard(suit, queen)] = 2
				if debug {
					fmt.Println("DoubleMarriageInTrump")
				}
			case count[CreateCard(suit, king)] >= 1 && count[CreateCard(suit, queen)] >= 1:
				meld += 4
				show[CreateCard(suit, king)] = max(show[CreateCard(suit, king)], 1)
				show[CreateCard(suit, queen)] = max(show[CreateCard(suit, queen)], 1)
				if debug {
					fmt.Println("SingleMarriageInTrump")
				}
			}
		case count[CreateCard(suit, king)] == 2 && count[CreateCard(suit, queen)] == 2:
			show[CreateCard(suit, king)] = 2
			show[CreateCard(suit, queen)] = 2
			meld += 4
			if debug {
				fmt.Println("DoubleMarriage")
			}
		case count[CreateCard(suit, king)] >= 1 && count[CreateCard(suit, queen)] >= 1:
			show[CreateCard(suit, king)] = max(show[CreateCard(suit, king)], 1)
			show[CreateCard(suit, queen)] = max(show[CreateCard(suit, queen)], 1)
			if debug {
				fmt.Println("SingleMarriage")
			}
			meld += 2
		}
		for _, face := range Faces() { // looking for "around" meld
			//						fmt.Printf("Looking for %d in suit %d\n", value, suit)
			around[face] = min(count[CreateCard(suit, face)], around[face])
		}
	}
	for _, face := range []Face{ace, king, queen, jack} {
		if around[face] > 0 {
			var worth int
			switch face {
			case ace:
				worth = acearound
			case king:
				worth = kingaround
			case queen:
				worth = queenaround
			case jack:
				worth = jackaround
			}
			if around[face] == 2 {
				worth *= 10
			}
			for _, suit := range Suits() {
				show[CreateCard(suit, face)] = max(show[CreateCard(suit, face)], around[face])
			}
			meld += worth
			if debug {
				fmt.Printf("Around-%d\n", worth)
			}
		}
	}
	switch { // pinochle
	case count[CreateCard(diamonds, jack)] == 2 && count[CreateCard(spades, queen)] == 2:
		meld += 30
		show[CreateCard(spades, queen)] = 2
		show[CreateCard(diamonds, jack)] = 2
		if debug {
			fmt.Println("DoubleNochle")
		}
	case count[CreateCard(diamonds, jack)] >= 1 && count[CreateCard(spades, queen)] >= 1:
		meld += 4
		show[CreateCard(diamonds, jack)] = max(show[CreateCard(diamonds, jack)], 1)
		show[CreateCard(spades, queen)] = max(show[CreateCard(spades, queen)], 1)
		if debug {
			fmt.Println("Nochle")
		}
	}
	return
}
