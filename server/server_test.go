// server_test.go
package main

import (
	sdz "github.com/mzimmerman/sdzpinochle"
	pt "github.com/remogatto/prettytest"
	"sort"
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
	ai := createAI(1)
	ai.SetHand(hand, 0)
	t.Equal(12, len(*ai.Hand()))
	t.True(ai.Hand().Remove(C("JD")))
	t.True(ai.Hand().Remove(C("JD")))
	t.False(ai.Hand().Remove(C("9D")))
	t.Equal(10, len(*ai.Hand()))
}

func (t *testSuite) TestBidding() {
	// 9D QD TD TD AD JC QC KC 9H AH AH KS
	hand := sdz.Hand{C("9D"), C("QD"), C("TD"), C("TD"), C("AD"), C("JC"), C("QC"), C("KC"), C("9H"), C("AH"), C("AH"), C("KS")}
	sort.Sort(hand)
	ai := createAI(1)
	ai.SetHand(hand, 0)
	go ai.Go()
	ai.Tell(sdz.CreateBid(0, 1))
	action, _ := ai.Listen()
	t.False(21 > action.Bid || action.Bid > 23)
}
