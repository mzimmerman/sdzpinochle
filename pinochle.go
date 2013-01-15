// pinochle.go
package sdzpinochle

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

const (
	debug       = false
	Ace         = "A"
	Ten         = "T"
	King        = "K"
	Queen       = "Q"
	Jack        = "J"
	Nine        = "9"
	Spades      = "S"
	Hearts      = "H"
	Clubs       = "C"
	Diamonds    = "D"
	acearound   = 10
	kingaround  = 8
	queenaround = 6
	jackaround  = 4
)

type Card string // two chars Face + String
type Suit string // one char
type Face string // one char

func Faces() [6]Face {
	return [6]Face{Ace, Ten, King, Queen, Jack, Nine}
}

func Suits() [4]Suit {
	return [4]Suit{Spades, Hearts, Clubs, Diamonds}
}

type Deck [48]Card
type Hand []Card

func CreateCard(suit Suit, face Face) Card {
	return Card(string(face) + string(suit))
}

func (a Card) Beats(b Card, trump Suit) bool {
	// a is the challenging card
	switch {
	case a.Suit() == b.Suit():
		return a.Face().Less(b.Face())
	case a.Suit() == trump:
		return true
	}
	return false
}

func (c Card) Suit() Suit {
	return Suit(c[1])
}

func (c Card) Face() Face {
	return Face(c[0])
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
	case b == Ace:
		return false
	case a == Ace:
		return true
	case b == Ten:
		return false
	case a == Ten:
		return true
	case b == King:
		return false
	case a == King:
		return true
	case b == Queen:
		return false
	case a == Queen:
		return true
	case b == Jack:
		return false
	case a == Jack:
		return true
	case b == Nine:
		return false
	}
	return true
}

