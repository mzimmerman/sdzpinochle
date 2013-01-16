package sdzpinochle

import (
	pt "github.com/remogatto/prettytest"
	"reflect"
	"sort"
	"testing"
)

type testSuite struct {
	pt.Suite
}

func TestFoo(t *testing.T) {
	pt.RunWithFormatter(
		t,
		new(pt.TDDFormatter),
		new(testSuite),
	)
}

func checkForDupes(h []Hand, t *testSuite) {
	check := map[Card]int{}
	for x := 0; x < 4; x++ {
		for y := 0; y < len(h[x]); y++ {
			check[h[x][y]]++
		}
	}
	for _, value := range check {
		t.Equal(value, 2)
	}
}

func fakeDeal(d *Deck) (h []Hand) {
	h = make([]Hand, 4)
	for x := 0; x < 4; x++ {
		h[x] = d[x*12 : x*12+12]
	}
	return
}

func (t *testSuite) TestAction() {
	var action Action
	action = &BidAction{bid: 20}
	t.Equal(action.(*BidAction).Bid(), 20)
	t.Equal(reflect.TypeOf(action).String(), "*sdzpinochle.BidAction")
	switch action.(type) {
	default:
		t.True(false)
	case *BidAction:

	}
}

func (t *testSuite) TestDeal() {
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

func (t *testSuite) TestMin() {
	t.Equal(min(1, 2), 1)
	t.Equal(min(2, 1), 1)
	t.Equal(min(5, 5), 5)
}

func C(c string) Card {
	return Card(c)
}

func (t *testSuite) TestRemove() {
	hand := Hand{C("JD"), C("QD"), C("KD"), C("AD"), C("TD"), C("JD"), C("QS"), C("QS"), C("KS"), C("AS"), C("TS"), C("JS")}
	sort.Sort(hand)
	t.Equal(len(hand), 12)
	t.True(hand.Remove(C("JD")))
	t.Equal(len(hand), 11)
	t.True(hand.Remove(C("JD")))
	t.Equal(len(hand), 10)
	t.False(hand.Remove(C("JD")))
	t.False(hand.Remove(C("QH")))
	t.True(hand.Remove(C("KD")))
	t.False(hand.Remove(C("KD")))
	t.True(hand.Remove(C("QD")))
	t.True(hand.Remove(C("AD")))
	t.True(hand.Remove(C("TD")))
	t.True(hand.Remove(C("QS")))
	t.True(hand.Remove(C("QS")))
	t.True(hand.Remove(C("KS")))
	t.True(hand.Remove(C("AS")))
	t.True(hand.Remove(C("TS")))
	t.True(hand.Remove(C("JS")))
	t.Equal(len(hand), 0)
	t.False(hand.Remove(C("9D")))
}

func (t *testSuite) TestValidPlay() {
	// playedCard, winningCard Card, leadSuit Suit, hand Hand, trump Suit
	hand := Hand{C("JD"), C("QS"), C("9D"), C("TH")}
	t.True(ValidPlay(C("JD"), C("9D"), Diamonds, &hand, Diamonds))
	t.False(ValidPlay(C("QD"), C("9D"), Diamonds, &hand, Diamonds))
	t.True(ValidPlay(C("JD"), C("9D"), Diamonds, &hand, Spades))
	t.False(ValidPlay(C("JD"), C("9D"), Hearts, &hand, Diamonds))
	t.True(ValidPlay(C("TH"), C("9D"), Hearts, &hand, Diamonds))
	t.True(ValidPlay(C("QS"), C("KS"), Spades, &hand, Diamonds))
	t.False(ValidPlay(C("JD"), C("9D"), Hearts, &hand, Spades))
	t.True(ValidPlay(C("QS"), C("9D"), Clubs, &hand, Spades))
	t.False(ValidPlay(C("QS"), C("9D"), Diamonds, &hand, Spades))
	t.False(ValidPlay(C("JD"), C("9C"), Clubs, &hand, Spades))
	t.True(ValidPlay(C("JD"), C("KD"), Clubs, &hand, Diamonds))
	hand.Remove(C("QS"))
	t.True(ValidPlay(C("JD"), C("9S"), Spades, &hand, Clubs))
}

func (t *testSuite) TestCount() {
	hand := Hand{C("JD"), C("QD"), C("KD"), C("AD"), C("JD"), C("JS"), C("QD"), C("KS"), C("AS"), C("TS"), C("JS"), C("TD")}
	count := hand.Count()
	t.Equal(count[C("JS")], 2)
	t.Equal(count[C("KD")], 1)
	t.Equal(count[C("AD")], 1)
	t.Equal(count[C("TD")], 1)
	t.Equal(count[C("QS")], 0)
	t.Equal(count[C("KS")], 1)
	t.Equal(count[C("AS")], 1)
	t.Equal(count[C("TS")], 1)
	t.Equal(count[C("JS")], 2)
	t.Equal(count[C("JH")], 0)
}

func (t *testSuite) TestMeld2() {
	// spades, hearts, clubs, diamonds
	shown := Hand{
		C("JD"), C("JD"),
		C("QD"),
		C("KD"),
		C("QS"), C("QS"),
		C("KS"),
	}
	sort.Sort(shown)
	hand := Hand{C("JD"), C("QD"), C("KD"), C("AD"), C("TD"), C("JD"), C("QS"), C("QS"), C("KS"), C("AS"), C("TS"), C("JS")}
	sort.Sort(hand)
	_, results := hand.Meld(Hearts)
	sort.Sort(results)
	for x := range results {
		t.Equal(results[x], shown[x])
	}
}

func (t *testSuite) TestSuitLess() {
	t.False(Spades.Less(Hearts))
	t.False(Spades.Less(Diamonds))
	t.True(Diamonds.Less(Spades))
	t.False(Hearts.Less(Hearts))
	t.True(Diamonds.Less(Hearts))
	t.False(Diamonds.Less(Diamonds))
}

func (t *testSuite) TestBeats() {
	t.True(C("JD").Beats(C("9D"), Diamonds))
	t.True(C("JD").Beats(C("9D"), Spades))
	t.True(C("JD").Beats(C("9S"), Diamonds))
	t.True(C("9S").Beats(C("JD"), Spades))
	t.False(C("9D").Beats(C("9D"), Diamonds))
	t.False(C("9D").Beats(C("9D"), Diamonds))
}

func (t *testSuite) TestCardInHand() {
	hand := Hand{C("JD"), C("QD"), C("KD"), C("AD"), C("TD"), C("JD"), C("QS"), C("QS"), C("KS"), C("AS"), C("TS"), C("JS")}
	t.True(IsCardInHand(C("JD"), hand))
	t.False(IsCardInHand(C("9S"), hand))
}

func (t *testSuite) TestMeld() {
	hands := []Hand{
		Hand{C("JD"), C("QD"), C("KD"), C("AD"), C("TD"), C("JD"), C("QS"), C("QS"), C("KS"), C("AS"), C("TS"), C("JS")},
		Hand{C("JD"), C("QD"), C("KD"), C("QD"), C("KD"), C("JH"), C("JC"), C("QS"), C("KS"), C("AS"), C("TS"), C("JS")},
		Hand{C("9D"), C("QD"), C("KD"), C("AD"), C("TD"), C("JD"), C("QS"), C("QS"), C("KS"), C("AS"), C("9S"), C("9S")},
		Hand{C("JD"), C("QD"), C("KD"), C("AD"), C("TD"), C("JD"), C("QD"), C("KD"), C("AD"), C("TD"), C("TS"), C("JS")},
		Hand{C("AD"), C("TD"), C("KD"), C("KD"), C("QD"), C("QD"), C("JD"), C("AS"), C("KS"), C("TH"), C("QS"), C("9H")},
		Hand{C("AD"), C("AH"), C("AS"), C("AC"), C("KD"), C("KH"), C("KS"), C("KC"), C("QS"), C("QS"), C("JD"), C("JD")},
		Hand{C("QD"), C("QH"), C("QS"), C("QC"), C("QD"), C("QH"), C("QS"), C("QC"), C("9S"), C("9S"), C("JD"), C("9D")},
	}
	// spades, hearts, clubs, diamonds
	results := []map[Suit]int{
		map[Suit]int{Spades: 47, Hearts: 34, Clubs: 34, Diamonds: 47},
		map[Suit]int{Spades: 27, Hearts: 14, Clubs: 14, Diamonds: 18},
		map[Suit]int{Spades: 12, Hearts: 8, Clubs: 8, Diamonds: 22},
		map[Suit]int{Spades: 4, Hearts: 4, Clubs: 4, Diamonds: 150},
		map[Suit]int{Spades: 12, Hearts: 11, Clubs: 10, Diamonds: 25},
		map[Suit]int{Spades: 52, Hearts: 50, Clubs: 50, Diamonds: 50},
		map[Suit]int{Spades: 66, Hearts: 64, Clubs: 64, Diamonds: 65},
	}
	for x := 0; x < len(hands); x++ {
		for _, trump := range Suits() {
			sort.Sort(hands[x])
			meld, _ := hands[x].Meld(trump)
			t.Equal(results[x][trump], meld)
		}
	}
}
