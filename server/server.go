package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"runtime"
	"sort"
	"time"

	. "github.com/mzimmerman/sdzpinochle"
)

const (
	StateNew   = "new"
	StateBid   = "bid"
	StateTrump = "trump"
	StateMeld  = "meld"
	StatePlay  = "play"
	cookieName = "sdzpinochle"
	debugLog   = false
	Nothing    = iota
	TrumpLose
	TrumpWin
	FollowLose
	FollowWin
	None    = uint8(0)
	Unknown = uint8(3)
)

//var sem = make(chan bool, runtime.NumCPU())

var Hands = make(chan Hand, 1000)

var logBuffer bytes.Buffer

func init() {
	rand.Seed(0)
}

func getHand() Hand {
	var h Hand
	select {
	case h = <-Hands:
		h = h[:0] // empty the slice
	default:
		h = make(Hand, 0, 24)
	}
	return h
}

func getHT(owner uint8) (*HandTracker, error) {
	return htstack.Pop()
}

func (ai *AI) populate() {
	ai.HT.reset(ai.Playerid)
	for _, card := range *ai.RealHand {
		ai.HT.Cards[ai.Playerid].inc(card)
		ai.HT.calculateCard(card)
	}
	ai.HT.calculateHand(ai.Playerid)
}

func (ht *HandTracker) noSuit(playerid uint8, suit Suit) {
	//Log(ht.Owner, "No suit start on %s", suit)
	card := CreateCard(suit, Ace)
	for x := 0; x < 6; x++ {
		//Log(ht.Owner, "Card=%s", card)
		ht.Cards[playerid][card] = None
		ht.calculateCard(card)
		card++
	}
	//Log(ht.Owner, "No suit end")
}

func (ht *HandTracker) calculateHand(hand uint8) (totalCards uint8) {
	for x := AS; int8(x) <= AllCards; x++ {
		if ht.Cards[hand][x] != Unknown {
			totalCards += ht.Cards[hand][x]
		}
	}
	if totalCards > 12 {
		Log(ht.Owner, "Player %d has more than 12 cards!", hand)
		panic("Player has more than 12 cards")
	}
	if totalCards == 12 {
		for x := AS; int8(x) <= AllCards; x++ {
			if ht.Cards[hand][x] == Unknown {
				ht.Cards[hand][x] = None
				ht.calculateCard(x)
			}
		}
	}
	return totalCards
}

func (ht *HandTracker) calculateCard(cardIndex Card) {
	sum := ht.sum(cardIndex)
	//if cardIndex == TH {
	//	Log(ht.Owner, "htcardset - Sum for %s is %d", cardIndex, sum)
	//	debug.PrintStack()
	//}
	if sum > 2 || sum < 0 {
		Log(ht.Owner, "htcardset - Card=%s,sum=%d", cardIndex, sum)
		Log(ht.Owner, "ht.PlayedCards = %s", ht.PlayedCards)
		for x := 0; x < 4; x++ {
			Log(ht.Owner, "Player%d - %s", x, ht.Cards[x])
		}
		panic("Cannot have more cards than 2 or less than 0 - " + string(sum))
	}
	if sum == 2 {
		for x := 0; x < 4; x++ {
			if val := ht.Cards[x][cardIndex]; val == Unknown {
				ht.Cards[x][cardIndex] = None
			}
		}
	} else {
		//TODO: implement "has at least one" status
		//unknown := -1
		//hasCard := -1
		//for x := 0; x < 4; x++ {
		//	if sum == 1 && ht.Cards[x][cardIndex] == 1 {
		//		hasCard = x
		//	}
		//	if val := ht.Cards[x][cardIndex]; val == Unknown {
		//		if unknown == -1 {
		//			unknown = x
		//		} else {
		//			// at least two unknowns
		//			unknown = -1
		//			hasCard = -1
		//			break
		//		}
		//	}
		//}
		////Log(4, "unknown = %d", unknown)
		//if unknown >= 0 {
		//	//if ht.PlayedCards[cardIndex] > 0 || ht.Cards[ht.Owner][cardIndex] == 1 {
		//	ht.Cards[unknown][cardIndex] = 2 - sum
		//	if ht == oright && unknown == 2 {
		//		Log(ht.Owner, "Set playerid=%d to have card %s at value = %d", unknown, cardIndex, 2-sum)
		//	}
		//	//} else if sum == 0 {
		//	//ht.Cards[unknown][cardIndex] = 2
		//	//}
		//} else if hasCard >= 0 && sum == 1 {
		//	//Log(ht.Owner, "Setting ht.Cards[%d][%s] to 2", hasCard, cardIndex)
		//	ht.Cards[hasCard][cardIndex] = 2
		//	if ht == oright && hasCard == 2 {
		//		Log(ht.Owner, "Playerid=%d is the only one to have card %s, set value to 2", hasCard, cardIndex)
		//	}
		//}
	}

	//if cardIndex == QC {
	//	if ht.PlayedCards[cardIndex] != Unknown {
	//		if ht.PlayedCards[cardIndex] == None {
	//			Log(ht.Owner, "PC[%s]=%d", cardIndex, 0)
	//		} else {
	//			Log(ht.Owner, "PC[%s]=%d", cardIndex, ht.PlayedCards[cardIndex])
	//		}
	//	}
	//	for x := 0; x < 4; x++ {
	//		if val := ht.Cards[x][cardIndex]; val != Unknown {
	//			if val == None {
	//				Log(ht.Owner, "P%d[%s]=%d", x, cardIndex, 0)
	//			} else {
	//				Log(ht.Owner, "P%d[%s]=%d", x, cardIndex, ht.Cards[x][cardIndex])
	//			}
	//		}
	//	}
	//}
}

type AI struct {
	RealHand        *Hand
	Trump           Suit
	BiddingStrategy BiddingStrategy
	PlayingStrategy PlayingStrategy
	BidAmount       uint8
	HighBid         uint8
	HighBidder      uint8
	NumBidders      uint8
	Bids            []uint8
	PlayerImpl
	HT    *HandTracker
	name  string
	score [2]uint8 // current score of the game
}

func (ai *AI) Name() string {
	return ai.name
}

func (a *AI) reset() {
	var err error
	if a.HT == nil {
		a.HT, err = getHT(a.Playerid)
		if err != nil {
			panic("not going to run out of memory here right?!")
		}
	}
	a.name = ""
	a.HT.reset(a.Playerid)
}

