// sdzpinochle-client project main.go
package main

import (
	//	"encoding/json"
	"fmt"
	sdz "github.com/mzimmerman/sdzpinochle"
	"math/rand"
	"sort"
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

type AI struct {
	hand       sdz.Hand
	c          chan sdz.Action
	trump      sdz.Suit
	bidAmount  int
	highBid    int
	highBidder int
	numBidders int
	show       map[sdz.Card]int
	sdz.PlayerImpl
}

func Log(m string, v ...interface{}) {
	fmt.Printf(m+"\n", v...)
}

func (ai AI) Close() {
	close(ai.c)
}

func (ai AI) powerBid(suit sdz.Suit) (count int) {
	count = 7 // your partner's good for at least this right?!?
	suitMap := make(map[sdz.Suit]int)
	for _, card := range ai.hand {
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

func (ai AI) calculateBid() (amount int, trump sdz.Suit, show map[sdz.Card]int) {
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
				if ai.highBid < action.Value().(int) {
					ai.highBid = action.Value().(int)
					ai.highBidder = action.Playerid()
				}
				ai.numBidders++
			}
		case *sdz.PlayAction:
			if action.Playerid() == ai.Playerid() {
				action = sdz.CreatePlay(ai.hand.Play(0), ai.Playerid())
				Log("Player %d played card %s", ai.Playerid(), action.Value())
				ai.c <- action
			} else {
				// TODO: Keep track of what has been played already
				// received someone else's play'
			}
		case *sdz.TrumpAction:
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
				Log("Player %d was told trump", ai.Playerid())
				ai.trump = action.Value().(sdz.Suit)
			}
		case *sdz.ThrowinAction:
			Log("Player %d saw that player %d threw in", ai.Playerid(), action.Playerid())
		case *sdz.DealAction: // should not happen as the server can set the Hand automagically
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
func (a AI) Hand() sdz.Hand {
	return a.hand
}

func (a *AI) SetHand(h sdz.Hand, dealer int) {
	a.hand = h
	a.highBid = 20
	a.highBidder = dealer
	a.numBidders = 0
}

type Human struct {
	hand sdz.Hand
	c    chan sdz.Action
	sdz.PlayerImpl
}

func createHuman(x int) (h *Human) {
	h = new(Human)
	h.Id = x
	h.c = make(chan sdz.Action)
	return h
}

func (human Human) Playerid() int {
	return human.Playerid()
}

func (human Human) Tell(action sdz.Action) {
	human.c <- action
	// TODO: do some network stuff here
}
func (human Human) Listen() (action sdz.Action, closed bool) {
	// TODO: do some network stuff here
	action, closed = <-human.c
	return
}
func (human Human) Hand() sdz.Hand {
	return human.Hand()
}
func (human Human) SetHand(h sdz.Hand) {
	human.hand = h
	human.Tell(sdz.CreateDeal(h, human.Playerid()))
}

func createAI(x int) (a *AI) {
	a = new(AI)
	a.Id = x
	a.c = make(chan sdz.Action)
	return a
}

func main() {
	game := createGame()
	game.Players = make([]sdz.Player, 4)
	// connect players
	for x := 0; x < 4; x++ {
		game.Players[x] = createAI(x)
		go game.Players[x].Go()
	}
	for {
		// shuffle & deal
		game.Deck.Shuffle()
		hands := game.Deck.Deal()
		next := game.Dealer
		for x := 0; x < 4; x++ {
			next = (next + 1) % 4
			sort.Sort(hands[x])
			game.Players[next].SetHand(hands[x], game.Dealer)
			Log("Dealing player %d hand %s", next, game.Players[next].Hand())
		}
		// ask players to bid
		game.HighBid = 20
		game.HighPlayer = game.Dealer
		next = game.Dealer
		for x := 0; x < 4; x++ {
			next = (next + 1) % 4
			game.Players[next].Tell(sdz.CreateBid(0, next))
			bid, _ := game.Players[next].Listen()
			game.Broadcast(bid, next)
			if bid.Value().(int) > game.HighBid {
				game.HighBid = bid.Value().(int)
				game.HighPlayer = next
			}
		}
		// ask trump
		game.Players[game.HighPlayer].Tell(sdz.CreateTrump(*new(sdz.Suit), game.HighPlayer))
		response, _ := game.Players[game.HighPlayer].Listen()
		switch response.(type) {
		case *sdz.ThrowinAction:
			game.Broadcast(response, response.Playerid())
			// TODO: adjust the score
		case *sdz.TrumpAction:
			game.Trump = response.Value().(sdz.Suit)
			game.Broadcast(response, game.HighPlayer)
		}
		// TODO: send meld of each player out to clients
		next = game.HighPlayer
		for trick := 0; trick < 12; trick++ {
			for x := 0; x < 4; x++ {
				// play the hand
				// TODO: handle possible throwin
				var action sdz.Action
				action = sdz.CreatePlay(*new(sdz.Card), next)
				game.Players[next].Tell(action)
				action, _ = game.Players[next].Listen()
				// TODO: verify legal move for player
				game.Broadcast(action, next)
				next = (next + 1) % 4
			}
			// next = winnerOfTheTrick
			// add counters to winner's points'
		}
		for x := 0; x < 4; x++ {
			game.Dealer = (game.Dealer + 1) % 4
			game.Players[game.Dealer].Close()
		}
		return
		// check the score for a winner
		game.Dealer = (game.Dealer + 1) % 4
	}
}

func createGame() (game *sdz.Game) {
	game = &sdz.Game{}
	game.Deck = sdz.CreateDeck()
	game.Dealer = 0
	return
}
