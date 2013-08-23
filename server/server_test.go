// server_test.go
package sdzpinochleserver

import (
	"github.com/icub3d/appenginetesting"
	sdz "github.com/mzimmerman/sdzpinochle"
	pt "github.com/remogatto/prettytest"
	"sort"
	//"strconv"
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

func BenchmarkFullGame(b *testing.B) {
	c, err := appenginetesting.NewContext(&appenginetesting.Options{Debug: "critical"})
	if err != nil {
		b.Fatalf("Could not start up appenginetesting")
	}
	defer c.Close()
	for y := 0; y < b.N; y++ {
		game := NewGame(4)
		for x := 0; x < len(game.Players); x++ {
			game.Players[x] = createAI()
		}
		game.NextHand(nil, c)
	}
}

func (t *testSuite) TestPotentialCards() {
	ai := createAI()
	ht := ai.HT
	for card := range sdz.AllCards {
		ht.Cards[0][card] = 0
	}
	ht.Cards[0][index("AD")] = Unknown
	ht.Cards[0][index("TD")] = 1
	ht.Cards[0][index("KD")] = 2
	ht.Cards[0][index("QD")] = None

	potentials := potentialCards(0, ht, sdz.NACard, sdz.NASuit, sdz.Spades)
	Log(4, "Potentials = %s", potentials)
	t.True(potentials.Contains("AD"))
	t.True(potentials.Contains("TD"))
	t.True(potentials.Contains("KD"))
	t.Equal(ht.Cards[0][index("QD")], None)
	t.False(potentials.Contains("QD"))

	return

	ht.Cards[0][index("TS")] = Unknown
	potentials = potentialCards(0, ht, C("KD"), sdz.Diamonds, sdz.Spades)

	t.True(potentials.Contains("AD"))
	t.True(potentials.Contains("TD"))
	t.False(potentials.Contains("KD"))
	t.False(potentials.Contains("TS"))

	//func potentialCards(playerid, ht *HandTracker, winning sdz.Card, lead sdz.Suit, trump sdz.Suit) map[sdz.Card]int {
	ht.Cards[0][index("AH")] = Unknown
	ht.Cards[0][index("TH")] = Unknown
	ht.Cards[0][index("QH")] = Unknown
	ht.Cards[0][index("AS")] = 1
	ht.Cards[0][index("TS")] = 0
	potentials = potentialCards(0, ht, C("KH"), sdz.Hearts, sdz.Spades)
	t.True(potentials.Contains("AH"))
	t.True(potentials.Contains("TH"))
	t.True(potentials.Contains("AS"))
	t.True(potentials.Contains("QH"))
	t.False(potentials.Contains("TS"))

	ht.Cards[0][index("AH")] = 1
	potentials = potentialCards(0, ht, C("KH"), sdz.Hearts, sdz.Spades)
	t.True(potentials.Contains("AH"))
	t.True(potentials.Contains("TH"))
	t.False(potentials.Contains("AS"))
	t.False(potentials.Contains("QH"))
	t.False(potentials.Contains("TS"))

	ai = createAI()
	ht = ai.HT
	for card := range sdz.AllCards {
		ht.Cards[0][card] = 0
	}
	ht.Cards[0][index("AD")] = 2
	ht.Cards[0][index("TD")] = 1
	ht.Cards[0][index("JD")] = 1
	ht.Cards[0][index("TC")] = 1
	ht.Cards[0][index("KC")] = 1
	ht.Cards[0][index("QC")] = 1
	ht.Cards[0][index("TH")] = 1
	ht.Cards[0][index("JH")] = 1
	ht.Cards[0][index("9H")] = 1
	ht.Cards[0][index("KS")] = 1
	ht.Cards[0][index("QS")] = 1
	potentials = potentialCards(0, ht, sdz.NACard, sdz.NASuit, sdz.Hearts)
	t.Equal(11, len(potentials))

	ai = createAI()
	ht = ai.HT
	for card := range sdz.AllCards {
		ht.Cards[0][card] = 0
	}
	ht.Cards[0][index("AD")] = 2
	ht.Cards[0][index("TD")] = 1
	ht.Cards[0][index("JD")] = 1
	ht.Cards[0][index("TC")] = 1
	ht.Cards[0][index("KC")] = 1
	ht.Cards[0][index("QC")] = 1
	ht.Cards[0][index("TH")] = 1
	ht.Cards[0][index("JH")] = 1
	ht.Cards[0][index("9H")] = 1
	ht.Cards[0][index("KS")] = 1
	ht.Cards[0][index("QS")] = 1
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
	for card := range sdz.AllCards {
		ht.Cards[0][card] = 0
	}
	ht.Cards[0][index("TD")] = 2
	ht.Cards[0][index("9D")] = 2
	ht.Cards[0][index("QD")] = 1
	ht.Cards[0][index("JD")] = 1
	ht.Cards[0][index("JS")] = 1
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

func convert(oldMap map[sdz.Card]int) CardMap {
	var cm CardMap
	for card, val := range oldMap {
		cm[index(card)] = val
	}
	return cm
}

func (t *testSuite) TestPlayHandWithCard() {
	//func playHandWithCard(playerid int, ht *HandTracker, trick *Trick, trump sdz.Suit) (sdz.Card, [2]int) {
	ht := NewHandTracker(0)
	for x := 0; x < len(ht.Cards); x++ {
		for card := range sdz.AllCards {
			ht.Cards[x][card] = 3
		}
	}
	ht.PlayedCards = convert(map[sdz.Card]int{"AD": 0, "TD": 0, "KD": 0, "QD": 0, "JD": 2, "9D": 2, "AS": 2, "TS": 2, "KS": 2, "QS": 2, "JS": 2, "9S": 2, "AH": 2, "TH": 2, "KH": 2, "QH": 2, "JH": 2, "9H": 2, "AC": 2, "TC": 2, "KC": 2, "QC": 2, "JC": 2, "9C": 2})
	ht.Cards[0][index("AD")] = 1
	ht.Cards[1][index("TD")] = 1
	ht.Cards[2][index("KD")] = 1
	ht.Cards[3][index("QD")] = 1

	before := len(ht.Cards[0])
	card, value := playHandWithCard(0, ht, NewTrick(), sdz.Diamonds)
	t.Equal(before, len(ht.Cards[0]))
	t.Equal(card, sdz.CreateCard("D", "A"))
	t.Equal(value, 4)

	ht.Cards[1][index("AD")] = 1
	ht.Cards[2][index("TD")] = 1
	ht.Cards[3][index("KD")] = 1
	ht.Cards[0][index("QD")] = 1

	card, value = playHandWithCard(0, ht, NewTrick(), sdz.Diamonds)
	t.Equal(card, C("AD"))
	t.Equal(value, 3)

	ht.Cards[1][index("AD")] = 1
	ht.Cards[2][index("TD")] = 1
	ht.Cards[3][index("KD")] = 1
	ht.Cards[0][index("QD")] = 1

	ht.PlayedCards[index("AS")] = 1
	ht.Cards[0][index("AS")] = 1
	ht.PlayedCards[index("TS")] = 1
	ht.Cards[1][index("TS")] = 1
	ht.PlayedCards[index("QS")] = 1
	ht.Cards[2][index("QS")] = 1
	ht.PlayedCards[index("JS")] = 1
	ht.Cards[3][index("JS")] = 1

	card, value = playHandWithCard(0, ht, NewTrick(), sdz.Diamonds)
	t.Equal(card, C("AD"))
	t.Equal(value, 6)

	ht.PlayedCards[index("AC")] = None
	ht.PlayedCards[index("TC")] = None

	ht.Cards[0][index("AC")] = 1
	ht.Cards[1][index("AC")] = Unknown
	ht.Cards[2][index("AC")] = Unknown
	ht.Cards[3][index("AC")] = Unknown
	ht.Cards[1][index("TC")] = Unknown
	ht.Cards[2][index("TC")] = Unknown
	ht.Cards[3][index("TC")] = Unknown

	card, value = playHandWithCard(0, ht, NewTrick(), sdz.Diamonds)
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
	ai.HT.PlayedCards[index("AD")] = 1
	ai.HT.PlayedCards[index("QS")] = 1
	ai.HT.PlayedCards[index("KH")] = 1
	ai.HT.PlayedCards[index("QH")] = 1
	ai.HT.PlayedCards[index("9H")] = None
	ai.HT.PlayedCards[index("KD")] = 1
	ai.HT.PlayedCards[index("QD")] = 1
	action := sdz.CreatePlayRequest(sdz.NACard, sdz.NASuit, sdz.Hearts, 3, ai.Hand())
	card := ai.findCardToPlay(action)
	t.True(card == C("AD") || card == C("QS"))
}

func (t *testSuite) TestFindCardToPlay() {
	//func (ai *AI) findCardToPlay(action *sdz.Action) sdz.Card {
	ai := createAI()
	ai.SetHand(nil, nil, nil, sdz.Hand{C("AD"), C("AD"), C("TD"), C("JD"), C("TC"), C("KC"), C("QC"), C("TH"), C("JH"), C("9H"), C("KS"), C("QS")}, 0, 3)
	ai.Trump = sdz.Hearts
	ai.HT.Cards[0][index("KH")] = 1
	ai.HT.Cards[0][index("QH")] = 1
	ai.HT.Cards[1][index("9H")] = 1
	ai.HT.Cards[2][index("KD")] = 1
	ai.HT.Cards[2][index("QD")] = 1
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

	t.Equal(1, ai.HT.Cards[1][index("JD")])
	t.Equal(1, ai.HT.Cards[1][index("QS")])
	t.Equal(1, ai.HT.Cards[2][index("JD")])
	t.Equal(1, ai.HT.Cards[2][index("QS")])
	t.Equal(None, ai.HT.Cards[3][index("QS")])
	t.Equal(None, ai.HT.PlayedCards[index("JD")])
	t.Equal(None, ai.HT.PlayedCards[index("QS")])
	t.Equal(None, ai.HT.PlayedCards[index("QD")])
	t.Equal(1, ai.HT.Cards[1][index("QD")])
	t.Equal(None, ai.HT.Cards[2][index("QD")])
	t.Equal(None, ai.HT.Cards[3][index("QD")])
	t.Equal(1, ai.HT.Cards[1][index("KD")])

	ai.Trick.Lead = 1
	ai.Tell(nil, nil, nil, sdz.CreatePlay(C("JD"), 1))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(C("KD"), 2))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(C("AD"), 3))
	val := ai.HT.Cards[1][index("JD")]
	t.Equal(None, val)

	val = ai.HT.Cards[1][index("JD")]
	t.Equal(None, val)
	t.Equal(1, ai.HT.Cards[1][index("QS")])
	t.Equal(1, ai.HT.Cards[2][index("JD")])
	t.Equal(1, ai.HT.Cards[2][index("QS")])
	t.Equal(None, ai.HT.Cards[3][index("QS")])
	t.Equal(1, ai.HT.PlayedCards[index("JD")])
	t.Equal(None, ai.HT.PlayedCards[index("QS")])
	t.Equal(1, ai.HT.PlayedCards[index("KD")])
	t.Equal(1, ai.HT.PlayedCards[index("AD")])

	ai.Trick.Lead = 1
	ai.Tell(nil, nil, nil, sdz.CreatePlay(C("QD"), 1))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(C("9H"), 2))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(C("9H"), 3))
	val = ai.HT.Cards[1][index("QD")]
	t.Equal(None, val)

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
	t.Equal(None, ai.HT.Cards[0][index("JD")])
	t.Equal(2, ai.HT.Cards[1][index("JD")])
	t.Equal(None, ai.HT.Cards[2][index("JD")])
	t.Equal(None, ai.HT.Cards[3][index("JD")])
}