func CreateAI(bs BiddingStrategy, ps PlayingStrategy, name string) (a *AI) {
	a = new(AI)
	a.reset()
	a.BiddingStrategy = bs
	a.PlayingStrategy = ps
	a.name = name
	return a
}

func powerBid(realHand *Hand, suit Suit) (count uint8) {
	count = 5 // your partner's good for at least this right?!?
	suitMap := make(map[Suit]int)
	for _, card := range *realHand {
		suitMap[card.Suit()]++
		if card.Suit() == suit {
			switch card.Face() {
			case Ace:
				count += 3
			case Ten:
				count += 2
			case King:
				fallthrough
			case Queen:
				fallthrough
			case Jack:
				fallthrough
			case Nine:
				count += 1
			}
		} else if card.Face() == Ace {
			count += 2
		} else if card.Face() == Jack || card.Face() == Nine {
			count -= 1
		}
	}
	for _, x := range Suits {
		if x == suit {
			continue
		}
		if suitMap[x] == 0 {
			count++
		}
	}
	return
}

func (ai AI) calculateBid() (amount uint8, trump Suit) {
	return ai.BiddingStrategy(ai.RealHand, ai.Bids, ai.score)
}

func max(a, b uint8) uint8 {
	if a > b {
		return a
	}
	return b
}

func min(a, b uint8) uint8 {
	if a < b {
		return a
	}
	return b
}

type Trick struct {
	Played        [4]Card
	WinningPlayer uint8
	Lead          uint8
	Plays         uint8
	Next          uint8 // the next player that needs to play
}

func (t *Trick) PlayCard(card Card, trump Suit) {
	if t.Plays == 4 {
		t.reset()
	}
	t.Played[t.Next] = card
	if t.Plays == 0 {
		t.Lead = t.Next
		t.WinningPlayer = t.Next
	} else if card.Beats(t.Played[t.WinningPlayer], trump) {
		t.WinningPlayer = t.Next
	}
	t.Plays++
	t.Next = (t.Next + 1) % uint8(len(t.Played))
	if t.Plays == 4 {
		t.Next = t.WinningPlayer
	}
	//Log(4, "After trick.PlayCard - %s", t)
	//Log(4, "After trick.PlayCard - %#v", t)
}

func (t *Trick) reset() {
	t.Plays = 0
}

func (pw *PlayWalker) Worth(trump Suit) (worth int8) {
	worth = int8(pw.Counters[pw.Me%2]) * 3
	count := int8(0)
	face := NAFace
	suit := NASuit
	for card := AS; int8(card) <= AllCards; card++ {
		// teamate
		suit = card.Suit()
		face = card.Face()
		count = pw.TeamCards[pw.Me%2].Count(card)
		if suit == trump {
			worth -= count
		}
		if face == Ace {
			worth -= count * 2
		} else if face == Ten {
			worth -= count
		}
		// opponent
		count = pw.TeamCards[(pw.Me+1)%2].Count(card)
		if suit == trump {
			worth += count
		}
		if face == Ace {
			worth += count * 2
		} else if face == Ten {
			worth += count
		}
	}
	return
}

func (t *Trick) String() string {
	if t == nil {
		return ""
	}
	var str bytes.Buffer
	str.WriteString("-")
	if t.Plays == 0 {
		return "-----"
	}
	var printme [4]bool
	walker := t.Lead - 1
	for x := uint8(0); x < t.Plays; x++ {
		walker = (walker + 1) % 4
		printme[walker] = true
	}
	for y := uint8(0); y < 4; y++ {
		if printme[y] {
			if t.Lead == y {
				str.WriteString("l")
			}
			if t.WinningPlayer == y {
				str.WriteString("w")
			}
			str.WriteString(fmt.Sprintf("%s-", t.Played[y]))
		} else {
			str.WriteString("-")
		}
	}
	return str.String()
}

func (trick *Trick) LeadSuit() Suit {
	if trick.Plays == 0 {
		return NASuit
	}
	return trick.Played[trick.Lead].Suit()
}

func (trick *Trick) WinningCard() Card {
	if trick.Plays == 0 {
		return NACard
	}
	return trick.Played[trick.WinningPlayer]
}

func (trick *Trick) counters() (counters uint8) {
	if trick.Plays != 4 {
		panic("can't get counters before the trick is finished")
	}
	for _, card := range trick.Played {
		if card.Counter() {
			counters++
		}
	}
	return
}

func CardBeatsTrick(card Card, trick *Trick, trump Suit) bool {
	return card.Beats(trick.WinningCard(), trump)
}

type PlayWalker struct {
	Parent    *PlayWalker
	Children  []*PlayWalker
	Card      Card
	Hands     [4]*SmallHand
	TeamCards [2]*SmallHand
	Counters  [2]uint8
	Trick     *Trick
	PlayCount uint8
	Me        uint8
	//Best      *PlayWalker // used for debugging
	//Count     uint // used for debugging
}

func (walker *PlayWalker) PlayTrail() string {
	if walker == nil {
		return ""
	}
	var str bytes.Buffer
	//str.WriteString(strconv.Itoa(int(walker.Count)))
	tricks := make([]*Trick, 0)
	tricks = append([]*Trick{walker.Trick}, tricks...)
	for {
		if walker.Parent == nil {
			break
		}
		walker = walker.Parent
		if walker.Trick != nil && walker.Trick.Plays == 4 {
			tricks = append([]*Trick{walker.Trick}, tricks...)
		}
	}
	for x := range tricks {
		str.WriteString(tricks[x].String())
		str.WriteString(" ")
	}
	return str.String()
}

