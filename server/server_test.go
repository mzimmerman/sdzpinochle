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
	t.Not(t.True(ai.Hand().Remove(C("9D"))))
	t.Equal(10, len(*ai.Hand()))
}

func (t *testSuite) TestBidding() {
	// 9D 9D QD TD TD AD JC QC KC AH AH KS
	hand := sdz.Hand{C("9D"), C("9D"), C("QD"), C("TD"), C("TD"), C("AD"), C("JC"), C("QC"), C("KC"), C("AH"), C("AH"), C("KS")}
	sort.Sort(hand)
	ai := createAI()
	ai.SetHand(hand, 0, 1)
	action := ai.Tell(sdz.CreateBid(0, 1))
	t.Not(t.True(22 > action.Bid || action.Bid > 24))
}

func (t *testSuite) TestFullGame() {
	game := NewGame(4)
	for x := 0; x < len(game.Players); x++ {
		game.Players[x] = createAI()
	}
	game.NextHand(nil)
}

func (t *testSuite) TestPotentialCards() {
	ai := createAI()
	ht := ai.ht
	for _, card := range sdz.AllCards() {
		ht.cards[0][card] = 0
	}
	delete(ht.cards[0], C("AD"))
	ht.cards[0][C("TD")] = 1
	ht.cards[0][C("KD")] = 2
	potentials := potentialCards(0, ht, sdz.NACard, sdz.NASuit, sdz.Spades)
	t.True(potentials[C("AD")])
	t.True(potentials[C("TD")])
	t.True(potentials[C("KD")])
	t.False(potentials[C("QD")])

	delete(ht.cards[0], C("TS"))
	potentials = potentialCards(0, ht, C("KD"), sdz.Diamonds, sdz.Spades)

	t.True(potentials[C("AD")])
	t.True(potentials[C("TD")])
	t.False(potentials[C("KD")])
	t.False(potentials[C("TS")])

	//func potentialCards(playerid, ht *HandTracker, winning sdz.Card, lead sdz.Suit, trump sdz.Suit) map[sdz.Card]int {
	delete(ht.cards[0], C("AH"))
	delete(ht.cards[0], C("TH"))
	delete(ht.cards[0], C("QH"))
	ht.cards[0][C("AS")] = 1
	ht.cards[0][C("TS")] = 0
	potentials = potentialCards(0, ht, C("KH"), sdz.Hearts, sdz.Spades)
	t.True(potentials[C("AH")])
	t.True(potentials[C("TH")])
	t.True(potentials[C("AS")])
	t.True(potentials[C("QH")])
	t.False(potentials[C("TS")])

	ht.cards[0][C("AH")] = 1
	potentials = potentialCards(0, ht, C("KH"), sdz.Hearts, sdz.Spades)
	t.True(potentials[C("AH")])
	t.True(potentials[C("TH")])
	t.False(potentials[C("AS")])
	t.False(potentials[C("QH")])
	t.False(potentials[C("TS")])

	ai = createAI()
	ht = ai.ht
	for _, card := range sdz.AllCards() {
		ht.cards[0][card] = 0
	}
	ht.cards[0][C("AD")] = 2
	ht.cards[0][C("TD")] = 1
	ht.cards[0][C("JD")] = 1
	ht.cards[0][C("TC")] = 1
	ht.cards[0][C("KC")] = 1
	ht.cards[0][C("QC")] = 1
	ht.cards[0][C("TH")] = 1
	ht.cards[0][C("JH")] = 1
	ht.cards[0][C("9H")] = 1
	ht.cards[0][C("KS")] = 1
	ht.cards[0][C("QS")] = 1
	potentials = potentialCards(0, ht, sdz.NACard, sdz.NASuit, sdz.Hearts)
	t.Equal(11, len(potentials))

	ai = createAI()
	ht = ai.ht
	for _, card := range sdz.AllCards() {
		ht.cards[0][card] = 0
	}
	ht.cards[0][C("AD")] = 2
	ht.cards[0][C("TD")] = 1
	ht.cards[0][C("JD")] = 1
	ht.cards[0][C("TC")] = 1
	ht.cards[0][C("KC")] = 1
	ht.cards[0][C("QC")] = 1
	ht.cards[0][C("TH")] = 1
	ht.cards[0][C("JH")] = 1
	ht.cards[0][C("9H")] = 1
	ht.cards[0][C("KS")] = 1
	ht.cards[0][C("QS")] = 1
	potentials = potentialCards(0, ht, sdz.NACard, sdz.NASuit, sdz.Hearts)
	t.Equal(11, len(potentials))

	//Playerid:2, Bid:0, PlayedCard:"", WinningCard:"JS", Lead:"S", Trump:"C", Amount:0, Message:"", Hand:sdzpinochle.Hand{"TD", "TD", "QD", "JD", "9D", "9D", "JS"}, Option:0, GameOver:false, Win:false, Score:[]int(nil), Dealer:0, WinningPlayer:0}

	//Starting calculate() - map[KD:1 9S:1 QS:1 JH:2 9D:0 AC:2 JS:1 JD:0 KS:1 QD:0 JC:0 AD:0 KH:1 QC:1 9C:2 QH:1 TH:1 TD:1 AH:2 AS:1 KC:0 TS:0 TC:1 9H:2]
	//Player0 - map[JS:0 9H:0 9D:0 AH:0 KH:0 KS:0 QC:0 9C:0 AC:0 KC:1 TD:0 JH:0 KD:1]
	//Player1 - map[KD:0 QC:0 AC:0 JS:0 JC:0 TC:0 9C:0 9H:0 9D:0 TD:0 KC:0 JH:0 AH:0]
	//Player2 - map[TH:0 JH:0 JD:1 KS:0 QS:0 9C:0 JS:1 AD:0 QD:1 TS:0 9H:0 AS:0 KD:0 TD:1 KC:0 TC:0 9S:0 AH:0 9D:2 JC:0 QH:0 AC:0 QC:0 KH:0]
	//Player3 - map[JH:0 AH:0 KD:0 9C:0 JS:0 KC:1 TD:0 AC:0 9H:0 QC:1 9D:0]
	ai = createAI()
	ht = ai.ht
	for _, card := range sdz.AllCards() {
		ht.cards[0][card] = 0
	}
	ht.cards[0][C("TD")] = 2
	ht.cards[0][C("9D")] = 2
	ht.cards[0][C("QD")] = 1
	ht.cards[0][C("JD")] = 1
	ht.cards[0][C("JS")] = 1
	potentials = potentialCards(0, ht, C("JS"), sdz.Spades, sdz.Clubs)
	t.Equal(1, len(potentials))
	t.True(potentials[C("JS")])
	t.False(potentials[C("TD")])

	//PotentialCards called with 0,winning=AS,lead=D,trump=C,
	//ht=&main.HandTracker{cards:[4]map[sdzpinochle.Card]int{map[sdzpinochle.Card]int{"9C":0, "AC":0, "AS":0, "KD":0, "QD":0, "QS":0, "TH":0, "JC":0, "AD":0, "9D":0, "KC":0, "TS":0, "9H":0, "TC":0, "TD":0, "QH":0, "9S":0, "JD":0, "QC":0, "KH":0, "JH":0, "KS":0, "AH":0, "JS":0}, map[sdzpinochle.Card]int{"9C":0, "QC":0, "9D":0, "AH":0, "AC":0, "QH":1, "JD":0, "KS":0, "JC":0, "AS":0, "KC":0, "TH":0, "TC":0, "QS":0, "KH":0, "TS":0}, map[sdzpinochle.Card]int{"KS":0, "JD":0, "JC":0, "TH":0, "KH":0, "AS":0, "QD":0, "TC":0, "AC":0, "AH":0, "QH":0, "9C":0, "KD":1, "QC":0, "9D":0, "KC":0, "QS":0, "TS":0}, map[sdzpinochle.Card]int{"TH":0, "AS":0, "AH":0, "TS":0, "KC":0, "9C":0, "QS":0, "TC":0, "9D":0, "AC":0, "KS":0, "QC":0, "KH":0, "JD":0, "QH":0, "JC":0}}, playedCards:map[sdzpinochle.Card]int{"KH":2, "KD":0, "JC":2, "AH":2, "TH":2, "TD":1, "9S":1, "TC":2, "9D":2, "AS":2, "KS":2, "JS":1, "QC":2, "KC":2, "QD":1, "QS":2, "9H":1, "QH":1, "AD":0, "JD":2, "AC":2, "JH":1, "9C":2, "TS":2}}
	//ht = NewHandTracker(0, make(sdz.Hand, 0))
	//for _, card := range sdz.AllCards() {
	//	ht.cards[0][card] = 0
	//}
	//ht.cards[0][C("TD")] = 2
	//ht.cards[0][C("9D")] = 2
	//ht.cards[0][C("QD")] = 1
	//ht.cards[0][C("JD")] = 1
	//ht.cards[0][C("JS")] = 1
	//potentials = potentialCards(0, ht, C("JS"), sdz.Spades, sdz.Clubs)
	//t.Equal(1, len(potentials))
	//t.True(potentials[C("JS")])
	//t.False(potentials[C("TD")])

}

