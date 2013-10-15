// server_test.go
package sdzpinochleserver

import (
	"github.com/icub3d/appenginetesting"
	"github.com/mzimmerman/goon"
	sdz "github.com/mzimmerman/sdzpinochle"
	pt "github.com/remogatto/prettytest"
	"sort"
	//"strconv"
	"testing"
)

const (
	AS     sdz.Card = iota
	TS     sdz.Card = iota
	KS     sdz.Card = iota
	QS     sdz.Card = iota
	JS     sdz.Card = iota
	NS     sdz.Card = iota
	AH     sdz.Card = iota
	TH     sdz.Card = iota
	KH     sdz.Card = iota
	QH     sdz.Card = iota
	JH     sdz.Card = iota
	NH     sdz.Card = iota
	AC     sdz.Card = iota
	TC     sdz.Card = iota
	KC     sdz.Card = iota
	QC     sdz.Card = iota
	JC     sdz.Card = iota
	NC     sdz.Card = iota
	AD     sdz.Card = iota
	TD     sdz.Card = iota
	KD     sdz.Card = iota
	QD     sdz.Card = iota
	JD     sdz.Card = iota
	ND     sdz.Card = iota
	NACard sdz.Card = -1
)

func TestFoo(t *testing.T) {
	pt.RunWithFormatter(
		t,
		new(pt.TDDFormatter),
		new(testSuite),
	)
}

type testSuite struct {
	pt.Suite
}

func (t *testSuite) TestRemoveShort() {
	hand := sdz.Hand{JD, QD, KD, AD, TD, JD, QS, QS, KS, AS, TS, JS}
	sort.Sort(hand)
	ai := createAI()
	ai.SetHand(nil, nil, nil, hand, 0, 0)
	t.Equal(12, len(*ai.Hand()))
	t.True(ai.Hand().Remove(JD))
	t.True(ai.Hand().Remove(JD))
	t.Not(t.True(ai.Hand().Remove(ND)))
	t.Equal(10, len(*ai.Hand()))
}

func (t *testSuite) TestBiddingShort() {
	// ND ND QD TD TD AD JC QC KC AH AH KS
	hand := sdz.Hand{ND, ND, QD, TD, TD, AD, JC, QC, KC, AH, AH, KS}
	sort.Sort(hand)
	ai := createAI()
	ai.SetHand(nil, nil, nil, hand, 0, 1)
	action := ai.Tell(nil, nil, nil, sdz.CreateBid(0, 1))
	t.Not(t.True(22 > action.Bid || action.Bid > 24))
}

func (t *testSuite) TestTrickStringShort() {
	trump := sdz.Suit(sdz.Diamonds)
	trick := new(Trick)
	t.Equal("-----", trick.String())
	trick.PlayCard(TC, trump)
	t.Equal("-lwTC----", trick.String())
	trick.PlayCard(AC, trump)
	t.Equal("-lTC-wAC---", trick.String())
	trick.PlayCard(KD, trump)
	t.Equal("-lTC-AC-wKD--", trick.String())
	trick.PlayCard(QS, trump)
	t.Equal("-lTC-AC-wKD-QS-", trick.String())
	trick.reset()
	trick.Next = 1
	t.Equal("-----", trick.String())
	trick.PlayCard(TC, trump)
	t.Equal("--lwTC---", trick.String())
	trick.PlayCard(AC, trump)
	t.Equal("--lTC-wAC--", trick.String())
	trick.PlayCard(KD, trump)
	t.Equal("--lTC-AC-wKD-", trick.String())
	trick.PlayCard(QS, trump)
	t.Equal("-QS-lTC-AC-wKD-", trick.String())
	trick.reset()
	trick.Next = 1 // 3 won the last trick, but we're starting with 1
	trick.PlayCard(AD, trump)
	t.Equal("--lwAD---", trick.String())
	trick.reset()
	trick.Next = 2
	trick.PlayCard(AD, trump)
	t.Equal("---lwAD--", trick.String())
	trick.reset()
	trick.Next = 3
	trick.PlayCard(AD, trump)
	t.Equal("----lwAD-", trick.String())
	trick.reset()
	trick.Next = 0
	trick.PlayCard(AD, trump)
	t.Equal("-lwAD----", trick.String())
}