// Deal fills in the gaps in the HT object based off the status of the hand and does not change the HandTracker
// it is used so potentialCards doesn't play sequences that aren't possible due to having to follow the rules of pinochle
func (ht *HandTracker) Deal() (sh [4]*SmallHand) {
	unknownCards := getHand()
	sum := uint8(0)
	//Log(ht.Owner, "Calling Deal()")
	//ht.Debug()
	for x := AS; int8(x) <= AllCards; x++ {
		sum = ht.sum(x)
		if sum == 2 {
			continue
		}
		//Log(ht.Owner, "Sum for %s = %d", x, sum)
		for {
			if sum == 2 {
				break
			}
			unknownCards = append(unknownCards, x)
			sum++
		}
	}
	unknownCards.Shuffle()
	//Log(ht.Owner, "Have to add %d cards of %s to players' hands", len(unknownCards), unknownCards)
	playerWalker := ht.Trick.Next
	card := Card(NACard)
	addHands := make([]Hand, 4)
	baseNeedCards := uint8((48 - ht.PlayCount) / 4)
	addExtra := uint8(1)
	needs := make([]uint8, 4)
	for x := range addHands {
		if (ht.PlayCount+uint8(x))%4 == 0 {
			addExtra = 0
		}
		numNeedsCards := baseNeedCards + addExtra - ht.calculateHand(playerWalker)
		//Log(ht.Owner, "Player %d needs %d cards added to his hand to make %d", playerWalker, numNeedsCards, baseNeedCards+addExtra)
		needs[playerWalker] = numNeedsCards
		playerWalker = (playerWalker + 1) % 4
	}
largeLoop:
	for {
		//Log(ht.Owner, "Entering large loop")
		if len(unknownCards) == 0 {
			//Log(ht.Owner, "Exiting largeLoop, no more cards left")
			break // all cards identified homes
		}
		if needs[playerWalker] == uint8(len(addHands[playerWalker])) {
			//Log(ht.Owner, "Done adding cards to %d", playerWalker)
			playerWalker = (playerWalker + 1) % 4
			continue
		}
		for _, card = range unknownCards {
			if ht.Cards[playerWalker][card] == Unknown || ht.Cards[playerWalker][card] == 1 {
				//Log(ht.Owner, "Adding %s to player %d, value = %d", card, playerWalker, ht.Cards[playerWalker][card])
				addHands[playerWalker] = append(addHands[playerWalker], card)
				//Log(ht.Owner, "Removing card %s from unknownHand", card)
				unknownCards.Remove(card)
				continue largeLoop
			} else {
				//Log(ht.Owner, "Player %d can't take %s", playerWalker, card)
			}
		}
		// didn't find a location in the current player
		for x := range addHands {
			if uint8(x) == playerWalker {
				continue
			}
			for y, tmpCard := range addHands[x] {
				if ht.Cards[playerWalker][tmpCard] == Unknown || ht.Cards[playerWalker][tmpCard] == 1 {
					addHands[playerWalker] = append(addHands[playerWalker], tmpCard)
					addHands[x][y] = card
					//Log(ht.Owner, "Moving %s to player %d from player %d", tmpCard, playerWalker, x)
					//Log(ht.Owner, "Adding %s to player %d", card, x)
					//Log(ht.Owner, "Removing card %s from unknownHand", card)
					unknownCards.Remove(card)
					continue largeLoop
				}
			}
		}
		// didn't find a card we could switch with, give up!  It's a bug!
		ht.Debug()
		for x := range addHands {
			Log(ht.Owner, "addHands[%d] = %s", x, addHands[x])
		}
		panic(fmt.Sprintf("Nowhere for %s to go!", card))

	}
	// found homes for all cards, let's put them there and create what we knew too
	for x := range addHands {
		sh[x] = NewSmallHand()
		sh[x].Append(addHands[x]...)
		for y := AS; int8(y) <= AllCards; y++ {
			if ht.Cards[x][y] == 2 {
				sh[x].Append(y, y)
			} else if ht.Cards[x][y] == 1 {
				sh[x].Append(y)
			}
		}
	}
	//Log(ht.Owner, "Ending Deal()")
	return
}

