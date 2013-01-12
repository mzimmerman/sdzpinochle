// server_test.go
package main

import (
	sdz "github.com/mzimmerman/sdzpinochle"
	"sort"
	"testing"
)

func C(c string) sdz.Card {
	return sdz.Card(c)
}

func TestBidding(t *testing.T) {
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
	if 21 > action.Amount || action.Amount > 23 { // bid has 3 points of randomness
		t.Errorf("Power bid should be between 21 and 23, not %d, for hand %s", action.Amount, hand)
	}
}