func (t *testSuite) TestNoSuit() {
	ht := NewHandTracker(0)
	ht.Cards[1][index("9D")] = 1
	ht.noSuit(1, sdz.Diamonds)

	t.Equal(None, ht.Cards[1][index("9D")])
	t.Equal(None, ht.Cards[1][index("JD")])
	t.Equal(None, ht.Cards[1][index("QD")])
	t.Equal(None, ht.Cards[1][index("KD")])
	t.Equal(None, ht.Cards[1][index("TD")])
	t.Equal(None, ht.Cards[1][index("AD")])
	t.Equal(Unknown, ht.Cards[1][index("AS")])
}

func (t *testSuite) TestCalculate() {
	hand := sdz.Hand{C("9D"), C("9D"), C("QD"), C("TD"), C("TD"), C("AD"), C("JC"), C("QC"), C("KC"), C("AH"), C("AH"), C("KS")}
	sort.Sort(hand)
	ai := createAI()
	// dealer 0, playerid 1
	ai.SetHand(nil, nil, nil, hand, 0, 1)
	for x := 0; x < 4; x++ {
		if x == 1 {
			t.Equal(ai.HT.Cards[x][index("9D")], 2)
		} else {
			t.Equal(ai.HT.Cards[x][index("9D")], None)
		}
		val := ai.HT.Cards[x][index("QD")]
		if x == 1 {
			t.Equal(val, 1)
		} else {
			t.Equal(val, Unknown)
		}
		val = ai.HT.Cards[x][index("KH")]
		if x == 1 {
			t.Equal(val, None)
		} else {
			t.Equal(val, Unknown)
		}
	}
	ai.HT.Cards[2][index("QD")] = 1
	ai.HT.Cards[2][index("KH")] = 1
	ai.HT.Cards[3][index("KH")] = 1
	ai.HT.Cards[3][index("KD")] = 1
	ai.HT.Cards[0][index("9S")] = 1
	ai.HT.PlayedCards[index("9S")] = 1
	ai.HT.PlayedCards[index("TS")] = 2
	ai.HT.PlayedCards[index("JS")] = 1
	ai.HT.Cards[0][index("JS")] = None
	ai.HT.Cards[2][index("JS")] = None
	ai.HT.PlayedCards[index("QS")] = None
	ai.HT.Cards[0][index("QS")] = None
	ai.HT.Cards[1][index("QS")] = None
	ai.HT.Cards[2][index("QS")] = None

	for _, card := range []sdz.Card{C("QD"), C("KH"), C("KD"), C("9S"), C("TS"), C("JS"), C("QS")} {
		ai.HT.calculateCard(index(card))
	}

	t.Equal(1, ai.HT.Cards[3][index("JS")])
	t.Equal(2, ai.HT.Cards[3][index("QS")])

	for x := 0; x < 4; x++ {
		val := ai.HT.Cards[x][index("QD")]
		t.Not(t.Equal(val, Unknown))
		val = ai.HT.Cards[x][index("KH")]
		t.Not(t.Equal(val, Unknown))
		val = ai.HT.Cards[x][index("KD")]
		if x == 3 || x == 1 {
			t.Not(t.Equal(val, Unknown))
		} else {
			t.Equal(val, Unknown)
		}
		val = ai.HT.Cards[x][index("9S")]
		t.Not(t.Equal(val, Unknown))
		val = ai.HT.Cards[x][index("TS")]
		t.Not(t.Equal(val, Unknown))
	}

	ai.HT.PlayedCards[index("JD")] = None
	ai.HT.Cards[0][index("JD")] = 1
	ai.HT.Cards[1][index("JD")] = None
	ai.HT.Cards[2][index("JD")] = None
	ai.HT.calculateCard(index("JD"))
	val := ai.HT.Cards[0][index("JD")]
	t.Equal(1, val)
	val = ai.HT.Cards[1][index("JD")]
	t.Equal(None, val)
	val = ai.HT.Cards[2][index("JD")]
	t.Equal(None, val)
	val = ai.HT.Cards[3][index("JD")]
	t.Equal(val, 1)
}
