// sdzpinochle-client project main.go
package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"fmt"
	sdz "github.com/mzimmerman/sdzpinochle"
	"html/template"
	"math/rand"
	"net"
	"net/http"
	"runtime/debug"
	"sort"
	"strconv"
	"time"
)

const (
	ace      = "A"
	ten      = "T"
	king     = "K"
	queen    = "Q"
	jack     = "J"
	nine     = "9"
	spades   = "S"
	hearts   = "H"
	clubs    = "C"
	diamonds = "D"
)

func Log(playerid int, m string, v ...interface{}) {
	if playerid == 4 {
		fmt.Printf("NP - "+m+"\n", v...)
	} else {
		fmt.Printf("P"+strconv.Itoa(playerid)+" - "+m+"\n", v...)
	}
}

type HandTracker struct {
	cards [4]map[sdz.Card]int
	// missing entry = know nothing
	// 0 = does not have any of this card
	// 1 = has this card
	// 2 = has two of these cards
	playedCards map[sdz.Card]int
}

type Decision map[sdz.Card]bool

func (dec Decision) Sort() sdz.Hand {
	hand := make(sdz.Hand, len(dec))
	x := 0
	for card := range dec {
		hand[x] = card
		x++
	}
	sort.Sort(hand)
	return hand
}

type HTString map[sdz.Card]int

func (hts HTString) Sort() sdz.Hand {
	hand := make(sdz.Hand, len(hts))
	x := 0
	for card := range hts {
		hand[x] = card
		x++
	}
	sort.Sort(hand)
	return hand
}

func (hts HTString) String() (output string) {
	hand := hts.Sort()
	for x := range hand {
		output += fmt.Sprintf("%s:%d ", hand[x], hts[hand[x]])
	}
	return output
}

func (ai *AI) PlayCard(c sdz.Card, playerid int) {
	Log(ai.Playerid(), "In ht.PlayCard for %d-%s on player %d", playerid, c, ai.Playerid())
	Log(ai.Playerid(), "ht.playedCards = %v", HTString(ai.ht.playedCards))
	for x := 0; x < 4; x++ {
		Log(ai.Playerid(), "Player%d - %v", x, ai.ht.cards[x])
		Log(ai.Playerid(), "Player%d - %s", x, HTString(ai.ht.cards[x]))
	}
	if ai.ht.playedCards[c] >= 2 {
		Log(ai.Playerid(), "Player %d has card %s", playerid, c)
		panic("Played cards cannot be greater than 2")
	}
	ai.ht.playedCards[c]++
	if val, ok := ai.ht.cards[playerid][c]; ok {
		if val != 0 {
			ai.ht.cards[playerid][c]--
		} else {
			Log(ai.Playerid(), "Player %d has card %s", playerid, c)
			panic("Player is supposed to have 0 cards, how can he have played it?!")
		}
		if val == 1 && ai.ht.playedCards[c] == 1 && playerid != ai.Playerid() {
			// Other player could have only shown one in meld, but has two - now we don't know who has the last one
			Log(ai.Playerid(), "htcardset - deleted card %s for player %d", c, playerid)
			delete(ai.ht.cards[playerid], c)
		}
	}
	ai.calculateCard(c)
	Log(ai.Playerid(), "Player %d played card %s", playerid, c)
}

func (ai *AI) populate() {
	for _, suit := range sdz.Suits() {
		for _, face := range sdz.Faces() {
			card := sdz.CreateCard(suit, face)
			ai.ht.cards[ai.Playerid()][card] = 0
		}
	}
	for _, card := range *ai.hand {
		ai.ht.cards[ai.Playerid()][card]++
	}
	ai.calculate()
}

func (ht *HandTracker) noSuit(playerid int, suit sdz.Suit) {
	for _, face := range sdz.Faces() {
		ht.cards[playerid][sdz.CreateCard(suit, face)] = 0
	}
}

