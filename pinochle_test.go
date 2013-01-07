package sdzpinochle

import (
	"fmt"
	"sort"
	"testing"
)

func checkForDupes(h []Hand, t *testing.T) {
	check := map[string]int{}
	for x := 0; x < 4; x++ {
		for y := 0; y < len(h[x]); y++ {
			check[h[x][y].String()]++
		}
	}
	for key, value := range check {
		if value != 2 {
			t.Fatalf("there are %d %s", value, key)
		}
	}
}

func fakeDeal(d *Deck) (h []Hand) {
	h = make([]Hand, 4)
	for x := 0; x < 4; x++ {
		h[x] = d[x*12 : x*12+12]
	}
	return
}

func TestHands(t *testing.T) {
	deck := createDeck()
	var h []Hand
	h = fakeDeal(&deck)
	fmt.Println("Deck Created")
	checkForDupes(h, t)
	deck.Shuffle()
	fmt.Println("Deck Shuffled")
	h = fakeDeal(&deck)
	checkForDupes(h, t)
	h = deck.Deal()
	fmt.Println("Deck Dealt")
	checkForDupes(h, t)
	sort.Sort(h[0])
	sort.Sort(h[1])
	sort.Sort(h[2])
	sort.Sort(h[3])
	fmt.Println("Hands Sorted")
	checkForDupes(h, t)
}
