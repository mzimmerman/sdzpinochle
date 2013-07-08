// sdzpinochle-client project main.go
package main

import (
	"appengine"
	//"appengine/datastore"
	"appengine/mail"
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"errors"
	"fmt"
	sdz "github.com/mzimmerman/sdzpinochle"
	//"html/template"
	"math/rand"
	//"net"
	//"net/http"
	"runtime/debug"
	"sort"
	"strconv"
	//"time"
)

const (
	ace       = "A"
	ten       = "T"
	king      = "K"
	queen     = "Q"
	jack      = "J"
	nine      = "9"
	spades    = "S"
	hearts    = "H"
	clubs     = "C"
	diamonds  = "D"
	StateDeal = iota
	StateBid
	StateTrump
	StateMeld
	StatePlay
)

func Log(playerid int, m string, v ...interface{}) {
	return
	if playerid == 4 {
		fmt.Printf("NP - "+m+"\n", v...)
	} else {
		fmt.Printf("P"+strconv.Itoa(playerid)+" - "+m+"\n", v...)
	}
}

type HandTracker struct {
	Cards [4]map[sdz.Card]int
	// missing entry = know nothing
	// 0 = does not have any of this card
	// 1 = has this card
	// 2 = has two of these cards
	PlayedCards map[sdz.Card]int
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
	Log(ai.Playerid(), "ht.PlayedCards = %v", HTString(ai.ht.PlayedCards))
	for x := 0; x < 4; x++ {
		Log(ai.Playerid(), "Player%d - %s", x, HTString(ai.ht.Cards[x]))
	}
	if ai.ht.PlayedCards[c] >= 2 {
		Log(ai.Playerid(), "Player %d has card %s", playerid, c)
		panic("Played cards cannot be greater than 2")
	}
	ai.ht.PlayedCards[c]++
	if val, ok := ai.ht.Cards[playerid][c]; ok {
		if val > 0 {
			ai.ht.Cards[playerid][c]--
		} else {
			Log(ai.Playerid(), "Player %d has card %s", playerid, c)
			panic("Player is supposed to have 0 cards, how can he have played it?!")
		}
		if val == 1 && ai.ht.PlayedCards[c] == 1 && playerid != ai.Playerid() {
			// Other player could have only shown one in meld, but has two - now we don't know who has the last one
			Log(ai.Playerid(), "htcardset - deleted card %s for player %d", c, playerid)
			delete(ai.ht.Cards[playerid], c)
		}
	}
	ai.calculateCard(c)
	Log(ai.Playerid(), "Player %d played card %s", playerid, c)
}

func (ai *AI) populate() {
	for _, suit := range sdz.Suits() {
		for _, face := range sdz.Faces() {
			card := sdz.CreateCard(suit, face)
			ai.ht.Cards[ai.Playerid()][card] = 0
		}
	}
	for _, card := range *ai.hand {
		ai.ht.Cards[ai.Playerid()][card]++
		ai.calculateCard(card)
	}
}

func (ai *AI) noSuit(playerid int, suit sdz.Suit) {
	Log(ai.Playerid(), "No suit start")
	for _, face := range sdz.Faces() {
		card := sdz.CreateCard(suit, face)
		ai.ht.Cards[playerid][card] = 0
		ai.calculateCard(card)
	}
	Log(ai.Playerid(), "No suit end")
}

