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

func TestDeal(t *testing.T) {
	deck := CreateDeck()
	var h []Hand
	h = fakeDeal(&deck)
	//	fmt.Println("Deck Created")
	checkForDupes(h, t)
	deck.Shuffle()
	//	fmt.Println("Deck Shuffled")
	h = fakeDeal(&deck)
	checkForDupes(h, t)
	h = deck.Deal()
	//	fmt.Println("Deck Dealt")
	checkForDupes(h, t)
	sort.Sort(h[0])
	sort.Sort(h[1])
	sort.Sort(h[2])
	sort.Sort(h[3])
	//	fmt.Println("Hands Sorted")
	checkForDupes(h, t)
}

func TestMin(t *testing.T) {
	if min(1, 2) != 1 {
		t.FailNow()
	}
	if min(2, 1) != 1 {
		t.FailNow()
	}
	if min(5, 5) != 5 {
		t.FailNow()
	}
}
func TestPlay(t *testing.T) {
	hand := Hand{
		Card{jack, diamonds},
		Card{queen, diamonds},
		Card{king, diamonds},
		Card{ace, diamonds},
		Card{ten, diamonds},
		Card{jack, diamonds},
		Card{queen, spades},
		Card{queen, spades},
		Card{king, spades},
		Card{ace, spades},
		Card{ten, spades},
		Card{jack, spades},
	}
	card := hand.Play(11)
	if card.card != jack || card.suit != spades {
		t.Errorf("Should have gotten a jack of spades - %s", card)
	}
	card = hand.Play(0)
	if card.card != jack || card.suit != diamonds {
		t.Errorf("Should have gotten a jack of diamonds - %s", card)
	}
	card = hand.Play(3)
	if card.card != ten || card.suit != diamonds {
		t.Errorf("Should have gotten a ten of diamonds - %s", card)
	}
	card = hand.Play(8)
	if card.card != ten || card.suit != spades {
		t.Errorf("Should have gotten a ten of spades - %s", card)
	}
}

func TestCount(t *testing.T) {
	hand := Hand{
		Card{jack, diamonds},
		Card{queen, diamonds},
		Card{king, diamonds},
		Card{ace, diamonds},
		Card{ten, diamonds},
		Card{jack, diamonds},
		Card{queen, spades},
		Card{queen, spades},
		Card{king, spades},
		Card{ace, spades},
		Card{ten, spades},
		Card{jack, spades},
	}
	count := hand.Count()
	if count[Card{jack, diamonds}] != 2 {
		t.Errorf("There should be two jacks of diamonds")
	}
	if count[Card{queen, diamonds}] != 1 {
		t.Errorf("There should be one jacks of diamonds")
	}
	if count[Card{king, diamonds}] != 1 {
		t.Errorf("There should be one jacks of diamonds")
	}
	if count[Card{ace, diamonds}] != 1 {
		t.Errorf("There should be one jacks of diamonds")
	}
	if count[Card{ten, diamonds}] != 1 {
		t.Errorf("There should be one jacks of diamonds")
	}
	if count[Card{queen, spades}] != 2 {
		t.Errorf("There should be two queen of spades")
	}
	if count[Card{king, spades}] != 1 {
		t.Errorf("There should be one king of spades")
	}
	if count[Card{ace, spades}] != 1 {
		t.Errorf("There should be one ace of spades")
	}
	if count[Card{ten, spades}] != 1 {
		t.Errorf("There should be one ten of spades")
	}
	if count[Card{jack, spades}] != 1 {
		t.Errorf("There should be one jack of spades")
	}
	if count[Card{jack, hearts}] != 0 {
		t.Errorf("There should be zero jacks of hearts")
	}
}

func TestMeld2(t *testing.T) {
	// spades, hearts, clubs, diamonds
	shown := map[Card]int{
		Card{jack, diamonds}:  2,
		Card{queen, diamonds}: 1,
		Card{king, diamonds}:  1,
		Card{ace, diamonds}:   0,
		Card{ten, diamonds}:   0,
		Card{queen, spades}:   2,
		Card{king, spades}:    1,
		Card{ace, spades}:     0,
		Card{ten, spades}:     0,
		Card{jack, spades}:    0,
	}
	hand := Hand{
		Card{jack, diamonds},
		Card{queen, diamonds},
		Card{king, diamonds},
		Card{ace, diamonds},
		Card{ten, diamonds},
		Card{jack, diamonds},
		Card{queen, spades},
		Card{queen, spades},
		Card{king, spades},
		Card{ace, spades},
		Card{ten, spades},
		Card{jack, spades},
	}
	_, results := hand.Meld(hearts)
	for _, card := range []int{ace, ten, king, queen, jack, nine} {
		for _, suit := range []int{spades, hearts, clubs, diamonds} {
			realCard := Card{suit: suit, card: card}
			if shown[realCard] != results[realCard] {
				t.Errorf("shown[realCard]=%d != results[realCard]=%d for %s", shown[realCard], results[realCard], realCard)
			}
		}
	}
}

func TestMeld(t *testing.T) {
	hands := []Hand{
		Hand{
			Card{jack, diamonds},
			Card{queen, diamonds},
			Card{king, diamonds},
			Card{ace, diamonds},
			Card{ten, diamonds},
			Card{jack, diamonds},
			Card{queen, spades},
			Card{queen, spades},
			Card{king, spades},
			Card{ace, spades},
			Card{ten, spades},
			Card{jack, spades},
		},
		Hand{
			Card{jack, diamonds},
			Card{queen, diamonds},
			Card{king, diamonds},
			Card{queen, diamonds},
			Card{king, diamonds},
			Card{jack, hearts},
			Card{jack, clubs},
			Card{queen, spades},
			Card{king, spades},
			Card{ace, spades},
			Card{ten, spades},
			Card{jack, spades},
		},
		Hand{
			Card{nine, diamonds},
			Card{queen, diamonds},
			Card{king, diamonds},
			Card{ace, diamonds},
			Card{ten, diamonds},
			Card{jack, diamonds},
			Card{queen, spades},
			Card{queen, spades},
			Card{king, spades},
			Card{ace, spades},
			Card{nine, spades},
			Card{nine, spades},
		},
		Hand{
			Card{jack, diamonds},
			Card{queen, diamonds},
			Card{king, diamonds},
			Card{ace, diamonds},
			Card{ten, diamonds},
			Card{jack, diamonds},
			Card{queen, diamonds},
			Card{king, diamonds},
			Card{ace, diamonds},
			Card{ten, diamonds},
			Card{ten, spades},
			Card{jack, spades},
		},
	}
	// spades, hearts, clubs, diamonds
	results := [][]int{
		[]int{47, 34, 34, 47},
		[]int{27, 14, 14, 18},
		[]int{12, 8, 8, 22},
		[]int{4, 4, 4, 150},
	}

	for x := 0; x < len(hands); x++ {
		for _, suit := range []int{spades, hearts, clubs, diamonds} {
			var trump string
			switch suit {
			case hearts:
				trump = "hearts"
			case spades:
				trump = "spades"
			case diamonds:
				trump = "diamonds"
			case clubs:
				trump = "clubs"
			}
			sort.Sort(hands[x])
			meld, _ := hands[x].Meld(suit)
			if meld != results[x][suit] {
				fmt.Printf("Tested hand #%d %v with %s\n", x, hands[x], trump)
				t.Errorf("Trump is %s, hand %d, %s, should be %d", trump, x, hands[x], results[x][suit])
			}
		}
	}
}