func BenchmarkFindCardToPlay(b *testing.B) {
	//func (ai *AI) findCardToPlay(action *sdz.Action) sdz.Card {
	trump := sdz.Suit(sdz.Diamonds)
	p0 := createAI()
	p0.SetHand(nil, nil, nil, sdz.Hand{QS, NC, ND, ND, KH, JS, QD, AS, JC, JC, QH, JD}, 0, 0)
	p1 := createAI()
	p1.SetHand(nil, nil, nil, sdz.Hand{AD, KS, NH, TD, JD, QH, QC, AD, KD, TC, AS, AH}, 0, 1)
	p2 := createAI()
	p2.SetHand(nil, nil, nil, sdz.Hand{KS, NC, NS, AH, KC, AC, TH, TH, TS, KH, KC, QC}, 0, 2)
	p3 := createAI()
	p3.SetHand(nil, nil, nil, sdz.Hand{JS, JH, TC, JH, QS, NH, TD, KD, AC, NS, QD, TS}, 0, 3)
	p1Amt, p1Meld := p1.Hand().Meld(trump)
	p2Amt, p2Meld := p2.Hand().Meld(trump)
	p3Amt, p3Meld := p3.Hand().Meld(trump)
	p0.Tell(nil, nil, nil, sdz.CreateMeld(p1Meld, p1Amt, 1))
	p0.Tell(nil, nil, nil, sdz.CreateMeld(p2Meld, p2Amt, 2))
	p0.Tell(nil, nil, nil, sdz.CreateMeld(p3Meld, p3Amt, 3))
	//Log(0, "Starting")
	//Log(0, "PlayedCards=%s", p0.HT.PlayedCards)
	//Log(0, "Cards[0]=%s", p0.HT.Cards[0])
	//Log(0, "Cards[1]=%s", p0.HT.Cards[1])
	//Log(0, "Cards[2]=%s", p0.HT.Cards[2])
	//Log(0, "Cards[3]=%s", p0.HT.Cards[3])

	trick := new(Trick)
	//p0.HT.PlayCard(AD, 1, trick, trump)
	//p0.HT.PlayCard(NC, 2, trick, trump)
	//p0.HT.PlayCard(TD, 3, trick, trump)
	//p0.HT.PlayCard(ND, 0, trick, trump)
	//Log(0, "Trick1 = %s", trick)
	trick.reset()
	//p0.HT.PlayCard(AD, 1, trick, trump)
	//p0.HT.PlayCard(QC, 2, trick, trump)
	//p0.HT.PlayCard(KD, 3, trick, trump)
	//p0.HT.PlayCard(ND, 0, trick, trump)
	////Log(0, "Trick2 = %s", trick)
	//trick.reset()
	//p0.HT.PlayCard(AH, 1, trick, trump)
	//p0.HT.PlayCard(KH, 2, trick, trump)
	//p0.HT.PlayCard(NH, 3, trick, trump)
	//p0.HT.PlayCard(QH, 0, trick, trump)
	////Log(0, "Trick3 = %s", trick)
	//trick.reset()
	//p0.HT.PlayCard(QH, 1, trick, trump)
	//p0.HT.PlayCard(TH, 2, trick, trump)
	//p0.HT.PlayCard(JH, 3, trick, trump)
	//p0.HT.PlayCard(KH, 0, trick, trump)
	////Log(0, "Trick4 = %s", trick)
	//trick.reset()
	//p0.HT.PlayCard(AC, 2, trick, trump)
	//p0.HT.PlayCard(TC, 3, trick, trump)
	//p0.HT.PlayCard(NC, 0, trick, trump)
	//p0.HT.PlayCard(QC, 1, trick, trump)
	////Log(0, "Trick5 = %s", trick)
	//trick.reset()
	//p0.HT.PlayCard(KS, 2, trick, trump)
	//p0.HT.PlayCard(TS, 3, trick, trump)
	//p0.HT.PlayCard(AS, 0, trick, trump)
	//p0.HT.PlayCard(KS, 1, trick, trump)
	////Log(0, "Trick6 = %s", trick)
	//trick.reset()
	//p0.HT.PlayCard(JC, 0, trick, trump)
	//p0.HT.PlayCard(TC, 1, trick, trump)
	//p0.HT.PlayCard(KC, 2, trick, trump)
	//p0.HT.PlayCard(AC, 3, trick, trump)
	////Log(0, "Trick7 = %s", trick)
	//trick.reset()
	//p0.HT.PlayCard(JH, 3, trick, trump)
	//p0.HT.PlayCard(JD, 0, trick, trump)
	//p0.HT.PlayCard(NH, 1, trick, trump)
	//p0.HT.PlayCard(TH, 2, trick, trump)
	////Log(0, "Trick8 = %s", trick)
	//trick.reset()
	//p0.HT.PlayCard(JC, 0, trick, trump)
	//p0.HT.PlayCard(JD, 1, trick, trump)
	//p0.HT.PlayCard(KC, 2, trick, trump)
	//p0.HT.PlayCard(QD, 3, trick, trump)
	//Log(0, "Trick9 = %s", trick)
	//trick.reset()
	//p0.HT.PlayCard(QS, 3, trick, trump)
	//p0.HT.PlayCard(JS, 0, trick, trump)
	//p0.HT.PlayCard(AS, 1, trick, trump)
	//p0.HT.PlayCard(TS, 2, trick, trump)
	//Log(0, "Trick10 = %s", trick)
	//trick.reset()
	//p0.HT.PlayCard(TD, 1, trick, trump)
	//p0.HT.PlayCard(NS, 2, trick, trump)
	//p0.HT.PlayCard(NS, 3, trick, trump)
	//p0.HT.PlayCard(QD, 0, trick, trump)
	//Log(0, "Trick11 = %s", trick)
	//trick.reset()
	//p0.HT.PlayCard(KD, 1, trick, trump)
	//p0.HT.PlayCard(AH, 2, trick, trump)
	//p0.HT.PlayCard(JS, 3, trick, trump)
	//p0.HT.PlayCard(QS, 0, trick, trump)
	//trick.reset()
	action := sdz.CreatePlayRequest(sdz.NACard, sdz.NASuit, sdz.Hearts, 0, p0.Hand())
	b.ResetTimer()
	for x := 0; x < b.N; x++ {
		p0.findCardToPlay(action)
	}
}

func BenchmarkKnownCards(b *testing.B) {
	//func (ai *AI) findCardToPlay(action *sdz.Action) sdz.Card {
	p0 := createAI()
	p0.SetHand(nil, nil, nil, sdz.Hand{QS, NC, ND, ND, KH, JS, QD, AS, JC, JC, QH, JD}, 0, 0)
	p1 := createAI()
	p1.SetHand(nil, nil, nil, sdz.Hand{AD, KS, NH, TD, JD, QH, QC, AD, KD, TC, AS, AH}, 0, 1)
	p2 := createAI()
	p2.SetHand(nil, nil, nil, sdz.Hand{KS, NC, NS, AH, KC, AC, TH, TH, TS, KH, KC, QC}, 0, 2)
	p3 := createAI()
	p3.SetHand(nil, nil, nil, sdz.Hand{JS, JH, TC, JH, QS, NH, TD, KD, AC, NS, QD, TS}, 0, 3)
	p0.Tell(nil, nil, nil, sdz.CreateMeld(*p1.Hand(), 0, 1))
	p0.Tell(nil, nil, nil, sdz.CreateMeld(*p2.Hand(), 0, 2))
	p0.Tell(nil, nil, nil, sdz.CreateMeld(*p3.Hand(), 0, 3))
	action := sdz.CreatePlayRequest(sdz.NACard, sdz.NASuit, sdz.Hearts, 0, p0.Hand())
	b.ResetTimer()
	for x := 0; x < b.N; x++ {
		p0.findCardToPlay(action)
	}
}

func BenchmarkFullGame(b *testing.B) {
	c, err := appenginetesting.NewContext(&appenginetesting.Options{Debug: "critical"})
	if err != nil {
		b.Fatalf("Could not start up appenginetesting")
	}
	defer c.Close()
	b.ResetTimer()
	for y := 0; y < b.N; y++ {
		game := NewGame(4)
		for x := 0; x < len(game.Players); x++ {
			game.Players[x] = createAI()
		}
		game.NextHand(nil, c)
	}
	b.StopTimer() // stopping so not to count the defer call above
}