func (a Suit) Less(b Suit) bool { // only for sorting the suits for display in the hand
	switch {
	case a == Spades:
		return false
	case b == Spades:
		return true
	case a == Hearts:
		return false
	case b == Hearts:
		return true
	case a == Clubs:
		return false
	case b == Clubs:
		return true
	case a == Diamonds:
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

func CreateMeld(hand Hand, amount, playerid int) *MeldAction {
	ma := MeldAction{hand: hand, amount: amount}
	ma.SetPlayer(playerid)
	return &ma
}

type MeldAction struct {
	ActionImpl
	hand   Hand
	amount int
}

func (x MeldAction) Value() interface{} {
	return []interface{}{x.hand, x.amount}
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
	Team() int
}

type PlayerImpl struct {
	Id int
}

func (p PlayerImpl) Playerid() int {
	return p.Id
}

func (p PlayerImpl) Team() int {
	return p.Playerid() % 2
}

func (p PlayerImpl) IsPartner(player int) bool {
	return p.Playerid()%2 == player%2
}

type Game struct {
	Deck       Deck
	Players    []Player
	Dealer     int
	Score      []int
	Meld       []int
	Counters   []int
	MeldHands  []Hand
	HighBid    int
	HighPlayer int
	Trump      Suit
}

func CreateGame() (game *Game) {
	game = &Game{}
	game.Deck = CreateDeck()
	game.Dealer = 0
	return
}

func (g Game) Broadcast(a Action, p int) {
	for x, player := range g.Players {
		if p != x {
			player.Tell(a)
		}
	}
}

func (g Game) BroadcastAll(a Action) {
	g.Broadcast(a, -1)
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

func (h Hand) Meld(trump Suit) (meld int, result Hand) {
	// hand does not have to be sorted
	count := h.Count()
	if debug {
		fmt.Printf("Count is %v\n", count)
	}
	show := make(map[Card]int)
	around := make(map[Face]int)
	for _, value := range Faces() {
		around[value] = 2
	}
	//	fmt.Printf("AroundBefore = %v\n", around)
	for _, suit := range Suits() { // look through each suit
		switch { // straights & marriages
		case trump == suit:
			if debug {
				fmt.Printf("Scoring %d nine(s) in trump %s\n", count[CreateCard(suit, Nine)], trump)
			}
			meld += count[CreateCard(suit, Nine)] // 9s in trump
			show[CreateCard(suit, Nine)] = count[CreateCard(suit, Nine)]
			switch {
			// double straight
			case count[CreateCard(suit, Ace)] == 2 && count[CreateCard(suit, Ten)] == 2 && count[CreateCard(suit, King)] == 2 && count[CreateCard(suit, Queen)] == 2 && count[CreateCard(suit, Jack)] == 2:
				meld += 150
				for _, face := range Faces() {
					show[CreateCard(suit, face)] = 2
				}
				if debug {
					fmt.Println("DoubleStraight")
				}
			// single straight
			case count[CreateCard(suit, Ace)] >= 1 && count[CreateCard(suit, Ten)] >= 1 && count[CreateCard(suit, King)] >= 1 && count[CreateCard(suit, Queen)] >= 1 && count[CreateCard(suit, Jack)] >= 1:
				for _, face := range []Face{Ace, Ten, King, Queen, Jack} {
					show[CreateCard(suit, face)] = max(show[CreateCard(suit, face)], 1)
				}
				if count[CreateCard(suit, King)] == 2 && count[CreateCard(suit, Queen)] == 2 {
					show[CreateCard(suit, King)] = 2
					show[CreateCard(suit, Queen)] = 2
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
			case count[CreateCard(suit, King)] == 2 && count[CreateCard(suit, Queen)] == 2:
				meld += 8
				show[CreateCard(suit, King)] = 2
				show[CreateCard(suit, Queen)] = 2
				if debug {
					fmt.Println("DoubleMarriageInTrump")
				}
			case count[CreateCard(suit, King)] >= 1 && count[CreateCard(suit, Queen)] >= 1:
				meld += 4
				show[CreateCard(suit, King)] = max(show[CreateCard(suit, King)], 1)
				show[CreateCard(suit, Queen)] = max(show[CreateCard(suit, Queen)], 1)
				if debug {
					fmt.Println("SingleMarriageInTrump")
				}
			}
		case count[CreateCard(suit, King)] == 2 && count[CreateCard(suit, Queen)] == 2:
			show[CreateCard(suit, King)] = 2
			show[CreateCard(suit, Queen)] = 2
			meld += 4
			if debug {
				fmt.Println("DoubleMarriage")
			}
		case count[CreateCard(suit, King)] >= 1 && count[CreateCard(suit, Queen)] >= 1:
			show[CreateCard(suit, King)] = max(show[CreateCard(suit, King)], 1)
			show[CreateCard(suit, Queen)] = max(show[CreateCard(suit, Queen)], 1)
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
	for _, face := range []Face{Ace, King, Queen, Jack} {
		if around[face] > 0 {
			var worth int
			switch face {
			case Ace:
				worth = acearound
			case King:
				worth = kingaround
			case Queen:
				worth = queenaround
			case Jack:
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
	case count[CreateCard(Diamonds, Jack)] == 2 && count[CreateCard(Spades, Queen)] == 2:
		meld += 30
		show[CreateCard(Spades, Queen)] = 2
		show[CreateCard(Diamonds, Jack)] = 2
		if debug {
			fmt.Println("DoubleNochle")
		}
	case count[CreateCard(Diamonds, Jack)] >= 1 && count[CreateCard(Spades, Queen)] >= 1:
		meld += 4
		show[CreateCard(Diamonds, Jack)] = max(show[CreateCard(Diamonds, Jack)], 1)
		show[CreateCard(Spades, Queen)] = max(show[CreateCard(Spades, Queen)], 1)
		if debug {
			fmt.Println("Nochle")
		}
	}
	result = make([]Card, 0, 12)
	for card, amount := range show {
		for {
			if amount > 0 {
				result = append(result, card)
				amount--
			} else {
				break
			}
		}
	}
	sort.Sort(result)
	return
}
