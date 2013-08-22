// server_test.go
package sdzpinochleserver

import (
	//"github.com/icub3d/appenginetesting"
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
	ai.SetHand(nil, nil, nil, hand, 0, 0)
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
	ai.SetHand(nil, nil, nil, hand, 0, 1)
	action := ai.Tell(nil, nil, nil, sdz.CreateBid(0, 1))
	t.Not(t.True(22 > action.Bid || action.Bid > 24))
}

//func BenchmarkFullGame(b *testing.B) {
//	c, err := appenginetesting.NewContext(&appenginetesting.Options{Debug: "critical"})
//	if err != nil {
//		b.Fatalf("Could not start up appenginetesting")
//	}
//	defer c.Close()
//	for y := 0; y < b.N; y++ {
//		game := NewGame(4)
//		for x := 0; x < len(game.Players); x++ {
//			game.Players[x] = createAI()
//		}
//		game.NextHand(nil, c)
//	}
//}

func (t *testSuite) TestPotentialCards() {
	ai := createAI()
	ht := ai.HT
	for _, card := range sdz.AllCards() {
		ht.Cards[0][card] = 0
	}
	delete(ht.Cards[0], C("AD"))
	ht.Cards[0][C("TD")] = 1
	ht.Cards[0][C("KD")] = 2
	potentials := potentialCards(0, ht, sdz.NACard, sdz.NASuit, sdz.Spades)

	t.True(potentials.Contains("AD"))
	t.True(potentials.Contains("TD"))
	t.True(potentials.Contains("KD"))
	t.False(potentials.Contains("QD"))

	delete(ht.Cards[0], C("TS"))
	potentials = potentialCards(0, ht, C("KD"), sdz.Diamonds, sdz.Spades)

	t.True(potentials.Contains("AD"))
	t.True(potentials.Contains("TD"))
	t.False(potentials.Contains("KD"))
	t.False(potentials.Contains("TS"))

	//func potentialCards(playerid, ht *HandTracker, winning sdz.Card, lead sdz.Suit, trump sdz.Suit) map[sdz.Card]int {
	delete(ht.Cards[0], C("AH"))
	delete(ht.Cards[0], C("TH"))
	delete(ht.Cards[0], C("QH"))
	ht.Cards[0][C("AS")] = 1
	ht.Cards[0][C("TS")] = 0
	potentials = potentialCards(0, ht, C("KH"), sdz.Hearts, sdz.Spades)
	t.True(potentials.Contains("AH"))
	t.True(potentials.Contains("TH"))
	t.True(potentials.Contains("AS"))
	t.True(potentials.Contains("QH"))
	t.False(potentials.Contains("TS"))

	ht.Cards[0][C("AH")] = 1
	potentials = potentialCards(0, ht, C("KH"), sdz.Hearts, sdz.Spades)
	t.True(potentials.Contains("AH"))
	t.True(potentials.Contains("TH"))
	t.False(potentials.Contains("AS"))
	t.False(potentials.Contains("QH"))
	t.False(potentials.Contains("TS"))

	ai = createAI()
	ht = ai.HT
	for _, card := range sdz.AllCards() {
		ht.Cards[0][card] = 0
	}
	ht.Cards[0][C("AD")] = 2
	ht.Cards[0][C("TD")] = 1
	ht.Cards[0][C("JD")] = 1
	ht.Cards[0][C("TC")] = 1
	ht.Cards[0][C("KC")] = 1
	ht.Cards[0][C("QC")] = 1
	ht.Cards[0][C("TH")] = 1
	ht.Cards[0][C("JH")] = 1
	ht.Cards[0][C("9H")] = 1
	ht.Cards[0][C("KS")] = 1
	ht.Cards[0][C("QS")] = 1
	potentials = potentialCards(0, ht, sdz.NACard, sdz.NASuit, sdz.Hearts)
	t.Equal(11, len(potentials))

	ai = createAI()
	ht = ai.HT
	for _, card := range sdz.AllCards() {
		ht.Cards[0][card] = 0
	}
	ht.Cards[0][C("AD")] = 2
	ht.Cards[0][C("TD")] = 1
	ht.Cards[0][C("JD")] = 1
	ht.Cards[0][C("TC")] = 1
	ht.Cards[0][C("KC")] = 1
	ht.Cards[0][C("QC")] = 1
	ht.Cards[0][C("TH")] = 1
	ht.Cards[0][C("JH")] = 1
	ht.Cards[0][C("9H")] = 1
	ht.Cards[0][C("KS")] = 1
	ht.Cards[0][C("QS")] = 1
	potentials = potentialCards(0, ht, sdz.NACard, sdz.NASuit, sdz.Hearts)
	t.Equal(11, len(potentials))

	//Playerid:2, Bid:0, PlayedCard:"", WinningCard:"JS", Lead:"S", Trump:"C", Amount:0, Message:"", Hand:sdzpinochle.Hand{"TD", "TD", "QD", "JD", "9D", "9D", "JS"}, Option:0, GameOver:false, Win:false, Score:[]int(nil), Dealer:0, WinningPlayer:0}

	//Starting calculate() - map[KD:1 9S:1 QS:1 JH:2 9D:0 AC:2 JS:1 JD:0 KS:1 QD:0 JC:0 AD:0 KH:1 QC:1 9C:2 QH:1 TH:1 TD:1 AH:2 AS:1 KC:0 TS:0 TC:1 9H:2]
	//Player0 - map[JS:0 9H:0 9D:0 AH:0 KH:0 KS:0 QC:0 9C:0 AC:0 KC:1 TD:0 JH:0 KD:1]
	//Player1 - map[KD:0 QC:0 AC:0 JS:0 JC:0 TC:0 9C:0 9H:0 9D:0 TD:0 KC:0 JH:0 AH:0]
	//Player2 - map[TH:0 JH:0 JD:1 KS:0 QS:0 9C:0 JS:1 AD:0 QD:1 TS:0 9H:0 AS:0 KD:0 TD:1 KC:0 TC:0 9S:0 AH:0 9D:2 JC:0 QH:0 AC:0 QC:0 KH:0]
	//Player3 - map[JH:0 AH:0 KD:0 9C:0 JS:0 KC:1 TD:0 AC:0 9H:0 QC:1 9D:0]
	ai = createAI()
	ht = ai.HT
	for _, card := range sdz.AllCards() {
		ht.Cards[0][card] = 0
	}
	ht.Cards[0][C("TD")] = 2
	ht.Cards[0][C("9D")] = 2
	ht.Cards[0][C("QD")] = 1
	ht.Cards[0][C("JD")] = 1
	ht.Cards[0][C("JS")] = 1
	potentials = potentialCards(0, ht, C("JS"), sdz.Spades, sdz.Clubs)
	t.Equal(1, len(potentials))
	t.True(potentials.Contains("JS"))
	t.False(potentials.Contains("TD"))

	//PotentialCards called with 0,winning=AS,lead=D,trump=C,
	//ht=&main.HandTracker{cards:[4]map[sdzpinochle.Card]int{map[sdzpinochle.Card]int{"9C":0, "AC":0, "AS":0, "KD":0, "QD":0, "QS":0, "TH":0, "JC":0, "AD":0, "9D":0, "KC":0, "TS":0, "9H":0, "TC":0, "TD":0, "QH":0, "9S":0, "JD":0, "QC":0, "KH":0, "JH":0, "KS":0, "AH":0, "JS":0}, map[sdzpinochle.Card]int{"9C":0, "QC":0, "9D":0, "AH":0, "AC":0, "QH":1, "JD":0, "KS":0, "JC":0, "AS":0, "KC":0, "TH":0, "TC":0, "QS":0, "KH":0, "TS":0}, map[sdzpinochle.Card]int{"KS":0, "JD":0, "JC":0, "TH":0, "KH":0, "AS":0, "QD":0, "TC":0, "AC":0, "AH":0, "QH":0, "9C":0, "KD":1, "QC":0, "9D":0, "KC":0, "QS":0, "TS":0}, map[sdzpinochle.Card]int{"TH":0, "AS":0, "AH":0, "TS":0, "KC":0, "9C":0, "QS":0, "TC":0, "9D":0, "AC":0, "KS":0, "QC":0, "KH":0, "JD":0, "QH":0, "JC":0}}, playedCards:map[sdzpinochle.Card]int{"KH":2, "KD":0, "JC":2, "AH":2, "TH":2, "TD":1, "9S":1, "TC":2, "9D":2, "AS":2, "KS":2, "JS":1, "QC":2, "KC":2, "QD":1, "QS":2, "9H":1, "QH":1, "AD":0, "JD":2, "AC":2, "JH":1, "9C":2, "TS":2}}
	//ht = NewHandTracker(0, make(sdz.Hand, 0))
	//for _, card := range sdz.AllCards() {
	//	ht.Cards[0][card] = 0
	//}
	//ht.Cards[0][C("TD")] = 2
	//ht.Cards[0][C("9D")] = 2
	//ht.Cards[0][C("QD")] = 1
	//ht.Cards[0][C("JD")] = 1
	//ht.Cards[0][C("JS")] = 1
	//potentials = potentialCards(0, ht, C("JS"), sdz.Spades, sdz.Clubs)
	//t.Equal(1, len(potentials))
	//t.True(potentials[C("JS")])
	//t.False(potentials[C("TD")])

}