func PlayHandWithCard(ht *HandTracker, trump Suit) Card {
	var start = time.Now()
	count := uint(0)
	tierSlice := make([][]*PlayWalker, 48-ht.PlayCount+2)
	length := int(ht.calculateHand(ht.Owner))
	// TODO: update length to be the count of "unknown" cards in the HandTracker
	tierSlice[0] = make([]*PlayWalker, length)
	for x := 0; x < length; x++ {
		tierSlice[0][x] = &PlayWalker{
			Hands:     ht.Deal(),
			Card:      NACard,
			Trick:     new(Trick),
			PlayCount: ht.PlayCount,
			Me:        ht.Owner,
			TeamCards: [2]*SmallHand{NewSmallHand(), NewSmallHand()},
		}
		*tierSlice[0][x].Trick = *ht.Trick
	}
	end := time.Now().Add(time.Millisecond * 500)
	var pw *PlayWalker
tierLoop:
	for tier := 0; tier < len(tierSlice); tier++ {
		//Log(ht.Owner, "Working on tier %d", tier)
		if tierSlice[tier] == nil {
			tierSlice[tier] = make([]*PlayWalker, 0)
		}
		for _, pw = range tierSlice[tier] {
			if time.Now().After(end) {
				break tierLoop // ran out of time generating tricks, calculate results
			}
			//Log(ht.Owner, "Evaluating pw = %#v", pw)
			decisionMap := pw.potentialCards(pw.Trick, trump)
			if len(decisionMap) == 0 {
				if pw.PlayCount != 48 {
					panic("hand is at the end but 48 plays haven't been made!")
				}
				//Log(ht.Owner, "************** Hand is at the end! - %s", pw.PlayTrail())
				continue // no need to make children and append them if they don't exist!
			}
			if tier == 0 && len(decisionMap) == 1 {
				// no need to continue any further, this was the only legal play
				Log(ht.Owner, "Returning the only legal play of %s", decisionMap[0])
				log.Printf("Logged %d unique paths in %s", 0, time.Now().Sub(start))
				return decisionMap[0]
			}
			pw.Children = make([]*PlayWalker, len(decisionMap))
			for x := range decisionMap {
				pw.Children[x] = &PlayWalker{
					Card:      decisionMap[x],
					Parent:    pw,
					Hands:     pw.Hands,
					Trick:     new(Trick),
					PlayCount: pw.PlayCount + 1,
					Me:        pw.Trick.Next,
					//Count:     count,
					Counters: pw.Counters,
				}
				if pw.PlayCount < 47 { // end of the hand, use only counters to make your "best" decision, else TeamCards=nil
					pw.Children[x].TeamCards = pw.TeamCards
					pw.Children[x].TeamCards[pw.Children[x].Me%2] = pw.TeamCards[pw.Children[x].Me%2].CopySmallHand()
					pw.Children[x].TeamCards[pw.Children[x].Me%2].Append(decisionMap[x])
				}
				pw.Children[x].Hands[pw.Trick.Next] = pw.Hands[pw.Trick.Next].CopySmallHand()
				pw.Children[x].Hands[pw.Trick.Next].Remove(decisionMap[x])
				count++
				*pw.Children[x].Trick = *pw.Trick // copy the trick
				pw.Children[x].Trick.PlayCard(pw.Children[x].Card, trump)
				if pw.Children[x].Trick.Plays == 4 {
					pw.Children[x].Counters[pw.Children[x].Trick.WinningPlayer%2] += pw.Children[x].Trick.counters()
					if pw.Children[x].PlayCount == 48 {
						// add one for the last trick
						pw.Children[x].Counters[pw.Children[x].Trick.WinningPlayer%2]++
					}
				}
				//Log(ht.Owner, "Tier %d - Created PlayWalker for %d of card %s for %s", tier, pw.Children[x].Me, pw.Children[x].Card, pw.Children[x].PlayTrail())
			}
			tierSlice[tier+1] = append(tierSlice[tier+1], pw.Children...)
		}
	} // if end==false, we generated all the possibilities
	// the whole hand is played, now we score it
	aggregateScore := make([]int8, len(tierSlice[0][0].Children))
	for tier := len(tierSlice) - 1; tier >= 0; tier-- {
		for _, pw = range tierSlice[tier] {
			if len(pw.Children) > 0 {
				bestChild := uint8(0)
				bestWorth := pw.Children[0].Worth(trump)
				if tier == 0 { // since each "root" will have the same potentialCards, find out which one did the best when accounting for all scenarios played
					aggregateScore[0] += bestWorth
					//Log(ht.Owner, "Child is %d %s", 0, pw.Children[0].Card)
				}
				//Log(ht.Owner, "Found initial child [%d]%s for player %d", bestWorth, pw.Children[0].Best.PlayTrail(), pw.Children[0].Me)
				for c := uint8(1); c < uint8(len(pw.Children)); c++ {
					worth := pw.Children[c].Worth(trump)
					if tier == 0 { // since each "root" will have the same potentialCards, find out which one did the best when accounting for all scenarios played
						aggregateScore[c] += worth
						//Log(ht.Owner, "Child is %d %s", c, pw.Children[c].Card)
					}
					if (pw.Children[0].Me%2 == ht.Owner%2 && worth > bestWorth) || (pw.Children[0].Me%2 != ht.Owner%2 && worth < bestWorth) {
						//Log(ht.Owner, "Child [%d]%s is better than [%d]%s for player %d", bestWorth, pw.Children[c].Best.PlayTrail(), worth, pw.Children[bestChild].Best.PlayTrail(), pw.Children[0].Me)
						bestWorth = worth
						bestChild = c
					} else {
						//Log(ht.Owner, "Incumbent child [%d]%s is better than [%d]%s for player %d", bestWorth, pw.Children[c].Best.PlayTrail(), worth, pw.Children[bestChild].Best.PlayTrail(), pw.Children[0].Me)
					}
				}
				//if pw.Children[bestChild].Best == nil {
				//pw.Best = pw.Children[bestChild]
				//} else {
				//pw.Best = pw.Children[bestChild].Best
				//}
				pw.Counters = pw.Children[bestChild].Counters
				pw.TeamCards = pw.Children[bestChild].TeamCards
				//Log(ht.Owner, "Found best child %s for player %d on tier %d with %d - %s", pw.Children[bestChild].Card, pw.Children[0].Me, tier, bestWorth, pw.Best.PlayTrail())
				if pw.Parent == nil { // || tier == 0
					pw.Card = pw.Children[bestChild].Card
				}
			}
		}
	}
	bestChild := uint8(0)
	for c := uint8(0); c < uint8(len(aggregateScore)); c++ {
		if aggregateScore[c] > aggregateScore[bestChild] {
			bestChild = c
		}
	}
	//Log(ht.Owner, "bestChild = %d", bestChild)
	//Log(ht.Owner, "len(tierSlice[0]) = %d", len(tierSlice[0]))
	Log(ht.Owner, "Returning best play #%d %s with worth %d for the following path(s):", bestChild, tierSlice[0][0].Children[bestChild].Card, aggregateScore[bestChild])
	//for _, pw := range tierSlice[0] {
	//Log(ht.Owner, pw.Best.PlayTrail())
	//}
	log.Printf("Logged %d unique paths in %s", count, time.Now().Sub(start))
	return tierSlice[0][0].Children[bestChild].Card
}

func (ai *AI) findCardToPlay(action *Action) Card {
	ai.HT.Trick.Next = action.Playerid
	card := ai.PlayingStrategy(ai.HT, action.Trump)
	runtime.GC() // since we created so much garbage, we need to have the GC mark it as unlinked/unused so next round it can be reused
	//Log(ai.Playerid, "PlayHandWithCard returned %s for %d points.", card, points)
	return card
}

func (pw *PlayWalker) potentialCards(trick *Trick, trump Suit) Hand {
	//Log(ht.Owner, "PotentialCards called with %d,winning=%s,lead=%s,trump=%s", playerid, winning, lead, trump)
	//Log(ht.Owner, "PotentialCards Player%d - %s", playerid, ht.Cards[playerid])
	validHand := getHand()
	handStatus := Nothing
	winning := NACard
	lead := NASuit
	if trick.Plays != 4 {
		winning = trick.WinningCard()
		lead = trick.LeadSuit()
	}
allCardLoop:
	for card := AS; int8(card) <= AllCards; card++ {
		suit := card.Suit()
		if pw.Hands[trick.Next].Contains(card) {
			cardStatus := Nothing
			switch {
			case winning == NACard:
				// do nothing, just be the default case
			case suit == lead && card.Beats(winning, trump):
				cardStatus = FollowWin
			case suit == lead:
				cardStatus = FollowLose
			case suit == trump && card.Beats(winning, trump):
				cardStatus = TrumpWin
			case suit == trump:
				cardStatus = TrumpLose
			}
			if cardStatus > handStatus {
				handStatus = cardStatus
				validHand = validHand[:1]
				validHand[0] = card
			} else if cardStatus == handStatus {
				if (cardStatus == FollowLose || cardStatus == TrumpLose) ||
					((cardStatus == FollowWin || cardStatus == TrumpWin) && trick.Plays == 3) {
					// there should be a maximum of two cards in validHand, counter and non-counter
					//Log(ht.Owner, "ValidHand=%s", validHand)
					for y, vhc := range validHand {
						//Log(4, "Comparing vhc=%s to card=%s", vhc, card)
						if (vhc.Counter() && card.Counter()) || (!vhc.Counter() && !card.Counter()) {
							if card > vhc {
								//Log(4, "Replacing %s with %s", vhc, card)
								validHand[y] = card
								continue allCardLoop
							}
						}
					}
				}
				//Log(4, "Appending card valid normal")
				validHand = append(validHand, card)
			}
		}
	}
	//sLog(4, "Returning %d potential plays of %s for playerid %d on trick %s", len(validHand), validHand, pw.Trick.Next, pw.Trick)
	//if len(validHand) == 0 && ht.PlayCount != 48 {
	//	ht.Debug()
	//	panic("hand is not at the end but still returning 0 potential cards")
	//}
	return validHand
}

