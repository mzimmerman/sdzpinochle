package sdzpinochleserver

import (
	"appengine"
	"appengine/channel"
	"appengine/datastore"
	"bytes"
	"encoding/gob"
	"github.com/gorilla/sessions"
	"log"
	//"appengine/datastore"
	"appengine/mail"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mzimmerman/goon"
	sdz "github.com/mzimmerman/sdzpinochle"
	"math/rand"
	//"net"
	"net/http"
	"runtime/debug"
	"sort"
	"strconv"
	"time"
)

const (
	ace        = "A"
	ten        = "T"
	king       = "K"
	queen      = "Q"
	jack       = "J"
	nine       = "9"
	spades     = "S"
	hearts     = "H"
	clubs      = "C"
	diamonds   = "D"
	StateNew   = "new"
	StateBid   = "bid"
	StateTrump = "trump"
	StateMeld  = "meld"
	StatePlay  = "play"
	cookieName = "sdzpinochle"
)

var store = sessions.NewCookieStore([]byte("sdzpinochle"))

func init() {
	http.HandleFunc("/connect", connect)
	http.HandleFunc("/_ah/channel/connected/", connected)
	http.HandleFunc("/receive", receive)
	http.HandleFunc("/remind", remind)
	store.Options = &sessions.Options{
		Path:   "/",
		MaxAge: 3600, // keep the cookie for one hour
	}
	gob.Register(new(AI))
	//gob.Register(AI{})
	gob.Register(new(Human))
	//gob.Register(Human{})
}

func remind(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	g := goon.FromContext(c)
	late := time.Now().Add(-time.Minute)
	query := datastore.NewQuery("Game").Filter("Updated < ", late).KeysOnly()
	gameKeys, err := g.GetAll(query, nil)
	if logError(c, err) {
		return
	}
	games := make([]*Game, len(gameKeys))
	for x := range gameKeys {
		games[x] = &Game{Id: gameKeys[x].IntID()}
	}
	err = g.GetMulti(&games)
	for _, game := range games {
		if game.Updated.After(late) {
			continue
		}
		game.retell(g, c)
		endGameTime := time.Now().Add(-30 * time.Minute)
		if game.Updated.Before(endGameTime) {
			for _, player := range game.Players {
				if human, ok := player.(*Human); ok {
					human.Client.TableId = 0
					_, err := g.Put(human.Client)
					if logError(c, err) {
						return
					}
				}
			}
		}
	}
}

func connected(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	client := new(Client)
	client.setId(r.FormValue("from"))
	g := goon.FromContext(c)
	c.Debugf("connected - Getting client %d", client.Id)
	err := g.Get(client)
	if err == datastore.ErrNoSuchEntity {
		client.Tell(g, c, nil, &sdz.Action{Type: "Error", Message: "Your client does not exist, please hit /connect again"})
	} else if logError(c, err) {
		return
	} else {
		client.Connected = true
		_, err = g.Put(client)
		if logError(c, err) {
			return
		}
		// figure out what we need to tell the client -- table list or put them back in their game
		//human := StubHuman(int64(id))
		//human.Tell(c, sdz.CreateMessage("Do you want to join a game, create a new game, or quit? (join, create, quit)"))
		if client.Name == "" {
			client.Tell(g, c, nil, sdz.CreateName())
			// request a name and load it later
		}
		client.SendTables(g, c, nil)
	}
	fmt.Fprintf(w, "Success")
}

func connect(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	g := goon.FromContext(c)
	client := new(Client)
	cookie, _ := store.Get(r, cookieName)
	var ok bool
	var err error

	client.Id, ok = cookie.Values["ClientId"].(int64)
	if ok && client.Id != 0 {
		c.Debugf("connect - Getting client %d", client.Id)
		err = g.Get(client)
		if err != datastore.ErrNoSuchEntity && logError(c, err) {
			return
		}
	}
	client.Connected = false // the client is only connecting now, need to setup the channel first
	c.Debugf("Putting client %d", client.Id)
	_, err = g.Put(client)
	c.Debugf("Put client %d", client.Id)
	if logError(c, err) {
		return
	}
	cookie.Values["ClientId"] = client.Id
	cookie.Save(r, w)
	token, err := channel.Create(c, client.getId())
	if logError(c, err) {
		return
	}
	w.Header().Set("Content-type", " application/json")
	rj, err := json.Marshal(token)
	if logError(c, err) {
		return
	}
	fmt.Fprintf(w, "%s", rj)
	client.SendTables(g, c, nil)
}