func (t *testSuite) TestPlayHandWithCard() {
	//func playHandWithCard(playerid int, ht *HandTracker, trick *Trick, trump sdz.Suit) (sdz.Card, [2]int) {
	ht := new(HandTracker)
	ht.Owner = 0
	for x := 0; x < len(ht.Cards); x++ {
		ht.Cards[x] = make(map[sdz.Card]int)
		for _, card := range sdz.AllCards() {
			ht.Cards[x][card] = 0
		}
	}
	ht.PlayedCards = map[sdz.Card]int{"AD": 0, "TD": 0, "KD": 0, "QD": 0, "JD": 2, "9D": 2, "AS": 2, "TS": 2, "KS": 2, "QS": 2, "JS": 2, "9S": 2, "AH": 2, "TH": 2, "KH": 2, "QH": 2, "JH": 2, "9H": 2, "AC": 2, "TC": 2, "KC": 2, "QC": 2, "JC": 2, "9C": 2}
	//for _, card := range sdz.AllCards() {
	//	ht.PlayedCards[card] = 0
	//}
	ht.Cards[0]["AD"] = 1
	ht.Cards[1]["TD"] = 1
	ht.Cards[2]["KD"] = 1
	ht.Cards[3]["QD"] = 1

	before := len(ht.Cards[0])
	card, value := playHandWithCard(true, 0, ht, NewTrick(), sdz.Diamonds)
	t.Equal(before, len(ht.Cards[0]))
	t.Equal(card, sdz.CreateCard("D", "A"))
	t.Equal(value, 4)

	ht.Cards[1]["AD"] = 1
	ht.Cards[2]["TD"] = 1
	ht.Cards[3]["KD"] = 1
	ht.Cards[0]["QD"] = 1

	card, value = playHandWithCard(true, 0, ht, NewTrick(), sdz.Diamonds)
	t.Equal(card, C("AD"))
	t.Equal(value, 3)

	ht.Cards[1]["AD"] = 1
	ht.Cards[2]["TD"] = 1
	ht.Cards[3]["KD"] = 1
	ht.Cards[0]["QD"] = 1

	ht.PlayedCards["AS"] = 1
	ht.Cards[0]["AS"] = 1
	ht.PlayedCards["TS"] = 1
	ht.Cards[1]["TS"] = 1
	ht.PlayedCards["QS"] = 1
	ht.Cards[2]["QS"] = 1
	ht.PlayedCards["JS"] = 1
	ht.Cards[3]["JS"] = 1

	card, value = playHandWithCard(true, 0, ht, NewTrick(), sdz.Diamonds)
	t.Equal(card, C("AD"))
	t.Equal(value, 6)

	ht.PlayedCards["AC"] = 0
	ht.PlayedCards["TC"] = 0

	ht.Cards[0]["AC"] = 1
	delete(ht.Cards[1], "AC")
	delete(ht.Cards[2], "AC")
	delete(ht.Cards[3], "AC")
	delete(ht.Cards[1], "TC")
	delete(ht.Cards[2], "TC")
	delete(ht.Cards[3], "TC")

	card, value = playHandWithCard(true, 0, ht, NewTrick(), sdz.Diamonds)
	t.Equal(card, C("AC"))
	t.Equal(value, 10)

}