//func (t *testSuite) TestPotentialCardsShort() {
//	ht := new(HandTracker)
//	ht.reset(0)
//	pw := new(PlayWalker)

//	ht.PlayCount = 0
//	ht.Trick.Next = 3
//	ht.Trick.reset()
//	ht.PlayCard(KD, sdz.Spades)
//	pw.Hands = ht.Deal()
//	pw.Me = ht.Trick.Next
//	pw.PlayCount = ht.PlayCount
//	*pw.Trick = *ht.Trick

//	potentials = pw.potentialCards(pw.Trick, sdz.Spades)

//	t.True(potentials.Contains(TD))
//	t.False(potentials.Contains(KD))
//	t.False(potentials.Contains(TS))

//	ht.Cards[0][AS] = 1
//	//ht.Cards[0][TS] = None
//	ht.PlayCount = 0
//	ht.Trick.Next = 3
//	ht.Trick.reset()
//	ht.PlayCard(KH, sdz.Spades)
//	pw.Hands = ht.Deal()
//	pw.Me = ht.Trick.Next
//	pw.PlayCount = ht.PlayCount
//	*pw.Trick = *ht.Trick
//	potentials = pw.potentialCards(pw.Trick, sdz.Spades)
//	t.False(potentials.Contains(AH))
//	t.False(potentials.Contains(TH))
//	t.True(potentials.Contains(AS))
//	t.False(potentials.Contains(QH))
//	t.False(potentials.Contains(TS))
//	t.False(potentials.Contains(KD))
//	t.False(potentials.Contains(TD))

//	ht.Cards[0][AH] = 1
//	ht.PlayCount = 0
//	ht.Trick.Next = 3
//	ht.Trick.reset()
//	ht.PlayCard(KH, sdz.Spades)
//	pw.Hands = ht.Deal()
//	pw.Me = ht.Trick.Next
//	pw.PlayCount = ht.PlayCount
//	*pw.Trick = *ht.Trick

//	potentials = pw.potentialCards(pw.Trick, sdz.Spades)
//	t.True(potentials.Contains(AH))
//	t.False(potentials.Contains(TH))
//	t.False(potentials.Contains(AS))
//	t.False(potentials.Contains(QH))
//	t.False(potentials.Contains(TS))

//	ht.reset(0)
//	ht.Cards[0][AD] = 2
//	ht.Cards[0][TD] = 1
//	ht.Cards[0][JD] = 1
//	ht.Cards[0][TC] = 1
//	ht.Cards[0][KC] = 1
//	ht.Cards[0][QC] = 1
//	ht.Cards[0][TH] = 1
//	ht.Cards[0][JH] = 1
//	ht.Cards[0][NH] = 1
//	ht.Cards[0][KS] = 1
//	ht.Cards[0][QS] = 1
//	ht.PlayCount = 0
//	ht.Trick.Next = 0
//	ht.Trick.reset()
//	pw.Hands = ht.Deal()
//	pw.Me = ht.Trick.Next
//	pw.PlayCount = ht.PlayCount
//	*pw.Trick = *ht.Trick

//	potentials = pw.potentialCards(pw.Trick, sdz.Hearts)
//	t.Equal(11, len(potentials))

//	ht.reset(0)
//	ht.Cards[0][TD] = 2
//	ht.Cards[0][ND] = 2
//	ht.Cards[0][QD] = 1
//	ht.Cards[0][JD] = 1
//	ht.Cards[0][JS] = 1
//	ht.PlayCount = 0
//	ht.Trick.Next = 3
//	ht.PlayCard(JS, sdz.Clubs)
//	pw.Hands = ht.Deal()
//	pw.Me = ht.Trick.Next
//	pw.PlayCount = ht.PlayCount
//	*pw.Trick = *ht.Trick
//	potentials = pw.potentialCards(pw.Trick, sdz.Clubs)
//	t.Equal(1, len(potentials))
//	t.True(potentials.Contains(JS))
//	t.False(potentials.Contains(TD))

//	ht.reset(0)
//	ht.Cards[0][TD] = 1
//	ht.Cards[0][KD] = 1
//	ht.Cards[0][QD] = 1
//	ht.Cards[0][JD] = 1
//	// follow suit and lose
//	ht.PlayCount = 0
//	ht.Trick.Next = 3
//	ht.PlayCard(AD, sdz.Spades)
//	pw.Hands = ht.Deal()
//	pw.Me = ht.Trick.Next
//	pw.PlayCount = ht.PlayCount
//	*pw.Trick = *ht.Trick
//	potentials = pw.potentialCards(pw.Trick, sdz.Spades)
//	t.False(potentials.Contains(TD))
//	t.True(potentials.Contains(KD))
//	t.False(potentials.Contains(QD))
//	t.True(potentials.Contains(JD))

//	// play trump and lose
//	ht.PlayCount = 0
//	ht.Trick.Next = 3
//	ht.PlayCard(AD, sdz.Diamonds)
//	pw.Hands = ht.Deal()
//	pw.Me = ht.Trick.Next
//	pw.PlayCount = ht.PlayCount
//	*pw.Trick = *ht.Trick
//	potentials = pw.potentialCards(pw.Trick, sdz.Diamonds)
//	t.False(potentials.Contains(AD))
//	t.False(potentials.Contains(TD))
//	t.True(potentials.Contains(KD))
//	t.False(potentials.Contains(QD))
//	t.True(potentials.Contains(JD))

//	// follow suit and win
//	ht.PlayCount = 0
//	ht.Trick.Next = 3
//	ht.Trick.reset()
//	ht.PlayCard(ND, sdz.Spades)
//	pw.Hands = ht.Deal()
//	pw.Me = ht.Trick.Next
//	pw.PlayCount = ht.PlayCount
//	*pw.Trick = *ht.Trick
//	potentials = pw.potentialCards(pw.Trick, sdz.Spades)
//	t.True(potentials.Contains(TD))
//	t.True(potentials.Contains(KD))
//	t.True(potentials.Contains(QD))
//	t.True(potentials.Contains(JD))