func receive(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	g := goon.FromContext(c)
	client := new(Client)
	cookie, _ := store.Get(r, cookieName)
	var ok bool
	client.Id, ok = cookie.Values["ClientId"].(int64)

	if !ok || client.Id == 0 {
		err := errors.New("Tried to receive a message from an unknown client")
		logError(c, err)
		w.WriteHeader(500)
		fmt.Fprintf(w, "Error - %v", err)
		return
	}
	err := g.Get(client)
	if datastore.ErrNoSuchEntity == err {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Error - you don't exist")
		return
	} else if logError(c, err) {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Error - %v", err)
		return
	}
	decoder := json.NewDecoder(r.Body)
	action := new(sdz.Action)
	err = decoder.Decode(action)
	if logError(c, err) {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Error - %v", err)
		return
	}
	c.Debugf("Received %s", action)
	var game *Game
	if client.TableId != 0 {
		game = &Game{Id: client.TableId}
		err = g.Get(game)
		if err == datastore.ErrNoSuchEntity {
			game = nil
		} else if logError(c, err) {
			w.WriteHeader(500)
			fmt.Fprintf(w, "Error - %v", err)
			return
		}
	}
	game, err = game.processAction(g, c, client, action)
	if logError(c, err) {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Error - %v", err)
		return
	}
	fmt.Fprintf(w, "Success")
}

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
	Log(ai.Playerid, "In ht.PlayCard for %d-%s on player %d", playerid, c, ai.Playerid)
	Log(ai.Playerid, "ht.PlayedCards = %v", HTString(ai.HT.PlayedCards))
	for x := 0; x < 4; x++ {
		Log(ai.Playerid, "Player%d - %s", x, HTString(ai.HT.Cards[x]))
	}
	if ai.HT.PlayedCards[c] >= 2 {
		Log(ai.Playerid, "Player %d has card %s", playerid, c)
		panic("Played cards cannot be greater than 2")
	}
	ai.HT.PlayedCards[c]++
	if val, ok := ai.HT.Cards[playerid][c]; ok {
		if val > 0 {
			ai.HT.Cards[playerid][c]--
		} else {
			Log(ai.Playerid, "Player %d does not have card %s, panicking", playerid, c)
			panic("panic")
		}
		if val == 1 && ai.HT.PlayedCards[c] == 1 && playerid != ai.Playerid {
			// Other player could have only shown one in meld, but has two - now we don't know who has the last one
			Log(ai.Playerid, "htcardset - deleted card %s for player %d", c, playerid)
			delete(ai.HT.Cards[playerid], c)
		}
	}
	ai.calculateCard(c)
	Log(ai.Playerid, "Player %d played card %s", playerid, c)
}

func (ai *AI) populate() {
	for _, suit := range sdz.Suits() {
		for _, face := range sdz.Faces() {
			card := sdz.CreateCard(suit, face)
			ai.HT.Cards[ai.Playerid][card] = 0
		}
	}
	for _, card := range *ai.RealHand {
		ai.HT.Cards[ai.Playerid][card]++
		ai.calculateCard(card)
	}
}

func (ai *AI) noSuit(playerid int, suit sdz.Suit) {
	Log(ai.Playerid, "No suit start")
	for _, face := range sdz.Faces() {
		card := sdz.CreateCard(suit, face)
		ai.HT.Cards[playerid][card] = 0
		ai.calculateCard(card)
	}
	Log(ai.Playerid, "No suit end")
}