func (ai *AI) Tell(action *Action) *Action {
	//Log(ai.Playerid, "Action received - %+v", action)
	switch action.Type {
	case "Bid":
		if action.Playerid == ai.Playerid {
			//Log(ai.Playerid, "------------------Player %d asked to bid against player %d", ai.Playerid, ai.HighBidder)
			ai.BidAmount, ai.Trump = ai.calculateBid()
			if ai.NumBidders == 1 && ai.IsPartner(ai.HighBidder) && ai.BidAmount < 21 && ai.BidAmount+5 > 20 {
				// save our parter
				//Log(ai.Playerid, "Saving our partner with a recommended bid of %d", ai.BidAmount)
				ai.BidAmount = 21
			}
			switch {
			case ai.Playerid == ai.HighBidder: // this should only happen if I was the dealer and I got stuck
				ai.BidAmount = 20
			case ai.HighBid > ai.BidAmount:
				ai.BidAmount = 0
			case ai.HighBid == ai.BidAmount && !ai.IsPartner(ai.HighBidder): // if equal with an opponent, bid one over them for spite!
				ai.BidAmount++
			case ai.NumBidders == 3: // I'm last to bid, but I want it
				ai.BidAmount = ai.HighBid + 1
			}
			//meld, _ := ai.RealHand.Meld(ai.Trump)
			//Log(ai.Playerid, "------------------Player %d bid %d over %d with recommendation of %d and %d meld", ai.Playerid, ai.BidAmount, ai.HighBid, bidAmountOld, meld)
			return CreateBid(ai.BidAmount, ai.Playerid)
		} else {
			// received someone else's bid value'
			if ai.HighBid < action.Bid {
				ai.HighBid = action.Bid
				ai.HighBidder = action.Playerid
			}
			ai.Bids = append(ai.Bids, ai.BidAmount)
			ai.NumBidders++
		}
	case "Play":
		fallthrough
	case "PlayRequest":
		//Log(ai.Playerid, "Trick = %s", ai.Trick)
		var response *Action
		if action.Playerid == ai.Playerid {
			card := ai.findCardToPlay(action)
			response = CreatePlay(card, ai.Playerid)
			action.PlayedCard = response.PlayedCard
		}
		ai.HT.Trick.Next = action.Playerid
		ai.HT.PlayCard(action.PlayedCard, ai.Trump)
		//Log(ai.Playerid, "Player %d played card %s on %s", action.Playerid, action.PlayedCard, ai.HT.Trick)
		return response
	case "Trump":
		if action.Playerid == ai.Playerid {
			//meld, _ := ai.RealHand.Meld(ai.Trump)
			//Log(ai.Playerid, "Player %d being asked to name trump on hand %s and have %d meld", ai.Playerid, ai.RealHand, meld)
			switch {
			// TODO add case for the end of the game like if opponents will coast out
			case ai.BidAmount < 15:
				return CreateThrowin(ai.Playerid)
			default:
				return CreateTrump(ai.Trump, ai.Playerid)
			}
		} else {
			ai.Trump = action.Trump
			//Log(ai.Playerid, "Trump is %s", ai.Trump)
		}
	case "Throwin":
		//Log(ai.Playerid, "Player %d saw that player %d threw in", ai.Playerid, action.Playerid)
	case "Deal":
		ai.reset()
		ai.RealHand = &action.Hand
		//Log(ai.Playerid, "Set playerid")
		//Log(ai.Playerid, "Dealt Hand = %s", ai.RealHand.String())
		ai.populate()
		ai.HighBid = 20
		ai.HighBidder = action.Dealer
		ai.NumBidders = 0
		ai.Bids = make([]uint8, 0)
	case "Meld":
		//Log(ai.Playerid, "Received meld action - %#v", action)
		if action.Playerid == ai.Playerid {
			return nil // seeing our own meld, we don't care
		}
		for _, cardIndex := range action.Hand {
			val := ai.HT.Cards[action.Playerid][cardIndex]
			if val == Unknown {
				ai.HT.Cards[action.Playerid][cardIndex] = 1
			} else if val == 1 {
				ai.HT.Cards[action.Playerid][cardIndex] = 2
			}
			ai.HT.calculateCard(cardIndex)
		}
		ai.HT.calculateHand(ai.Playerid)
	case "Message": // nothing to do here, no one to read it
	case "Trick": // nothing to do here, nothing to display
		//Log(ai.Playerid, "playedCards=%v", ai.HT.PlayedCards)
		ai.HT.Trick.reset()
	case "Score": // TODO: save score to use for future bidding techniques
	default:
		//Log(ai.Playerid, "Received an action I didn't understand - %v", action)
	}
	return nil
}

func (a *AI) Hand() *Hand {
	return a.RealHand
}

func (a *AI) SetHand(h Hand, dealer, playerid uint8) {
	a.Playerid = playerid
	hand := make(Hand, len(h))
	copy(hand, h)
	a.Tell(CreateDeal(hand, playerid, dealer))
}

type Human struct {
	conn     *net.Conn
	enc      *json.Encoder
	dec      *json.Decoder
	RealHand *Hand
	PlayerImpl
}

func (h *Human) Name() string {
	return fmt.Sprintf("Human - %d", h.PlayerImpl)
}

func (h Human) Hand() *Hand {
	return h.RealHand
}

func (a *Human) SetHand(h Hand, dealer, playerid uint8) {
	hand := make(Hand, len(h))
	copy(hand, h)
	a.RealHand = &hand
	a.Playerid = playerid
	a.Tell(CreateDeal(hand, a.Playerid, dealer))
}