//	// play trump and win
//	ht.PlayCount = 0
//	ht.Trick.Next = 3
//	ht.Trick.reset()
//	ht.PlayCard(ND, sdz.Diamonds)
//	pw.Hands = ht.Deal()
//	pw.Me = ht.Trick.Next
//	pw.PlayCount = ht.PlayCount
//	*pw.Trick = *ht.Trick
//	potentials = pw.potentialCards(pw.Trick, sdz.Diamonds)
//	t.True(potentials.Contains(TD))
//	t.True(potentials.Contains(KD))
//	t.True(potentials.Contains(QD))
//	t.True(potentials.Contains(JD))

//	// now we do the 4 blocks over again, but this time it's the last play
//	ht.reset(0)
//	ht.Cards[0][TD] = 1
//	ht.Cards[0][KD] = 1
//	ht.Cards[0][QD] = 1
//	ht.Cards[0][JD] = 1
//	// follow suit and lose
//	ht.PlayCount = 2
//	ht.Trick.Next = 3
//	ht.Trick.reset()
//	ht.PlayCard(AD, sdz.Spades)
//	pw.Hands = ht.Deal()
//	pw.Me = ht.Trick.Next
//	pw.PlayCount = ht.PlayCount
//	*pw.Trick = *ht.Trick
//	potentials = pw.potentialCards(pw.Trick, sdz.Spades)
//	t.False(potentials.Contains(AD))
//	t.False(potentials.Contains(TD))
//	t.True(potentials.Contains(KD))
//	t.False(potentials.Contains(QD))
//	t.True(potentials.Contains(JD))

//	// play trump and lose
//	ht.PlayCount = 1
//	ht.Trick.Next = 2
//	ht.Trick.reset()
//	ht.PlayCard(JS, sdz.Diamonds)
//	ht.PlayCard(AD, sdz.Diamonds)
//	pw.Hands = ht.Deal()
//	pw.Me = ht.Trick.Next
//	pw.PlayCount = ht.PlayCount
//	*pw.Trick = *ht.Trick

//	potentials = pw.potentialCards(pw.Trick, sdz.Diamonds)
//	t.False(potentials.Contains(AD))
//	t.False(potentials.Contains(TD))
//	t.True(potentials.Contains(KD))
//	t.False(potentials.Contains(QD))
//	t.True(potentials.Contains(JD))

//	// follow suit and win
//	ht.PlayCount = 2
//	ht.Trick.Next = 3
//	ht.Trick.reset()
//	ht.PlayCard(ND, sdz.Spades)
//	pw.Hands = ht.Deal()
//	pw.Me = ht.Trick.Next
//	pw.PlayCount = ht.PlayCount
//	*pw.Trick = *ht.Trick

//	potentials = pw.potentialCards(pw.Trick, sdz.Spades)
//	t.False(potentials.Contains(AD))
//	t.False(potentials.Contains(TD))
//	t.True(potentials.Contains(KD))
//	t.False(potentials.Contains(QD))
//	t.True(potentials.Contains(JD))

//	// play trump and win
//	ht.PlayCount = 1
//	ht.Trick.Next = 2
//	ht.Trick.reset()
//	ht.PlayCard(JS, sdz.Diamonds)
//	ht.PlayCard(ND, sdz.Diamonds)
//	pw.Hands = ht.Deal()
//	pw.Me = ht.Trick.Next
//	pw.PlayCount = ht.PlayCount
//	*pw.Trick = *ht.Trick

//	potentials = pw.potentialCards(pw.Trick, sdz.Diamonds)
//	t.False(potentials.Contains(AD))
//	t.False(potentials.Contains(TD))
//	t.True(potentials.Contains(KD))
//	t.False(potentials.Contains(QD))
//	t.True(potentials.Contains(JD))
//}

//func convert(oldMap map[sdz.Card]int) CardMap {
//	var cm CardMap
//	for card, val := range oldMap {
//		cm[card] = val
//	}
//	return cm
//}

//func (t *testSuite) TestPlayHandWithCard() {
//	//func playHandWithCard(playerid int, ht *HandTracker, trick *Trick, trump sdz.Suit) (sdz.Card, [2]int) {
//	ht := getHT(0)
//	for x := 0; x < len(ht.Cards); x++ {
//		for card := 0; card < sdz.AllCards; card++ {
//			ht.Cards[x][card] = 3
//		}
//	}
//	ht.PlayedCards = convert(map[sdz.Card]int{"AD": 0, "TD": 0, "KD": 0, "QD": 0, "JD": 2, "ND": 2, "AS": 2, "TS": 2, "KS": 2, "QS": 2, "JS": 2, "NS": 2, "AH": 2, "TH": 2, "KH": 2, "QH": 2, "JH": 2, "NH": 2, "AC": 2, "TC": 2, "KC": 2, "QC": 2, "JC": 2, "NC": 2})
//	ht.Cards[0][AD] = 1
//	ht.Cards[1][TD] = 1
//	ht.Cards[2][KD] = 1
//	ht.Cards[3][QD] = 1

//	before := len(ht.Cards[0])
//	card, value := playHandWithCard(0, ht, NewTrick(), sdz.Diamonds)
//	t.Equal(before, len(ht.Cards[0]))
//	t.Equal(card, AD)
//	t.Equal(value, 4)

//	ht.Cards[1][AD] = 1
//	ht.Cards[2][TD] = 1
//	ht.Cards[3][KD] = 1
//	ht.Cards[0][QD] = 1

//	card, value = playHandWithCard(0, ht, NewTrick(), sdz.Diamonds)
//	t.Equal(card, AD)
//	t.Equal(value, 3)

//	ht.Cards[1][AD] = 1
//	ht.Cards[2][TD] = 1
//	ht.Cards[3][KD] = 1
//	ht.Cards[0][QD] = 1

//	ht.PlayedCards[AS] = 1
//	ht.Cards[0][AS] = 1
//	ht.PlayedCards[TS] = 1
//	ht.Cards[1][TS] = 1
//	ht.PlayedCards[QS] = 1
//	ht.Cards[2][QS] = 1
//	ht.PlayedCards[JS] = 1
//	ht.Cards[3][JS] = 1

//	card, value = playHandWithCard(0, ht, NewTrick(), sdz.Diamonds)
//	t.Equal(card, AD)
//	t.Equal(value, 6)

//	ht.PlayedCards[AC] = None
//	ht.PlayedCards[TC] = None

//	ht.Cards[0][AC] = 1
//	ht.Cards[1][AC] = Unknown
//	ht.Cards[2][AC] = Unknown
//	ht.Cards[3][AC] = Unknown
//	ht.Cards[1][TC] = Unknown
//	ht.Cards[2][TC] = Unknown
//	ht.Cards[3][TC] = Unknown