func (t *testSuite) TestFindCardToPlay() {
	//func (ai *AI) findCardToPlay(action *sdz.Action) sdz.Card {
	ai := createAI()
	ai.SetHand(sdz.Hand{C("AD"), C("AD"), C("TD"), C("JD"), C("TC"), C("KC"), C("QC"), C("TH"), C("JH"), C("9H"), C("KS"), C("QS")}, 0, 3)
	ai.ht.cards[0][C("KH")] = 1
	ai.ht.cards[0][C("QH")] = 1
	ai.ht.cards[1][C("9H")] = 1
	ai.ht.cards[2][C("KD")] = 1
	ai.ht.cards[2][C("QD")] = 1
	action := sdz.CreatePlayRequest(sdz.NACard, sdz.NASuit, sdz.Hearts, 3, ai.Hand())
	card := ai.findCardToPlay(action)
	t.True(card == C("TD"))
}

func (t *testSuite) TestRankCard() {
	//func rankCard(playerid int, ht *HandTracker, trick *Trick, lead, trump sdz.Suit) *Trick {
	ai := createAI()
	ht := ai.ht
	for _, card := range sdz.AllCards() {
		ht.cards[3][card] = 0
	}
	ht.cards[0][C("AD")] = 1
	ht.cards[0][C("TD")] = 1
	ht.cards[0][C("QD")] = 2
	ht.cards[1][C("AD")] = 1
	ht.cards[2][C("KD")] = 1
	ht.cards[2][C("TD")] = 1
	ht.cards[2][C("QS")] = 1
	ht.cards[3][C("KD")] = 1
	for _, card := range []sdz.Card{C("AD"), C("TD"), C("QD"), C("KD"), C("QS")} {
		ai.calculateCard(card)
	}

	trick := NewTrick()
	trick.played[0] = C("AD")
	trick.played[1] = C("AD")
	trick.played[2] = C("KD")
	trick.winningPlayer = 0
	trick.lead = 0
	// 3 has options of KD
	victim := rankCard(3, ht, trick, sdz.Diamonds)
	t.Equal(C("AD"), victim.played[0])
	t.Equal(C("AD"), victim.played[1])
	t.Equal(C("KD"), victim.played[2])
	t.Equal(C("KD"), victim.played[3])
	t.Equal(-32, victim.worth(3, sdz.Diamonds))

	trick = NewTrick()
	trick.played[0] = C("AD")
	trick.played[1] = C("AD")
	trick.winningPlayer = 0
	trick.lead = 0
	// 2 has real options of KD, TD, QD
	// 3 has real options of KD
	victim = rankCard(2, ht, trick, sdz.Diamonds)
	t.Equal(C("AD"), victim.played[0])
	t.Equal(C("AD"), victim.played[1])
	t.Equal(C("KD"), victim.played[2])
	t.Equal(C("KD"), victim.played[3])
	t.Equal(32, victim.worth(2, sdz.Diamonds))

	trick = NewTrick()
	trick.played[0] = C("TD")
	trick.played[1] = C("AD")
	trick.winningPlayer = 1
	trick.lead = 0
	ht.cards[2][C("9D")] = 0
	// 2 has real options of KD, TD, QD and no option of a 9D
	ht.cards[3][C("KD")] = 1
	ht.cards[3][C("TD")] = 1
	ht.cards[3][C("AD")] = 1
	ht.cards[3][C("9D")] = 1
	ht.cards[3][C("QD")] = 1
	// 3 has options of KD, TD, AD, 9D, and QD
	victim = rankCard(2, ht, trick, sdz.Diamonds)
	t.Equal(C("TD"), victim.played[0])
	t.Equal(C("AD"), victim.played[1])
	t.Equal(C("JD"), victim.played[2])
	t.Equal(C("KD"), victim.played[3])

}

