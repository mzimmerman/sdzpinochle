// sdzpinochle-client project main.go
package main

import (
	//	"encoding/json"
	"fmt"
	sdz "github.com/mzimmerman/sdzpinochle"
	"math/rand"
	//"sort"
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

func Log(m string, v ...interface{}) {
	fmt.Printf(m+"\n", v...)
}

type AI struct {
	hand       *sdz.Hand
	c          chan sdz.Action
	trump      sdz.Suit
	bidAmount  int
	highBid    int
	highBidder int
	numBidders int
	show       sdz.Hand
	sdz.PlayerImpl
}

func createAI(x int) (a *AI) {
	a = new(AI)
	a.Id = x
	a.c = make(chan sdz.Action)
	return a
}

func (ai AI) Close() {
	close(ai.c)
}

func (ai AI) powerBid(suit sdz.Suit) (count int) {
	count = 7 // your partner's good for at least this right?!?
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

func (ai *AI) Go() {
	for {
		action, open := ai.Listen()
		//Log("Action received by player %d with hand %s - %+v", ai.Playerid(), ai.hand, action)
		if !open {
			return
		}
		switch action.(type) {
		case *sdz.BidAction:
			bidAction := action.(*sdz.BidAction)
			if action.Playerid() == ai.Playerid() {
				Log("------------------Player %d asked to bid against player %d", ai.Playerid(), ai.highBidder)
				ai.bidAmount, ai.trump, ai.show = ai.calculateBid()
				if ai.numBidders == 1 && ai.IsPartner(ai.highBidder) && ai.bidAmount < 21 && ai.bidAmount+5 > 20 {
					// save our parter
					Log("Saving our partner with a recommended bid of %d", ai.bidAmount)
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
				Log("------------------Player %d bid %d over %d with recommendation of %d and %d meld", ai.Playerid(), ai.bidAmount, ai.highBid, bidAmountOld, meld)
				ai.c <- sdz.CreateBid(ai.bidAmount, ai.Playerid())
			} else {
				// received someone else's bid value'
				if ai.highBid < bidAction.Bid() {
					ai.highBid = bidAction.Bid()
					ai.highBidder = action.Playerid()
				}
				ai.numBidders++
			}
		case *sdz.PlayAction:
			if action.Playerid() == ai.Playerid() {
				playRequest := action.(*sdz.PlayAction)
				if playRequest.WinningCard() == "" { // nothing to compute as far as legal moves
					playRequest = sdz.CreatePlay((*ai.hand)[0], ai.Playerid())
				} else {
					for _, card := range *ai.hand {
						// playedCard, winningCard Card, leadSuit Suit, hand Hand, trump Suit
						if sdz.ValidPlay(card, playRequest.WinningCard(), playRequest.Lead(), ai.hand, playRequest.Trump()) {
							playRequest = sdz.CreatePlay(card, ai.Playerid())
							break
						}
					}
				}
				//Log("Player %d played card %s", ai.Playerid(), playRequest.PlayedCard())
				ai.c <- playRequest
			} else {
				// TODO: Keep track of what has been played already
				// received someone else's play'
			}
		case *sdz.TrumpAction:
			trumpAction := action.(*sdz.TrumpAction)
			if action.Playerid() == ai.Playerid() {
				meld, _ := ai.hand.Meld(ai.trump)
				Log("Player %d being asked to name trump on hand %s and have %d meld", ai.Playerid(), ai.hand, meld)
				switch {
				// TODO add case for the end of the game like if opponents will coast out
				case ai.bidAmount < 15:
					ai.c <- sdz.CreateThrowin(ai.Playerid())
				default:
					ai.c <- sdz.CreateTrump(ai.trump, ai.Playerid())
				}
			} else {
				//Log("Player %d was told trump", ai.Playerid())
				ai.trump = trumpAction.Trump()
			}
		case *sdz.ThrowinAction:
			Log("Player %d saw that player %d threw in", ai.Playerid(), action.Playerid())
		case *sdz.DealAction: // should not happen as the server can set the Hand automagically for AI
		case *sdz.MeldAction:
		default:
			Log("Received an action I didn't understand - %v", action)
		}
	}
}

func (ai AI) Tell(action sdz.Action) {
	ai.c <- action
}
func (a AI) Listen() (action sdz.Action, open bool) {
	action, open = <-a.c
	return
}
func (a AI) Hand() *sdz.Hand {
	return a.hand
}

func (a *AI) SetHand(h sdz.Hand, dealer int) {
	hand := make(sdz.Hand, len(h))
	copy(hand, h)
	a.hand = &hand
	a.highBid = 20
	a.highBidder = dealer
	a.numBidders = 0
}

type Human struct {
	hand *sdz.Hand
	c    chan sdz.Action
	//trump      sdz.Suit
	//bidAmount  int
	//highBid    int
	//highBidder int
	//numBidders int
	//show       sdz.Hand
	sdz.PlayerImpl
}

func createHuman(x int) (a *Human) {
	a = new(Human)
	a.Id = x
	a.c = make(chan sdz.Action)
	return a
}

func (h Human) Close() {
	close(h.c)
}

func (h *Human) Go() {
	Log("Welcome to Single-Deck Zimmerman Pinochle!  You are player #%d!", h.Playerid())
	for {
		var bidAmount int
		action, open := h.Listen()
		if !open {
			return
		}
		switch action.(type) {
		case *sdz.BidAction:
			bidAction := action.(*sdz.BidAction)
			if action.Playerid() == h.Playerid() {
				Log("How much would you like to bid?:")
				fmt.Scan(&bidAmount)
				h.c <- sdz.CreateBid(bidAmount, h.Playerid())
			} else {
				// received someone else's bid value'
				Log("Player #%d bid %d", bidAction.Playerid(), bidAction.Bid())
			}
		case *sdz.PlayAction:
			playRequest := action.(*sdz.PlayAction)
			if action.Playerid() == h.Playerid() {
				var card sdz.Card
				Log("Your turn, in your hand is %s - what would you like to play?:", h.Hand())
				fmt.Scan(&card)
				Log("Received input %s", card)
				playRequest = sdz.CreatePlay(card, h.Playerid())
				//if playRequest.WinningCard() == "" { // nothing to compute as far as legal moves
				//	playRequest = sdz.CreatePlay((*ai.hand)[0], ai.Playerid())
				//} else {
				//	for _, card := range *ai.hand {
				//		// playedCard, winningCard Card, leadSuit Suit, hand Hand, trump Suit
				//		if sdz.ValidPlay(card, playRequest.WinningCard(), playRequest.Lead(), ai.hand, playRequest.Trump()) {
				//			playRequest = sdz.CreatePlay(card, ai.Playerid())
				//			break
				//		}
				//	}
				//}
				//Log("Player %d played card %s", ai.Playerid(), playRequest.PlayedCard())
				h.c <- playRequest
			} else {
				Log("Player %d played card %s", playRequest.Playerid(), playRequest.PlayedCard())
				// TODO: Keep track of what has been played already
				// received someone else's play'
			}
		case *sdz.TrumpAction:
			trumpAction := action.(*sdz.TrumpAction)
			if action.Playerid() == h.Playerid() {
				var trump sdz.Suit
				Log("What would you like to make trump?")
				fmt.Scan(&trump)
				h.c <- sdz.CreateTrump(trump, h.Playerid())
			} else {
				Log("Player %d says trump is %s", trumpAction.Playerid(), trumpAction.Trump())
			}
		case *sdz.ThrowinAction:
			Log("Player %d saw that player %d threw in", h.Playerid(), action.Playerid())
		case *sdz.DealAction:
			dealAction := action.(*sdz.DealAction)
			Log("Your hand is - %s", dealAction.Hand())
		case *sdz.MeldAction:
			meldAction := action.(*sdz.MeldAction)
			Log("Player %d is melding %s for %d points", meldAction.Playerid(), meldAction.Hand(), meldAction.Amount())
		default:
			Log("Received an action I didn't understand - %v", action)
		}
	}
}

func (h Human) Tell(action sdz.Action) {
	h.c <- action
}

func (h Human) Listen() (action sdz.Action, open bool) {
	action, open = <-h.c
	return
}
func (h Human) Hand() *sdz.Hand {
	return h.hand
}

func (a *Human) SetHand(h sdz.Hand, dealer int) {
	hand := make(sdz.Hand, len(h))
	copy(hand, h)
	a.hand = &hand
}

func main() {
	game := sdz.CreateGame()
	players := make([]sdz.Player, 4)
	// connect players
	players[0] = createHuman(0)
	go players[0].Go()
	for x := 1; x < len(players); x++ {
		players[x] = createAI(x)
		go players[x].Go()
	}
	game.Go(players)
}
