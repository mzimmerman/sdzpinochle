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

func checkForDupes(h []Hand, t *testSuite) bool {
	check := map[Card]int{}
	for x := 0; x < 4; x++ {
		for y := 0; y < len(h[x]); y++ {
			check[h[x][y]]++
		}
	}
	for _, value := range check {
		t.Equal(value, 2)
	}
	return !t.Failed()
}

func fakeDeal(d *Deck) (h []Hand) {
	h = make([]Hand, 4)
	for x := 0; x < 4; x++ {
		h[x] = d[x*12 : x*12+12]
	}
	return
}

func (t *testSuite) TestSuit() {
	t.True(Card(ND).Suit() == Diamonds)
	t.True(Card(AD).Suit() == Diamonds)
	t.True(Card(NS).Suit() == Spades)
	t.True(Card(AS).Suit() == Spades)
	t.True(Card(NH).Suit() == Hearts)
	t.True(Card(AH).Suit() == Hearts)
	t.True(Card(NC).Suit() == Clubs)
	t.True(Card(AC).Suit() == Clubs)
}

func (t *testSuite) TestDeal() {
	deck := CreateDeck()
	var h []Hand
	h = fakeDeal(&deck)
	//	fmt.Println("Deck Created")
	t.True(checkForDupes(h, t))
	deck.Shuffle()
	//	fmt.Println("Deck Shuffled")
	h = fakeDeal(&deck)
	t.True(checkForDupes(h, t))
	h = deck.Deal()
	//	fmt.Println("Deck Dealt")
	t.True(checkForDupes(h, t))
	sort.Sort(h[0])
	sort.Sort(h[1])
	sort.Sort(h[2])
	sort.Sort(h[3])
	//	fmt.Println("Hands Sorted")
	t.True(checkForDupes(h, t))
}

func (t *testSuite) TestMin() {
	t.Equal(min(1, 2), 1)
	t.Equal(min(2, 1), 1)
	t.Equal(min(5, 5), 5)
}

func (t *testSuite) TestRemove() {
	hand := Hand{JD, QD, KD, AD, TD, JD, QS, QS, KS, AS, TS, JS}
	sort.Sort(hand)
	t.Equal(len(hand), 12)
	t.True(hand.Remove(JD))
	t.Equal(len(hand), 11)
	t.True(hand.Remove(JD))
	t.Equal(len(hand), 10)
	t.False(hand.Remove(JD))
	t.False(hand.Remove(QH))
	t.True(hand.Remove(KD))
	t.False(hand.Remove(KD))
	t.True(hand.Remove(QD))
	t.True(hand.Remove(AD))
	t.True(hand.Remove(TD))
	t.True(hand.Remove(QS))
	t.True(hand.Remove(QS))
	t.True(hand.Remove(KS))
	t.True(hand.Remove(AS))
	t.True(hand.Remove(TS))
	t.True(hand.Remove(JS))
	t.Equal(len(hand), 0)
	t.False(hand.Remove(ND))
}

func (t *testSuite) TestValidPlay() {
	// playedCard, winningCard Card, leadSuit Suit, hand Hand, trump Suit
	hand := Hand{JD, QS, ND, TH}
	t.True(ValidPlay(JD, ND, Diamonds, &hand, Diamonds))
	t.False(ValidPlay(QD, ND, Diamonds, &hand, Diamonds))
	t.True(ValidPlay(JD, ND, Diamonds, &hand, Spades))
	t.False(ValidPlay(JD, ND, Hearts, &hand, Diamonds))
	t.True(ValidPlay(TH, ND, Hearts, &hand, Diamonds))
	t.True(ValidPlay(QS, KS, Spades, &hand, Diamonds))
	t.False(ValidPlay(JD, ND, Hearts, &hand, Spades))
	t.True(ValidPlay(QS, ND, Clubs, &hand, Spades))
	t.False(ValidPlay(QS, ND, Diamonds, &hand, Spades))
	t.False(ValidPlay(JD, NC, Clubs, &hand, Spades))
	t.True(ValidPlay(JD, KD, Clubs, &hand, Diamonds))
	hand.Remove(QS)
	t.True(ValidPlay(JD, NS, Spades, &hand, Clubs))
	t.True(ValidPlay(NS, NACard, NASuit, &hand, Clubs))
	t.True(ValidPlay(ND, NACard, NASuit, &hand, Clubs))
}

func (t *testSuite) TestCount() {
	hand := Hand{JD, QD, KD, AD, JD, JS, QD, KS, AS, TS, JS, TD}
	count := hand.Count()
	t.Equal(count[JS], 2)
	t.Equal(count[KD], 1)
	t.Equal(count[AD], 1)
	t.Equal(count[TD], 1)
	t.Equal(count[QS], 0)
	t.Equal(count[KS], 1)
	t.Equal(count[AS], 1)
	t.Equal(count[TS], 1)
	t.Equal(count[JS], 2)
	t.Equal(count[JH], 0)
}

func (t *testSuite) TestMeld2() {
	// spades, hearts, clubs, diamonds
	shown := Hand{JD, JD, QD, KD, QS, QS, KS}
	sort.Sort(shown)
	hand := Hand{JD, QD, KD, AD, TD, JD, QS, QS, KS, AS, TS, JS}
	sort.Sort(hand)
	_, results := hand.Meld(Hearts)
	sort.Sort(results)
	for x := range results {
		t.Equal(results[x], shown[x])
	}
}

func (t *testSuite) TestSuitLess() {
	t.False(Suit(Spades).Less(Hearts))
	t.False(Suit(Spades).Less(Diamonds))
	t.True(Suit(Diamonds).Less(Spades))
	t.False(Suit(Hearts).Less(Hearts))
	t.True(Suit(Diamonds).Less(Hearts))
	t.False(Suit(Diamonds).Less(Diamonds))
}

func (t *testSuite) TestBeats() {
	t.True(Card(JD).Beats(ND, Diamonds))
	t.True(Card(JD).Beats(ND, Spades))
	t.True(Card(JD).Beats(NS, Diamonds))
	t.True(Card(NS).Beats(JD, Spades))
	t.False(Card(ND).Beats(ND, Diamonds))
	t.False(Card(ND).Beats(ND, Diamonds))
	t.False(Card(AD).Beats(AD, Diamonds))
}

func (t *testSuite) TestCardInHand() {
	hand := Hand{JD, QD, KD, AD, TD, JD, QS, QS, KS, AS, TS, JS}
	t.True(IsCardInHand(JD, hand))
	t.False(IsCardInHand(NS, hand))
}

func (t *testSuite) TestMeld() {
	hands := []Hand{
		Hand{JD, QD, KD, AD, TD, JD, QS, QS, KS, AS, TS, JS},
		Hand{JD, QD, KD, QD, KD, JH, JC, QS, KS, AS, TS, JS},
		Hand{ND, QD, KD, AD, TD, JD, QS, QS, KS, AS, NS, NS},
		Hand{JD, QD, KD, AD, TD, JD, QD, KD, AD, TD, TS, JS},
		Hand{AD, TD, KD, KD, QD, QD, JD, AS, KS, TH, QS, NH},
		Hand{AD, AH, AS, AC, KD, KH, KS, KC, QS, QS, JD, JD},
		Hand{QD, QH, QS, QC, QD, QH, QS, QC, NS, NS, JD, ND},
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
		for _, trump := range Suits {
			sort.Sort(hands[x])
			meld, _ := hands[x].Meld(trump)
			t.Equal(results[x][trump], meld)
		}
	}
}