func (t *testSuite) TestWorth() {
	trick := NewTrick()
	trick.played[0] = C("AS")
	trick.played[1] = C("9S")
	trick.played[2] = C("KS")
	trick.played[3] = C("QS")
	trick.winningPlayer = 0
	t.Equal(12, trick.worth(0, sdz.Diamonds))
	t.Equal(12, trick.worth(2, sdz.Diamonds))
	t.Equal(-12, trick.worth(1, sdz.Diamonds))
	t.Equal(-12, trick.worth(3, sdz.Diamonds))
	trick.certain = false
	t.Equal(6, trick.worth(0, sdz.Diamonds))
	t.Equal(6, trick.worth(2, sdz.Diamonds))
	t.Equal(-6, trick.worth(1, sdz.Diamonds))
	t.Equal(-6, trick.worth(3, sdz.Diamonds))
	t.Equal(6, trick.worth(0, sdz.Spades))
	t.Equal(6, trick.worth(2, sdz.Spades))
	t.Equal(-6, trick.worth(1, sdz.Spades))
	t.Equal(-6, trick.worth(3, sdz.Spades))

	trick.played[0] = C("9S")
	trick.played[1] = C("AS")
	trick.played[2] = C("KS")
	trick.played[3] = C("TS")
	trick.winningPlayer = 1
	trick.certain = false
	t.Equal(-9, trick.worth(0, sdz.Diamonds))
	t.Equal(-9, trick.worth(2, sdz.Diamonds))
	t.Equal(9, trick.worth(1, sdz.Diamonds))
	t.Equal(9, trick.worth(3, sdz.Diamonds))
	t.Equal(-9, trick.worth(0, sdz.Spades))
	t.Equal(-9, trick.worth(2, sdz.Spades))
	t.Equal(9, trick.worth(1, sdz.Spades))
	t.Equal(9, trick.worth(3, sdz.Spades))

	trick.played[0] = C("9S")
	trick.played[1] = C("AS")
	trick.played[2] = C("JD")
	trick.played[3] = C("TS")
	trick.winningPlayer = 2
	trick.certain = false
	t.Equal(10, trick.worth(0, sdz.Diamonds))
	t.Equal(10, trick.worth(2, sdz.Diamonds))
	t.Equal(-10, trick.worth(1, sdz.Diamonds))
	t.Equal(-10, trick.worth(3, sdz.Diamonds))
	trick.winningPlayer = 1
	t.Equal(-4, trick.worth(0, sdz.Spades))
	t.Equal(-4, trick.worth(2, sdz.Spades))
	t.Equal(4, trick.worth(1, sdz.Spades))
	t.Equal(4, trick.worth(3, sdz.Spades))

}