func (ai *AI) calculateCard(c sdz.Card) {
	sum := ai.ht.PlayedCards[c]
	Log(ai.Playerid(), "htcardset - Sum for %s is %d", c, sum)
	for x := 0; x < 4; x++ {
		if val, ok := ai.ht.Cards[x][c]; ok {
			sum += val
			Log(ai.Playerid(), "htcardsetIterative%d - Sum for %s is now %d", x, c, sum)
		}
	}
	if sum > 2 || sum < 0 {
		sdz.Log("htcardset - Card=%s,sum=%d", c, sum)
		Log(ai.Playerid(), "ht.PlayedCards = %v", HTString(ai.ht.PlayedCards))
		for x := 0; x < 4; x++ {
			Log(ai.Playerid(), "Player%d - %s", x, HTString(ai.ht.Cards[x]))
		}
		panic("Cannot have more cards than 2 or less than 0 - " + string(sum))
	}
	if sum == 2 {
		for x := 0; x < 4; x++ {
			if _, ok := ai.ht.Cards[x][c]; !ok {
				ai.ht.Cards[x][c] = 0
			}
		}
	} else {
		unknown := -1
		for x := 0; x < 4; x++ {
			if _, ok := ai.ht.Cards[x][c]; !ok {
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
			if ai.ht.PlayedCards[c] > 0 || ai.ht.Cards[ai.Playerid()][c] == 1 {
				if unknown != -1 {
					ai.ht.Cards[unknown][c] = 2 - sum
				}
			} else if sum == 0 {
				ai.ht.Cards[unknown][c] = 2
			}
		}
	}
	Log(ai.Playerid(), "TT[%s]=%d", c, ai.ht.PlayedCards[c])
	for x := 0; x < 4; x++ {
		if _, ok := ai.ht.Cards[x][c]; ok {
			Log(ai.Playerid(), "P%d[%s]=%d", x, c, ai.ht.Cards[x][c])
		}
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

func (a *AI) reset() {
	a.ht = new(HandTracker)
	for x := 0; x < 4; x++ {
		a.ht.Cards[x] = make(map[sdz.Card]int)
	}
	a.ht.PlayedCards = make(map[sdz.Card]int)
	for _, suit := range sdz.Suits() {
		for _, face := range sdz.Faces() {
			a.ht.PlayedCards[sdz.CreateCard(suit, face)] = 0
		}
	}
	a.trick = NewTrick()
}

func createAI() (a *AI) {
	a = new(AI)
	a.reset()
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
			//rand.Seed(time.Now().UnixNano())
			if rand.Intn(2) == 0 { // returns one in the set of [0,2)
				trump = suit
			} // else - stay with trump as it was
		}
	}
	//rand.Seed(time.Now().UnixNano())
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
	Played        map[int]sdz.Card
	WinningPlayer int
	certain       bool // only the AI uses this to determine what card to play, not relevant for the server
	Lead          int
}

func (t *Trick) String() string {
	var str = "-"
	for x := 0; x < 4; x++ {
		if t.Lead == x {
			str += "s"
		}
		if t.WinningPlayer == x {
			str += "w"
		}
		str += string(t.Played[x]) + "-"
	}
	return fmt.Sprintf("%s Winning=%s Lead=%s certain=%v", str, t.winningCard(), t.leadSuit(), t.certain)
}

func NewTrick() *Trick {
	trick := new(Trick)
	trick.Played = make(map[int]sdz.Card)
	trick.certain = true
	return trick
}

func (trick *Trick) leadSuit() sdz.Suit {
	if leadCard, ok := trick.Played[trick.Lead]; ok {
		return leadCard.Suit()
	}
	return sdz.NASuit
}

func (trick *Trick) winningCard() sdz.Card {
	if winningCard, ok := trick.Played[trick.WinningPlayer]; ok {
		return winningCard
	}
	return sdz.NACard
}

func (trick *Trick) counters() (counters int) {
	for _, card := range trick.Played {
		if card.Counter() {
			counters++
		}
	}
	return
}

func (trick *Trick) worth(playerid int, trump sdz.Suit) (worth int) {
	if len(trick.Played) != 4 {
		sdz.Log("Trick = %s", trick)
		panic("worth should only be called at the theoretical end of the trick")
	}
	for x := range trick.Played {
		if playerid%2 == x%2 {
			if trick.Played[x].Suit() == trump {
				worth--
			}
			switch trick.Played[x].Face() {
			case sdz.Ace:
				worth -= 2
			case sdz.Ten:
				worth--
			}
		} else {
			if trick.Played[x].Suit() == trump {
				worth++
			}
			switch trick.Played[x].Face() {
			case sdz.Ace:
				worth += 2
			case sdz.Ten:
				worth++
			}

		}
	}
	if trick.WinningPlayer%2 == playerid%2 {
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
	return card.Beats(trick.winningCard(), trump)
}

// PRE Condition: Initial call, trick.certain should be "true" - the cards have already been played
func rankCard(playerid int, ht *HandTracker, trick *Trick, trump sdz.Suit) *Trick {
	//sdz.Log("calling Player%d rankCard on trick %s", playerid, trick)
	decisionMap := potentialCards(playerid, ht, trick.winningCard(), trick.leadSuit(), trump)
	if len(decisionMap) == 0 {
		Log(4, "playerid = %d", playerid)
		Log(4, "ht.PlayedCards = %v", HTString(ht.PlayedCards))
		for x := 0; x < 4; x++ {
			Log(4, "Player%d -------- %s", x, HTString(ht.Cards[x]))
		}
		panic("decisionMap should not be empty")
	}
	//sdz.Log("Player%d - Potential cards to play - %v", playerid, decisionMap)
	//sdz.Log("Received Trick %s", trick)
	var topTrick *Trick
	nextPlayer := (playerid + 1) % 4
	for _, card := range Decision(decisionMap).Sort() {
		tempTrick := new(Trick)
		*tempTrick = *trick // make a copy
		trick.Played = make(map[int]sdz.Card)
		for x := range tempTrick.Played { // now copy the map
			trick.Played[x] = tempTrick.Played[x]
		}
		tempTrick.Played[playerid] = card
		if tempTrick.leadSuit() == sdz.NASuit || tempTrick.winningCard() == sdz.NACard {
			tempTrick.Lead = playerid
			tempTrick.WinningPlayer = playerid
		}
		if CardBeatsTrick(card, tempTrick, trump) {
			tempTrick.WinningPlayer = playerid
			if _, ok := ht.Cards[playerid][card]; !ok {
				tempTrick.certain = false
			}
		}
		if len(tempTrick.Played) < 4 {
			tempTrick = rankCard(nextPlayer, ht, tempTrick, trump)
		}
		//sdz.Log("Playerid = %d - Top = %s, Temp = %s", playerid, topTrick, tempTrick)
		if topTrick == nil {
			topTrick = tempTrick
		} else {
			topWorth := topTrick.worth(playerid, trump)
			tempWorth := tempTrick.worth(playerid, trump)
			switch {
			case topWorth < tempWorth:
				topTrick = tempTrick
			case !topTrick.certain && !tempTrick.certain && (topWorth == tempWorth) && (card.Face().Less(topTrick.Played[playerid].Face())):
				topTrick = tempTrick
			case topWorth == tempWorth && topTrick.Played[playerid].Face().Less(card.Face()):
				topTrick = tempTrick
			}
			if topWorth < tempWorth || (topWorth == tempWorth && topTrick.Played[playerid].Face().Less(card.Face())) {
				topTrick = tempTrick
			}
		}
	}
	//sdz.Log("Returning worth %d - %s", topTrick.worth(playerid, trump), topTrick)
	return topTrick
}

func (ai *AI) findCardToPlay(action *sdz.Action) sdz.Card {
	ai.trick.certain = true
	//sdz.Log("htcardset - Player%d is calculating", ai.Playerid())
	//sdz.Log("Before rankCard as player %d", ai.Playerid())
	//sdz.Log("Playerid %d choosing card %s", ai.Playerid(), ai.trick.played[ai.Playerid()])
	return rankCard(ai.Playerid(), ai.ht, ai.trick, ai.trump).Played[ai.Playerid()]
}

func potentialCards(playerid int, ht *HandTracker, winning sdz.Card, lead sdz.Suit, trump sdz.Suit) map[sdz.Card]bool {
	//sdz.Log("PotentialCards called with %d,winning=%s,lead=%s,trump=%s", playerid, winning, lead, trump)
	//sdz.Log("PotentialCards Player%d - %#v", playerid, ht.Cards[playerid])
	trueHand := make(sdz.Hand, 0)
	potentialHand := make(sdz.Hand, 0)
	for _, card := range sdz.AllCards() {
		val, ok := ht.Cards[playerid][card]
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

func (ai *AI) Tell(action *sdz.Action) *sdz.Action {
	Log(ai.Playerid(), "Action received - %+v", action)
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
			return sdz.CreateBid(ai.bidAmount, ai.Playerid())
		} else {
			// received someone else's bid value'
			if ai.highBid < action.Bid {
				ai.highBid = action.Bid
				ai.highBidder = action.Playerid
			}
			ai.numBidders++
		}
	case "Play":
		Log(ai.Playerid(), "Trick = %s", ai.trick)
		if action.Playerid == ai.Playerid() {
			ai.action = sdz.CreatePlay(ai.findCardToPlay(action), ai.Playerid())
			action.PlayedCard = ai.action.PlayedCard
		}
		ai.trick.Played[action.Playerid] = action.PlayedCard
		if ai.trick.leadSuit() == sdz.NASuit || ai.trick.winningCard() == sdz.NACard {
			ai.trick.Lead = action.Playerid
			ai.trick.WinningPlayer = action.Playerid
			Log(ai.Playerid(), "Set lead to %s", ai.trick.leadSuit())
		} else if action.PlayedCard.Beats(ai.trick.winningCard(), ai.trump) {
			ai.trick.WinningPlayer = action.Playerid
		}
		ai.PlayCard(action.PlayedCard, action.Playerid)
		if action.Playerid != ai.Playerid() && action.PlayedCard.Suit() != ai.trick.leadSuit() {
			Log(ai.Playerid(), "nofollow - nosuit on Player%d on suit %s with trick %#v", action.Playerid, ai.trick.leadSuit(), ai.trick)
			ai.noSuit(action.Playerid, ai.trick.leadSuit())
			if ai.trick.leadSuit() != ai.trump && action.PlayedCard.Suit() != ai.trump {
				Log(ai.Playerid(), "notrump - nosuit on Player%d on suit %s", action.Playerid, ai.trump)
				ai.noSuit(action.Playerid, ai.trump)
			}
		}
		return ai.action
	case "Trump":
		if action.Playerid == ai.Playerid() {
			meld, _ := ai.hand.Meld(ai.trump)
			Log(ai.Playerid(), "Player %d being asked to name trump on hand %s and have %d meld", ai.Playerid(), ai.hand, meld)
			switch {
			// TODO add case for the end of the game like if opponents will coast out
			case ai.bidAmount < 15:
				return sdz.CreateThrowin(ai.Playerid())
			default:
				return sdz.CreateTrump(ai.trump, ai.Playerid())
			}
		} else {
			ai.trump = action.Trump
			Log(ai.Playerid(), "Trump is %s", ai.trump)
		}
	case "Throwin":
		Log(ai.Playerid(), "Player %d saw that player %d threw in", ai.Playerid(), action.Playerid)
	case "Deal":
		ai.reset()
		ai.hand = &action.Hand
		Log(ai.Playerid(), "Set playerid")
		Log(ai.Playerid(), "Dealt Hand = %s", ai.hand.String())
		ai.populate()
		ai.highBid = 20
		ai.highBid = action.Dealer
		ai.numBidders = 0
	case "Meld":
		Log(ai.Playerid(), "Received meld action - %#v", action)
		if action.Playerid == ai.Playerid() {
			return nil // seeing our own meld, we don't care
		}
		for _, card := range action.Hand {
			val, ok := ai.ht.Cards[action.Playerid][card]
			if !ok {
				ai.ht.Cards[action.Playerid][card] = 1
			} else if val == 1 {
				ai.ht.Cards[action.Playerid][card] = 2
			}
		}
	case "Message": // nothing to do here, no one to read it
	case "Trick": // nothing to do here, nothing to display
		Log(ai.Playerid(), "playedCards=%v", ai.ht.PlayedCards)
		ai.trick = NewTrick()
	case "Score": // TODO: save score to use for future bidding techniques
	default:
		Log(ai.Playerid(), "Received an action I didn't understand - %v", action)
	}
	return nil
}

func (a *AI) Listen() (action *sdz.Action, open bool) {
	Log(a.Playerid(), "Listen for playerid %d returning %+v", a.Playerid(), a.action)
	return a.action, true
}

func (a *AI) Hand() *sdz.Hand {
	return a.hand
}

func (a *AI) SetHand(h sdz.Hand, dealer, playerid int) {
	a.Id = playerid
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

func (h *Human) Tell(action *sdz.Action) *sdz.Action {
	jsonData, _ := json.Marshal(action)
	Log(h.Playerid(), "--> %s", jsonData)
	err := websocket.JSON.Send(h.conn, action)
	if err != nil {
		sdz.Log("Error in Send - %v", err)
		h.Close()
	}
	return nil
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

//func (h *Human) createGame(option int, cp *ConnectionPool) {
//	game := new(Game)
//	players := make([]sdz.Player, 4)
//	// connect players
//	players[0] = h
//	switch option {
//	case 1:
//		// Option 1 - Play against three AI players and start immediately
//		for x := 1; x < 4; x++ {
//			players[x] = createAI()
//		}
//	case 2:
//		// Option 2 - Play with a human partner against two AI players
//		players[1] = createAI()
//		players[2] = cp.Pop()
//		players[3] = createAI()
//	case 3:
//		// Option 3 - Play with a human partner against one AI players and 1 Human
//		players[1] = createAI()
//		players[2] = cp.Pop()
//		players[3] = cp.Pop()

//	case 4:
//		// Option 4 - Play with a human partner against two humans
//		for x := 1; x < 4; x++ {
//			players[x] = cp.Pop()
//		}
//	case 5:
//		// Option 5 - Play against a human with AI partners
//		players[1] = cp.Pop()
//		players[2] = createAI()
//		players[3] = createAI()
//	}
//	game.KickOff()
//}

//type ConnectionPool struct {
//	connections chan *Human
//}

//func (cp *ConnectionPool) Push(h *Human) {
//	cp.connections <- h
//	return
//}

//func (cp *ConnectionPool) Pop() *Human {
//	return <-cp.connections
//}

//func setupGame(net *websocket.Conn, cp *ConnectionPool) {
//	Log(4, "Connection received")
//	human := createHuman(net)
//	for {
//		for {
//			human.Tell(sdz.CreateMessage("Do you want to join a game, create a new game, or quit? (join, create, quit)"))
//			action := sdz.CreateHello("")
//			human.Tell(action)
//			action, open := human.Listen()
//			if !open {
//				return
//			}
//			if action.Message == "create" {
//				break
//			} else if action.Message == "join" {
//				human.Tell(sdz.CreateMessage("Waiting on a game to be started that you can join..."))
//				cp.Push(human)
//				<-human.finished // this will block to keep the websocket open
//				continue
//			} else if action.Message == "quit" {
//				human.Tell(sdz.CreateMessage("Ok, bye bye!"))
//				human.Close()
//				return
//			}
//		}
//		for {
//			human.Tell(sdz.CreateMessage("Option 1 - Play against three AI players and start immediately"))
//			human.Tell(sdz.CreateMessage("Option 2 - Play with a human partner against two AI players"))
//			human.Tell(sdz.CreateMessage("Option 3 - Play with a human partner against one AI players and 1 Human"))
//			human.Tell(sdz.CreateMessage("Option 4 - Play with a human partner against two humans"))
//			human.Tell(sdz.CreateMessage("Option 5 - Play against a human with AI partners"))
//			human.Tell(sdz.CreateMessage("Option 6 - Go back"))
//			human.Tell(sdz.CreateGame(0))
//			action, open := human.Listen()
//			if !open {
//				return
//			}
//			switch action.Option {
//			case 1:
//				fallthrough
//			case 2:
//				fallthrough
//			case 3:
//				fallthrough
//			case 4:
//				fallthrough
//			case 5:
//				human.createGame(action.Option, cp)
//			case 6:
//				break
//			default:
//				human.Tell(sdz.CreateMessage("Not a valid option"))
//			}
//			break // after their game is over, let's set them up again'
//		}
//	}
//}

//func wshandler(ws *websocket.Conn) {
//	//cookie, err := ws.Request().Cookie("pinochle")
//	//if err != nil {
//	//	Log("Could not get cookie - %v", err)
//	//	return
//	//}
//	setupGame(ws, cp)
//}

func logError(c appengine.Context, err error) bool {
	if err != nil {
		if appengine.IsOverQuota(err) {
			mail.SendToAdmins(c, &mail.Message{
				Sender:  "mzimmerman@gmail.com",
				Subject: "SDZPinochle is over quota!",
				Body:    fmt.Sprintf("SDZPinochle is over quota.  The error is:\n\n%#v", err),
			})
		}
		c.Errorf("Error - %v", err)
		c.Debugf("Stack = %s", debug.Stack())
		return true
	}
	return false
}

type Game struct {
	Trick       *Trick
	Deck        sdz.Deck
	Players     []sdz.Player
	Dealer      int
	Score       []int
	Meld        []int
	CountMeld   []bool
	Counters    []int
	HighBid     int
	HighPlayer  int
	Trump       sdz.Suit
	State       int
	Next        int
	Hands       []sdz.Hand
	HandsPlayed int
}

func NewGame(players int) *Game {
	game := new(Game)
	game.Players = make([]sdz.Player, players)
	game.Score = make([]int, players/2)
	game.Deck = sdz.CreateDeck()
	return game
}

// PRE : Players are already created and set
func (game *Game) NextHand(c appengine.Context) {
	game.Meld = make([]int, len(game.Players)/2)
	game.Trick = NewTrick()
	game.CountMeld = make([]bool, len(game.Players)/2)
	game.Counters = make([]int, len(game.Players)/2)
	game.HighBid = 20
	game.HighPlayer = game.Dealer
	game.State = StateBid
	game.Next = game.Dealer
	game.Deck.Shuffle()
	Log(4, "Dealer is %d", game.Dealer)
	hands := game.Deck.Deal()
	for x := 0; x < len(game.Players); x++ {
		game.Next = game.inc()
		sort.Sort(hands[x])
		game.Players[game.Next].SetHand(hands[x], game.Dealer, game.Next)
		Log(4, "Dealing player %d hand %s", game.Next, game.Players[game.Next].Hand())
	}
	game.Next = game.inc() // increment so that Dealer + 1 is asked to bid first
	game.processAction(c, game.Players[game.Next].Tell(sdz.CreateBid(0, game.Next)))
	// processAction will write the game to the datastore when it's done processing the action(s)
}

func (game *Game) inc() int {
	return (game.Next + 1) % len(game.Players)
}

//func (game *Game) Go(players []Player) {
//	game.Deck = CreateDeck()
//	game.Players = players
//	game.Score = make([]int, 2)
//	handsPlayed := 0
//	game.Dealer = 0
//	for {
//		handsPlayed++
//		// shuffle & deal
//		game.Deck.Shuffle()
//		hands := game.Deck.Deal()
//		next := game.Dealer
//		game.Meld = make([]int, len(game.Players))
//		game.MeldHands = make([]Hand, len(game.Players))
//		game.Counters = make([]int, len(game.Players))
//		for x := 0; x < len(game.Players); x++ {
//			next = (next + 1) % 4
//			sort.Sort(hands[x])
//			game.Players[next].SetHand(hands[x], game.Dealer, next)
//			//Log("Dealing player %d hand %s", next, game.Players[next].Hand())
//		}
//		// ask players to bid
//		game.HighBid = 20
//		game.HighPlayer = game.Dealer
//		next = game.Dealer
//		for x := 0; x < 4; x++ {
//			var bidAction *Action
//			next = (next + 1) % 4
//			if !(next == game.Dealer && game.HighBid == 20) { // no need to ask the dealer to bid if they've already won
//				game.Players[next].Tell(CreateBid(0, next))
//				var open bool
//				bidAction, open = game.Players[next].Listen()
//				if !open {
//					game.Broadcast(CreateMessage("Player disconnected"), next)
//					return
//				}
//			} else {
//				bidAction = CreateBid(game.HighBid, game.Dealer)
//			}
//			game.Broadcast(bidAction, next)
//			if bidAction.Bid > game.HighBid {
//				game.HighBid = bidAction.Bid
//				game.HighPlayer = next
//			}
//		}
//		// ask trump
//		game.Players[game.HighPlayer].Tell(CreateTrump(*new(Suit), game.HighPlayer))
//		response, open := game.Players[game.HighPlayer].Listen()
//		if !open {
//			game.Broadcast(CreateMessage("Player disconnected"), game.HighPlayer)
//			return
//		}
//		switch response.Type {
//		case "Throwin":
//			game.Broadcast(response, response.Playerid)
//			game.Score[game.HighPlayer%2] -= game.HighBid
//			game.BroadcastAll(CreateMessage(fmt.Sprintf("Scores are now Team0 = %d to Team1 = %d, played %d hands", game.Score[0], game.Score[1], handsPlayed)))
//			Log("Scores are now Team0 = %d to Team1 = %d, played %d hands", game.Score[0], game.Score[1], handsPlayed)
//			game.Dealer = (game.Dealer + 1) % 4
//			Log("-----------------------------------------------------------------------------")
//			continue
//		case "Trump":
//			game.Trump = response.Trump
//			Log("Trump is set to %s", game.Trump)
//			game.Broadcast(response, game.HighPlayer)
//		default:
//			panic("Didn't receive either expected response")
//		}
//		for x := 0; x < len(game.Players); x++ {
//			game.Meld[x], game.MeldHands[x] = game.Players[x].Hand().Meld(game.Trump)
//			meldAction := CreateMeld(game.MeldHands[x], game.Meld[x], x)
//			game.BroadcastAll(meldAction)
//		}
//		next = game.HighPlayer
//		countMeld := make([]bool, 2)
//		for trick := 0; trick < 12; trick++ {
//			var winningCard Card
//			var cardPlayed Card
//			var leadSuit Suit
//			winningPlayer := next
//			counters := 0
//			for x := 0; x < 4; x++ {
//				//Log("*******************************************************************************NEXT CARD")
//				// play the hand
//				// TODO: handle possible throwin
//				var action *Action
//				for {
//					action = CreatePlayRequest(winningCard, leadSuit, game.Trump, next, game.Players[next].Hand())
//					game.Players[next].Tell(action)
//					action, open = game.Players[next].Listen()
//					if !open {
//						game.Broadcast(CreateMessage("Player disconnected"), next)
//						return
//					}
//					cardPlayed = action.PlayedCard
//					if x > 0 {
//						if ValidPlay(cardPlayed, winningCard, leadSuit, game.Players[next].Hand(), game.Trump) &&
//							game.Players[next].Hand().Remove(cardPlayed) {
//							// playedCard, winningCard Card, leadSuit Suit, hand Hand, trump Suit
//							break
//						}
//					} else if game.Players[next].Hand().Remove(cardPlayed) {
//						break
//					}
//				}
//				switch cardPlayed.Face() {
//				case Ace:
//					fallthrough
//				case Ten:
//					fallthrough
//				case King:
//					counters++
//				}
//				if x == 0 {
//					winningCard = cardPlayed
//					leadSuit = cardPlayed.Suit()
//				} else {
//					if cardPlayed.Beats(winningCard, game.Trump) {
//						winningCard = cardPlayed
//						winningPlayer = next
//					}
//				}
//				game.Broadcast(action, next)
//				next = (next + 1) % 4
//			}
//			next = winningPlayer
//			if trick == 11 {
//				counters++
//			}
//			countMeld[winningPlayer%2] = true
//			game.BroadcastAll(CreateMessage(fmt.Sprintf("Player %d wins trick #%d with %s for %d points", winningPlayer, trick+1, winningCard, counters)))
//			game.BroadcastAll(CreateTrick(winningPlayer))
//			Log("Player %d wins trick #%d with %s for %d points", winningPlayer, trick+1, winningCard, counters)
//			game.Counters[game.Players[winningPlayer].Team()] += counters
//			//Log("*******************************************************************************NEXT TRICK")
//		}
//		game.Meld[0] += game.Meld[2]
//		game.Counters[0] += game.Counters[2]
//		game.Meld[1] += game.Meld[3]
//		game.Counters[1] += game.Counters[3]
//		if game.HighBid <= game.Meld[game.HighPlayer%2]+game.Counters[game.HighPlayer%2] {
//			game.Score[game.HighPlayer%2] += game.Meld[game.HighPlayer%2] + game.Counters[game.HighPlayer%2]
//		} else {
//			game.Score[game.HighPlayer%2] -= game.HighBid
//		}
//		if countMeld[(game.HighPlayer+1)%2] {
//			game.Score[(game.HighPlayer+1)%2] += game.Meld[(game.HighPlayer+1)%2] + game.Counters[(game.HighPlayer+1)%2]
//		}
//		// check the score for a winner
//		game.BroadcastAll(CreateMessage(fmt.Sprintf("Scores are now Team0 = %d to Team1 = %d, played %d hands", game.Score[0], game.Score[1], handsPlayed)))
//		Log("Scores are now Team0 = %d to Team1 = %d, played %d hands", game.Score[0], game.Score[1], handsPlayed)
//		win := make([]bool, 2)
//		gameOver := false
//		if game.Score[game.HighPlayer%2] >= 120 {
//			win[game.HighPlayer%2] = true
//			gameOver = true
//		} else if game.Score[(game.HighPlayer+1)%2] >= 120 {
//			win[(game.HighPlayer+1)%2] = true
//			gameOver = true
//		}
//		for x := 0; x < len(game.Players); x++ {
//			game.Players[x].Tell(CreateScore(x, game.Score, gameOver, win[x%2]))
//		}
//		if gameOver {
//			return
//		}
//		game.Dealer = (game.Dealer + 1) % 4
//		Log("-----------------------------------------------------------------------------")
//	}
//	for x := 0; x < 4; x++ {
//		game.Players[x].Close()
//	}
//}

func (g Game) Broadcast(a *sdz.Action, p int) {
	for x, player := range g.Players {
		if p != x {
			player.Tell(a)
		}
	}
}

func (g Game) BroadcastAll(a *sdz.Action) {
	g.Broadcast(a, -1)
}

//func gameAction(w http.ResponseWriter, r *http.Request) {
//	c := appengine.NewContext(r)
//	game := new(Game)
//	err := datastore.Get("id", game)
//	action := new(sdz.Action)
//	logError(c, err)
//}

func (game *Game) processAction(c appengine.Context, action *sdz.Action) {
	for {
		if action == nil {
			// waiting on a human, save the state and exit
			return
		}
		switch game.State {
		case StateBid:
			if action.Type != "Bid" {
				logError(c, errors.New("Received non bid action"))
				action = nil
				continue
			}
			if action.Playerid != game.Next {
				logError(c, errors.New("It's not your turn!"))
				action = nil
				continue
			}
			game.Broadcast(action, game.Next)
			if action.Bid > game.HighBid {
				game.HighBid = action.Bid
				game.HighPlayer = game.Next
			}
			if game.HighPlayer == game.Dealer && game.inc() == game.Dealer { // dealer was stuck, tell everyone
				game.Broadcast(sdz.CreateBid(game.HighBid, game.Dealer), game.Dealer)
				game.Next = game.inc()
			}
			if game.Next == game.Dealer { // the bidding is done
				game.State = StateTrump
				game.Next = game.HighPlayer
				action = game.Players[game.HighPlayer].Tell(sdz.CreateTrump(sdz.NASuit, game.HighPlayer))
				continue
			}
			game.Next = game.inc()
			action = game.Players[game.Next].Tell(sdz.CreateBid(0, game.Next))
			continue
		case StateTrump:
			switch action.Type {
			case "Throwin":
				game.Broadcast(action, action.Playerid)
				game.Score[game.HighPlayer%2] -= game.HighBid
				game.BroadcastAll(sdz.CreateMessage(fmt.Sprintf("Scores are now Team0 = %d to Team1 = %d, played %d hands", game.Score[0], game.Score[1], game.HandsPlayed)))
				Log(4, "Scores are now Team0 = %d to Team1 = %d, played %d hands", game.Score[0], game.Score[1], game.HandsPlayed)
				game.Dealer = (game.Dealer + 1) % 4
				Log(4, "-----------------------------------------------------------------------------")
				game.NextHand(c)
				continue
			case "Trump":
				game.Trump = action.Trump
				Log(4, "Trump is set to %s", game.Trump)
				game.Broadcast(action, game.HighPlayer)
				for x := 0; x < len(game.Players); x++ {
					meld, meldHand := game.Players[x].Hand().Meld(game.Trump)
					meldAction := sdz.CreateMeld(meldHand, meld, x)
					game.BroadcastAll(meldAction)
					game.Meld[x%2] += meld
				}
				game.Next = game.HighPlayer
				game.Counters = make([]int, 2)
				game.State = StatePlay
				action = game.Players[game.Next].Tell(sdz.CreatePlayRequest(game.Trick.winningCard(), game.Trick.leadSuit(), game.Trump, game.Next, game.Players[game.Next].Hand()))
				continue
			}
		case StatePlay:
			// TODO: check for throw in
			if sdz.ValidPlay(action.PlayedCard, game.Trick.winningCard(), game.Trick.leadSuit(), game.Players[game.Next].Hand(), game.Trump) &&
				game.Players[game.Next].Hand().Remove(action.PlayedCard) {
				game.Trick.Played[game.Next] = action.PlayedCard
				game.Broadcast(action, game.Next)
				if len(game.Trick.Played) == 1 {
					game.Trick.Lead = game.Next
				}
				if game.Trick.Played[game.Next].Beats(game.Trick.winningCard(), game.Trump) {
					game.Trick.WinningPlayer = game.Next
				}
			}
			if len(game.Trick.Played) == len(game.Players) {
				for _, card := range game.Trick.Played {
					if card.Counter() {
						game.Counters[game.Trick.WinningPlayer%2]++
					}
				}
				game.CountMeld[game.Trick.WinningPlayer%2] = true
				game.Next = game.Trick.WinningPlayer
				game.BroadcastAll(sdz.CreateMessage(fmt.Sprintf("Player %d wins trick with %s", game.Trick.WinningPlayer, game.Trick.winningCard())))
				game.BroadcastAll(sdz.CreateTrick(game.Trick.WinningPlayer))
				Log(4, "Player %d wins trick with %s", game.Trick.WinningPlayer, game.Trick.winningCard())
				if len(*game.Players[0].Hand()) == 0 {
					game.Counters[game.Trick.WinningPlayer%2]++ // last trick
					// end of hand
					game.HandsPlayed++
					if game.HighBid <= game.Meld[game.HighPlayer%2]+game.Counters[game.HighPlayer%2] {
						game.Score[game.HighPlayer%2] += game.Meld[game.HighPlayer%2] + game.Counters[game.HighPlayer%2]
					} else {
						game.Score[game.HighPlayer%2] -= game.HighBid
					}
					if game.CountMeld[(game.HighPlayer+1)%2] {
						game.Score[(game.HighPlayer+1)%2] += game.Meld[(game.HighPlayer+1)%2] + game.Counters[(game.HighPlayer+1)%2]
					}
					// check the score for a winner
					game.BroadcastAll(sdz.CreateMessage(fmt.Sprintf("Scores are now Team0 = %d to Team1 = %d, played %d hands", game.Score[0], game.Score[1], game.HandsPlayed)))
					Log(4, "Scores are now Team0 = %d to Team1 = %d, played %d hands", game.Score[0], game.Score[1], game.HandsPlayed)
					win := make([]bool, 2)
					gameOver := false
					if game.Score[game.HighPlayer%2] >= 120 {
						win[game.HighPlayer%2] = true
						gameOver = true
					} else if game.Score[(game.HighPlayer+1)%2] >= 120 {
						win[(game.HighPlayer+1)%2] = true
						gameOver = true
					}
					for x := 0; x < len(game.Players); x++ {
						game.Players[x].Tell(sdz.CreateScore(x, game.Score, gameOver, win[x%2]))
					}
					if gameOver {
						return // game over
					}
					game.Dealer = (game.Dealer + 1) % 4
					Log(4, "-----------------------------------------------------------------------------")
					game.NextHand(c)
					continue
				}
				game.Trick = NewTrick()
			}
			game.Next = game.inc()
			action = game.Players[game.Next].Tell(sdz.CreatePlayRequest(game.Trick.winningCard(), game.Trick.leadSuit(), game.Trump, game.Next, game.Players[game.Next].Hand()))
			continue
		}
	}
}

//func serveGame(w http.ResponseWriter, r *http.Request) {
//	listTmpl, err := template.New("tempate").ParseGlob("*.html")
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
//	err = listTmpl.ExecuteTemplate(w, "game", nil)
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
//}

//var cp *ConnectionPool

//func listenForFlash() {
//	response := []byte("<cross-domain-policy><allow-access-from domain=\"*\" to-ports=\"*\" /></cross-domain-policy>")
//	ln, err := net.Listen("tcp", ":843")
//	if err != nil {
//		sdz.Log("Cannot listen on port 843 for flash policy file, will not serve non WebSocket clients, check permissions or run as root - " + err.Error())
//		return
//	}
//	for {
//		conn, err := ln.Accept()
//		if err != nil {
//			// handle error
//			continue
//		}
//		conn.Write(response)
//		conn.Close()
//	}
//}

//func main() {
//	go listenForFlash()
//	cp = &ConnectionPool{connections: make(chan *Human, 100)}
//	http.Handle("/connect", websocket.Handler(wshandler))
//	http.Handle("/cards/", http.StripPrefix("/cards/", http.FileServer(http.Dir("cards"))))
//	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("js"))))
//	http.Handle("/web-socket-js/", http.StripPrefix("/web-socket-js/", http.FileServer(http.Dir("web-socket-js"))))
//	http.HandleFunc("/", serveGame)
//	err := http.ListenAndServe(":80", nil)
//	if err != nil {
//		sdz.Log("Cannot listen on port 80, check permissions or run as root - " + err.Error())
//	}
//	err = http.ListenAndServe(":1080", nil)
//	if err != nil {
//		panic("Cannot listen for http requests - " + err.Error())
//	}
//}