type Game struct {
	Trick              Trick    `datastore:"-" json:"-"`
	Players            []Player `datastore:"-"`
	Dealer             uint8    `datastore:"-" json:"-"`
	Score              []int16  `datastore:"-"`
	Meld               []uint8  `datastore:"-"`
	CountMeld          []bool   `datastore:"-" json:"-"`
	Counters           []uint8  `datastore:"-" json:"-"`
	HighBid            uint8    `datastore:"-"`
	HighPlayer         uint8    `datastore:"-"`
	Trump              Suit     `datastore:"-"`
	State              string
	Next               uint8  `datastore:"-"`
	Hands              []Hand `datastore:"-" json:"-"`
	HandsPlayed        uint8  `datastore:"-" json:"-"`
	WinningPartnership uint8
}

func NewGame(players int) *Game {
	game := new(Game)
	game.Players = make([]Player, players)
	//	for x := range game.Players {
	//		game.Players[x] = CreateAI()
	//	}
	game.Score = make([]int16, players/2)
	game.Meld = make([]uint8, players/2)
	game.State = StateNew
	return game
}

// PRE : Players are already created and set
func (game *Game) NextHand() (*Game, error) {
	game.Meld = make([]uint8, len(game.Players)/2)
	game.Trick = Trick{}
	game.CountMeld = make([]bool, len(game.Players)/2)
	game.Counters = make([]uint8, len(game.Players)/2)
	game.HighBid = 20
	game.HighPlayer = game.Dealer
	game.State = StateBid
	game.Next = game.Dealer
	Log(4, "Dealer is %d", game.Dealer)
	deck := CreateDeck()
	deck.Shuffle()
	hands := deck.Deal()
	for x := uint8(0); x < uint8(len(game.Players)); x++ {
		game.Next = game.Inc()
		sort.Sort(hands[x])
		game.Players[game.Next].SetHand(hands[x], game.Dealer, game.Next)
		//Log(4, "Dealing player %d hand %s", game.Next, game.Players[game.Next].Hand())
	}
	game.Next = game.Inc() // increment so that Dealer + 1 is asked to bid first
	return game.ProcessAction(game.Players[game.Next].Tell(CreateBid(0, game.Next)))
	// ProcessAction will write the game to the datastore when it's done processing the action(s)
}

func (game *Game) Inc() uint8 {
	return (game.Next + 1) % uint8(len(game.Players))
}

func (game *Game) Broadcast(a *Action, p uint8) {
	for x, player := range game.Players {
		if p != uint8(x) {
			player.Tell(a)
		}
	}
}

func (game *Game) BroadcastAll(a *Action) {
	game.Broadcast(a, uint8(len(game.Players)))
}

func (game *Game) retell() {
	switch game.State {
	case StateNew:
		// do nothing, we're not waiting on anyone in particular
	case StateBid:
		game.Players[game.Next].Tell(CreateDeal(*game.Players[game.Next].Hand(), game.Next, game.Dealer))
		game.Players[game.Next].Tell(CreateBid(game.HighBid, game.Next))
	case StateTrump:
		game.Players[game.Next].Tell(CreateDeal(*game.Players[game.Next].Hand(), game.Next, game.Dealer))
		game.Players[game.Next].Tell(CreateTrump(NASuit, game.Next))
	case StateMeld:
		// never going to be stuck here on a user action
	case StatePlay:
		if game.Trick.Plays != 0 {
			x := game.Trick.Lead
			for y := uint8(0); y < game.Trick.Plays; y++ {
				game.Players[game.Next].Tell(CreatePlay(game.Trick.Played[x], x))
				x = (x + 1) % uint8(len(game.Trick.Played))
			}
		}
		game.Players[game.Next].Tell(CreatePlayRequest(game.Trick.WinningCard(), game.Trick.LeadSuit(), game.Trump, game.Next, game.Players[game.Next].Hand()))
	}
}

