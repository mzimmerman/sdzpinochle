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

type HandTracker struct {
	cards [4]map[sdz.Card]int
	// missing entry = know nothing
	// 0 = does not have any of this card
	// 1 = has this card
	// 2 = has two of these cards
	playedCards map[sdz.Card]int
}

// Returns a new HandTracker with the initial population and calculation done
func NewHandTracker(playerid int, hand sdz.Hand) *HandTracker {
	ht := new(HandTracker)
	for x := 0; x < 4; x++ {
		ht.cards[x] = make(map[sdz.Card]int)
	}
	ht.playedCards = make(map[sdz.Card]int)
	for _, suit := range sdz.Suits() {
		for _, face := range sdz.Faces() {
			card := sdz.CreateCard(suit, face)
			ht.playedCards[card] = 0
		}
	}
	ht.populate(playerid, hand)
	return ht
}

func (ht *HandTracker) populate(playerid int, hand sdz.Hand) {
	for _, suit := range sdz.Suits() {
		for _, face := range sdz.Faces() {
			card := sdz.CreateCard(suit, face)
			ht.cards[playerid][card] = 0
		}
	}
	for _, card := range hand {
		ht.cards[playerid][card]++
	}
	ht.calculate()
}

func (ht *HandTracker) noSuit(playerid int, suit sdz.Suit) {
	for _, face := range sdz.Faces() {
		ht.cards[playerid][sdz.CreateCard(suit, face)] = 0
	}
}

func (ht *HandTracker) calculate() {
	for _, suit := range sdz.Suits() {
		for _, face := range sdz.Faces() {
			card := sdz.CreateCard(suit, face)
			sum := ht.playedCards[card]
			for x := 0; x < 4; x++ {
				if val, ok := ht.cards[x][card]; ok {
					sum += val
				}
			}
			if sum == 2 {
				for x := 0; x < 4; x++ {
					if _, ok := ht.cards[x][card]; !ok {
						ht.cards[x][card] = 0
					}
				}
			} else {
				unknown := -1
				for x := 0; x < 4; x++ {
					if _, ok := ht.cards[x][card]; !ok {
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
					ht.cards[unknown][card] = 2 - sum
				}
			}
		}
	}
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
	ht        *HandTracker
	playCount int
}

func createAI() (a *AI) {
	a = new(AI)
	a.c = make(chan *sdz.Action)
	return a
}

func (ai AI) Close() {
	close(ai.c)
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
				decisionMap := make(map[sdz.Card]int)
				for _, card := range *ai.hand {
					if ai.playCount == 0 || sdz.ValidPlay(card, action.WinningCard, action.Lead, ai.hand, action.Trump) {
						decisionMap[card] = 1
					}
				}
				action = sdz.CreatePlay(ai.findCardToPlay(action, decisionMap), ai.Playerid())
				ai.ht.playedCards[action.PlayedCard]++
				ai.c <- action
			} else {
				ai.playCount++
				ai.ht.playedCards[action.PlayedCard]++
				if action.PlayedCard.Suit() != action.Lead {
					ai.ht.noSuit(action.Playerid, action.Lead)
					if action.PlayedCard.Suit() != action.Trump {
						ai.ht.noSuit(action.Playerid, action.Trump)
					}
				}
				// TODO: find all the cards that can beat the lead card and set those
				// received someone else's play
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
		case "Deal": // nothing to do here, this is set automagically
		case "Meld": // nothing to do here, no one to read it
		case "Message": // nothing to do here, no one to read it
		case "Trick": // nothing to do here, nothing to display
			ai.playCount = 0
		case "Score": // TODO: save score to use for future bidding techniques
		default:
			Log("Received an action I didn't understand - %v", action)
		}
	}
}

func (ai *AI) findCardToPlay(action *sdz.Action, decisionMap map[sdz.Card]int) sdz.Card {
	var selection sdz.Card
	for card := range decisionMap {
		if selection == "" {
			selection = card
		}
		if ai.playCount == 0 {
			// TODO: Anticipate opponents trumping in
			if card.Face() == sdz.Ace {
				// choose the Ace with the least amount of cards in the suit
				decisionMap[card] += 12 - ai.hand.CountSuit(card.Suit())
			}
		} else if card.Beats(action.WinningCard, action.Trump) {
			if ai.hand.Highest(card) {
				decisionMap[card]++
			}
		} else if ai.playCount > 0 && action.WinningPlayer%2 == ai.Playerid()%2 {
			if card.Counter() {
				decisionMap[card]++
			} else {
				decisionMap[card]--
			}
		} else {
			// TODO: Anticipate partner winning the hand through trump or otherwise
			if ai.hand.Lowest(card) {
				decisionMap[card]++
			}
		}
		if decisionMap[card] > decisionMap[selection] {
			selection = card
		}
	}
	Log("%d - Playing %s - Decision map = %v", ai.Playerid(), selection, decisionMap)
	return selection
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

func (a *AI) SetHand(h sdz.Hand, dealer, playerid int) {
	hand := make(sdz.Hand, len(h))
	copy(hand, h)
	a.hand = &hand
	a.Id = playerid
	a.highBid = 20
	a.highBidder = dealer
	a.numBidders = 0
	a.ht = NewHandTracker(playerid, h)
	a.playCount = 0
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

func (h *Human) Go() {
	// this "thread" runs on the client
}

func (h *Human) Tell(action *sdz.Action) {
	jsonData, _ := json.Marshal(action)
	Log("--> %s", jsonData)
	err := websocket.JSON.Send(h.conn, action)
	if err != nil {
		sdz.Log("Error in Human.Go Send - %v", err)
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
	Log("<-- %s", jsonData)
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
	for x := 0; x < len(players); x++ {
		go players[x].Go() // the humans will just start and die immediately
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
	Log("Connection received")
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
