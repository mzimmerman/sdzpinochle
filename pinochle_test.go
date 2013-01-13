package sdzpinochle

import (
	pt "github.com/remogatto/prettytest"
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
	t.True(card.Face() == jack && card.Suit() == spades)
	card = hand.Play(0)
	t.True(card.Face() == jack && card.Suit() == diamonds)
	card = hand.Play(3)
	t.True(card.Face() == ten && card.Suit() == diamonds)
	card = hand.Play(8)
	t.True(card.Face() == ten && card.Suit() == spades)
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
	_, results := hand.Meld(hearts)
	for _, face := range Faces() {
		for _, suit := range Suits() {
			realCard := CreateCard(suit, face)
			t.Equal(shown[realCard], results[realCard])
		}
	}
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
		map[string]int{spades: 47, hearts: 34, clubs: 34, diamonds: 47},
		map[string]int{spades: 27, hearts: 14, clubs: 14, diamonds: 18},
		map[string]int{spades: 12, hearts: 8, clubs: 8, diamonds: 22},
		map[string]int{spades: 4, hearts: 4, clubs: 4, diamonds: 150},
	}
	for x := 0; x < len(hands); x++ {
		for _, trump := range Suits() {
			sort.Sort(hands[x])
			meld, _ := hands[x].Meld(trump)
			t.Equal(meld, results[x][string(trump)])
		}
	}
}
