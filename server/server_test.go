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

const (
	AS     = iota
	TS     = iota
	KS     = iota
	QS     = iota
	JS     = iota
	NS     = iota
	AH     = iota
	TH     = iota
	KH     = iota
	QH     = iota
	JH     = iota
	NH     = iota
	AC     = iota
	TC     = iota
	KC     = iota
	QC     = iota
	JC     = iota
	NC     = iota
	AD     = iota
	TD     = iota
	KD     = iota
	QD     = iota
	JD     = iota
	ND     = iota
	NACard = -1
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

func (t *testSuite) TestPotentialCardsShort() {
	ht := new(HandTracker)
	ht.reset(0)
	ht.Cards[0][AD] = Unknown
	ht.Cards[0][TD] = 1
	ht.Cards[0][KD] = 2
	ht.Cards[0][QD] = None

	potentials := ht.potentialCards(0, sdz.NACard, sdz.NASuit, sdz.Spades, false)

	t.True(potentials.Contains(AD))
	t.True(potentials.Contains(TD))
	t.True(potentials.Contains(KD))
	t.Equal(ht.Cards[0][QD], None)
	t.False(potentials.Contains(QD))

	ht.Cards[0][TS] = Unknown
	potentials = ht.potentialCards(0, KD, sdz.Diamonds, sdz.Spades, false)

	t.True(potentials.Contains(AD))
	t.True(potentials.Contains(TD))
	t.False(potentials.Contains(KD))
	t.False(potentials.Contains(TS))

	ht.Cards[0][AH] = Unknown
	ht.Cards[0][TH] = Unknown
	ht.Cards[0][QH] = Unknown
	ht.Cards[0][AS] = 1
	ht.Cards[0][TS] = None
	potentials = ht.potentialCards(0, KH, sdz.Hearts, sdz.Spades, false)
	t.True(potentials.Contains(AH))
	t.True(potentials.Contains(TH))
	t.True(potentials.Contains(AS))
	t.True(potentials.Contains(QH))
	t.False(potentials.Contains(TS))

	ht.Cards[0][AH] = 1
	potentials = ht.potentialCards(0, KH, sdz.Hearts, sdz.Spades, false)
	t.True(potentials.Contains(AH))
	t.True(potentials.Contains(TH))
	t.False(potentials.Contains(AS))
	t.False(potentials.Contains(QH))
	t.False(potentials.Contains(TS))

	ht.reset(0)
	ht.Cards[0][AD] = 2
	ht.Cards[0][TD] = 1
	ht.Cards[0][JD] = 1
	ht.Cards[0][TC] = 1
	ht.Cards[0][KC] = 1
	ht.Cards[0][QC] = 1
	ht.Cards[0][TH] = 1
	ht.Cards[0][JH] = 1
	ht.Cards[0][NH] = 1
	ht.Cards[0][KS] = 1
	ht.Cards[0][QS] = 1
	potentials = ht.potentialCards(0, sdz.NACard, sdz.NASuit, sdz.Hearts, false)
	t.Equal(11, len(potentials))

	ht.reset(0)
	ht.Cards[0][AD] = 2
	ht.Cards[0][TD] = 1
	ht.Cards[0][JD] = 1
	ht.Cards[0][TC] = 1
	ht.Cards[0][KC] = 1
	ht.Cards[0][QC] = 1
	ht.Cards[0][TH] = 1
	ht.Cards[0][JH] = 1
	ht.Cards[0][NH] = 1
	ht.Cards[0][KS] = 1
	ht.Cards[0][QS] = 1
	potentials = ht.potentialCards(0, sdz.NACard, sdz.NASuit, sdz.Hearts, false)
	t.Equal(11, len(potentials))

	//Playerid:2, Bid:0, PlayedCard:"", WinningCard:"JS", Lead:"S", Trump:"C", Amount:0, Message:"", Hand:sdzpinochle.Hand{"TD", "TD", "QD", "JD", "ND", "ND", "JS"}, Option:0, GameOver:false, Win:false, Score:[]int(nil), Dealer:0, WinningPlayer:0}

	//Starting calculate() - map[KD:1 NS:1 QS:1 JH:2 ND:0 AC:2 JS:1 JD:0 KS:1 QD:0 JC:0 AD:0 KH:1 QC:1 NC:2 QH:1 TH:1 TD:1 AH:2 AS:1 KC:0 TS:0 TC:1 NH:2]
	//Player0 - map[JS:0 NH:0 ND:0 AH:0 KH:0 KS:0 QC:0 NC:0 AC:0 KC:1 TD:0 JH:0 KD:1]
	//Player1 - map[KD:0 QC:0 AC:0 JS:0 JC:0 TC:0 NC:0 NH:0 ND:0 TD:0 KC:0 JH:0 AH:0]
	//Player2 - map[TH:0 JH:0 JD:1 KS:0 QS:0 NC:0 JS:1 AD:0 QD:1 TS:0 NH:0 AS:0 KD:0 TD:1 KC:0 TC:0 NS:0 AH:0 ND:2 JC:0 QH:0 AC:0 QC:0 KH:0]
	//Player3 - map[JH:0 AH:0 KD:0 NC:0 JS:0 KC:1 TD:0 AC:0 NH:0 QC:1 ND:0]
	ht.reset(0)
	ht.Cards[0][TD] = 2
	ht.Cards[0][ND] = 2
	ht.Cards[0][QD] = 1
	ht.Cards[0][JD] = 1
	ht.Cards[0][JS] = 1
	potentials = ht.potentialCards(0, JS, sdz.Spades, sdz.Clubs, false)
	t.Equal(1, len(potentials))
	t.True(potentials.Contains(JS))
	t.False(potentials.Contains(TD))

	ht.reset(0)
	ht.Cards[0][AD] = Unknown
	ht.Cards[0][TD] = 1
	ht.Cards[0][KD] = 1
	ht.Cards[0][QD] = 1
	ht.Cards[0][JD] = 1
	// follow suit and lose
	potentials = ht.potentialCards(0, AD, sdz.Diamonds, sdz.Spades, false)
	t.False(potentials.Contains(AD))
	t.False(potentials.Contains(TD))
	t.True(potentials.Contains(KD))
	t.False(potentials.Contains(QD))
	t.True(potentials.Contains(JD))

	// play trump and lose
	potentials = ht.potentialCards(0, AD, sdz.Spades, sdz.Diamonds, false)
	t.False(potentials.Contains(AD))
	t.False(potentials.Contains(TD))
	t.True(potentials.Contains(KD))
	t.False(potentials.Contains(QD))
	t.True(potentials.Contains(JD))

	// follow suit and win
	potentials = ht.potentialCards(0, ND, sdz.Diamonds, sdz.Spades, false)
	t.True(potentials.Contains(AD))
	t.True(potentials.Contains(TD))
	t.True(potentials.Contains(KD))
	t.True(potentials.Contains(QD))
	t.True(potentials.Contains(JD))

	// play trump and win
	potentials = ht.potentialCards(0, ND, sdz.Spades, sdz.Diamonds, false)
	t.True(potentials.Contains(AD))
	t.True(potentials.Contains(TD))
	t.True(potentials.Contains(KD))
	t.True(potentials.Contains(QD))
	t.True(potentials.Contains(JD))

	// now we do the 4 blocks over again, but this time it's the last play
	ht.reset(0)
	ht.Cards[0][AD] = Unknown
	ht.Cards[0][TD] = 1
	ht.Cards[0][KD] = 1
	ht.Cards[0][QD] = 1
	ht.Cards[0][JD] = 1
	// follow suit and lose
	potentials = ht.potentialCards(0, AD, sdz.Diamonds, sdz.Spades, true)
	t.False(potentials.Contains(AD))
	t.False(potentials.Contains(TD))
	t.True(potentials.Contains(KD))
	t.False(potentials.Contains(QD))
	t.True(potentials.Contains(JD))

	// play trump and lose
	potentials = ht.potentialCards(0, AD, sdz.Spades, sdz.Diamonds, true)
	t.False(potentials.Contains(AD))
	t.False(potentials.Contains(TD))
	t.True(potentials.Contains(KD))
	t.False(potentials.Contains(QD))
	t.True(potentials.Contains(JD))

	// follow suit and win
	potentials = ht.potentialCards(0, ND, sdz.Diamonds, sdz.Spades, true)
	t.False(potentials.Contains(AD))
	t.False(potentials.Contains(TD))
	t.True(potentials.Contains(KD))
	t.False(potentials.Contains(QD))
	t.True(potentials.Contains(JD))

	// play trump and win
	potentials = ht.potentialCards(0, ND, sdz.Spades, sdz.Diamonds, true)
	t.False(potentials.Contains(AD))
	t.False(potentials.Contains(TD))
	t.True(potentials.Contains(KD))
	t.False(potentials.Contains(QD))
	t.True(potentials.Contains(JD))

	//PotentialCards called with 0,winning=AS,lead=D,trump=C,
	//ht=&main.HandTracker{cards:[4]map[sdzpinochle.Card]int{map[sdzpinochle.Card]int{"NC":0, "AC":0, "AS":0, "KD":0, "QD":0, "QS":0, "TH":0, "JC":0, "AD":0, "ND":0, "KC":0, "TS":0, "NH":0, "TC":0, "TD":0, "QH":0, "NS":0, "JD":0, "QC":0, "KH":0, "JH":0, "KS":0, "AH":0, "JS":0}, map[sdzpinochle.Card]int{"NC":0, "QC":0, "ND":0, "AH":0, "AC":0, "QH":1, "JD":0, "KS":0, "JC":0, "AS":0, "KC":0, "TH":0, "TC":0, "QS":0, "KH":0, "TS":0}, map[sdzpinochle.Card]int{"KS":0, "JD":0, "JC":0, "TH":0, "KH":0, "AS":0, "QD":0, "TC":0, "AC":0, "AH":0, "QH":0, "NC":0, "KD":1, "QC":0, "ND":0, "KC":0, "QS":0, "TS":0}, map[sdzpinochle.Card]int{"TH":0, "AS":0, "AH":0, "TS":0, "KC":0, "NC":0, "QS":0, "TC":0, "ND":0, "AC":0, "KS":0, "QC":0, "KH":0, "JD":0, "QH":0, "JC":0}}, playedCards:map[sdzpinochle.Card]int{"KH":2, "KD":0, "JC":2, "AH":2, "TH":2, "TD":1, "NS":1, "TC":2, "ND":2, "AS":2, "KS":2, "JS":1, "QC":2, "KC":2, "QD":1, "QS":2, "NH":1, "QH":1, "AD":0, "JD":2, "AC":2, "JH":1, "NC":2, "TS":2}}
	//ht = NewHandTracker(0, make(sdz.Hand, 0))
	//for _, card := range sdz.AllCards() {
	//	ht.Cards[0][card] = 0
	//}
	//ht.Cards[0][TD] = 2
	//ht.Cards[0][ND] = 2
	//ht.Cards[0][QD] = 1
	//ht.Cards[0][JD] = 1
	//ht.Cards[0][JS] = 1
	//potentials = potentialCards(0, ht, JS, sdz.Spades, sdz.Clubs)
	//t.Equal(1, len(potentials))
	//t.True(potentials[JS])
	//t.False(potentials[TD])

}

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

func (t *testSuite) TestFindCardToPlayShort() {
	//func (ai *AI) findCardToPlay(action *sdz.Action) sdz.Card {
	ai := createAI()
	ai.SetHand(nil, nil, nil, sdz.Hand{AD, QS}, 0, 3)
	for card := range ai.HT.PlayedCards {
		ai.HT.PlayedCards[card] = 2
	}
	ai.HT.PlayedCards[AD] = 1
	ai.HT.PlayedCards[QS] = 1
	ai.HT.PlayedCards[KH] = 1
	ai.HT.PlayedCards[QH] = 1
	ai.HT.PlayedCards[NH] = None
	ai.HT.PlayedCards[KD] = 1
	ai.HT.PlayedCards[QD] = 1
	action := sdz.CreatePlayRequest(sdz.NACard, sdz.NASuit, sdz.Hearts, 3, ai.Hand())
	card := ai.findCardToPlay(action)
	t.True(card == AD || card == QS)
}

func (t *testSuite) TestFindCardToPlay() {
	//func (ai *AI) findCardToPlay(action *sdz.Action) sdz.Card {
	ai := createAI()
	ai.SetHand(nil, nil, nil, sdz.Hand{AD, AD, TD, JD, TC, KC, QC, TH, JH, NH, KS, QS}, 0, 3)
	ai.Trump = sdz.Hearts
	ai.HT.Cards[0][KH] = 1
	ai.HT.Cards[0][QH] = 1
	ai.HT.Cards[1][NH] = 1
	ai.HT.Cards[2][KD] = 1
	ai.HT.Cards[2][QD] = 1
	action := sdz.CreatePlayRequest(sdz.NACard, sdz.NASuit, sdz.Hearts, 3, ai.Hand())
	card := ai.findCardToPlay(action)
	t.True(card == TD)
}

func (t *testSuite) TestAITracking() {
	ai := createAI()
	hand := sdz.Hand{ND, ND, QD, TD, TD, AD, JC, QC, KC, AH, AH, KS}
	ai.SetHand(nil, nil, nil, hand, 0, 0)
	ai.Trump = sdz.Spades
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{JD, QS, KD, QD}, 6, 1))
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{JD, QS}, 4, 2))
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{}, 0, 3))

	t.Equal(1, ai.HT.Cards[1][JD])
	t.Equal(1, ai.HT.Cards[1][QS])
	t.Equal(1, ai.HT.Cards[2][JD])
	t.Equal(1, ai.HT.Cards[2][QS])
	t.Equal(None, ai.HT.Cards[3][QS])
	t.Equal(None, ai.HT.PlayedCards[JD])
	t.Equal(None, ai.HT.PlayedCards[QS])
	t.Equal(None, ai.HT.PlayedCards[QD])
	t.Equal(1, ai.HT.Cards[1][QD])
	t.Equal(None, ai.HT.Cards[2][QD])
	t.Equal(None, ai.HT.Cards[3][QD])
	t.Equal(1, ai.HT.Cards[1][KD])

	ai.Trick.Lead = 1
	ai.Tell(nil, nil, nil, sdz.CreatePlay(JD, 1))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(KD, 2))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(AD, 3))
	val := ai.HT.Cards[1][JD]
	t.Equal(None, val)

	val = ai.HT.Cards[1][JD]
	t.Equal(None, val)
	t.Equal(1, ai.HT.Cards[1][QS])
	t.Equal(1, ai.HT.Cards[2][JD])
	t.Equal(1, ai.HT.Cards[2][QS])
	t.Equal(None, ai.HT.Cards[3][QS])
	t.Equal(1, ai.HT.PlayedCards[JD])
	t.Equal(None, ai.HT.PlayedCards[QS])
	t.Equal(1, ai.HT.PlayedCards[KD])
	t.Equal(1, ai.HT.PlayedCards[AD])

	ai.Trick.Lead = 1
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
	//ai.calculate()
	ai.Trick.Lead = 1

	ai.Tell(nil, nil, nil, sdz.CreatePlay(JD, 1))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(QD, 2))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(KD, 3))
	play := ai.Tell(nil, nil, nil, sdz.CreatePlayRequest(ai.Trick.winningCard(), ai.Trick.leadSuit(), ai.Trump, ai.PlayerID(), &sdz.Hand{}))
	t.Equal(TD, play.PlayedCard)
	ai.Tell(nil, nil, nil, sdz.CreateTrick(1))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(JD, 1))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(KD, 2))
	ai.Tell(nil, nil, nil, sdz.CreatePlay(KH, 3))
	play = ai.Tell(nil, nil, nil, sdz.CreatePlayRequest(ai.Trick.winningCard(), ai.Trick.leadSuit(), ai.Trump, ai.PlayerID(), &sdz.Hand{}))
	t.Equal(TD, play.PlayedCard)

	ai = createAI()
	hand = sdz.Hand{ND, ND, QD, TD, TD, AD, JC, QC, KC, AH, AH, KS}
	ai.SetHand(nil, nil, nil, hand, 0, 0)
	ai.Trump = sdz.Spades
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{JD, JD, QS, QS, KD, QD}, 32, 1))
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{}, 0, 2))
	ai.Tell(nil, nil, nil, sdz.CreateMeld(sdz.Hand{}, 0, 3))
	//ai.calculate()
	t.Equal(None, ai.HT.Cards[0][JD])
	t.Equal(2, ai.HT.Cards[1][JD])
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

func (t *testSuite) TestCalculateShort() {
	hand := sdz.Hand{ND, ND, QD, TD, TD, AD, JC, QC, KC, AH, AH, KS}
	sort.Sort(hand)
	ai := createAI()
	// dealer 0, playerid 1
	ai.SetHand(nil, nil, nil, hand, 0, 1)
	for x := 0; x < 4; x++ {
		if x == 1 {
			t.Equal(ai.HT.Cards[x][ND], 2)
		} else {
			t.Equal(ai.HT.Cards[x][ND], None)
		}
		val := ai.HT.Cards[x][QD]
		if x == 1 {
			t.Equal(val, 1)
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

	t.Equal(1, ai.HT.Cards[3][JS])
	t.Equal(2, ai.HT.Cards[3][QS])

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
	t.Equal(1, val)
	val = ai.HT.Cards[1][JD]
	t.Equal(None, val)
	val = ai.HT.Cards[2][JD]
	t.Equal(None, val)
	val = ai.HT.Cards[3][JD]
	t.Equal(val, 1)
}
