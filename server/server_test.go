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

func (t *testSuite) TestBidding() {
	// 9D QD TD TD AD JC QC KC 9H AH AH KS
	hand := sdz.Hand{C("9D"), C("QD"), C("TD"), C("TD"), C("AD"), C("JC"), C("QC"), C("KC"), C("9H"), C("AH"), C("AH"), C("KS")}
	sort.Sort(hand)
	ai := createAI(1)
	ai.SetHand(hand, 0)
	go ai.Go()
	ai.c <- sdz.Action{
		Action:   sdz.Bid,
		Playerid: ai.playerid,
	}
	action := <-ai.c
	t.False(21 > action.Amount || action.Amount > 23)
}