//func (t *testSuite) TestHands() {
//	p1 := createAI()
//	p2 := createAI()
//	p3 := createAI()
//	p0 := createAI()
//	game := &sdz.Game{Players: []sdz.Player{p0, p1, p2, p3}}
//	game.MeldHands = make([]sdz.Hand, len(game.Players))
//	game.Meld = make([]int, len(game.Players))
//	game.Dealer = 0
//	p1.SetHand(sdz.Hand{C("AD"), C("QD"), C("TC"), C("KC"), C("KC"), C("QC"), C("QC"), C("AH"), C("JH"), C("9H"), C("QS"), C("9S")}, game.Dealer, 1)
//	p2.SetHand(sdz.Hand{C("KD"), C("JD"), C("9D"), C("9D"), C("JC"), C("AH"), C("TH"), C("KH"), C("QH"), C("AS"), C("QS"), C("JS")}, game.Dealer, 2)
//	p3.SetHand(sdz.Hand{C("TD"), C("KD"), C("QD"), C("JD"), C("AC"), C("AC"), C("TC"), C("KH"), C("9H"), C("TS"), C("KS"), C("9S")}, game.Dealer, 3)
//	p0.SetHand(sdz.Hand{C("AD"), C("TD"), C("JC"), C("9C"), C("9C"), C("TH"), C("QH"), C("JH"), C("AS"), C("TS"), C("KS"), C("JS")}, game.Dealer, 0)
//	game.Broadcast(sdz.CreateBid(22, 1), 3)
//	game.Trump = sdz.Clubs
//	game.BroadcastAll(sdz.CreateTrump(game.Trump, 3))
//	p1.trump = game.Trump
//	p2.trump = game.Trump
//	p3.trump = game.Trump
//	p0.trump = game.Trump
//	for x := 0; x < len(game.Players); x++ {
//		game.Meld[x], game.MeldHands[x] = game.Players[x].Hand().Meld(game.Trump)
//		meldAction := sdz.CreateMeld(game.MeldHands[x], game.Meld[x], x)
//		game.BroadcastAll(meldAction)
//	}
//	next := 3
//	for trick := 0; trick < 12; trick++ {
//		var winningCard sdz.Card
//		var cardPlayed sdz.Card
//		var leadSuit sdz.Suit
//		winningPlayer := next
//		counters := 0
//		for x := 0; x < 4; x++ {
//			// play the hand
//			// TODO: handle possible throwin
//			action := sdz.CreatePlayRequest(winningCard, leadSuit, game.Trump, next, game.Players[next].Hand())
//			game.Players[next].Tell(action)
//			action, open := game.Players[next].Listen()
//			if !open {
//				game.Broadcast(sdz.CreateMessage("Player disconnected"), next)
//				return
//			}
//			cardPlayed = action.PlayedCard
//			game.Players[next].Hand().Remove(cardPlayed)
//			if x == 0 {
//				winningCard = cardPlayed
//				leadSuit = cardPlayed.Suit()
//			} else {
//				if cardPlayed.Beats(winningCard, game.Trump) {
//					winningCard = cardPlayed
//					winningPlayer = next
//				}
//			}
//			game.Broadcast(action, next)
//			next = (next + 1) % 4
//		}
//		next = winningPlayer
//		game.BroadcastAll(sdz.CreateMessage(fmt.Sprintf("Player %d wins trick #%d with %s for %d points", winningPlayer, trick+1, winningCard, counters)))
//		game.BroadcastAll(sdz.CreateTrick(winningPlayer))
//		//Log("Player %d wins trick #%d with %s for %d points", winningPlayer, trick+1, winningCard, counters)
//	}

