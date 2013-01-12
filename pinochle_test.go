package sdzpinochle

import (
	"fmt"
	"sort"
	"testing"
)

func checkForDupes(h []Hand, t *testing.T) {
	check := map[Card]int{}
	for x := 0; x < 4; x++ {
		for y := 0; y < len(h[x]); y++ {
			check[h[x][y]]++
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

func C(c string) Card {
	return Card(c)
}

func TestPlay(t *testing.T) {
	hand := Hand{C("JD"), C("QD"), C("KD"), C("AD"), C("TD"), C("JD"), C("QS"), C("QS"), C("KS"), C("AS"), C("TS"), C("JS")}
	card := hand.Play(11)
	if card.Face() != jack || card.Suit() != spades {
		t.Errorf("Should have gotten a jack of spades - %s", card)
	}
	card = hand.Play(0)
	if card.Face() != jack || card.Suit() != diamonds {
		t.Errorf("Should have gotten a jack of diamonds - %s", card)
	}
	card = hand.Play(3)
	if card.Face() != ten || card.Suit() != diamonds {
		t.Errorf("Should have gotten a ten of diamonds - %s", card)
	}
	card = hand.Play(8)
	if card.Face() != ten || card.Suit() != spades {
		t.Errorf("Should have gotten a ten of spades - %s", card)
	}
}

func TestCount(t *testing.T) {
	hand := Hand{C("JD"), C("QD"), C("KD"), C("AD"), C("JD"), C("JS"), C("QD"), C("KS"), C("AS"), C("TS"), C("JS"), C("TD")}
	count := hand.Count()
	if count[C("JS")] != 2 {
		t.Errorf("There should be two jacks of diamonds")
	}
	if count[C("QD")] != 2 {
		t.Errorf("There should be two queens of diamonds")
	}
	if count[C("KD")] != 1 {
		t.Errorf("There should be one king of diamonds")
	}
	if count[C("AD")] != 1 {
		t.Errorf("There should be one ace of diamonds")
	}
	if count[C("TD")] != 1 {
		t.Errorf("There should be one ten of diamonds")
	}
	if count[C("QS")] != 0 {
		t.Errorf("There should be 0 queens of spades")
	}
	if count[C("KS")] != 1 {
		t.Errorf("There should be one king of spades")
	}
	if count[C("AS")] != 1 {
		t.Errorf("There should be one ace of spades")
	}
	if count[C("TS")] != 1 {
		t.Errorf("There should be one ten of spades")
	}
	if count[C("JS")] != 2 {
		t.Errorf("There should be two jack of spades")
	}
	if count[C("JH")] != 0 {
		t.Errorf("There should be zero jacks of hearts")
	}
}

func TestMeld2(t *testing.T) {
	// spades, hearts, clubs, diamonds
	shown := map[Card]int{
		C("JD"): 2,
		C("QD"): 1,
		C("KD"): 1,
		C("AD"): 0,
		C("TD"): 0,
		C("QS"): 2,
		C("KS"): 1,
		C("AS"): 0,
		C("TS"): 0,
		C("JS"): 0,
	}
	hand := Hand{C("JD"), C("QD"), C("KD"), C("AD"), C("TD"), C("JD"), C("QS"), C("QS"), C("KS"), C("AS"), C("TS"), C("JS")}
	_, results := hand.Meld(hearts)
	for _, face := range Faces() {
		for _, suit := range Suits() {
			realCard := CreateCard(suit, face)
			if shown[realCard] != results[realCard] {
				t.Errorf("shown[realCard]=%d != results[realCard]=%d for %s", shown[realCard], results[realCard], realCard)
			}
		}
	}
}

func TestMeld(t *testing.T) {
	hands := []Hand{
		Hand{C("JD"), C("QD"), C("KD"), C("AD"), C("TD"), C("JD"), C("QS"), C("QS"), C("KS"), C("AS"), C("TS"), C("JS")},
		Hand{C("JD"), C("QD"), C("KD"), C("QD"), C("KD"), C("JH"), C("JC"), C("QS"), C("KS"), C("AS"), C("TS"), C("JS")},
		Hand{C("9D"), C("QD"), C("KD"), C("AD"), C("TD"), C("JD"), C("QS"), C("QS"), C("KS"), C("AS"), C("9S"), C("9S")},
		Hand{C("JD"), C("QD"), C("KD"), C("AD"), C("TD"), C("JD"), C("QD"), C("KD"), C("AD"), C("TD"), C("TS"), C("JS")},
	}
	// spades, hearts, clubs, diamonds
	results := []map[string]int{
		map[string]int{spades: 47, hearts: 34, clubs: 34, diamonds: 47},
		map[string]int{spades: 27, hearts: 14, clubs: 14, diamonds: 18},
		map[string]int{spades: 12, hearts: 8, clubs: 8, diamonds: 22},
		map[string]int{spades: 4, hearts: 4, clubs: 4, diamonds: 150},
	}
	for x := 0; x < len(hands); x++ {
		for _, trump := range Suits() {
			sort.Sort(hands[x])
			meld, _ := hands[x].Meld(trump)
			if meld != results[x][string(trump)] {
				fmt.Printf("Tested hand #%d %v with %s\n", x, hands[x], trump)
				t.Errorf("Trump is %s, hand %d, %s, should be %d", trump, x, hands[x], results[x][string(trump)])
			}
		}
	}
}
