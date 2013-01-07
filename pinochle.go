// pinochle.go
package sdzpinochle

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

const (
	ace = iota
	ten
	king
	queen
	jack
	nine
)

const (
	spades = iota
	hearts
	clubs
	diamonds
)

const (
	acearound   = 10
	kingaround  = 8
	queenaround = 6
	jackaround  = 4
)

type Card struct {
	card, suit int
}

type Deck [48]Card
type Hand []Card

func (c Card) String() string {
	var str string
	switch c.card {
	case nine:
		str = "9"
	case ten:
		str = "10"
	case jack:
		str = "J"
	case queen:
		str = "Q"
	case king:
		str = "K"
	case ace:
		str = "A"
	}
	switch c.suit {
	case spades:
		return str + "S"
	case clubs:
		return str + "C"
	case diamonds:
		return str + "D"
	case hearts:
		return str + "H"
	}
	return "Card does not exist"
}

func (c Card) isTrump(trump int) bool {
	return c.suit == trump
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
		cards += h[x].String() + " "
	}
}

func (h Hand) Len() int {
	return len(h)
}

func (h Hand) Less(i, j int) bool {
	if h[i].suit == h[j].suit {
		return h[i].card < h[j].card
	}
	return h[i].suit < h[j].suit
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

func createDeck() (deck Deck) {
	cards := [6]int{nine, ten, jack, queen, king, ace}
	suits := [4]int{spades, hearts, diamonds, clubs}
	index := 0
	for x := 0; x < len(cards); x++ {
		for y := 0; y < len(suits); y++ {
			for z := 0; z < 2; z++ {
				deck[index] = Card{card: cards[x], suit: suits[y]}
				index++
			}
		}
	}
	return
}

type Player struct {
	hand Hand
}

type AI struct {
	Player
}

type Game struct {
	deck Deck
	p    []Player
}

func createGame() (game *Game) {
	game.deck = createDeck()
	game.deck.Shuffle()
	hands := game.deck.Deal()
	for x := 0; x < 4; x++ {
		game.p[x].hand = hands[x]
		sort.Sort(game.p[x].hand)
	}
	return
}

func (h Hand) Count() (cards map[Card]int) {
	cards = make(map[Card]int)
	//	for card := range []int{ace, ten, king, queen, jack, nine} {
	//		for suit := range []int{spades, hearts, clubs, diamonds} {
	//			cards[Card{suit:suit, card:card}] = 0
	//		}
	//	}
	for x := 0; x < len(h); x++ {
		cards[h[x]]++
	}
	return
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (h Hand) Meld(trump int) (meld int) {
	return h.MeldDebug(trump, false)
}

func (h Hand) MeldDebug(trump int, debug bool) (meld int) {
	// hand does not have to be sorted
	count := h.Count()
	around := [6]int{}
	for _, value := range []int{ace, king, queen, jack} {
		around[value] = 2
	}
	//	fmt.Printf("ace %d ten %d king %d queen %d jack %d nine %d\n", ace, ten, king, queen, jack, nine)
	//	fmt.Printf("AroundBefore = %v\n", around)
	for _, suit := range []int{spades, hearts, clubs, diamonds} { // look through each suit
		switch { // straights & marriages
		case trump == suit:
			if debug {
				fmt.Printf("Scoring %d nine(s) in trump %d\n", count[Card{suit: suit, card: nine}], suit)
			}
			meld += count[Card{suit: suit, card: nine}] // 9s in trump
			switch {
			// double straight
			case count[Card{suit: suit, card: ace}] == 2 && count[Card{suit: suit, card: ten}] == 2 && count[Card{suit: suit, card: king}] == 2 && count[Card{suit: suit, card: queen}] == 2 && count[Card{suit: suit, card: jack}] == 2:
				meld += 150
				if debug {
					fmt.Println("DoubleStraight")
				}
			// single straight
			case count[Card{suit: suit, card: ace}] >= 1 && count[Card{suit: suit, card: ten}] >= 1 && count[Card{suit: suit, card: king}] >= 1 && count[Card{suit: suit, card: queen}] >= 1 && count[Card{suit: suit, card: jack}] >= 1:
				if count[Card{suit: suit, card: king}] == 2 && count[Card{suit: suit, card: queen}] == 2 {
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
			case count[Card{suit: suit, card: king}] == 2 && count[Card{suit: suit, card: queen}] == 2:
				meld += 8
				if debug {
					fmt.Println("DoubleMarriageInTrump")
				}
			case count[Card{suit: suit, card: king}] >= 1 && count[Card{suit: suit, card: queen}] >= 1:
				meld += 4
				if debug {
					fmt.Println("SingleMarriageInTrump")
				}
			}
		case count[Card{suit: suit, card: king}] == 2 && count[Card{suit: suit, card: queen}] == 2:
			meld += 4
			if debug {
				fmt.Println("DoubleMarriage")
			}
		case count[Card{suit: suit, card: king}] >= 1 && count[Card{suit: suit, card: queen}] >= 1:
			if debug {
				fmt.Println("SingleMarriage")
			}
			meld += 2
		}
		for _, value := range [4]int{ace, king, queen, jack} { // looking for "around" meld
			//						fmt.Printf("Looking for %d in suit %d\n", value, suit)
			around[value] = min(count[Card{suit: suit, card: value}], around[value])
		}
	}
	for _, value := range [4]int{ace, king, queen, jack} {
		if around[value] > 0 {
			var worth int
			switch value {
			case ace:
				worth = acearound
			case king:
				worth = kingaround
			case queen:
				worth = queenaround
			case jack:
				worth = jackaround
			}
			if around[value] == 2 {
				worth *= 10
			}
			meld += worth
			if debug {
				fmt.Printf("Around-%d\n", worth)
			}
		}
	}
	switch { // pinochle
	case count[Card{suit: diamonds, card: jack}] == 2 && count[Card{suit: spades, card: queen}] == 2:
		meld += 30
		if debug {
			fmt.Println("DoubleNochle")
		}
	case count[Card{suit: diamonds, card: jack}] >= 1 && count[Card{suit: spades, card: queen}] >= 1:
		meld += 4
		if debug {
			fmt.Println("Nochle")
		}
	}
	return
}
