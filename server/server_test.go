// server_test.go
package main

import (
	sdz "github.com/mzimmerman/sdzpinochle"
	pt "github.com/remogatto/prettytest"
	"sort"
	"strconv"
	"testing"
)

func TestFoo(t *testing.T) {
	pt.RunWithFormatter(
		t,
		new(pt.TDDFormatter),
		new(testSuite),
	)
}

func C(c string) sdz.Card {
	return sdz.Card(c)
}

type testSuite struct {
	pt.Suite
}

func (t *testSuite) TestRemove() {
	hand := sdz.Hand{C("JD"), C("QD"), C("KD"), C("AD"), C("TD"), C("JD"), C("QS"), C("QS"), C("KS"), C("AS"), C("TS"), C("JS")}
	sort.Sort(hand)
	ai := createAI()
	ai.SetHand(hand, 0, 0)
	t.Equal(12, len(*ai.Hand()))
	t.True(ai.Hand().Remove(C("JD")))
	t.True(ai.Hand().Remove(C("JD")))
	t.Not(ai.Hand().Remove(C("9D")))
	t.Equal(10, len(*ai.Hand()))
}

func (t *testSuite) TestBidding() {
	// 9D 9D QD TD TD AD JC QC KC AH AH KS
	hand := sdz.Hand{C("9D"), C("9D"), C("QD"), C("TD"), C("TD"), C("AD"), C("JC"), C("QC"), C("KC"), C("AH"), C("AH"), C("KS")}
	sort.Sort(hand)
	ai := createAI()
	ai.SetHand(hand, 0, 1)
	go ai.Go()
	ai.Tell(sdz.CreateBid(0, 1))
	action, _ := ai.Listen()
	t.Not(22 > action.Bid || action.Bid > 24)
}

func (t *testSuite) TestTracking() {
	hand := sdz.Hand{C("9D"), C("9D"), C("QD"), C("TD"), C("TD"), C("AD"), C("JC"), C("QC"), C("KC"), C("AH"), C("AH"), C("KS")}
	sort.Sort(hand)
	ai := createAI()
	ai.SetHand(hand, 0, 1)
	for x := 0; x < 4; x++ {
		if x == 1 {
			t.Equal(ai.ht.cards[x][C("9D")], 2)
		} else {
			t.Equal(ai.ht.cards[x][C("9D")], 0)
		}
		_, ok := ai.ht.cards[x][C("QD")]
		if x == 1 {
			t.True(ok, "Should have a record of 1 for QD")
		} else {
			t.Not(ok, "Should not have a record for QD")
		}
		_, ok = ai.ht.cards[x][C("KH")]
		t.Not(ok, "Should not have a record of a KH")
	}
	ai.ht.cards[2][C("QD")] = 1
	ai.ht.cards[2][C("KH")] = 1
	ai.ht.cards[3][C("KH")] = 1
	ai.ht.cards[3][C("KD")] = 1
	ai.ht.cards[0][C("9S")] = 1
	ai.ht.playedCards[C("9S")] = 1
	ai.ht.playedCards[C("TS")] = 2
	ai.ht.playedCards[C("JS")] = 1
	ai.ht.cards[0][C("JS")] = 0
	ai.ht.cards[2][C("JS")] = 0
	ai.ht.playedCards[C("QS")] = 0
	ai.ht.cards[0][C("QS")] = 0
	ai.ht.cards[2][C("QS")] = 0

	ai.ht.calculate()

	t.Equal(1, 2)
	t.Equal(1, ai.ht.cards[3][C("JS")])
	t.Equal(2, ai.ht.cards[3][C("QS")])

	for x := 0; x < 4; x++ {
		_, ok := ai.ht.cards[x][C("QD")]
		t.True(ok, "Value should be true for player "+strconv.Itoa(x)+" with QD")
		_, ok = ai.ht.cards[x][C("KH")]
		t.True(ok, "Value should be true for player "+strconv.Itoa(x)+" with KH")
		_, ok = ai.ht.cards[x][C("KD")]
		if x == 3 || x == 1 {
			t.True(ok, "Value should be true for player "+strconv.Itoa(x)+" with KD")
		} else {
			t.Not(ok, "Value should be false for player "+strconv.Itoa(x)+" with KD")
		}
		_, ok = ai.ht.cards[x][C("9S")]
		t.True(ok, "All 9S cards should have been found")
		_, ok = ai.ht.cards[x][C("TS")]
		t.True(ok, "All TS cards should have been found")
	}
	t.Equal(1, ai.ht.cards[3][C("JS")])
	t.Equal(2, ai.ht.cards[3][C("QS")])
}