// client parameter only required for actions that modify the client, like sitting at a table, setting your name, etc
func (game *Game) ProcessAction(action *Action) (*Game, error) {
	for {
		if debugLog {
			log.Printf("ProcessAction on %s", action)
		}
		if action == nil {
			// waiting on a human, exit
			log.Printf("ProcessAction returning %#v", game)
			return game, nil
		}
		switch {
		case action.Type == "Tables":
			//			client.SendTables(game)
			return game, nil
		case action.Type == "Name":
			//			client.Name = action.Message
			//			if game != nil {
			//				for _, player := range game.Players {
			//					log.Printf("Checking player %#v", player)
			//					if human, ok := player.(*Human); ok && human.Client.Id == client.Id {
			//						human.Client = client
			//						log.Printf("%s sitting at table %d", human.Client.Name, game.Id)
			//					} else {
			//						log.Printf("Not updating the client")
			//					}
			//				}
			//				action = nil
			//				continue
			//			}
			return game, nil
		case action.Type == "Start":
			//			c.Debugf("Game is %#v", game)
			//			if game.State != StateNew {
			//				return game, errors.New("Game is already started")
			//			}
			//			for x := range game.Players {
			//				if game.Players[x] == nil {
			//					game.Players[x] = CreateAI()
			//				}
			//			}
			//			return game.NextHand(g, c)
		case action.Type == "Sit":
			//			if action.TableId == 0 { // create a new table/game
			//				game = NewGame(4)
			//			} else {
			//				game = &Game{Id: action.TableId}
			//				err := g.Get(game)
			//				if logError(c, err) {
			//					return game, err
			//				}
			//			}
			//			c.Debugf("%s - %d sitting at table %d", client.Name, client.Id, game.Id)
			//			openSlot := -1
			//			var meHuman *Human
			//			for x, player := range game.Players {
			//				human, ok := player.(*Human)
			//				if ok && human.Client.Id == client.Id {
			//					meHuman = human
			//					game.Players[x] = nil
			//					openSlot = x
			//					break
			//				}
			//				if _, ok := player.(*AI); ok {
			//					openSlot = x
			//				}
			//			}
			//			if openSlot == -1 {
			//				logError(c, errors.New("Game is full!"))
			//				return game, nil
			//			}
			//			if meHuman == nil {
			//				meHuman = &Human{Client: client}
			//			}
			//			game.Players[openSlot] = game.Players[action.Playerid]
			//			game.Players[action.Playerid] = meHuman
			//			for _, player := range game.Players {
			//				if human, ok := player.(*Human); ok {
			//					human.Client.SendTables(g, c, game)
			//				}
			//			}
			//			var err error
			//			game, err = game.processAction(g, c, nil, nil) // save it to the datastore
			//			logError(c, err)
			//			client.TableId = game.Id
			//			_, err = g.Put(client)
			//			logError(c, err)
			//			return game, err
		case game.State == StateBid && action.Type != "Bid":
			log.Printf("Received non bid action")
			action = nil
			continue
		case game.State == StateBid && action.Type == "Bid" && action.Playerid != game.Next:
			log.Printf("It's not your turn!")
			action = nil
			continue
		case game.State == StateBid && action.Type == "Bid" && action.Playerid == game.Next:
			game.Broadcast(action, game.Next)
			if action.Bid > game.HighBid {
				game.HighBid = action.Bid
				game.HighPlayer = game.Next
			}
			if game.HighPlayer == game.Dealer && game.Inc() == game.Dealer { // dealer was stuck, tell everyone
				game.Broadcast(CreateBid(game.HighBid, game.Dealer), game.Dealer)
				game.Next = game.Inc()
			}
			if game.Next == game.Dealer { // the bidding is done
				game.State = StateTrump
				game.Next = game.HighPlayer
				action = game.Players[game.HighPlayer].Tell(CreateTrump(NASuit, game.HighPlayer))
				continue
			}
			game.Next = game.Inc()
			action = game.Players[game.Next].Tell(CreateBid(0, game.Next))
			continue
		case game.State == StateTrump:
			switch action.Type {
			case "Throwin":
				game.Broadcast(action, action.Playerid)
				game.Score[game.HighPlayer%2] -= int16(game.HighBid)
				game.BroadcastAll(CreateMessage(fmt.Sprintf("Player %d threw in! Scores are now Team0 = %d to Team1 = %d, played %d hands", action.Playerid, game.Score[0], game.Score[1], game.HandsPlayed)))
				//Log(4, "Scores are now Team0 = %d to Team1 = %d, played %d hands", game.Score[0], game.Score[1], game.HandsPlayed)
				game.BroadcastAll(CreateScore(game.Score, false, false))
				game.Dealer = (game.Dealer + 1) % 4
				//Log(4, "-----------------------------------------------------------------------------")
				return game.NextHand()
			case "Trump":
				game.Trump = action.Trump
				//Log(4, "Trump is set to %s", game.Trump)
				game.Broadcast(action, game.HighPlayer)
				for x := uint8(0); x < uint8(len(game.Players)); x++ {
					meld, meldHand := game.Players[x].Hand().Meld(game.Trump)
					meldAction := CreateMeld(meldHand, meld, x)
					game.BroadcastAll(meldAction)
					game.Meld[x%2] += meld
				}
				game.Next = game.HighPlayer
				game.Counters = make([]uint8, 2)
				game.State = StatePlay
				action = game.Players[game.Next].Tell(CreatePlayRequest(game.Trick.WinningCard(), game.Trick.LeadSuit(), game.Trump, game.Next, game.Players[game.Next].Hand()))
				continue
			}
		case game.State == StatePlay:
			// TODO: check for throw in
			if ValidPlay(action.PlayedCard, game.Trick.WinningCard(), game.Trick.LeadSuit(), game.Players[game.Next].Hand(), game.Trump) &&
				game.Players[game.Next].Hand().Remove(action.PlayedCard) {
				game.Broadcast(action, game.Next)
				game.Trick.Next = game.Next
				game.Trick.PlayCard(action.PlayedCard, game.Trump)
			} else {
				action = game.Players[game.Next].Tell(CreatePlayRequest(game.Trick.WinningCard(), game.Trick.LeadSuit(), game.Trump, game.Next, game.Players[game.Next].Hand()))
				continue
			}
			if game.Trick.Plays == uint8(len(game.Players)) {
				game.Counters[game.Trick.WinningPlayer%2] += game.Trick.counters()
				game.CountMeld[game.Trick.WinningPlayer%2] = true
				game.Next = game.Trick.WinningPlayer
				game.BroadcastAll(CreateMessage(fmt.Sprintf("Player %d wins trick with %s", game.Trick.WinningPlayer, game.Trick.WinningCard())))
				game.BroadcastAll(CreateTrick(game.Trick.WinningPlayer))
				if debugLog {
					log.Printf("Player %d wins trick with %s", game.Trick.WinningPlayer, game.Trick.WinningCard())
				}
				if len(*game.Players[0].Hand()) == 0 {
					game.Counters[game.Trick.WinningPlayer%2]++ // last trick
					// end of hand
					game.HandsPlayed++
					if game.HighBid <= game.Meld[game.HighPlayer%2]+game.Counters[game.HighPlayer%2] {
						game.Score[game.HighPlayer%2] += int16(game.Meld[game.HighPlayer%2] + game.Counters[game.HighPlayer%2])
					} else {
						game.Score[game.HighPlayer%2] -= int16(game.HighBid)
					}
					if game.CountMeld[(game.HighPlayer+1)%2] {
						game.Score[(game.HighPlayer+1)%2] += int16(game.Meld[(game.HighPlayer+1)%2] + game.Counters[(game.HighPlayer+1)%2])
					}
					// check the score for a winner
					log.Printf("Scores are now Team0 = %d to Team1 = %d, played %d hands", game.Score[0], game.Score[1], game.HandsPlayed)
					game.BroadcastAll(CreateMessage(fmt.Sprintf("Scores are now Team0 = %d to Team1 = %d, played %d hands", game.Score[0], game.Score[1], game.HandsPlayed)))
					//Log(4, "Scores are now Team0 = %d to Team1 = %d, played %d hands", game.Score[0], game.Score[1], game.HandsPlayed)
					win := make([]bool, 2)
					gameOver := false
					if game.Score[game.HighPlayer%2] >= 120 {
						win[game.HighPlayer%2] = true
						game.WinningPartnership = game.HighPlayer % 2
						gameOver = true
					} else if game.Score[(game.HighPlayer+1)%2] >= 120 {
						win[(game.HighPlayer+1)%2] = true
						game.WinningPartnership = (game.HighPlayer + 1) % 2
						gameOver = true
					}
					for x := 0; x < len(game.Players); x++ {
						game.Players[x].Tell(CreateScore(game.Score, gameOver, win[x%2]))
					}
					log.Printf("Score = %v, GameOver=%t, win=%v", game.Score, gameOver, win)
					if gameOver {
						for _, player := range game.Players {
							if _, ok := player.(*Human); ok {
								//								logError(c, g.Get(human.Client))
								//								human.Client.TableId = 0
								//								_, err := g.Put(human.Client)
								//								logError(c, err)
							} else {
								htstack.Push(player.(*AI).HT)
							}
						}
						return nil, nil // game over
					}
					game.Dealer = (game.Dealer + 1) % 4
					//Log(4, "-----------------------------------------------------------------------------")
					return game.NextHand()
				}
				game.Trick.reset()
				action = game.Players[game.Next].Tell(CreatePlayRequest(game.Trick.WinningCard(), game.Trick.LeadSuit(), game.Trump, game.Next, game.Players[game.Next].Hand()))
				continue
			}
			game.Next = game.Inc()
			action = game.Players[game.Next].Tell(CreatePlayRequest(game.Trick.WinningCard(), game.Trick.LeadSuit(), game.Trump, game.Next, game.Players[game.Next].Hand()))
			continue
		}
	}
}