func (ai *AI) calculateCard(c sdz.Card) {
	sum := ai.HT.PlayedCards[c]
	Log(ai.Playerid, "htcardset - Sum for %s is %d", c, sum)
	for x := 0; x < 4; x++ {
		if val, ok := ai.HT.Cards[x][c]; ok {
			sum += val
			Log(ai.Playerid, "htcardsetIterative%d - Sum for %s is now %d", x, c, sum)
		}
	}
	if sum > 2 || sum < 0 {
		sdz.Log("htcardset - Card=%s,sum=%d", c, sum)
		Log(ai.Playerid, "ht.PlayedCards = %v", HTString(ai.HT.PlayedCards))
		for x := 0; x < 4; x++ {
			Log(ai.Playerid, "Player%d - %s", x, HTString(ai.HT.Cards[x]))
		}
		panic("Cannot have more cards than 2 or less than 0 - " + string(sum))
	}
	if sum == 2 {
		for x := 0; x < 4; x++ {
			if _, ok := ai.HT.Cards[x][c]; !ok {
				ai.HT.Cards[x][c] = 0
			}
		}
	} else {
		unknown := -1
		for x := 0; x < 4; x++ {
			if _, ok := ai.HT.Cards[x][c]; !ok {
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
			if ai.HT.PlayedCards[c] > 0 || ai.HT.Cards[ai.Playerid][c] == 1 {
				if unknown != -1 {
					ai.HT.Cards[unknown][c] = 2 - sum
				}
			} else if sum == 0 {
				ai.HT.Cards[unknown][c] = 2
			}
		}
	}
	Log(ai.Playerid, "TT[%s]=%d", c, ai.HT.PlayedCards[c])
	for x := 0; x < 4; x++ {
		if _, ok := ai.HT.Cards[x][c]; ok {
			Log(ai.Playerid, "P%d[%s]=%d", x, c, ai.HT.Cards[x][c])
		}
	}
}

type AI struct {
	RealHand   *sdz.Hand
	Trump      sdz.Suit
	BidAmount  int
	HighBid    int
	HighBidder int
	NumBidders int
	sdz.PlayerImpl
	HT    *HandTracker
	Trick *Trick
}

func (ai *AI) MarshalJSON() ([]byte, error) {
	return json.Marshal("AI")
}

func (a *AI) reset() {
	a.HT = new(HandTracker)
	for x := 0; x < 4; x++ {
		a.HT.Cards[x] = make(map[sdz.Card]int)
	}
	a.HT.PlayedCards = make(map[sdz.Card]int)
	for _, suit := range sdz.Suits() {
		for _, face := range sdz.Faces() {
			a.HT.PlayedCards[sdz.CreateCard(suit, face)] = 0
		}
	}
	a.Trick = NewTrick()
}

func createAI() (a *AI) {
	a = new(AI)
	a.reset()
	return a
}

func (ai AI) powerBid(suit sdz.Suit) (count int) {
	count = 5 // your partner's good for at least this right?!?
	suitMap := make(map[sdz.Suit]int)
	for _, card := range *ai.RealHand {
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
		bids[suit], show = ai.RealHand.Meld(suit)
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
	PlayedBlob    []byte
	Played        map[int]sdz.Card `datastore:"-"`
	WinningPlayer int
	certain       bool // only the AI uses this to determine what card to play, not relevant for the server
	Lead          int
}

func (x *Trick) Load(c <-chan datastore.Property) error {
	if err := datastore.LoadStruct(x, c); err != nil {
		return err
	}
	log.Printf("In trick.Load()\n")
	return gob.NewDecoder(bytes.NewReader(x.PlayedBlob)).Decode(x.Played)
}

func (x *Trick) Save(c chan<- datastore.Property) error {
	err := gob.NewEncoder(bytes.NewBuffer(x.PlayedBlob)).Encode(x.Played)
	if err != nil {
		close(c)
		return err
	}
	c <- datastore.Property{
		Name:  "PlayedBlob",
		Value: x.PlayedBlob,
	}
	return datastore.SaveStruct(x, c)
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

func (trick Trick) leadSuit() sdz.Suit {
	if leadCard, ok := trick.Played[trick.Lead]; ok {
		return leadCard.Suit()
	}
	return sdz.NASuit
}

func (trick Trick) winningCard() sdz.Card {
	if winningCard, ok := trick.Played[trick.WinningPlayer]; ok {
		return winningCard
	}
	return sdz.NACard
}

func (trick Trick) counters() (counters int) {
	for _, card := range trick.Played {
		if card.Counter() {
			counters++
		}
	}
	return
}

func (trick Trick) worth(playerid int, trump sdz.Suit) (worth int) {
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
	ai.Trick.certain = true
	//sdz.Log("htcardset - Player%d is calculating", ai.Playerid)
	//sdz.Log("Before rankCard as player %d", ai.Playerid)
	//sdz.Log("Playerid %d choosing card %s", ai.Playerid, ai.Trick.played[ai.Playerid])
	return rankCard(ai.Playerid, ai.HT, ai.Trick, ai.Trump).Played[ai.Playerid]
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

func (ai *AI) Tell(g *goon.Goon, c appengine.Context, game *Game, action *sdz.Action) *sdz.Action {
	Log(ai.Playerid, "Action received - %+v", action)
	switch action.Type {
	case "Bid":
		if action.Playerid == ai.Playerid {
			Log(ai.Playerid, "------------------Player %d asked to bid against player %d", ai.Playerid, ai.HighBidder)
			ai.BidAmount, ai.Trump, _ = ai.calculateBid()
			if ai.NumBidders == 1 && ai.IsPartner(ai.HighBidder) && ai.BidAmount < 21 && ai.BidAmount+5 > 20 {
				// save our parter
				Log(ai.Playerid, "Saving our partner with a recommended bid of %d", ai.BidAmount)
				ai.BidAmount = 21
			}
			bidAmountOld := ai.BidAmount
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
			meld, _ := ai.RealHand.Meld(ai.Trump)
			Log(ai.Playerid, "------------------Player %d bid %d over %d with recommendation of %d and %d meld", ai.Playerid, ai.BidAmount, ai.HighBid, bidAmountOld, meld)
			return sdz.CreateBid(ai.BidAmount, ai.Playerid)
		} else {
			// received someone else's bid value'
			if ai.HighBid < action.Bid {
				ai.HighBid = action.Bid
				ai.HighBidder = action.Playerid
			}
			ai.NumBidders++
		}
	case "Play":
		Log(ai.Playerid, "Trick = %s", ai.Trick)
		var response *sdz.Action
		if action.Playerid == ai.Playerid {
			response = sdz.CreatePlay(ai.findCardToPlay(action), ai.Playerid)
			action.PlayedCard = response.PlayedCard
		}
		ai.Trick.Played[action.Playerid] = action.PlayedCard
		if ai.Trick.leadSuit() == sdz.NASuit || ai.Trick.winningCard() == sdz.NACard {
			ai.Trick.Lead = action.Playerid
			ai.Trick.WinningPlayer = action.Playerid
			Log(ai.Playerid, "Set lead to %s", ai.Trick.leadSuit())
		} else if action.PlayedCard.Beats(ai.Trick.winningCard(), ai.Trump) {
			ai.Trick.WinningPlayer = action.Playerid
		}
		ai.PlayCard(action.PlayedCard, action.Playerid)
		if action.Playerid == ai.Playerid {
			// no cards to remove from others, I was asked to play
			return response
		}
		switch {
		case action.PlayedCard.Suit() != ai.Trick.leadSuit() && action.PlayedCard.Suit() != ai.Trump: // couldn't follow suit, couldn't lay trump
			ai.noSuit(action.Playerid, ai.Trump)
			fallthrough
		case action.PlayedCard.Suit() != ai.Trick.leadSuit(): // couldn't follow suit
			ai.noSuit(action.Playerid, ai.Trick.leadSuit())
		case action.Playerid != ai.Trick.WinningPlayer: // did not win
			for _, f := range sdz.Faces() {
				card := sdz.CreateCard(action.PlayedCard.Suit(), f)
				if card.Beats(ai.Trick.winningCard(), ai.Trump) {
					ai.HT.Cards[action.Playerid][card] = 0
					ai.calculateCard(card)
				}
			}
		}
		return response
	case "Trump":
		if action.Playerid == ai.Playerid {
			meld, _ := ai.RealHand.Meld(ai.Trump)
			Log(ai.Playerid, "Player %d being asked to name trump on hand %s and have %d meld", ai.Playerid, ai.RealHand, meld)
			switch {
			// TODO add case for the end of the game like if opponents will coast out
			case ai.BidAmount < 15:
				return sdz.CreateThrowin(ai.Playerid)
			default:
				return sdz.CreateTrump(ai.Trump, ai.Playerid)
			}
		} else {
			ai.Trump = action.Trump
			Log(ai.Playerid, "Trump is %s", ai.Trump)
		}
	case "Throwin":
		Log(ai.Playerid, "Player %d saw that player %d threw in", ai.Playerid, action.Playerid)
	case "Deal":
		ai.reset()
		ai.RealHand = &action.Hand
		Log(ai.Playerid, "Set playerid")
		Log(ai.Playerid, "Dealt Hand = %s", ai.RealHand.String())
		ai.populate()
		ai.HighBid = 20
		ai.HighBidder = action.Dealer
		ai.NumBidders = 0
	case "Meld":
		Log(ai.Playerid, "Received meld action - %#v", action)
		if action.Playerid == ai.Playerid {
			return nil // seeing our own meld, we don't care
		}
		for _, card := range action.Hand {
			val, ok := ai.HT.Cards[action.Playerid][card]
			if !ok {
				ai.HT.Cards[action.Playerid][card] = 1
			} else if val == 1 {
				ai.HT.Cards[action.Playerid][card] = 2
			}
		}
	case "Message": // nothing to do here, no one to read it
	case "Trick": // nothing to do here, nothing to display
		Log(ai.Playerid, "playedCards=%v", ai.HT.PlayedCards)
		ai.Trick = NewTrick()
	case "Score": // TODO: save score to use for future bidding techniques
	default:
		Log(ai.Playerid, "Received an action I didn't understand - %v", action)
	}
	return nil
}

func (a *AI) Hand() *sdz.Hand {
	return a.RealHand
}

func (a *AI) SetHand(g *goon.Goon, c appengine.Context, game *Game, h sdz.Hand, dealer, playerid int) {
	a.Playerid = playerid
	hand := make(sdz.Hand, len(h))
	copy(hand, h)
	a.Tell(g, c, game, sdz.CreateDeal(hand, playerid, dealer))
}

type Human struct {
	RealHand *sdz.Hand
	Client   *Client
	sdz.PlayerImpl
}

func (h *Human) MarshalJSON() ([]byte, error) {
	log.Printf("Logging from MarshalJSON in Human on %s\n", h.Client.Name)
	return json.Marshal(h.Client.Name)
}

func (h *Human) Tell(g *goon.Goon, c appengine.Context, game *Game, action *sdz.Action) *sdz.Action {
	return h.Client.Tell(g, c, game, action)
}

func (h Human) Hand() *sdz.Hand {
	return h.RealHand
}

func (a *Human) SetHand(g *goon.Goon, c appengine.Context, game *Game, h sdz.Hand, dealer, playerid int) {
	hand := make(sdz.Hand, len(h))
	copy(hand, h)
	a.RealHand = &hand
	a.Playerid = playerid
	a.Tell(g, c, game, sdz.CreateDeal(hand, a.Playerid, dealer))
}

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
		//debug.PrintStack()
		c.Debugf("Stack = %s", debug.Stack())
		return true
	}
	return false
}

type Game struct {
	Id          int64    `datastore:"-" goon:"id"`
	Trick       Trick    `datastore:"-" json:"-"`
	PlayerGob   []byte   `datastore:"-" json:"-"`
	Players     []Player `datastore:"-"`
	Dealer      int      `datastore:"-" json:"-"`
	Score       []int    `datastore:"-"`
	Meld        []int    `datastore:"-"`
	CountMeld   []bool   `datastore:"-" json:"-"`
	Counters    []int    `datastore:"-" json:"-"`
	HighBid     int      `datastore:"-"`
	HighPlayer  int      `datastore:"-"`
	Trump       sdz.Suit `datastore:"-"`
	State       string
	Next        int        `datastore:"-"`
	Hands       []sdz.Hand `datastore:"-" json:"-"`
	HandsPlayed int        `datastore:"-" json:"-"`
	Updated     time.Time  `json:"-"`
}

func (x *Game) Load(c <-chan datastore.Property) error {
	for {
		prop := <-c
		log.Printf("Property is %s", prop.Name)
		if prop.Name == "" {
			return nil
		}
		switch prop.Name {
		case "GameGob":
			if err := gob.NewDecoder(bytes.NewReader(prop.Value.([]byte))).Decode(&x); err != nil {
				return err
			}
		default:
			// skip, it'll get loaded in the Gob, I just want to query on it :)
		}
	}
}

func (x *Game) Save(c chan<- datastore.Property) error {
	if x.Players == nil {
		panic("Players should not be nil")
	}
	x.Updated = time.Now()
	var data bytes.Buffer
	err := gob.NewEncoder(&data).Encode(x)

	if err != nil {
		close(c)
		return err
	}
	c <- datastore.Property{
		Name:    "GameGob",
		Value:   data.Bytes(),
		NoIndex: true,
	}
	return datastore.SaveStruct(x, c)
}

func NewGame(players int) *Game {
	game := new(Game)
	game.Players = make([]Player, players)
	game.Score = make([]int, players/2)
	game.Meld = make([]int, players/2)
	game.State = StateNew
	return game
}

// PRE : Players are already created and set
func (game *Game) NextHand(g *goon.Goon, c appengine.Context) (*Game, error) {
	game.Meld = make([]int, len(game.Players)/2)
	game.Trick = *NewTrick()
	game.CountMeld = make([]bool, len(game.Players)/2)
	game.Counters = make([]int, len(game.Players)/2)
	game.HighBid = 20
	game.HighPlayer = game.Dealer
	game.State = StateBid
	game.Next = game.Dealer
	Log(4, "Dealer is %d", game.Dealer)
	deck := sdz.CreateDeck()
	deck.Shuffle()
	hands := deck.Deal()
	for x := 0; x < len(game.Players); x++ {
		game.Next = game.inc()
		sort.Sort(hands[x])
		game.Players[game.Next].SetHand(g, c, game, hands[x], game.Dealer, game.Next)
		Log(4, "Dealing player %d hand %s", game.Next, game.Players[game.Next].Hand())
	}
	game.Next = game.inc() // increment so that Dealer + 1 is asked to bid first
	return game.processAction(g, c, nil, game.Players[game.Next].Tell(g, c, game, sdz.CreateBid(0, game.Next)))
	// processAction will write the game to the datastore when it's done processing the action(s)
}

func (game *Game) inc() int {
	return (game.Next + 1) % len(game.Players)
}

func (game *Game) Broadcast(g *goon.Goon, c appengine.Context, a *sdz.Action, p int) {
	for x, player := range game.Players {
		if p != x {
			player.Tell(g, c, game, a)
		}
	}
}

func (game *Game) BroadcastAll(g *goon.Goon, c appengine.Context, a *sdz.Action) {
	game.Broadcast(g, c, a, -1)
}

func (game *Game) retell(g *goon.Goon, c appengine.Context) {
	switch game.State {
	case StateNew:
		// do nothing, we're not waiting on anyone in particular
	case StateBid:
		game.Players[game.Next].Tell(g, c, game, sdz.CreateDeal(*game.Players[game.Next].Hand(), game.Next, game.Dealer))
		game.Players[game.Next].Tell(g, c, game, sdz.CreateBid(game.HighBid, game.Next))
	case StateTrump:
		game.Players[game.Next].Tell(g, c, game, sdz.CreateDeal(*game.Players[game.Next].Hand(), game.Next, game.Dealer))
		game.Players[game.Next].Tell(g, c, game, sdz.CreateTrump(sdz.NASuit, game.Next))
	case StateMeld:
		// never going to be stuck here on a user action
	case StatePlay:
		for x, card := range game.Trick.Played {
			game.Players[game.Next].Tell(g, c, game, sdz.CreatePlay(card, x))
		}
		game.Players[game.Next].Tell(g, c, game, sdz.CreatePlayRequest(game.Trick.winningCard(), game.Trick.leadSuit(), game.Trump, game.Next, game.Players[game.Next].Hand()))
	}
}

// client parameter only required for actions that modify the client, like sitting at a table, setting your name, etc
func (game *Game) processAction(g *goon.Goon, c appengine.Context, client *Client, action *sdz.Action) (*Game, error) {
	for {
		if game == nil {
			c.Debugf("processAction on %s", action)
		} else {
			c.Debugf("processAction on %s with game.Id = %d", action, game.Id)
		}
		if action == nil {
			// waiting on a human, save the state and exit
			_, err := g.Put(game)
			logError(c, err)
			c.Debugf("ProcessAction returning %#v", game)
			return game, nil
		}
		switch {
		case action.Type == "Tables":
			client.SendTables(g, c, game)
			return game, nil
		case action.Type == "Name":
			client.Name = action.Message
			_, err := g.Put(client)
			c.Debugf("Saving name change")
			logError(c, err)
			if game != nil {
				for _, player := range game.Players {
					c.Debugf("Checking player %#v", player)
					if human, ok := player.(*Human); ok && human.Client.Id == client.Id {
						human.Client = client
						c.Debugf("%s sitting at table %d", human.Client.Name, game.Id)
					} else {
						c.Debugf("Not updating the client")
					}
				}
				action = nil
				continue
			}
			return game, nil
		case action.Type == "Start":
			c.Debugf("Game is %#v", game)
			if game.State != StateNew {
				return game, errors.New("Game is already started")
			}
			for x := range game.Players {
				if game.Players[x] == nil {
					game.Players[x] = createAI()
				}
			}
			return game.NextHand(g, c)
		case action.Type == "Sit":
			if action.TableId == 0 { // create a new table/game
				game = NewGame(4)
			} else {
				game = &Game{Id: action.TableId}
				err := g.Get(game)
				if logError(c, err) {
					return game, err
				}
			}
			c.Debugf("%s - %d sitting at table %d", client.Name, client.Id, game.Id)
			openSlot := -1
			var meHuman *Human
			for x, player := range game.Players {
				human, ok := player.(*Human)
				if ok && human.Client.Id == client.Id {
					meHuman = human
					game.Players[x] = nil
					openSlot = x
					break
				}
				if player == nil {
					openSlot = x
				}
			}
			if openSlot == -1 {
				logError(c, errors.New("Game is full!"))
				return game, nil
			}
			if meHuman == nil {
				meHuman = &Human{Client: client}
			}
			game.Players[openSlot] = game.Players[action.Playerid]
			game.Players[action.Playerid] = meHuman
			for _, player := range game.Players {
				if human, ok := player.(*Human); ok {
					human.Client.SendTables(g, c, game)
				}
			}
			var err error
			game, err = game.processAction(g, c, nil, nil) // save it to the datastore
			logError(c, err)
			client.TableId = game.Id
			_, err = g.Put(client)
			logError(c, err)
			return game, err
		case game.State == StateBid && action.Type != "Bid":
			logError(c, errors.New("Received non bid action"))
			action = nil
			continue
		case game.State == StateBid && action.Type == "Bid" && action.Playerid != game.Next:
			logError(c, errors.New("It's not your turn!"))
			action = nil
			continue
		case game.State == StateBid && action.Type == "Bid" && action.Playerid == game.Next:
			game.Broadcast(g, c, action, game.Next)
			if action.Bid > game.HighBid {
				game.HighBid = action.Bid
				game.HighPlayer = game.Next
			}
			if game.HighPlayer == game.Dealer && game.inc() == game.Dealer { // dealer was stuck, tell everyone
				game.Broadcast(g, c, sdz.CreateBid(game.HighBid, game.Dealer), game.Dealer)
				game.Next = game.inc()
			}
			if game.Next == game.Dealer { // the bidding is done
				game.State = StateTrump
				game.Next = game.HighPlayer
				action = game.Players[game.HighPlayer].Tell(g, c, game, sdz.CreateTrump(sdz.NASuit, game.HighPlayer))
				continue
			}
			game.Next = game.inc()
			action = game.Players[game.Next].Tell(g, c, game, sdz.CreateBid(0, game.Next))
			continue
		case game.State == StateTrump:
			switch action.Type {
			case "Throwin":
				game.Broadcast(g, c, action, action.Playerid)
				game.Score[game.HighPlayer%2] -= game.HighBid
				game.BroadcastAll(g, c, sdz.CreateMessage(fmt.Sprintf("Scores are now Team0 = %d to Team1 = %d, played %d hands", game.Score[0], game.Score[1], game.HandsPlayed)))
				Log(4, "Scores are now Team0 = %d to Team1 = %d, played %d hands", game.Score[0], game.Score[1], game.HandsPlayed)
				game.Dealer = (game.Dealer + 1) % 4
				Log(4, "-----------------------------------------------------------------------------")
				return game.NextHand(g, c)
			case "Trump":
				game.Trump = action.Trump
				Log(4, "Trump is set to %s", game.Trump)
				game.Broadcast(g, c, action, game.HighPlayer)
				for x := 0; x < len(game.Players); x++ {
					meld, meldHand := game.Players[x].Hand().Meld(game.Trump)
					meldAction := sdz.CreateMeld(meldHand, meld, x)
					game.BroadcastAll(g, c, meldAction)
					game.Meld[x%2] += meld
				}
				game.Next = game.HighPlayer
				game.Counters = make([]int, 2)
				game.State = StatePlay
				action = game.Players[game.Next].Tell(g, c, game, sdz.CreatePlayRequest(game.Trick.winningCard(), game.Trick.leadSuit(), game.Trump, game.Next, game.Players[game.Next].Hand()))
				continue
			}
		case game.State == StatePlay:
			// TODO: check for throw in
			if sdz.ValidPlay(action.PlayedCard, game.Trick.winningCard(), game.Trick.leadSuit(), game.Players[game.Next].Hand(), game.Trump) &&
				game.Players[game.Next].Hand().Remove(action.PlayedCard) {
				game.Trick.Played[game.Next] = action.PlayedCard
				game.Broadcast(g, c, action, game.Next)
				if len(game.Trick.Played) == 1 {
					game.Trick.Lead = game.Next
				}
				if game.Trick.Played[game.Next].Beats(game.Trick.winningCard(), game.Trump) {
					game.Trick.WinningPlayer = game.Next
				}
			} else {
				action = game.Players[game.Next].Tell(g, c, game, sdz.CreatePlayRequest(game.Trick.winningCard(), game.Trick.leadSuit(), game.Trump, game.Next, game.Players[game.Next].Hand()))
				continue
			}
			if len(game.Trick.Played) == len(game.Players) {
				game.Counters[game.Trick.WinningPlayer%2] += game.Trick.counters()
				game.CountMeld[game.Trick.WinningPlayer%2] = true
				game.Next = game.Trick.WinningPlayer
				game.BroadcastAll(g, c, sdz.CreateMessage(fmt.Sprintf("Player %d wins trick with %s", game.Trick.WinningPlayer, game.Trick.winningCard())))
				game.BroadcastAll(g, c, sdz.CreateTrick(game.Trick.WinningPlayer))
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
					game.BroadcastAll(g, c, sdz.CreateMessage(fmt.Sprintf("Scores are now Team0 = %d to Team1 = %d, played %d hands", game.Score[0], game.Score[1], game.HandsPlayed)))
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
						game.Players[x].Tell(g, c, game, sdz.CreateScore(x, game.Score, gameOver, win[x%2]))
					}
					if gameOver {
						g := goon.FromContext(c)
						for _, player := range game.Players {
							if human, ok := player.(*Human); ok {
								logError(c, g.Get(human.Client))
								human.Client.TableId = 0
								_, err := g.Put(human.Client)
								logError(c, err)
							}
						}
						key := g.Key(game)
						if key != nil {
							logError(c, g.Delete(key))
						}
						return nil, nil // game over
					}
					game.Dealer = (game.Dealer + 1) % 4
					Log(4, "-----------------------------------------------------------------------------")
					return game.NextHand(g, c)
				}
				game.Trick = *NewTrick()
				action = game.Players[game.Next].Tell(g, c, game, sdz.CreatePlayRequest(game.Trick.winningCard(), game.Trick.leadSuit(), game.Trump, game.Next, game.Players[game.Next].Hand()))
				continue
			}
			game.Next = game.inc()
			action = game.Players[game.Next].Tell(g, c, game, sdz.CreatePlayRequest(game.Trick.winningCard(), game.Trick.leadSuit(), game.Trump, game.Next, game.Players[game.Next].Hand()))
			continue
		}
	}
}