//	//game.Dealer = 3
//	//p1.SetHand(sdz.Hand{C("AD"), C("TD"), C("JD"), C("TC"), C("KC"), C("AH"), C("KH"), C("QH"), C("TS"), C("QS"), C("QS"), C("JS")}, game.Dealer, 1)
//	//p2.SetHand(sdz.Hand{C("AD"), C("QD"), C("QD"), C("AC"), C("QC"), C("QC"), C("JC"), C("JC"), C("TH"), C("QH"), C("KS"), C("JS")}, game.Dealer, 2)
//	//p3.SetHand(sdz.Hand{C("TD"), C("JD"), C("AC"), C("TC"), C("AH"), C("KH"), C("JH"), C("JH"), C("9H"), C("TS"), C("9S"), C("9S")}, game.Dealer, 3)
//	//p0.SetHand(sdz.Hand{C("KD"), C("KD"), C("9D"), C("9D"), C("KC"), C("9C"), C("9C"), C("TH"), C("9H"), C("AS"), C("AS"), C("KS")}, game.Dealer, 0)
//	//game.Broadcast(sdz.CreateBid(20, 3), 3)
//	//game.Trump = sdz.Clubs
//	//game.BroadcastAll(sdz.CreateTrump(game.Trump, 1))
//	//p1.trump = game.Trump
//	//p2.trump = game.Trump
//	//p3.trump = game.Trump
//	//p0.trump = game.Trump
//	//for x := 0; x < len(game.Players); x++ {
//	//	game.Meld[x], game.MeldHands[x] = game.Players[x].Hand().Meld(game.Trump)
//	//	meldAction := sdz.CreateMeld(game.MeldHands[x], game.Meld[x], x)
//	//	game.BroadcastAll(meldAction)
//	//}
//	//next = 3
//	//for trick := 0; trick < 12; trick++ {
//	//	var winningCard sdz.Card
//	//	var cardPlayed sdz.Card
//	//	var leadSuit sdz.Suit
//	//	winningPlayer := next
//	//	counters := 0
//	//	for x := 0; x < 4; x++ {
//	//		// play the hand
//	//		// TODO: handle possible throwin
//	//		action := sdz.CreatePlayRequest(winningCard, leadSuit, game.Trump, next, game.Players[next].Hand())
//	//		game.Players[next].Tell(action)
//	//		action, open := game.Players[next].Listen()
//	//		if !open {
//	//			game.Broadcast(sdz.CreateMessage("Player disconnected"), next)
//	//			return
//	//		}
//	//		cardPlayed = action.PlayedCard
//	//		game.Players[next].Hand().Remove(cardPlayed)
//	//		if x == 0 {
//	//			winningCard = cardPlayed
//	//			leadSuit = cardPlayed.Suit()
//	//		} else {
//	//			if cardPlayed.Beats(winningCard, game.Trump) {
//	//				winningCard = cardPlayed
//	//				winningPlayer = next
//	//			}
//	//		}
//	//		game.Broadcast(action, next)
//	//		next = (next + 1) % 4
//	//	}
//	//	next = winningPlayer
//	//	game.BroadcastAll(sdz.CreateMessage(fmt.Sprintf("Player %d wins trick #%d with %s for %d points", winningPlayer, trick+1, winningCard, counters)))
//	//	game.BroadcastAll(sdz.CreateTrick(winningPlayer))
//	//	//Log("Player %d wins trick #%d with %s for %d points", winningPlayer, trick+1, winningCard, counters)
//	//}