//	card, value = playHandWithCard(0, ht, NewTrick(), sdz.Diamonds)
//	t.Equal(card, AC)
//	t.Equal(value, 10)
//	HTs <- ht
//}

func (t *testSuite) TestHandTrackerDeal() {
	ai := createAI()
	ai.SetHand(nil, nil, nil, sdz.Hand{TD, TD, QD, TC, QC, AH, AH, KH, NH, TS, KS, QS}, 0, 0)

	result := ai.HT.Deal()
	t.True(result[0].Contains(TD))
	t.True(result[0].Contains(QD))
	t.True(result[0].Contains(TC))
	t.True(result[0].Contains(QC))
	t.True(result[0].Contains(AH))
	t.True(result[0].Contains(KH))
	t.True(result[0].Contains(NH))
	t.True(result[0].Contains(TS))
	t.True(result[0].Contains(KS))
	t.True(result[0].Contains(QS))
	t.False(result[0].Contains(JD))
	ai.HT.Cards[3][AC] = 2
	ai.HT.Cards[3][KC] = 1
	ai.HT.Cards[3][JC] = 2
	ai.HT.Cards[3][TH] = 1
	ai.HT.Cards[3][QH] = 2
	ai.HT.Cards[3][JH] = 1
	ai.HT.Cards[3][AS] = 1
	ai.HT.Cards[3][TS] = 1
	ai.HT.Cards[3][QS] = None
	ai.HT.Cards[3][JS] = None
	ai.HT.Cards[3][NS] = None
	ai.HT.Cards[3][KH] = None
	ai.HT.Cards[3][NH] = None
	ai.HT.Cards[3][TC] = None
	ai.HT.Cards[3][QC] = None
	ai.HT.Cards[3][NC] = None
	ai.HT.Cards[3][AD] = None
	ai.HT.Cards[3][KD] = None
	ai.HT.Cards[3][QD] = None
	ai.HT.Cards[3][JD] = None
	ai.HT.Cards[3][ND] = None
	//ai.HT.Cards[3][KS] = 1 - he should have this one as this is the only "unknown" for him
	for card := 0; card < sdz.AllCards; card++ {
		ai.HT.calculateCard(sdz.Card(card))
	}
	Log(ai.HT.Owner, "bah")
	ai.HT.Debug()
	result = ai.HT.Deal()
	t.True(result[0].Contains(TD))
	t.True(result[0].Contains(QD))
	t.True(result[0].Contains(TC))
	t.True(result[0].Contains(QC))
	t.True(result[0].Contains(AH))
	t.True(result[0].Contains(KH))
	t.True(result[0].Contains(NH))
	t.True(result[0].Contains(TS))
	t.True(result[0].Contains(KS))
	t.True(result[0].Contains(QS))
	t.False(result[0].Contains(JD))

	//t.True(result[3].Contains(KS)) // can't test this for truthfulness until I differentiate between 1 and has at least 1 in HandTracker
	t.True(result[3].Contains(AC))
	t.True(result[3].Contains(KC))
	t.True(result[3].Contains(JC))
	t.True(result[3].Contains(TH))
	t.True(result[3].Contains(QH))
	t.True(result[3].Contains(JH))
	t.True(result[3].Contains(AS))
	t.True(result[3].Contains(TS))

	ai.SetHand(nil, nil, nil, sdz.Hand{AD, AD, TD, JD, TC, KC, QC, TH, JH, NH, KS, QS}, 0, 3)
	ai.Trump = sdz.Diamonds
	ai.HT.Cards[0][KH] = 1
	ai.HT.Cards[0][QH] = 1
	ai.HT.Cards[1][NH] = 1
	ai.HT.Cards[2][KD] = 1
	ai.HT.Cards[2][QD] = 1

	result = ai.HT.Deal()
	t.True(result[3].Contains(AD))
	t.True(result[3].Contains(TD))
	t.True(result[3].Contains(JD))
	t.True(result[3].Contains(TC))
	t.True(result[3].Contains(KC))
	t.True(result[3].Contains(QC))
	t.True(result[3].Contains(TH))
	t.True(result[3].Contains(JH))
	t.True(result[3].Contains(NH))
	t.True(result[3].Contains(KS))
	t.True(result[3].Contains(QS))
	t.False(result[3].Contains(QD))

	t.True(result[0].Contains(KH))
	t.True(result[0].Contains(QH))
	t.True(result[1].Contains(NH))
	t.True(result[2].Contains(KD))
	t.True(result[2].Contains(QD))
}

func (t *testSuite) TestFindCardToPlayShort() {
	//func (ai *AI) findCardToPlay(action *sdz.Action) sdz.Card {
	ai := createAI()
	ai.SetHand(nil, nil, nil, sdz.Hand{AD, QS}, 0, 3)
	for card := range ai.HT.PlayedCards {
		ai.HT.PlayedCards[card] = 2
	}
	ai.HT.PlayCount = 48 - 8 // 8 cards have not been played according to below
	trump := sdz.Hearts
	ai.HT.PlayedCards[AD] = 1
	ai.HT.PlayedCards[QS] = 1
	ai.HT.PlayedCards[KH] = 1
	ai.HT.PlayedCards[QH] = 1
	ai.HT.PlayedCards[NH] = None
	ai.HT.PlayedCards[KD] = 1
	ai.HT.PlayedCards[QD] = 1
	for x := 0; x < sdz.AllCards; x++ {
		ai.HT.calculateCard(sdz.Card(x))
	}
	ai.HT.Trick.Next = 3
	action := sdz.CreatePlayRequest(sdz.NACard, sdz.NASuit, trump, 3, ai.Hand())
	card := ai.findCardToPlay(action)
	t.True(card == AD || card == QS)
}

