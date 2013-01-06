// pinochle.go
package main

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	nine = iota
	jack
	queen
	king
	ten
	ace
)

const (
	spades = iota
	hearts
	diamonds
	clubs
)

type Card struct {
	card int
	suit int
}

type Deck [48]Card
type Hand [12]Card

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
	victim := d[i]
	d[i] = d[j]
	d[j] = victim
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

func (d Deck) Deal() (hands [4]Hand) {
	for y := 0; y < 12; y++ {
		for x := 0; x < 4; x++ {
			hands[x][y] = d[y+x]
			fmt.Println("hands[%v][%v] = deck[%v]", x, y, (y*4)+x)
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
				deck[index] = Card{cards[x], suits[y]}
				index++
			}
		}
	}
	return
}

func main() {
	fmt.Println("Hello World!")
	deck := createDeck()
	deck.Shuffle()
	for x := 0; x < len(deck); x++ {
		fmt.Println(deck[x].String())
	}
	fmt.Println("----")
	hands := deck.Deal()
	for x := 0; x < 4; x++ {
		fmt.Println("Hand %d", x+1)
		for y := 0; y < len(hands[x]); y++ {
			fmt.Println(hands[x][y].String())
		}
	}
}