//	//p1 = createAI()
//	//p2 = createAI()
//	//p3 = createAI()
//	//p0 = createAI()
//	//game = &sdz.Game{Players: []sdz.Player{p0, p1, p2, p3}}
//	//game.MeldHands = make([]sdz.Hand, len(game.Players))
//	//game.Meld = make([]int, len(game.Players))
//	//p1.SetHand(sdz.Hand{C("AD"), C("QD"), C("JD"), C("9D"), C("AH"), C("TH"), C("KH"), C("KH"), C("JH"), C("KS"), C("JS"), C("JS")}, 0, 1)
//	//p2.SetHand(sdz.Hand{C("TD"), C("KD"), C("QC"), C("QC"), C("9C"), C("TH"), C("QH"), C("9H"), C("9H"), C("AS"), C("TS"), C("QS")}, 0, 2)
//	//p3.SetHand(sdz.Hand{C("KD"), C("QD"), C("JD"), C("AC"), C("AC"), C("TC"), C("TC"), C("JC"), C("AH"), C("JH"), C("AS"), C("KS")}, 0, 3)
//	//p0.SetHand(sdz.Hand{C("AD"), C("TD"), C("9D"), C("KC"), C("KC"), C("JC"), C("9C"), C("QH"), C("TS"), C("QS"), C("9S"), C("9S")}, 0, 0)
//	//game.Broadcast(sdz.CreateBid(21, 3), 3)
//	//game.BroadcastAll(sdz.CreateTrump(sdz.Clubs, 3))
//	//p1.trump = sdz.Clubs
//	//p2.trump = sdz.Clubs
//	//p3.trump = sdz.Clubs
//	//p0.trump = sdz.Clubs
//	//for x := 0; x < len(game.Players); x++ {
//	//	game.Meld[x], game.MeldHands[x] = game.Players[x].Hand().Meld(game.Trump)
//	//	meldAction := sdz.CreateMeld(game.MeldHands[x], game.Meld[x], x)
//	//	game.BroadcastAll(meldAction)
//	//}
//	//next = 1
//	//for trick := 0; trick < 12; trick++ {
//	//	var winningCard sdz.Card
//	//	var cardPlayed sdz.Card
//	//	var leadSuit sdz.Suit
//	//	winningPlayer := next
//	//	counters := 0
//	//	for x := 0; x < 4; x++ {
//	//		// play the hand
//	//		// TODO: handle possible throwin
//	//		action := sdz.CreatePlayRequest(winningCard, leadSuit, game.Trump, next, game.Players[next].Hand())
//	//		game.Players[next].Tell(action)
//	//		action, open := game.Players[next].Listen()
//	//		if !open {
//	//			game.Broadcast(sdz.CreateMessage("Player disconnected"), next)
//	//			return
//	//		}
//	//		cardPlayed = action.PlayedCard
//	//		game.Players[next].Hand().Remove(cardPlayed)
//	//		if x == 0 {
//	//			winningCard = cardPlayed
//	//			leadSuit = cardPlayed.Suit()
//	//		} else {
//	//			if cardPlayed.Beats(winningCard, game.Trump) {
//	//				winningCard = cardPlayed
//	//				winningPlayer = next
//	//			}
//	//		}
//	//		game.Broadcast(action, next)
//	//		next = (next + 1) % 4
//	//	}
//	//	next = winningPlayer
//	//	game.BroadcastAll(sdz.CreateMessage(fmt.Sprintf("Player %d wins trick #%d with %s for %d points", winningPlayer, trick+1, winningCard, counters)))
//	//	game.BroadcastAll(sdz.CreateTrick(winningPlayer))
//	//	//Log("Player %d wins trick #%d with %s for %d points", winningPlayer, trick+1, winningCard, counters)
//	//}

//	//p1 = createAI()
//	//p2 = createAI()
//	//p3 = createAI()
//	//p0 = createAI()
//	//game = &sdz.Game{Players: []sdz.Player{p0, p1, p2, p3}}
//	//game.MeldHands = make([]sdz.Hand, len(game.Players))
//	//game.Meld = make([]int, len(game.Players))
//	//p1.SetHand(sdz.Hand{C("AD"), C("TD"), C("KD"), C("KD"), C("QD"), C("JD"), C("AC"), C("QC"), C("9C"), C("QH"), C("9H"), C("KS")}, 0, 1)
//	//p2.SetHand(sdz.Hand{C("TD"), C("QD"), C("9D"), C("AC"), C("QC"), C("JC"), C("9C"), C("TH"), C("KH"), C("QH"), C("JH"), C("AS")}, 0, 2)
//	//p3.SetHand(sdz.Hand{C("AD"), C("TC"), C("KC"), C("KC"), C("JC"), C("AH"), C("AH"), C("TH"), C("TS"), C("KS"), C("JS"), C("9S")}, 0, 3)
//	//p0.SetHand(sdz.Hand{C("JD"), C("9D"), C("TC"), C("KH"), C("JH"), C("9H"), C("AS"), C("TS"), C("QS"), C("QS"), C("JS"), C("9S")}, 0, 0)
//	//game.Broadcast(sdz.CreateBid(31, 1), 1)
//	//game.BroadcastAll(sdz.CreateTrump(sdz.Diamonds, 1))
//	//p1.trump = sdz.Diamonds
//	//p2.trump = sdz.Diamonds
//	//p3.trump = sdz.Diamonds
//	//p0.trump = sdz.Diamonds
//	//for x := 0; x < len(game.Players); x++ {
//	//	game.Meld[x], game.MeldHands[x] = game.Players[x].Hand().Meld(game.Trump)
//	//	meldAction := sdz.CreateMeld(game.MeldHands[x], game.Meld[x], x)
//	//	game.BroadcastAll(meldAction)
//	//}
//	//next = 1
//	//for trick := 0; trick < 12; trick++ {
//	//	var winningCard sdz.Card
//	//	var cardPlayed sdz.Card
//	//	var leadSuit sdz.Suit
//	//	winningPlayer := next
//	//	counters := 0
//	//	for x := 0; x < 4; x++ {
//	//		// play the hand
//	//		// TODO: handle possible throwin
//	//		action := sdz.CreatePlayRequest(winningCard, leadSuit, game.Trump, next, game.Players[next].Hand())
//	//		game.Players[next].Tell(action)
//	//		action, open := game.Players[next].Listen()
//	//		if !open {
//	//			game.Broadcast(sdz.CreateMessage("Player disconnected"), next)
//	//			return
//	//		}
//	//		cardPlayed = action.PlayedCard
//	//		game.Players[next].Hand().Remove(cardPlayed)
//	//		if x == 0 {
//	//			winningCard = cardPlayed
//	//			leadSuit = cardPlayed.Suit()
//	//		} else {
//	//			if cardPlayed.Beats(winningCard, game.Trump) {
//	//				winningCard = cardPlayed
//	//				winningPlayer = next
//	//			}
//	//		}
//	//		game.Broadcast(action, next)
//	//		next = (next + 1) % 4
//	//	}
//	//	next = winningPlayer
//	//	game.BroadcastAll(sdz.CreateMessage(fmt.Sprintf("Player %d wins trick #%d with %s for %d points", winningPlayer, trick+1, winningCard, counters)))
//	//	game.BroadcastAll(sdz.CreateTrick(winningPlayer))
//	//	//Log("Player %d wins trick #%d with %s for %d points", winningPlayer, trick+1, winningCard, counters)
//	//}
//}

