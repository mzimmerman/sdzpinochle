// sdzpinochle-client project main.go
package main

import (
	"encoding/json"
	"fmt"
	sdz "github.com/mzimmerman/sdzpinochle"
	"math/rand"
	//"sort"
	"net"
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
	c          chan *sdz.Action
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
	a.c = make(chan *sdz.Action)
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
		switch action.Type {
		case "Bid":
			if action.Playerid == ai.Playerid() {
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
				if ai.highBid < action.Bid {
					ai.highBid = action.Bid
					ai.highBidder = action.Playerid
				}
				ai.numBidders++
			}
		case "Play":
			if action.Playerid == ai.Playerid() {
				if action.WinningCard == "" { // nothing to compute as far as legal moves
					action = sdz.CreatePlay((*ai.hand)[0], ai.Playerid())
				} else {
					for _, card := range *ai.hand {
						// playedCard, winningCard Card, leadSuit Suit, hand Hand, trump Suit
						if sdz.ValidPlay(card, action.WinningCard, action.Lead, ai.hand, action.Trump) {
							action = sdz.CreatePlay(card, ai.Playerid())
							break
						}
					}
				}
				ai.c <- action
			} else {
				// TODO: Keep track of what has been played already
				// received someone else's play'
			}
		case "Trump":
			if action.Playerid == ai.Playerid() {
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
				ai.trump = action.Trump
			}
		case "Throwin":
			Log("Player %d saw that player %d threw in", ai.Playerid(), action.Playerid)
		case "Deal": // should not happen as the server can set the Hand automagically for AI
		case "Meld":
		default:
			Log("Received an action I didn't understand - %v", action)
		}
	}
}

func (ai AI) Tell(action *sdz.Action) {
	ai.c <- action
}

func (a AI) Listen() (action *sdz.Action, open bool) {
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
	conn *net.Conn
	enc  *json.Encoder
	dec  *json.Decoder
	//trump      sdz.Suit
	//bidAmount  int
	//highBid    int
	//highBidder int
	//numBidders int
	//show       sdz.Hand
	sdz.PlayerImpl
}

func createHuman(x int, conn *net.Conn, enc *json.Encoder, dec *json.Decoder) (a *Human) {
	human := &Human{sdz.PlayerImpl: sdz.PlayerImpl{Id: x}, conn: conn, enc: enc, dec: dec}
	//human.PlayerImpl.Id = x
	return human
}

func (h Human) Close() {
	//h.conn.Close()
}

func (h *Human) Go() {
	// nothing to do here, the client is where this "thread" runs
}

func (h *Human) Tell(action *sdz.Action) {
	h.enc.Encode(action)
}

func (h *Human) Listen() (action *sdz.Action, open bool) {
	action = new(sdz.Action)
	err := h.dec.Decode(action)
	if err != nil {
		return action, false
	}
	return nil, true
}

func (h Human) Hand() *sdz.Hand {
	return h.hand
}

func (a *Human) SetHand(h sdz.Hand, dealer int) {
	hand := make(sdz.Hand, len(h))
	copy(hand, h)
	a.hand = &hand
}

func (h *Human) createGame(humans int, cp *ConnectionPool) {
	game := new(sdz.Game)
	players := make([]sdz.Player, 4)
	// connect players
	players[0] = h
	go players[0].Go()
	for x := 1; x < humans; x++ {
		players[x] = cp.Pop()
		go players[x].Go()
	}
	for x := humans; x < len(players); x++ {
		players[x] = createAI(x)
		go players[x].Go()
	}
	game.Players = players
	game.Go(players)
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
	Log("Connection received")
	human := createHuman(0, net, json.NewEncoder(*net), json.NewDecoder(*net))
	action := sdz.CreateHello("Do you want to join or create a new game?")
	human.Tell(action)
	action, _ = human.Listen()
	if action.Message == "create" {
		human.createGame(1, cp)
	} else {
		// old
		cp.Push(human)
	}
}

func main() {
	cp := ConnectionPool{make(chan *Human, 100)}
	service := ":1201"
	tcpAddr, err := net.ResolveTCPAddr("tcp4", service)
	if err != nil {
		Log("Error - %v", err)
		return
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		Log("Error - %v", err)
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
