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
	t.Equal(action.Value(), 20)
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

func (t *testSuite) TestPlay() {
	hand := Hand{C("JD"), C("QD"), C("KD"), C("AD"), C("TD"), C("JD"), C("QS"), C("QS"), C("KS"), C("AS"), C("TS"), C("JS")}
	card := hand.Play(11)
	t.True(card.Face() == Jack && card.Suit() == Spades)
	card = hand.Play(0)
	t.True(card.Face() == Jack && card.Suit() == Diamonds)
	card = hand.Play(3)
	t.True(card.Face() == Ten && card.Suit() == Diamonds)
	card = hand.Play(8)
	t.True(card.Face() == Ten && card.Suit() == Spades)
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
	_, results := hand.Meld(Hearts)
	for _, face := range Faces() {
		for _, suit := range Suits() {
			realCard := CreateCard(suit, face)
			t.Equal(shown[realCard], results[realCard])
		}
	}
}

func (t *testSuite) TestBeats() {
	t.True(C("JD").Beats(C("9D"), Diamonds))
	t.True(C("JD").Beats(C("9D"), Spades))
	t.True(C("JD").Beats(C("9S"), Diamonds))
	t.True(C("9S").Beats(C("JD"), Spades))
}

func (t *testSuite) TestMeld() {
	hands := []Hand{
		Hand{C("JD"), C("QD"), C("KD"), C("AD"), C("TD"), C("JD"), C("QS"), C("QS"), C("KS"), C("AS"), C("TS"), C("JS")},
		Hand{C("JD"), C("QD"), C("KD"), C("QD"), C("KD"), C("JH"), C("JC"), C("QS"), C("KS"), C("AS"), C("TS"), C("JS")},
		Hand{C("9D"), C("QD"), C("KD"), C("AD"), C("TD"), C("JD"), C("QS"), C("QS"), C("KS"), C("AS"), C("9S"), C("9S")},
		Hand{C("JD"), C("QD"), C("KD"), C("AD"), C("TD"), C("JD"), C("QD"), C("KD"), C("AD"), C("TD"), C("TS"), C("JS")},
	}
	// spades, hearts, clubs, diamonds
	results := []map[string]int{
		map[string]int{Spades: 47, Hearts: 34, Clubs: 34, Diamonds: 47},
		map[string]int{Spades: 27, Hearts: 14, Clubs: 14, Diamonds: 18},
		map[string]int{Spades: 12, Hearts: 8, Clubs: 8, Diamonds: 22},
		map[string]int{Spades: 4, Hearts: 4, Clubs: 4, Diamonds: 150},
	}
	for x := 0; x < len(hands); x++ {
		for _, trump := range Suits() {
			sort.Sort(hands[x])
			meld, _ := hands[x].Meld(trump)
			t.Equal(meld, results[x][string(trump)])
		}
	}
}