func (t *testSuite) TestFindCardToPlayShort() {
	//func (ai *AI) findCardToPlay(action *sdz.Action) sdz.Card {
	ai := createAI()
	ai.SetHand(nil, nil, nil, sdz.Hand{C("AD"), C("QS")}, 0, 3)
	for card := range ai.HT.PlayedCards {
		ai.HT.PlayedCards[card] = 2
	}
	ai.HT.PlayedCards["AD"] = 1
	ai.HT.PlayedCards["QS"] = 1
	//ai.HT.Cards[0][C("KH")] = 1
	ai.HT.PlayedCards["KH"] = 1
	//ai.HT.Cards[0][C("QH")] = 1
	ai.HT.PlayedCards["QH"] = 1
	//ai.HT.Cards[1][C("9H")] = 2
	ai.HT.PlayedCards["9H"] = 0
	//ai.HT.Cards[2][C("KD")] = 1
	ai.HT.PlayedCards["KD"] = 1
	//ai.HT.Cards[2][C("QD")] = 1
	ai.HT.PlayedCards["QD"] = 1
	action := sdz.CreatePlayRequest(sdz.NACard, sdz.NASuit, sdz.Hearts, 3, ai.Hand())
	card := ai.findCardToPlay(action)
	t.True(card == C("AD") || card == C("QS"))
}