func (t *testSuite) TestFindCardToPlayLong() {
	//func (ai *AI) findCardToPlay(action *sdz.Action) sdz.Card {
	ai := createAI()
	ai.SetHand(nil, nil, nil, sdz.Hand{AD, AD, TD, JD, TC, KC, QC, TH, JH, NH, KS, QS}, 0, 3)
	ai.Trump = sdz.Diamonds
	ai.HT.Cards[0][KH] = 1
	ai.HT.Cards[0][QH] = 1
	ai.HT.Cards[1][NH] = 1
	ai.HT.Cards[2][KD] = 1
	ai.HT.Cards[2][QD] = 1
	for x := 0; x < sdz.AllCards; x++ {
		ai.HT.calculateCard(sdz.Card(x))
	}
	action := sdz.CreatePlayRequest(sdz.NACard, sdz.NASuit, ai.Trump, 3, ai.Hand())
	switch ai.findCardToPlay(action) {
	case AD:
		fallthrough
	case JD:
		t.MustFail()
	default:
		// all other plays are acceptable depending on strategy
		t.True(true)
	}
}

func (t *testSuite) TestGame() {
	c, err := appenginetesting.NewContext(&appenginetesting.Options{Debug: "debug"})
	if err != nil {
		t.Error("Could not start appenginetesting")
		return
	}
	defer c.Close()
	if c == nil {
		t.Error("Could not start testing framework")
		return
	}
	g := goon.FromContext(c)
	game := NewGame(4)
	game.Dealer = 0
	game.Players[1] = createAI()
	game.Players[1].SetHand(g, c, game, sdz.Hand{KD, QD, JD, JD, ND, TC, KC, QC, KH, NH, QS, NS}, 0, 1)
	game.Players[2] = createAI()
	game.Players[2].SetHand(g, c, game, sdz.Hand{AD, AD, KD, ND, NC, NC, TH, JH, AS, JS, JS, NS}, 0, 2)
	game.Players[3] = createAI()
	game.Players[3].SetHand(g, c, game, sdz.Hand{AC, AC, KC, JC, JC, TH, QH, QH, JH, AS, TS, KS}, 0, 3)
	game.Players[0] = createAI()
	game.Players[0].SetHand(g, c, game, sdz.Hand{TD, TD, QD, TC, QC, AH, AH, KH, NH, TS, KS, QS}, 0, 0)
	game.Meld = make([]uint8, len(game.Players)/2)
	game.Trick = Trick{}
	game.CountMeld = make([]bool, len(game.Players)/2)
	game.Counters = make([]uint8, len(game.Players)/2)
	game.HighBid = 20
	game.HighPlayer = game.Dealer
	game.State = StateBid
	game.Next = game.Dealer
	//oright = game.Players[0].(*AI).HT
	//Log(oright.Owner, "Start of game hands")
	//oright.Debug()
	game.inc() // so dealer's not the first to bid

	game.processAction(g, c, nil, game.Players[game.Next].Tell(nil, nil, game, sdz.CreateBid(0, game.Next)))
	t.True(true) // just getting to the end successfully counts!
}

func (t *testSuite) TestPlayCard() {
	trump := sdz.Suit(sdz.Diamonds)
	p0 := createAI()
	p0.SetHand(nil, nil, nil, sdz.Hand{QS, NC, ND, ND, KH, JS, QD, AS, JC, JC, QH, JD}, 0, 0)
	p1 := createAI()
	p1.SetHand(nil, nil, nil, sdz.Hand{AD, KS, NH, TD, JD, QH, QC, AD, KD, TC, AS, AH}, 0, 1)
	p2 := createAI()
	p2.SetHand(nil, nil, nil, sdz.Hand{KS, NC, NS, AH, KC, AC, TH, TH, TS, KH, KC, QC}, 0, 2)
	p3 := createAI()
	p3.SetHand(nil, nil, nil, sdz.Hand{JS, JH, TC, JH, QS, NH, TD, KD, AC, NS, QD, TS}, 0, 3)
	p1Amt, p1Meld := p1.Hand().Meld(trump)
	p2Amt, p2Meld := p2.Hand().Meld(trump)
	p3Amt, p3Meld := p3.Hand().Meld(trump)
	p0.Tell(nil, nil, nil, sdz.CreateMeld(p1Meld, p1Amt, 1))
	p0.Tell(nil, nil, nil, sdz.CreateMeld(p2Meld, p2Amt, 2))
	p0.Tell(nil, nil, nil, sdz.CreateMeld(p3Meld, p3Amt, 3))

	p0.HT.Trick.Next = 2
	t.True(p0.HT.Cards[2][TH] == Unknown)
	t.True(p0.HT.PlayedCards[TH] == None)
	p0.HT.PlayCard(TH, trump)
	t.True(p0.HT.Cards[2][TH] == Unknown)
	t.True(p0.HT.PlayedCards[TH] == 1)
	p0.HT.Trick.reset()
	p0.HT.Trick.Next = 2
	p0.HT.PlayCard(TH, trump)
	t.True(p0.HT.Cards[3][TH] == None)
	t.True(p0.HT.Cards[2][TH] == None)
	t.True(p0.HT.Cards[1][TH] == None)
	t.True(p0.HT.Cards[0][TH] == None)
	t.True(p0.HT.PlayedCards[TH] == 2)
}

func (t *testSuite) TestFindCardToPlayFull() {
	trump := sdz.Suit(sdz.Diamonds)
	p0 := createAI()
	p1 := createAI()
	p2 := createAI()
	p3 := createAI()
	p0.SetHand(nil, nil, nil, sdz.Hand{TD, TD, QD, TC, QC, AH, AH, KH, NH, TS, KS, QS}, 0, 0)
	p1.SetHand(nil, nil, nil, sdz.Hand{KD, QD, JD, JD, ND, TC, KC, QC, KH, NH, QS, NS}, 0, 1)
	p2.SetHand(nil, nil, nil, sdz.Hand{AD, AD, KD, ND, NC, NC, TH, JH, AS, JS, JS, NS}, 0, 2)
	p3.SetHand(nil, nil, nil, sdz.Hand{AC, AC, KC, JC, JC, TH, QH, QH, JH, AS, TS, KS}, 0, 3)
	p1Amt, p1Meld := p1.Hand().Meld(trump)
	p2Amt, p2Meld := p2.Hand().Meld(trump)
	p3Amt, p3Meld := p3.Hand().Meld(trump)
	p0.Tell(nil, nil, nil, sdz.CreateMeld(p1Meld, p1Amt, 1))
	p0.Tell(nil, nil, nil, sdz.CreateMeld(p2Meld, p2Amt, 2))
	p0.Tell(nil, nil, nil, sdz.CreateMeld(p3Meld, p3Amt, 3))
	p0.HT.Trick.Next = 2
	p0.HT.PlayCard(AD, trump)
	p0.HT.PlayCard(JC, trump)
	t.True(playHandWithCard(p0.HT, trump) == TD)
	p0.HT.PlayCard(TD, trump)

}