func (ai *AI) calculateCard(c sdz.Card) {
	if ai.Playerid() == 1 && c == "TD" {
		Log(ai.Playerid(), "FLAG")
	}
	sum := ai.ht.playedCards[c]
	Log(ai.Playerid(), "htcardset - Sum for %s is %d", c, sum)
	for x := 0; x < 4; x++ {
		if val, ok := ai.ht.cards[x][c]; ok {
			sum += val
			Log(ai.Playerid(), "htcardsetIterative%d - Sum for %s is now %d", x, c, sum)
		}
	}
	if sum > 2 || sum < 0 {
		sdz.Log("htcardset - Card=%s,sum=%d", c, sum)
		debug.PrintStack()
		panic("Cannot have more cards than 2 or less than 0 - " + string(sum))
	}
	if sum == 2 {
		for x := 0; x < 4; x++ {
			if _, ok := ai.ht.cards[x][c]; !ok {
				ai.ht.cards[x][c] = 0
			}
		}
	} else {
		unknown := -1
		for x := 0; x < 4; x++ {
			if _, ok := ai.ht.cards[x][c]; !ok {
				if unknown == -1 {
					unknown = x
				} else {
					// at least two unknowns
					unknown = -1
					break
				}
			}
		}
		if unknown != -1 {
			ai.ht.cards[unknown][c] = 2 - sum
		}
	}
	Log(ai.Playerid(), "TT[%s]=%d", c, ai.ht.playedCards[c])
	for x := 0; x < 4; x++ {
		if _, ok := ai.ht.cards[x][c]; ok {
			Log(ai.Playerid(), "P%d[%s]=%d", x, c, ai.ht.cards[x][c])
		}
	}
}

func (ai *AI) calculate() {
	//Log(ai.Playerid(), "Starting calculate() - %v", HTString(ai.ht.playedCards))
	for x := 0; x < 4; x++ {
		//Log(ai.Playerid(), "Player%d - %v", x, HTString(ai.ht.cards[x]))
	}
	for _, suit := range sdz.Suits() {
		for _, face := range sdz.Faces() {
			card := sdz.CreateCard(suit, face)
			ai.calculateCard(card)
		}
	}
	//Log(ai.Playerid(), "After calculate() - %v", HTString(ai.ht.playedCards))
	for x := 0; x < 4; x++ {
		Log(ai.Playerid(), "Player%d - %s", x, HTString(ai.ht.cards[x]))
	}

}

type AI struct {
	hand       *sdz.Hand
	action     *sdz.Action
	trump      sdz.Suit
	bidAmount  int
	highBid    int
	highBidder int
	numBidders int
	show       sdz.Hand
	sdz.PlayerImpl
	ht    *HandTracker
	trick *Trick
}

func createAI() (a *AI) {
	a = new(AI)
	a.ht = new(HandTracker)
	for x := 0; x < 4; x++ {
		a.ht.cards[x] = make(map[sdz.Card]int)
	}
	a.ht.playedCards = make(map[sdz.Card]int)
	for _, suit := range sdz.Suits() {
		for _, face := range sdz.Faces() {
			a.ht.playedCards[sdz.CreateCard(suit, face)] = 0
		}
	}
	return a
}

func (ai AI) Close() {
}

func (ai AI) powerBid(suit sdz.Suit) (count int) {
	count = 5 // your partner's good for at least this right?!?
	suitMap := make(map[sdz.Suit]int)
	for _, card := range *ai.hand {
		suitMap[card.Suit()]++
		if card.Suit() == suit {
			switch card.Face() {
			case ace:
				count += 3
			case ten:
				count += 2
			case king:
				fallthrough
			case queen:
				fallthrough
			case jack:
				fallthrough
			case nine:
				count += 1
			}
		} else if card.Face() == ace {
			count += 2
		} else if card.Face() == jack || card.Face() == nine {
			count -= 1
		}
	}
	for _, x := range sdz.Suits() {
		if x == suit {
			continue
		}
		if suitMap[x] == 0 {
			count++
		}
	}
	return
}