func (t *testSuite) TestFindCardToPlay() {
	//func (ai *AI) findCardToPlay(action *sdz.Action) sdz.Card {
	ai := createAI()
	ai.SetHand(nil, nil, nil, sdz.Hand{C("AD"), C("AD"), C("TD"), C("JD"), C("TC"), C("KC"), C("QC"), C("TH"), C("JH"), C("9H"), C("KS"), C("QS")}, 0, 3)
	ai.HT.Cards[0][C("KH")] = 1
	ai.HT.Cards[0][C("QH")] = 1
	ai.HT.Cards[1][C("9H")] = 1
	ai.HT.Cards[2][C("KD")] = 1
	ai.HT.Cards[2][C("QD")] = 1
	action := sdz.CreatePlayRequest(sdz.NACard, sdz.NASuit, sdz.Hearts, 3, ai.Hand())
	card := ai.findCardToPlay(action)
	t.True(card == C("TD"))
}

func (t *testSuite) TestAITracking() {
	ai := createAI()
	hand := sdz.Hand{C("9D"), C("9D"), C("QD"), C("TD"), C("TD"), C("AD"), C("JC"), C("QC"), C("KC"), C("AH"), C("AH"), C("KS")}
	ai.SetHand(nil, nil, nil, hand, 0, 0)
	ai.Trump = sdz.Spades
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{C("JD"), C("QS"), C("KD"), C("QD")}, 6, 1))
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{C("JD"), C("QS")}, 4, 2))
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{}, 0, 3))

	t.Equal(1, ai.HT.Cards[1][C("JD")])
	t.Equal(1, ai.HT.Cards[1][C("QS")])
	t.Equal(1, ai.HT.Cards[2][C("JD")])
	t.Equal(1, ai.HT.Cards[2][C("QS")])
	t.Equal(0, ai.HT.Cards[3][C("QS")])
	t.Equal(0, ai.HT.PlayedCards[C("JD")])
	t.Equal(0, ai.HT.PlayedCards[C("QS")])
	t.Equal(0, ai.HT.PlayedCards[C("QD")])
	t.Equal(1, ai.HT.Cards[1][C("QD")])
	t.Equal(0, ai.HT.Cards[2][C("QD")])
	t.Equal(0, ai.HT.Cards[3][C("QD")])
	t.Equal(1, ai.HT.Cards[1][C("KD")])

	ai.Trick.Lead = 1
	ai.Tell(nil, nil, nil, sdz.CreatePlay(C("JD"), 1))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(C("KD"), 2))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(C("AD"), 3))
	val, ok := ai.HT.Cards[1][C("JD")]
	t.True(ok)
	t.Equal(0, val)

	val, ok = ai.HT.Cards[1][C("JD")]
	t.True(ok)
	t.Equal(0, val)
	t.Equal(1, ai.HT.Cards[1][C("QS")])
	t.Equal(1, ai.HT.Cards[2][C("JD")])
	t.Equal(1, ai.HT.Cards[2][C("QS")])
	t.Equal(0, ai.HT.Cards[3][C("QS")])
	t.Equal(1, ai.HT.PlayedCards[C("JD")])
	t.Equal(0, ai.HT.PlayedCards[C("QS")])
	t.Equal(1, ai.HT.PlayedCards[C("KD")])
	t.Equal(1, ai.HT.PlayedCards[C("AD")])

	ai.Trick.Lead = 1
	ai.Tell(nil, nil, nil, sdz.CreatePlay(C("QD"), 1))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(C("9H"), 2))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(C("9H"), 3))
	val, ok = ai.HT.Cards[1][C("QD")]
	t.True(ok)
	t.Equal(0, val)

	ai = createAI()
	hand = sdz.Hand{C("9D"), C("9D"), C("QD"), C("TD"), C("TD"), C("AS"), C("JC"), C("QC"), C("KC"), C("AH"), C("AH"), C("KS")}
	ai.SetHand(nil, nil, nil, hand, 0, 0)
	ai.Trump = sdz.Spades
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{}, 0, 0))
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{}, 0, 1))
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{}, 0, 2))
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{}, 0, 3))
	//ai.calculate()
	ai.Trick.Lead = 1

	ai.Tell(nil, nil, nil, sdz.CreatePlay(C("JD"), 1))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(C("QD"), 2))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(C("KD"), 3))
	play := ai.Tell(nil, nil, nil, sdz.CreatePlayRequest(ai.Trick.winningCard(), ai.Trick.leadSuit(), ai.Trump, ai.PlayerID(), &sdz.Hand{}))
	t.Equal(C("TD"), play.PlayedCard)
	ai.Tell(nil, nil, nil, sdz.CreateTrick(1))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(C("JD"), 1))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(C("KD"), 2))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(C("KH"), 3))
	play = ai.Tell(nil, nil, nil, sdz.CreatePlayRequest(ai.Trick.winningCard(), ai.Trick.leadSuit(), ai.Trump, ai.PlayerID(), &sdz.Hand{}))
	t.Equal(C("TD"), play.PlayedCard)

	ai = createAI()
	hand = sdz.Hand{C("9D"), C("9D"), C("QD"), C("TD"), C("TD"), C("AD"), C("JC"), C("QC"), C("KC"), C("AH"), C("AH"), C("KS")}
	ai.SetHand(nil, nil, nil, hand, 0, 0)
	ai.Trump = sdz.Spades
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{C("JD"), C("JD"), C("QS"), C("QS"), C("KD"), C("QD")}, 32, 1))
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{}, 0, 2))
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{}, 0, 3))
	//ai.calculate()
	t.Equal(0, ai.HT.Cards[0][C("JD")])
	t.Equal(2, ai.HT.Cards[1][C("JD")])
	t.Equal(0, ai.HT.Cards[2][C("JD")])
	t.Equal(0, ai.HT.Cards[3][C("JD")])
}