func (t *testSuite) TestAITracking() {
	ai := createAI()
	hand := sdz.Hand{ND, ND, QD, TD, TD, AD, JC, QC, KC, AH, AH, KS}
	ai.SetHand(nil, nil, nil, hand, 0, 0)
	ai.Trump = sdz.Spades
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{JD, QS, KD, QD}, 6, 1))
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{JD, QS}, 4, 2))
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{}, 0, 3))

	t.Equal(uint8(1), ai.HT.Cards[1][JD])
	t.Equal(uint8(1), ai.HT.Cards[1][QS])
	t.Equal(uint8(1), ai.HT.Cards[2][JD])
	t.Equal(uint8(1), ai.HT.Cards[2][QS])
	t.Equal(None, ai.HT.Cards[3][QS])
	t.Equal(None, ai.HT.PlayedCards[JD])
	t.Equal(None, ai.HT.PlayedCards[QS])
	t.Equal(None, ai.HT.PlayedCards[QD])
	t.Equal(uint8(1), ai.HT.Cards[1][QD])
	t.Equal(None, ai.HT.Cards[2][QD])
	t.Equal(None, ai.HT.Cards[3][QD])
	t.Equal(uint8(1), ai.HT.Cards[1][KD])

	ai.Tell(nil, nil, nil, sdz.CreatePlay(JD, 1))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(KD, 2))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(AD, 3))
	ai.Tell(nil, nil, nil, sdz.CreateTrick(0))
	val := ai.HT.Cards[1][JD]
	t.Equal(None, val)

	val = ai.HT.Cards[1][JD]
	t.Equal(None, val)
	t.Equal(uint8(1), ai.HT.Cards[1][QS])
	t.Equal(uint8(1), ai.HT.Cards[2][JD])
	t.Equal(uint8(1), ai.HT.Cards[2][QS])
	t.Equal(None, ai.HT.Cards[3][QS])
	t.Equal(uint8(1), ai.HT.PlayedCards[JD])
	t.Equal(None, ai.HT.PlayedCards[QS])
	t.Equal(uint8(1), ai.HT.PlayedCards[KD])
	t.Equal(uint8(1), ai.HT.PlayedCards[AD])

	ai.Tell(nil, nil, nil, sdz.CreateTrick(0))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(QD, 1))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(NH, 2))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(NH, 3))

	val = ai.HT.Cards[1][QD]
	t.Equal(None, val)

	ai = createAI()
	hand = sdz.Hand{ND, ND, QD, TD, TD, AS, JC, QC, KC, AH, AH, KS}
	ai.SetHand(nil, nil, nil, hand, 0, 0)
	ai.Trump = sdz.Spades
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{}, 0, 0))
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{}, 0, 1))
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{}, 0, 2))
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{}, 0, 3))

	ai.Tell(nil, nil, nil, sdz.CreateTrick(0))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(JD, 1))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(QD, 2))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(KD, 3))
	t.Equal(ai.HT.Cards[2][QD], None)
	t.Equal(ai.HT.Cards[3][QD], None)
	t.Equal(ai.HT.Cards[1][QD], None)
	t.Equal(ai.HT.PlayedCards[QD], uint8(1))
	t.Equal(ai.HT.Cards[0][QD], uint8(1))

	play := ai.Tell(nil, nil, nil, sdz.CreatePlayRequest(ai.HT.Trick.winningCard(), ai.HT.Trick.leadSuit(), ai.Trump, ai.PlayerID(), &sdz.Hand{}))
	t.Equal(sdz.Card(TD), play.PlayedCard)
	ai.Tell(nil, nil, nil, sdz.CreateTrick(0))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(JD, 1))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(KD, 2))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(KH, 3))
	play = ai.Tell(nil, nil, nil, sdz.CreatePlayRequest(ai.HT.Trick.winningCard(), ai.HT.Trick.leadSuit(), ai.Trump, ai.PlayerID(), &sdz.Hand{}))
	t.Equal(sdz.Card(TD), play.PlayedCard)

	ai = createAI()
	hand = sdz.Hand{ND, ND, QD, TD, TD, AD, JC, QC, KC, AH, AH, KS}
	ai.SetHand(nil, nil, nil, hand, 0, 0)
	ai.Trump = sdz.Spades
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{JD, JD, QS, QS, KD, QD}, 32, 1))
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{}, 0, 2))
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{}, 0, 3))
	//ai.calculate()
	t.Equal(None, ai.HT.Cards[0][JD])
	t.Equal(uint8(2), ai.HT.Cards[1][JD])
	t.Equal(None, ai.HT.Cards[2][JD])
	t.Equal(None, ai.HT.Cards[3][JD])
}

func (t *testSuite) TestNoSuitShort() {
	ht := new(HandTracker)
	ht.reset(0)
	ht.Cards[1][ND] = 1
	ht.noSuit(1, sdz.Diamonds)

	t.Equal(None, ht.Cards[1][ND])
	t.Equal(None, ht.Cards[1][JD])
	t.Equal(None, ht.Cards[1][QD])
	t.Equal(None, ht.Cards[1][KD])
	t.Equal(None, ht.Cards[1][TD])
	t.Equal(None, ht.Cards[1][AD])
	t.Equal(Unknown, ht.Cards[1][AS])
}