func (t *testSuite) TestAITracking() {
	ai := createAI()
	hand := sdz.Hand{C("9D"), C("9D"), C("QD"), C("TD"), C("TD"), C("AD"), C("JC"), C("QC"), C("KC"), C("AH"), C("AH"), C("KS")}
	ai.SetHand(hand, 0, 0)
	ai.trump = sdz.Spades
	ai.Tell(sdz.CreateMeld(sdz.Hand{C("JD"), C("QS"), C("KD"), C("QD")}, 6, 1))
	ai.Tell(sdz.CreateMeld(sdz.Hand{C("JD"), C("QS")}, 4, 2))
	ai.Tell(sdz.CreateMeld(sdz.Hand{}, 0, 3))
	//ai.calculate()
	t.Equal(1, ai.ht.cards[1][C("JD")])
	t.Equal(1, ai.ht.cards[1][C("QS")])
	t.Equal(1, ai.ht.cards[2][C("JD")])
	t.Equal(1, ai.ht.cards[2][C("QS")])
	t.Equal(0, ai.ht.cards[3][C("QS")])
	t.Equal(0, ai.ht.playedCards[C("JD")])
	t.Equal(0, ai.ht.playedCards[C("QS")])
	t.Equal(0, ai.ht.playedCards[C("QD")])
	t.Equal(1, ai.ht.cards[1][C("QD")])
	t.Equal(0, ai.ht.cards[2][C("QD")])
	t.Equal(0, ai.ht.cards[3][C("QD")])
	t.Equal(1, ai.ht.cards[1][C("KD")])

	ai.trick.lead = 1
	ai.Tell(sdz.CreatePlay(C("JD"), 1))
	ai.Tell(sdz.CreatePlay(C("KD"), 2))
	ai.Tell(sdz.CreatePlay(C("AD"), 3))
	val, ok := ai.ht.cards[1][C("JD")]
	t.True(ok)
	t.Equal(0, val)
	//ai.calculate()
	val, ok = ai.ht.cards[1][C("JD")]
	t.True(ok)
	t.Equal(0, val)
	t.Equal(1, ai.ht.cards[1][C("QS")])
	t.Equal(1, ai.ht.cards[2][C("JD")])
	t.Equal(1, ai.ht.cards[2][C("QS")])
	t.Equal(0, ai.ht.cards[3][C("QS")])
	t.Equal(1, ai.ht.playedCards[C("JD")])
	t.Equal(0, ai.ht.playedCards[C("QS")])
	t.Equal(1, ai.ht.playedCards[C("KD")])
	t.Equal(1, ai.ht.playedCards[C("AD")])

	ai.trick.lead = 1
	ai.Tell(sdz.CreatePlay(C("QD"), 1))
	ai.Tell(sdz.CreatePlay(C("9H"), 2))
	ai.Tell(sdz.CreatePlay(C("9H"), 3))
	val, ok = ai.ht.cards[1][C("QD")]
	t.True(ok)
	t.Equal(0, val)

	ai = createAI()
	hand = sdz.Hand{C("9D"), C("9D"), C("QD"), C("TD"), C("TD"), C("AS"), C("JC"), C("QC"), C("KC"), C("AH"), C("AH"), C("KS")}
	ai.SetHand(hand, 0, 0)
	ai.trump = sdz.Spades
	ai.Tell(sdz.CreateMeld(sdz.Hand{}, 0, 0))
	ai.Tell(sdz.CreateMeld(sdz.Hand{}, 0, 1))
	ai.Tell(sdz.CreateMeld(sdz.Hand{}, 0, 2))
	ai.Tell(sdz.CreateMeld(sdz.Hand{}, 0, 3))
	//ai.calculate()
	ai.trick.lead = 1

	ai.Tell(sdz.CreatePlay(C("JD"), 1))
	ai.Tell(sdz.CreatePlay(C("JD"), 2))
	ai.Tell(sdz.CreatePlay(C("KD"), 3))
	ai.Tell(sdz.CreatePlayRequest(ai.trick.winningCard(), ai.trick.leadSuit(), ai.trump, ai.Playerid(), &sdz.Hand{}))
	play, _ := ai.Listen()
	t.Equal(C("TD"), play.PlayedCard)
	ai.Tell(sdz.CreateTrick(0))
	ai.Tell(sdz.CreatePlay(C("QD"), 1))
	ai.Tell(sdz.CreatePlay(C("KD"), 2))
	ai.Tell(sdz.CreatePlay(C("KH"), 3))
	ai.Tell(sdz.CreatePlayRequest(ai.trick.winningCard(), ai.trick.leadSuit(), ai.trump, ai.Playerid(), &sdz.Hand{}))
	play, _ = ai.Listen()
	t.Equal(C("TD"), play.PlayedCard)

	ai = createAI()
	hand = sdz.Hand{C("9D"), C("9D"), C("QD"), C("TD"), C("TD"), C("AD"), C("JC"), C("QC"), C("KC"), C("AH"), C("AH"), C("KS")}
	ai.SetHand(hand, 0, 0)
	ai.trump = sdz.Spades
	ai.Tell(sdz.CreateMeld(sdz.Hand{C("JD"), C("JD"), C("QS"), C("QS"), C("KD"), C("QD")}, 32, 1))
	ai.Tell(sdz.CreateMeld(sdz.Hand{}, 0, 2))
	ai.Tell(sdz.CreateMeld(sdz.Hand{}, 0, 3))
	//ai.calculate()
	t.Equal(0, ai.ht.cards[0][C("JD")])
	t.Equal(2, ai.ht.cards[1][C("JD")])
	t.Equal(0, ai.ht.cards[2][C("JD")])
	t.Equal(0, ai.ht.cards[3][C("JD")])
}