func (t *testSuite) TestNoSuit() {
	ai := createAI()
	ai.RealHand = &sdz.Hand{C("9D"), C("9D"), C("QD"), C("TD"), C("TD"), C("AD"), C("JC"), C("QC"), C("KC"), C("AH"), C("AH"), C("KS")}
	ai.populate()
	ai.HT.Cards[1][C("9D")] = 1
	ai.HT.noSuit(1, sdz.Diamonds)

	t.Equal(0, ai.HT.Cards[1][C("9D")])
	t.Equal(0, ai.HT.Cards[1][C("JD")])
	t.Equal(0, ai.HT.Cards[1][C("QD")])
	t.Equal(0, ai.HT.Cards[1][C("KD")])
	t.Equal(0, ai.HT.Cards[1][C("TD")])
	t.Equal(0, ai.HT.Cards[1][C("AD")])
	t.Equal(0, ai.HT.Cards[1][C("AS")])
}

func (t *testSuite) TestCalculate() {
	hand := sdz.Hand{C("9D"), C("9D"), C("QD"), C("TD"), C("TD"), C("AD"), C("JC"), C("QC"), C("KC"), C("AH"), C("AH"), C("KS")}
	sort.Sort(hand)
	ai := createAI()
	// dealer 0, playerid 1
	ai.SetHand(nil, nil, nil, hand, 0, 1)
	for x := 0; x < 4; x++ {
		if x == 1 {
			t.Equal(ai.HT.Cards[x][C("9D")], 2)
		} else {
			t.Equal(ai.HT.Cards[x][C("9D")], 0)
		}
		_, ok := ai.HT.Cards[x][C("QD")]
		if x == 1 {
			t.True(ok, "Should have a record of 1 for QD")
		} else {
			t.Not(t.True(ok), "Should not have a record for QD")
		}
		_, ok = ai.HT.Cards[x][C("KH")]
		if x == 1 {
			t.True(ok, "Should have a record of a KH")
		} else {
			t.True(!ok, "Should not have a record of a KH")
		}
	}
	ai.HT.Cards[2][C("QD")] = 1
	ai.HT.Cards[2][C("KH")] = 1
	ai.HT.Cards[3][C("KH")] = 1
	ai.HT.Cards[3][C("KD")] = 1
	ai.HT.Cards[0][C("9S")] = 1
	ai.HT.PlayedCards[C("9S")] = 1
	ai.HT.PlayedCards[C("TS")] = 2
	ai.HT.PlayedCards[C("JS")] = 1
	ai.HT.Cards[0][C("JS")] = 0
	ai.HT.Cards[2][C("JS")] = 0
	ai.HT.PlayedCards[C("QS")] = 0
	ai.HT.Cards[0][C("QS")] = 0
	ai.HT.Cards[1][C("QS")] = 0
	ai.HT.Cards[2][C("QS")] = 0

	for _, card := range []sdz.Card{C("QD"), C("KH"), C("KD"), C("9S"), C("TS"), C("JS"), C("QS")} {
		ai.HT.calculateCard(card)
	}

	t.Equal(1, ai.HT.Cards[3][C("JS")])
	t.Equal(2, ai.HT.Cards[3][C("QS")])

	for x := 0; x < 4; x++ {
		_, ok := ai.HT.Cards[x][C("QD")]
		t.True(ok, "Value should be true for player "+strconv.Itoa(x)+" with QD")
		_, ok = ai.HT.Cards[x][C("KH")]
		t.True(ok, "Value should be true for player "+strconv.Itoa(x)+" with KH")
		_, ok = ai.HT.Cards[x][C("KD")]
		if x == 3 || x == 1 {
			t.True(ok, "Value should be true for player "+strconv.Itoa(x)+" with KD")
		} else {
			t.Not(t.True(ok), "Value should be false for player "+strconv.Itoa(x)+" with KD")
		}
		_, ok = ai.HT.Cards[x][C("9S")]
		t.True(ok, "All 9S cards should have been found")
		_, ok = ai.HT.Cards[x][C("TS")]
		t.True(ok, "All TS cards should have been found")
	}

	ai.HT.PlayedCards[C("JD")] = 0
	ai.HT.Cards[0][C("JD")] = 1
	ai.HT.Cards[1][C("JD")] = 0
	ai.HT.Cards[2][C("JD")] = 0
	ai.HT.calculateCard(C("JD"))
	val, ok := ai.HT.Cards[0][C("JD")]
	t.Equal(1, val)
	t.True(ok)
	val, ok = ai.HT.Cards[1][C("JD")]
	t.Equal(0, val)
	t.True(ok)
	val, ok = ai.HT.Cards[2][C("JD")]
	t.Equal(0, val)
	t.True(ok)
	val, ok = ai.HT.Cards[3][C("JD")]
	t.False(ok)
}