type Player interface {
	Tell(*Action) *Action // if player is being asked to play, return response, otherwise nil
	Hand() *Hand
	SetHand(Hand, uint8, uint8)
	PlayerID() uint8
	Team() uint8
	Name() string
}

func createHuman(conn *net.Conn, enc *json.Encoder, dec *json.Decoder) (a *Human) {
	human := &Human{conn: conn, enc: enc, dec: dec}
	return human
}

func (h Human) Close() {
	(*h.conn).Close()
}

func (h *Human) Go() {
	// nothing to do here, the client is where this "thread" runs
}

func (h *Human) Tell(action *Action) *Action {
	h.enc.Encode(action)
	return nil
}

func (h *Human) Listen() (action *Action, open bool) {
	action = new(Action)
	err := h.dec.Decode(action)
	if err != nil {
		Log(h.Playerid, "Error receiving action from human - %v", err)
		return nil, false
	}
	Log(h.Playerid, "Received action %s", action)
	return action, true

}

const (
	constMattBid        = "MattBid"
	constMattSimulation = "MattSimulation"
)

func (h *Human) createGame(option int, cp *ConnectionPool) {
	game := new(Game)
	game.Players = make([]Player, 4)
	// connect players
	game.Players[0] = h
	switch option {
	case 1:
		// Option 1 - Play against three AI players and start immediately
		for x := 1; x < 4; x++ {
			game.Players[x] = CreateAI(biddingStrategies[constMattBid], playingStrategies[constMattSimulation], fmt.Sprintf("AI%d", x))
		}
	case 2:
		// Option 2 - Play with a human partner against two AI players
		game.Players[1] = CreateAI(biddingStrategies[constMattBid], playingStrategies[constMattSimulation], fmt.Sprintf("AI%d", 1))
		game.Players[2] = cp.Pop()
		game.Players[3] = CreateAI(biddingStrategies[constMattBid], playingStrategies[constMattSimulation], fmt.Sprintf("AI%d", 3))
	case 3:
		// Option 3 - Play with a human partner against one AI players and 1 Human
		game.Players[1] = CreateAI(biddingStrategies[constMattBid], playingStrategies[constMattSimulation], fmt.Sprintf("AI%d", 1))
		game.Players[2] = cp.Pop()
		game.Players[3] = cp.Pop()

	case 4:
		// Option 4 - Play with a human partner against two humans
		for x := 1; x < 4; x++ {
			game.Players[x] = cp.Pop()
		}
	case 5:
		// Option 5 - Play against a human with AI partners
		game.Players[1] = cp.Pop()
		game.Players[2] = CreateAI(biddingStrategies[constMattBid], playingStrategies[constMattSimulation], fmt.Sprintf("AI%d", 2))
		game.Players[3] = CreateAI(biddingStrategies[constMattBid], playingStrategies[constMattSimulation], fmt.Sprintf("AI%d", 3))
	}
	var err error
	game, err = game.NextHand()
	log.Printf("Error found in createGame from NextHand - %v", err)
}

type ConnectionPool struct {
	connections chan *Human
}

func (cp *ConnectionPool) Push(h *Human) {
	cp.connections <- h
	return
}

func (cp *ConnectionPool) Pop() *Human {
	return <-cp.connections
}

func setupGame(net *net.Conn, cp *ConnectionPool) {
	Log(4, "Connection received")
	human := createHuman(net, json.NewEncoder(*net), json.NewDecoder(*net))
	for {
		for {
			human.Tell(CreateMessage("Do you want to join a game, create a new game, or quit? (join, create, quit)"))
			action := CreateHello("Hello")
			human.Tell(action)
			action, ok := human.Listen()
			if !ok {
				Log(4, "Error receiving message from human")
				return
			}
			if action.Message == "create" {
				break
			} else if action.Message == "join" {
				human.Tell(CreateMessage("Waiting on a game to be started that you can join..."))
				cp.Push(human)
				return
				// wait for someone to pick me up
			} else if action.Message == "quit" {
				human.Tell(CreateMessage("Ok, bye bye!"))
				human.Close()
				return
			}
		}
		for {
			human.Tell(CreateMessage("Option 1 - Play against three AI players and start immediately"))
			human.Tell(CreateMessage("Option 2 - Play with a human partner against two AI players"))
			human.Tell(CreateMessage("Option 3 - Play with a human partner against one AI players and 1 Human"))
			human.Tell(CreateMessage("Option 4 - Play with a human partner against two humans"))
			human.Tell(CreateMessage("Option 5 - Play against a human with AI partners"))
			human.Tell(CreateMessage("Option 6 - Go back"))
			human.Tell(CreateGame(0))
			if action, ok := human.Listen(); ok {
				switch action.Option {
				case 1, 2, 3, 4, 5:
					human.createGame(1, cp)
				case 6:
					break
				default:
					human.Tell(CreateMessage("Not a valid option"))
				}
				break // after their game is over, let's set them up again
			} else {
				return
			}
		}
	}
}

func main() {
	cp := ConnectionPool{make(chan *Human, 100)}
	tcpAddr, err := net.ResolveTCPAddr("tcp", ":1201")
	if err != nil {
		Log(4, "Error - %v", err)
		return
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		Log(4, "Error - %v", err)
		return
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go setupGame(&conn, &cp)
	}
}