type Client struct {
	Id        int64 `datastore:"-" goon:"id"`
	Connected bool
	Name      string
	TableId   int64
}

func (c Client) getId() string {
	return strconv.Itoa(int(c.Id))
}

func (c *Client) setId(id string) {
	tmp, _ := strconv.Atoi(id)
	c.Id = int64(tmp)
}

func (client *Client) SendTables(g *goon.Goon, c appengine.Context, game *Game) {
	if client.Name == "" {
		client.Tell(g, c, game, sdz.CreateName())
	}
	if game == nil && client.TableId == 0 {
		var tables []*Game
		query := datastore.NewQuery("Game").Filter("State = ", "new").Limit(30)
		_, err := g.GetAll(query, &tables)
		if err != datastore.ErrNoSuchEntity && logError(c, err) {
			return
		}
		tables = append(tables, NewGame(4))
		c.Debugf("Sending first table to %d - %s %#v", client.Id, client.Name, tables[0])
		logError(c, channel.SendJSON(c, client.getId(), struct{ Type, Tables interface{} }{Type: "Tables", Tables: tables}))
	} else {
		if game == nil {
			game = &Game{Id: client.TableId}
			if logError(c, g.Get(game)) {
				return
			}
		}
		me := 0
		for x, player := range game.Players {
			human, ok := player.(*Human)
			if ok {
				if human.Client.Id == client.Id {
					me = x
					break
				}
			}
		}
		c.Debugf("Sending MyTable to %d-%s %#v", client.Id, client.Name, game)
		logError(c, channel.SendJSON(c, client.getId(), struct{ Type, MyTable, Playerid interface{} }{Type: "MyTable", MyTable: game, Playerid: me}))
		game.retell(g, c)
	}
}

func (client *Client) Tell(g *goon.Goon, c appengine.Context, game *Game, action *sdz.Action) *sdz.Action {
	if !client.Connected {
		// client is not connected, can't tell them
		return nil
	}
	err := channel.SendJSON(c, client.getId(), action)
	if err != nil {
		sdz.Log("Error in Send - %v", err)
		client.Connected = false
		_, err = g.Put(client)
		logError(c, err)
		if game != nil {
			me := 0
			for x, player := range game.Players {
				if human, ok := player.(*Human); ok {
					if human.Client.Id == client.Id {
						me = x
						human.Client.Connected = false // update the client in the game as well
						break
					}
				}
			}
			game.Broadcast(g, c, sdz.CreateDisconnect(me), me)
		}
		return nil
	}
	c.Debugf("Sent %s", action)
	return nil
}

type Player interface {
	Tell(*goon.Goon, appengine.Context, *Game, *sdz.Action) *sdz.Action // returns the response if known immediately
	Hand() *sdz.Hand
	SetHand(*goon.Goon, appengine.Context, *Game, sdz.Hand, int, int)
	PlayerID() int
	Team() int
	MarshalJSON() ([]byte, error)
}