func (t *testSuite) TestNoSuit() {
	ai := createAI()
	ai.hand = &sdz.Hand{C("9D"), C("9D"), C("QD"), C("TD"), C("TD"), C("AD"), C("JC"), C("QC"), C("KC"), C("AH"), C("AH"), C("KS")}
	ai.populate()
	ai.ht.cards[1][C("9D")] = 1
	ai.noSuit(1, sdz.Diamonds)

	t.Equal(0, ai.ht.cards[1][C("9D")])
	t.Equal(0, ai.ht.cards[1][C("JD")])
	t.Equal(0, ai.ht.cards[1][C("QD")])
	t.Equal(0, ai.ht.cards[1][C("KD")])
	t.Equal(0, ai.ht.cards[1][C("TD")])
	t.Equal(0, ai.ht.cards[1][C("AD")])
	t.Equal(0, ai.ht.cards[1][C("AS")])
}

func (t *testSuite) TestCalculate() {
	hand := sdz.Hand{C("9D"), C("9D"), C("QD"), C("TD"), C("TD"), C("AD"), C("JC"), C("QC"), C("KC"), C("AH"), C("AH"), C("KS")}
	sort.Sort(hand)
	ai := createAI()
	// dealer 0, playerid 1
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
			t.Not(t.True(ok), "Should not have a record for QD")
		}
		_, ok = ai.ht.cards[x][C("KH")]
		if x == 1 {
			t.True(ok, "Should have a record of a KH")
		} else {
			t.True(!ok, "Should not have a record of a KH")
		}
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
	ai.ht.cards[1][C("QS")] = 0
	ai.ht.cards[2][C("QS")] = 0

	for _, card := range []sdz.Card{C("QD"), C("KH"), C("KD"), C("9S"), C("TS"), C("JS"), C("QS")} {
		ai.calculateCard(card)
	}

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
			t.Not(t.True(ok), "Value should be false for player "+strconv.Itoa(x)+" with KD")
		}
		_, ok = ai.ht.cards[x][C("9S")]
		t.True(ok, "All 9S cards should have been found")
		_, ok = ai.ht.cards[x][C("TS")]
		t.True(ok, "All TS cards should have been found")
	}

	ai.ht.playedCards[C("JD")] = 0
	ai.ht.cards[0][C("JD")] = 1
	ai.ht.cards[1][C("JD")] = 0
	ai.ht.cards[2][C("JD")] = 0
	ai.calculateCard(C("JD"))
	val, ok := ai.ht.cards[0][C("JD")]
	t.Equal(1, val)
	t.True(ok)
	val, ok = ai.ht.cards[1][C("JD")]
	t.Equal(0, val)
	t.True(ok)
	val, ok = ai.ht.cards[2][C("JD")]
	t.Equal(0, val)
	t.True(ok)
	val, ok = ai.ht.cards[3][C("JD")]
	t.False(ok)
}