func (t *testSuite) TestPlayWalkerStringShort() {
	trump := sdz.Suit(sdz.Diamonds)
	ht := new(HandTracker)
	ht.reset(0)
	ht.Cards[0][AS] = 1
	ht.Cards[0][TH] = 1
	ht.Cards[0][KS] = 0
	ht.Cards[0][QH] = 1
	ht.Cards[0][JS] = 2
	ht.Cards[0][NH] = 1
	ht.Cards[0][AD] = 1
	ht.Cards[0][TC] = 1
	ht.Cards[0][KD] = 1
	ht.Cards[0][QC] = 1
	ht.Cards[0][JD] = 1
	ht.Cards[0][NC] = 1
	for x := 0; x < sdz.AllCards; x++ {
		ht.calculateCard(sdz.Card(x))
	}
	type ts struct {
		Cards  []sdz.Card
		Result []string
	}
	tests := []ts{
		ts{
			[]sdz.Card{AS, TS, KS, QS},
			[]string{"-lwAS---- ", "-lwAS-TS--- ", "-lwAS-TS-KS-- ", "-lwAS-TS-KS-QS- "},
		},
		ts{
			[]sdz.Card{JS, NS, AS, TS},
			[]string{"-lwAS-TS-KS-QS- -lwJS---- ", "-lwAS-TS-KS-QS- -lwJS-9S--- ", "-lwAS-TS-KS-QS- -lJS-9S-wAS-- ", "-lwAS-TS-KS-QS- -lJS-9S-wAS-TS- "},
		},
		ts{
			[]sdz.Card{KS, QS, JS, NS},
			[]string{"-lwAS-TS-KS-QS- -lJS-9S-wAS-TS- ---lwKS-- ", "-lwAS-TS-KS-QS- -lJS-9S-wAS-TS- ---lwKS-QS- ", "-lwAS-TS-KS-QS- -lJS-9S-wAS-TS- -JS--lwKS-QS- ", "-lwAS-TS-KS-QS- -lJS-9S-wAS-TS- -JS-9S-lwKS-QS- "},
		},
	}
	ht.Trick = new(Trick)
	pw := &PlayWalker{
		Hands: ht.Deal(),
		Card:  sdz.NACard,
		Trick: new(Trick),
	}
	t.Equal("----- ", pw.PlayTrail())
	for x := range tests {
		for y := range tests[x].Cards {
			//Log(4, "Card = %s", tests[x].Cards[y])
			pw.Children = []*PlayWalker{&PlayWalker{
				Card:   tests[x].Cards[y],
				Parent: pw,
				Hands:  pw.Hands,
				Trick:  new(Trick),
			}}
			*pw.Children[0].Trick = *pw.Trick // copy the trick
			pw = pw.Children[0]
			pw.Hands[pw.Trick.Next] = pw.Hands[pw.Trick.Next].Copy()
			pw.Hands[pw.Trick.Next].Remove(pw.Card)
			pw.Trick.PlayCard(pw.Card, trump)
			pw.PlayCount++
			//Log(ht.Owner, "Created trick %s", pw.Trick)
			t.Equal(tests[x].Result[y], pw.PlayTrail())
		}
	}
}

func (t *testSuite) TestCalculateShort() {
	hand := sdz.Hand{ND, ND, QD, TD, TD, AD, JC, QC, KC, AH, AH, KS}
	sort.Sort(hand)
	ai := createAI()
	// dealer 0, playerid 1
	ai.SetHand(nil, nil, nil, hand, 0, 1)
	for x := 0; x < 4; x++ {
		if x == 1 {
			t.Equal(ai.HT.Cards[x][ND], uint8(2))
		} else {
			t.Equal(ai.HT.Cards[x][ND], None)
		}
		val := ai.HT.Cards[x][QD]
		if x == 1 {
			t.Equal(val, uint8(1))
		} else {
			t.Equal(val, Unknown)
		}
		val = ai.HT.Cards[x][KH]
		if x == 1 {
			t.Equal(val, None)
		} else {
			t.Equal(val, Unknown)
		}
	}
	ai.HT.Cards[2][QD] = 1
	ai.HT.Cards[2][KH] = 1
	ai.HT.Cards[3][KH] = 1
	ai.HT.Cards[3][KD] = 1
	ai.HT.Cards[0][NS] = 1
	ai.HT.PlayedCards[NS] = 1
	ai.HT.PlayedCards[TS] = 2
	ai.HT.PlayedCards[JS] = 1
	ai.HT.Cards[0][JS] = None
	ai.HT.Cards[2][JS] = None
	ai.HT.PlayedCards[QS] = None
	ai.HT.Cards[0][QS] = None
	ai.HT.Cards[1][QS] = None
	ai.HT.Cards[2][QS] = None

	for _, card := range []sdz.Card{QD, KH, KD, NS, TS, JS, QS} {
		ai.HT.calculateCard(card)
	}

	//t.Equal(1, ai.HT.Cards[3][JS]) // Need to keep track of "have at least 1" status before this test will pass again, see HandTracker.calculateCard()
	//t.Equal(2, ai.HT.Cards[3][QS])

	for x := 0; x < 4; x++ {
		val := ai.HT.Cards[x][QD]
		t.Not(t.Equal(val, Unknown))
		val = ai.HT.Cards[x][KH]
		t.Not(t.Equal(val, Unknown))
		val = ai.HT.Cards[x][KD]
		if x == 3 || x == 1 {
			t.Not(t.Equal(val, Unknown))
		} else {
			t.Equal(val, Unknown)
		}
		val = ai.HT.Cards[x][NS]
		t.Not(t.Equal(val, Unknown))
		val = ai.HT.Cards[x][TS]
		t.Not(t.Equal(val, Unknown))
	}

	ai.HT.PlayedCards[JD] = None
	ai.HT.Cards[0][JD] = 1
	ai.HT.Cards[1][JD] = None
	ai.HT.Cards[2][JD] = None
	ai.HT.calculateCard(JD)
	val := ai.HT.Cards[0][JD]
	t.Equal(uint8(1), val)
	val = ai.HT.Cards[1][JD]
	t.Equal(None, val)
	val = ai.HT.Cards[2][JD]
	t.Equal(None, val)
	//val = ai.HT.Cards[3][JD]  // Need to keep track of "have at least 1" status before this test will pass again, see HandTracker.calculateCard()
	//t.Equal(val, 1)

	//ai.HT.PlayedCards[JC] = None
	//ai.HT.Cards[0][JC] = None
	//ai.HT.Cards[1][JC] = None
	//ai.HT.Cards[2][JC] = None
	//ai.HT.Cards[3][JC] = 1
	//ai.HT.calculateCard(JC)
	//t.Equal(2, ai.HT.Cards[3][JC])

	t.Equal(ai.HT.Cards[2][KS], Unknown)
	t.Equal(ai.HT.Cards[1][KS], uint8(1))
	ai.HT.Trick.Next = 2
	ai.HT.PlayCard(KS, sdz.Spades)
	t.Equal(ai.HT.Cards[1][KS], uint8(1))
	t.Equal(ai.HT.Cards[2][KS], None)
}