func (ai AI) calculateBid() (amount int, trump sdz.Suit, show sdz.Hand) {
	bids := make(map[sdz.Suit]int)
	for _, suit := range sdz.Suits() {
		bids[suit], show = ai.hand.Meld(suit)
		bids[suit] = bids[suit] + ai.powerBid(suit)
		//		Log("Could bid %d in %s", bids[suit], suit)
		if bids[trump] < bids[suit] {
			trump = suit
		} else if bids[trump] == bids[suit] {
			rand.Seed(time.Now().UnixNano())
			if rand.Intn(2) == 0 { // returns one in the set of [0,2)
				trump = suit
			} // else - stay with trump as it was
		}
	}
	rand.Seed(time.Now().UnixNano())
	bids[trump] += rand.Intn(3) // adds 0, 1, or 2 for a little spontanaeity
	return bids[trump], trump, show
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type Trick struct {
	played        map[int]sdz.Card
	winningPlayer int
	certain       bool
	playCount     int
	lead          int
}

func (t *Trick) String() string {
	return fmt.Sprintf("[%s %s %s %s] playCount=%d Winning=%s Lead=%s certain=%v", t.played[0], t.played[1], t.played[2], t.played[3], t.playCount, t.winningCard(), t.leadSuit(), t.certain)
}

func NewTrick() *Trick {
	trick := new(Trick)
	trick.played = make(map[int]sdz.Card)
	trick.certain = true
	return trick
}

func (trick *Trick) leadSuit() sdz.Suit {
	if leadCard, ok := trick.played[trick.lead]; ok {
		return leadCard.Suit()
	}
	return sdz.NASuit
}

func (trick *Trick) winningCard() sdz.Card {
	if winningCard, ok := trick.played[trick.winningPlayer]; ok {
		return winningCard
	}
	return sdz.NACard
}

func (trick *Trick) counters() (counters int) {
	for _, card := range trick.played {
		if card.Counter() {
			counters++
		}
	}
	return
}

func (trick *Trick) worth(playerid int, trump sdz.Suit) (worth int) {
	if len(trick.played) != 4 {
		for x := range trick.played {
			sdz.Log("[%d]=%s", x, trick.played[x])
		}
		debug.PrintStack()
		panic("worth should only be called at the theoretical end of the trick")
	}
	for x := range trick.played {
		if playerid%2 == x%2 {
			if trick.played[x].Suit() == trump {
				worth--
			}
			switch trick.played[x].Face() {
			case sdz.Ace:
				worth -= 2
			case sdz.Ten:
				worth--
			}
		} else {
			if trick.played[x].Suit() == trump {
				worth++
			}
			switch trick.played[x].Face() {
			case sdz.Ace:
				worth += 2
			case sdz.Ten:
				worth++
			}

		}
	}
	if trick.winningPlayer%2 == playerid%2 {
		worth += trick.counters() * 4
	} else {
		worth -= trick.counters() * 4
	}
	if trick.certain {
		worth *= 2
	}
	return
}

func CardBeatsTrick(card sdz.Card, trick *Trick, trump sdz.Suit) bool {
	winningCard, ok := trick.played[trick.winningPlayer]
	if !ok {
		return true
	}
	return card.Beats(winningCard, trump)
}

// PRE Condition: Initial call, trick.certain should be "true" - the cards have already been played
func rankCard(playerid int, ht *HandTracker, trick *Trick, trump sdz.Suit) *Trick {
	//sdz.Log("Player%d rankCard on trick %s", playerid, trick)
	decisionMap := potentialCards(playerid, ht, trick.winningCard(), trick.leadSuit(), trump)
	if len(decisionMap) == 0 {
		debug.PrintStack()
		sdz.Log("%#v", ht)
		panic("decisionMap should not be empty")
	}
	//sdz.Log("Player%d - Potential cards to play - %v", playerid, decisionMap)
	//sdz.Log("Received Trick %s", trick)
	var topTrick *Trick
	nextPlayer := (playerid + 1) % 4
	for _, card := range Decision(decisionMap).Sort() {
		tempTrick := new(Trick)
		*tempTrick = *trick // make a copy
		trick.played = make(map[int]sdz.Card)
		for x := range tempTrick.played { // now copy the map
			trick.played[x] = tempTrick.played[x]
		}
		if CardBeatsTrick(card, tempTrick, trump) {
			tempTrick.winningPlayer = playerid
			if _, ok := ht.cards[playerid][card]; !ok {
				tempTrick.certain = false
			}
		}
		tempTrick.played[playerid] = card
		if tempTrick.playCount < 3 {
			tempTrick.playCount++
			tempTrick = rankCard(nextPlayer, ht, tempTrick, trump)
		}
		//sdz.Log("Playerid = %d - Top = %s, Temp = %s", playerid, topTrick, tempTrick)
		if topTrick == nil {
			topTrick = tempTrick
		}
		topWorth := topTrick.worth(playerid, trump)
		tempWorth := tempTrick.worth(playerid, trump)
		switch {
		case topWorth < tempWorth:
			topTrick = tempTrick
		case !topTrick.certain && !tempTrick.certain && (topWorth == tempWorth) && (card.Face().Less(topTrick.played[playerid].Face())):
			topTrick = tempTrick
		case topWorth == tempWorth && topTrick.played[playerid].Face().Less(card.Face()):
			topTrick = tempTrick
		}
		if topWorth < tempWorth || (topWorth == tempWorth && topTrick.played[playerid].Face().Less(card.Face())) {
			topTrick = tempTrick
		}
	}
	//sdz.Log("Returning worth %d - %s", topTrick.worth(playerid, trump), topTrick)
	return topTrick
}

func (ai *AI) findCardToPlay(action *sdz.Action) sdz.Card {
	ai.trick.certain = true
	//sdz.Log("htcardset - Player%d is calculating", ai.Playerid())
	ai.calculate() // make sure we're making a decision based on the most up-to-date information
	//sdz.Log("Before rankCard as player %d", ai.Playerid())
	ai.trick = rankCard(ai.Playerid(), ai.ht, ai.trick, ai.trump)
	//sdz.Log("Playerid %d choosing card %s", ai.Playerid(), ai.trick.played[ai.Playerid()])
	return ai.trick.played[ai.Playerid()]
}

//// returns 2 if it will win
//// returns 1 if it will possibly win
//// returns 0 for not winning
//func rankCard(partner bool, playerid, playCount int, ht *HandTracker, winning, selected sdz.Card, lead, trump sdz.Suit) int {
//	message := fmt.Sprintf(" - Evaluated %s - player=%d, playCount=%d,winning=%s,lead=%s,trump=%s", selected, playerid, playCount, winning, lead, trump)
//	returnVal := 2
//	if _, ok := ht.cards[playerid][selected]; !ok {
//		returnVal = 1 // a fuzzy estimate
//	} else if partner {
//		returnVal = 0
//	}
//	if !selected.Beats(winning, trump) {
//		sdz.Log("2 - Returning %d - %s", returnVal, message)
//		return returnVal
//	}
//	if playCount == 3 {
//		sdz.Log("3 - Returning %d - %s", returnVal, message)
//		return returnVal
//	}
//	nextPlayer := (playerid + 1) % 4
//	playCount++
//	potCards := potentialCards(nextPlayer, ht, selected, lead, trump)
//	//sdz.Log("potCards = %v", potCards)
//	for card := range potCards {
//		if playCount == 1 {
//			lead = selected.Suit()
//		}
//		if partner {
//			returnVal = max(returnVal, rankCard(!partner, nextPlayer, playCount, ht, selected, card, lead, trump))
//		} else {
//			returnVal = min(returnVal, rankCard(!partner, nextPlayer, playCount, ht, selected, card, lead, trump))
//		}
//	}
//	sdz.Log("5 - Returning %d - %s", returnVal, message)
//	return returnVal
//}

func potentialCards(playerid int, ht *HandTracker, winning sdz.Card, lead sdz.Suit, trump sdz.Suit) map[sdz.Card]bool {
	//sdz.Log("PotentialCards called with %d,winning=%s,lead=%s,trump=%s", playerid, winning, lead, trump)
	//sdz.Log("PotentialCards Player%d - %#v", playerid, ht.cards[playerid])
	trueHand := make(sdz.Hand, 0)
	potentialHand := make(sdz.Hand, 0)
	for _, card := range sdz.AllCards() {
		val, ok := ht.cards[playerid][card]
		if ok && val > 0 {
			trueHand = append(trueHand, card)
		} else if !ok {
			potentialHand = append(potentialHand, card)
		}
	}
	//sdz.Log("TrueHand = %#v", trueHand)
	//sdz.Log("PotentialHand = %#v", potentialHand)
	validHand := make(sdz.Hand, 0)
	decisionMap := make(map[sdz.Card]bool)
	for _, card := range trueHand {
		if sdz.ValidPlay(card, winning, lead, &trueHand, trump) {
			decisionMap[card] = true
			validHand = append(validHand, card)
			continue
		}
	}
	//sdz.Log("validHand = %#v", validHand)
	for _, card := range potentialHand {
		tempHand := append(validHand, card)
		if sdz.ValidPlay(card, winning, lead, &tempHand, trump) {
			decisionMap[card] = true
			continue
		}
	}
	return decisionMap
}

//func (ai *AI) findCardToPlay(action *sdz.Action) sdz.Card {
//	ai.ht.calculate() // make sure our knowledge is updated
//	decisionMap := potentialCards(ai.Playerid(), ai.ht, action.WinningCard, action.Lead, action.Trump)
//	for card := range decisionMap {
//		decisionMap[card] = rankCard(true, ai.Playerid(), ai.playCount, ai.ht, action.WinningCard, card, action.Lead, action.Trump)
//		sdz.Log("First result=%d - evaluated %s - player=%d, playCount=%d,winning=%s,lead=%s,trump=%s", decisionMap[card], card, ai.Playerid(), ai.playCount, action.WinningCard, action.Lead, action.Trump)
//		if decisionMap[card] == 2 {
//			willWin = true
//			canWin = true
//		} else if decisionMap[card] == 1 {
//			canWin = true
//		}
//	}
//	sdz.Log("willWin = %v, canWin = %v", willWin, canWin)
//	sdz.Log("RankedDecisionMap = %v", decisionMap)
//	var selection sdz.Card
//	for card := range decisionMap {
//		if selection == "" {
//			selection = card
//		}
//		if (willWin && decisionMap[card] < 2) || (canWin && decisionMap[card] < 1) {
//			delete(decisionMap, card)
//			//decisionMap[card] -= 50
//			continue
//		}
//		if willWin {
//			switch card.Face() {
//			case sdz.Ace:
//				decisionMap[card] = 0
//			case sdz.Ten:
//				decisionMap[card] = 1
//			case sdz.King:
//				decisionMap[card] = 2
//			case sdz.Queen:
//				decisionMap[card] = 3
//			case sdz.Jack:
//				decisionMap[card] = 4
//			case sdz.Nine:
//				decisionMap[card] = 5
//			}
//		} else if canWin {
//			switch card.Face() {
//			case sdz.Ace:
//				decisionMap[card] = 5
//			case sdz.Ten:
//				decisionMap[card] = 1 // if we're not sure if it wins, we don't want to play it and lose it
//			case sdz.King:
//				decisionMap[card] = 0
//			case sdz.Queen:
//				decisionMap[card] = 4 // playing a queen first (if it wins) then we'll know that our king and 10 are good too)
//			case sdz.Jack:
//				decisionMap[card] = 3
//			case sdz.Nine:
//				decisionMap[card] = 2
//			}
//		}
//		if ai.playCount > 0 && action.WinningPlayer%2 == ai.Playerid()%2 {
//			if card.Counter() && rankCard(true, ai.Playerid(), 0, ai.ht, "", card, "", action.Trump) < 2 { // don't throw winners as counters
//				decisionMap[card] += 5
//			} else {
//				decisionMap[card] -= 5
//			}
//		} else {
//			// rankCard anticipates my partner winning the trick, so canWin or willWin will be set for that case
//			if ai.hand.Lowest(card) {
//				decisionMap[card]++
//			}
//		}
//		if decisionMap[card] > decisionMap[selection] {
//			selection = card
//		}
//	}
//	Log("%d - Playing %s - Decision map = %v", ai.Playerid(), selection, decisionMap)
//	return selection
//}

func (ai *AI) Tell(action *sdz.Action) {
	//Log(ai.Playerid(), "Action received by player %d with hand %s - %+v", ai.Playerid(), ai.hand, action)
	switch action.Type {
	case "Bid":
		if action.Playerid == ai.Playerid() {
			Log(ai.Playerid(), "------------------Player %d asked to bid against player %d", ai.Playerid(), ai.highBidder)
			ai.bidAmount, ai.trump, ai.show = ai.calculateBid()
			if ai.numBidders == 1 && ai.IsPartner(ai.highBidder) && ai.bidAmount < 21 && ai.bidAmount+5 > 20 {
				// save our parter
				Log(ai.Playerid(), "Saving our partner with a recommended bid of %d", ai.bidAmount)
				ai.bidAmount = 21
			}
			bidAmountOld := ai.bidAmount
			switch {
			case ai.Playerid() == ai.highBidder: // this should only happen if I was the dealer and I got stuck
				ai.bidAmount = 20
			case ai.highBid > ai.bidAmount:
				ai.bidAmount = 0
			case ai.highBid == ai.bidAmount && !ai.IsPartner(ai.highBidder): // if equal with an opponent, bid one over them for spite!
				ai.bidAmount++
			case ai.numBidders == 3: // I'm last to bid, but I want it
				ai.bidAmount = ai.highBid + 1
			}
			meld, _ := ai.hand.Meld(ai.trump)
			Log(ai.Playerid(), "------------------Player %d bid %d over %d with recommendation of %d and %d meld", ai.Playerid(), ai.bidAmount, ai.highBid, bidAmountOld, meld)
			ai.action = sdz.CreateBid(ai.bidAmount, ai.Playerid())
		} else {
			// received someone else's bid value'
			if ai.highBid < action.Bid {
				ai.highBid = action.Bid
				ai.highBidder = action.Playerid
			}
			ai.numBidders++
		}
	case "Play":
		if action.Playerid == ai.Playerid() {
			action = sdz.CreatePlay(ai.findCardToPlay(action), ai.Playerid())
			sdz.Log("Player%d is playing %s", ai.Playerid(), action.PlayedCard)
			ai.PlayCard(action.PlayedCard, ai.Playerid())
			ai.action = action
		} else {
			if ai.trick.leadSuit() == sdz.NASuit || ai.trick.winningCard() == sdz.NACard {
				ai.trick.lead = action.Playerid
				ai.trick.winningPlayer = action.Playerid
				sdz.Log("Player%d - Set lead to %s", ai.Playerid(), ai.trick.leadSuit())
			}
			ai.trick.playCount++
			ai.trick.played[action.Playerid] = action.PlayedCard
			ai.PlayCard(action.PlayedCard, action.Playerid)
			if action.PlayedCard.Suit() != ai.trick.leadSuit() {
				ai.ht.noSuit(action.Playerid, ai.trick.leadSuit())
				sdz.Log("nofollow - Player%d calling nosuit on Player%d on suit %s", ai.Playerid(), action.Playerid, ai.trick.leadSuit())
				if ai.trick.leadSuit() != ai.trump && action.PlayedCard.Suit() != ai.trump {
					sdz.Log("notrump - Player%d calling nosuit on Player%d on suit %s", ai.Playerid(), action.Playerid, ai.trick.leadSuit())
					ai.ht.noSuit(action.Playerid, ai.trump)
				}
			}
			// TODO: find all the cards that can beat the lead card and set those in the HandTracker
			// received someone else's play
		}
	case "Trump":
		if action.Playerid == ai.Playerid() {
			meld, _ := ai.hand.Meld(ai.trump)
			Log(ai.Playerid(), "Player %d being asked to name trump on hand %s and have %d meld", ai.Playerid(), ai.hand, meld)
			switch {
			// TODO add case for the end of the game like if opponents will coast out
			case ai.bidAmount < 15:
				ai.action = sdz.CreateThrowin(ai.Playerid())
			default:
				ai.action = sdz.CreateTrump(ai.trump, ai.Playerid())
			}
		} else {
			ai.trump = action.Trump
			Log(ai.Playerid(), "Trump is %s", ai.trump)
		}
	case "Throwin":
		Log(ai.Playerid(), "Player %d saw that player %d threw in", ai.Playerid(), action.Playerid)
	case "Deal":
		ai.hand = &action.Hand
		ai.Id = action.Playerid
		Log(ai.Playerid(), "Set playerid")
		Log(ai.Playerid(), "Dealt Hand = %s", ai.hand.String())
		ai.populate()
		ai.highBid = 20
		ai.highBid = action.Dealer
		ai.numBidders = 0
		ai.trick = NewTrick()
		ai.trick.playCount = 0
	case "Meld":
		sdz.Log("Received meld action - %#v", action)
		if action.Playerid == ai.Playerid() {
			return // seeing our own meld, we don't care
		}
		for _, card := range action.Hand {
			val, ok := ai.ht.cards[action.Playerid][card]
			if !ok {
				ai.ht.cards[action.Playerid][card] = 1
			} else if val == 1 {
				ai.ht.cards[action.Playerid][card] = 2
			}
		}
	case "Message": // nothing to do here, no one to read it
	case "Trick": // nothing to do here, nothing to display
		Log(ai.Playerid(), "playedCards=%v", ai.ht.playedCards)
		ai.trick = NewTrick()
	case "Score": // TODO: save score to use for future bidding techniques
	default:
		Log(ai.Playerid(), "Received an action I didn't understand - %v", action)
	}
}

func (a *AI) Listen() (action *sdz.Action, open bool) {
	Log(a.Playerid(), "Listen for playerid %d returning %+v", a.Playerid(), a.action)
	return a.action, true
}

func (a *AI) Hand() *sdz.Hand {
	return a.hand
}

func (a *AI) SetHand(h sdz.Hand, dealer, playerid int) {
	hand := make(sdz.Hand, len(h))
	copy(hand, h)
	a.Tell(sdz.CreateDeal(hand, playerid, dealer))
}

type Human struct {
	hand *sdz.Hand
	conn *websocket.Conn
	sdz.PlayerImpl
	finished chan bool
}

func createHuman(conn *websocket.Conn) (a *Human) {
	human := &Human{conn: conn, finished: make(chan bool)}
	return human
}

func (h Human) Close() {
	(*h.conn).Close()
}

func (h *Human) Tell(action *sdz.Action) {
	jsonData, _ := json.Marshal(action)
	Log(h.Playerid(), "--> %s", jsonData)
	err := websocket.JSON.Send(h.conn, action)
	if err != nil {
		sdz.Log("Error in Send - %v", err)
		h.Close()
		return
	}
}

func (h *Human) Listen() (action *sdz.Action, open bool) {
	action = new(sdz.Action)
	err := websocket.JSON.Receive(h.conn, action)
	if err != nil {
		sdz.Log("Error receiving action from human - %v", err)
		return nil, false
	}
	action.Playerid = h.Id
	jsonData, _ := json.Marshal(action)
	Log(h.Playerid(), "<-- %s", jsonData)
	return action, true
}

func (h Human) Hand() *sdz.Hand {
	return h.hand
}

func (a *Human) SetHand(h sdz.Hand, dealer, playerid int) {
	hand := make(sdz.Hand, len(h))
	copy(hand, h)
	a.hand = &hand
	a.Id = playerid
	a.Tell(sdz.CreateDeal(hand, a.Playerid(), dealer))
}

func (h *Human) createGame(option int, cp *ConnectionPool) {
	game := new(sdz.Game)
	players := make([]sdz.Player, 4)
	// connect players
	players[0] = h
	switch option {
	case 1:
		// Option 1 - Play against three AI players and start immediately
		for x := 1; x < 4; x++ {
			players[x] = createAI()
		}
	case 2:
		// Option 2 - Play with a human partner against two AI players
		players[1] = createAI()
		players[2] = cp.Pop()
		players[3] = createAI()
	case 3:
		// Option 3 - Play with a human partner against one AI players and 1 Human
		players[1] = createAI()
		players[2] = cp.Pop()
		players[3] = cp.Pop()

	case 4:
		// Option 4 - Play with a human partner against two humans
		for x := 1; x < 4; x++ {
			players[x] = cp.Pop()
		}
	case 5:
		// Option 5 - Play against a human with AI partners
		players[1] = cp.Pop()
		players[2] = createAI()
		players[3] = createAI()
	}
	game.Go(players)
	//h.finished <- true
	for x := 1; x < 4; x++ {
		if th, ok := players[x].(*Human); ok {
			th.finished <- true
		}
	}
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

func setupGame(net *websocket.Conn, cp *ConnectionPool) {
	Log(4, "Connection received")
	human := createHuman(net)
	for {
		for {
			human.Tell(sdz.CreateMessage("Do you want to join a game, create a new game, or quit? (join, create, quit)"))
			action := sdz.CreateHello("")
			human.Tell(action)
			action, open := human.Listen()
			if !open {
				return
			}
			if action.Message == "create" {
				break
			} else if action.Message == "join" {
				human.Tell(sdz.CreateMessage("Waiting on a game to be started that you can join..."))
				cp.Push(human)
				<-human.finished // this will block to keep the websocket open
				continue
			} else if action.Message == "quit" {
				human.Tell(sdz.CreateMessage("Ok, bye bye!"))
				human.Close()
				return
			}
		}
		for {
			human.Tell(sdz.CreateMessage("Option 1 - Play against three AI players and start immediately"))
			human.Tell(sdz.CreateMessage("Option 2 - Play with a human partner against two AI players"))
			human.Tell(sdz.CreateMessage("Option 3 - Play with a human partner against one AI players and 1 Human"))
			human.Tell(sdz.CreateMessage("Option 4 - Play with a human partner against two humans"))
			human.Tell(sdz.CreateMessage("Option 5 - Play against a human with AI partners"))
			human.Tell(sdz.CreateMessage("Option 6 - Go back"))
			human.Tell(sdz.CreateGame(0))
			action, open := human.Listen()
			if !open {
				return
			}
			switch action.Option {
			case 1:
				fallthrough
			case 2:
				fallthrough
			case 3:
				fallthrough
			case 4:
				fallthrough
			case 5:
				human.createGame(action.Option, cp)
			case 6:
				break
			default:
				human.Tell(sdz.CreateMessage("Not a valid option"))
			}
			break // after their game is over, let's set them up again'
		}
	}
}

func wshandler(ws *websocket.Conn) {
	//cookie, err := ws.Request().Cookie("pinochle")
	//if err != nil {
	//	Log("Could not get cookie - %v", err)
	//	return
	//}
	setupGame(ws, cp)
}

func serveGame(w http.ResponseWriter, r *http.Request) {
	listTmpl, err := template.New("tempate").ParseGlob("*.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = listTmpl.ExecuteTemplate(w, "game", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

var cp *ConnectionPool

func listenForFlash() {
	response := []byte("<cross-domain-policy><allow-access-from domain=\"*\" to-ports=\"*\" /></cross-domain-policy>")
	ln, err := net.Listen("tcp", ":843")
	if err != nil {
		sdz.Log("Cannot listen on port 843 for flash policy file, will not serve non WebSocket clients, check permissions or run as root - " + err.Error())
		return
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			// handle error
			continue
		}
		conn.Write(response)
		conn.Close()
	}
}

func main() {
	go listenForFlash()
	cp = &ConnectionPool{connections: make(chan *Human, 100)}
	http.Handle("/connect", websocket.Handler(wshandler))
	http.Handle("/cards/", http.StripPrefix("/cards/", http.FileServer(http.Dir("cards"))))
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("js"))))
	http.Handle("/web-socket-js/", http.StripPrefix("/web-socket-js/", http.FileServer(http.Dir("web-socket-js"))))
	http.HandleFunc("/", serveGame)
	err := http.ListenAndServe(":80", nil)
	if err != nil {
		sdz.Log("Cannot listen on port 80, check permissions or run as root - " + err.Error())
	}
	err = http.ListenAndServe(":1080", nil)
	if err != nil {
		panic("Cannot listen for http requests - " + err.Error())
	}
}
