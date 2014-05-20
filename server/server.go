package server

import (
	"bytes"
	"encoding/gob"
	"log"

	"github.com/gorilla/sessions"

	"appengine"
	"appengine/channel"
	"appengine/datastore"
	"appengine/urlfetch"

	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"time"

	"github.com/mjibson/goon"
	. "github.com/mzimmerman/sdzpinochle"

	"appengine/mail"
)

const (
	StateNew   = "new"
	StateBid   = "bid"
	StateTrump = "trump"
	StateMeld  = "meld"
	StatePlay  = "play"
	cookieName = "sdzpinochle"
	Nothing    = iota
	TrumpLose
	TrumpWin
	FollowLose
	FollowWin
	None    = uint8(0)
	Unknown = uint8(3)
)

var store = sessions.NewCookieStore([]byte("sdzpinochle"))

//var sem = make(chan bool, runtime.NumCPU())

var Hands = make(chan Hand, 1000)

var htstack *HTStack

var logBuffer bytes.Buffer

func init() {
	http.HandleFunc("/connect", connect)
	http.HandleFunc("/_ah/channel/connected/", connected)
	http.HandleFunc("/receive", receive)
	http.HandleFunc("/processAction", processActionHandler)
	http.HandleFunc("/tell", tell)
	http.HandleFunc("/remind", remind)
	store.Options = &sessions.Options{
		Path:   "/",
		MaxAge: 3600, // keep the cookie for one hour
	}
	gob.Register(new(AI))
	//gob.Register(AI{})
	gob.Register(new(Human))
	//gob.Register(Human{})
	//for x := 0; x < runtime.NumCPU(); x++ {
	//	sem <- true
	//}
	hts := make(HTStack, 0, 1000)
	htstack = &hts
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
		client.Tell(g, c, nil, &Action{Type: "Error", Message: "Your client does not exist, please hit /connect again"})
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
		//human.Tell(c, CreateMessage("Do you want to join a game, create a new game, or quit? (join, create, quit)"))
		if client.Name == "" {
			client.Tell(g, c, nil, CreateName())
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
	if client.Id == 0 {
		c.Debugf("Putting client %d", client.Id)
		_, err = g.Put(client)
		c.Debugf("Put client %d", client.Id)
		if logError(c, err) {
			return
		}
	}
	cookie.Values["ClientId"] = client.Id
	cookie.Save(r, w)
	client.Token, err = channel.Create(c, client.getId())
	if logError(c, err) {
		return
	}
	c.Debugf("Putting client %d with token %s", client.Id, client.Token)
	_, err = g.Put(client)
	c.Debugf("Put client %d with token %s", client.Id, client.Token)
	w.Header().Set("Content-type", " application/json")
	rj, err := json.Marshal(client.Token)
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
	action := new(Action)
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
	actionJson, err := action.MarshalJSON()
	if logError(c, err) {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Error - %v", err)
		return
	}
	hostname, err := appengine.ModuleHostname(c, "ai", "", "")
	if logError(c, err) {
		hostname = "localhost:8080"
	}
	_, err = urlfetch.Client(c).PostForm("http://"+hostname+"/processAction", url.Values{"Client": []string{fmt.Sprintf("%d", client.Id)}, "Action": []string{string(actionJson)}})
	//task := taskqueue.NewPOSTTask("/processAction", url.Values{"Client": []string{fmt.Sprintf("%d", client.Id)}, "Action": []string{string(actionJson)}})
	//_, err = taskqueue.Add(c, task, "AI")
	if logError(c, err) {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Error adding task- %v", err)
		return
	}
	fmt.Fprintf(w, "Success")
}

func tell(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	client, jsonString := r.FormValue("Client"), r.FormValue("JSON")
	err := channel.Send(c, client, jsonString)
	c.Debugf("message sent on channel, tell handler fired for client id %s and action %s", client, jsonString)
	if logError(c, err) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error - %v", err)
	} else {
		fmt.Fprintf(w, "Success")
	}
	return
}

func processActionHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	g := goon.FromContext(c)
	client := new(Client)
	var err error
	client.Id, err = strconv.ParseInt(r.FormValue("Client"), 10, 64)
	if logError(c, err) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error - %v", err)
		return
	}
	err = g.Get(client)
	if logError(c, err) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error - %v", err)
		return
	}
	action := new(Action)
	if logError(c, action.UnmarshalJSON([]byte(r.FormValue("Action")))) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.Debugf("Received in backend %s", action)
	var game *Game
	if client.TableId != 0 {
		game = &Game{Id: client.TableId}
		err = g.Get(game)
		if err == datastore.ErrNoSuchEntity {
			game = nil
		} else if logError(c, err) {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error - %v", err)
			return
		}
	}
	c.Debugf("Game before processAction is - %#v", game)
	_, err = game.processAction(g, c, client, action)
	if logError(c, err) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error - %v", err)
	}
	c.Debugf("buffer = %s", logBuffer.String())
	logBuffer.Reset()
	return
}

type CardMap [25]uint8

func (cm *CardMap) inc(x Card) {
	if cm[x] == Unknown {
		cm[x] = 1
	} else {
		cm[x]++
	}
}

func (cm *CardMap) dec(x Card) {
	//Log(4, "Before dec, %s = %d", x, cm[x])
	if cm[x] == 0 {
		//Log(4, "Attempting to decrement %s from %d", card(x), cm[x])
		panic("Cannot decrement past 0")
	}
	if cm[x] != Unknown {
		cm[x]--
	}
	//Log(4, "After dec, %s = %d", x, cm[x])
}

func (ht *HandTracker) reset(owner uint8) {
	ht.Owner = owner
	for x := 0; x < len(ht.PlayedCards); x++ {
		for y := uint8(0); y < 4; y++ {
			if y == ht.Owner {
				ht.PlayedCards[x] = None
				ht.Cards[y][x] = None
			} else {
				ht.Cards[y][x] = Unknown
			}
		}
	}
	ht.PlayCount = 0
	ht.Trick = new(Trick)
	ht.Trick.reset()
}

type HTStack []*HandTracker

func (hts *HTStack) Push(ht *HandTracker) {
	*hts = append(*hts, ht)
}

func (hts *HTStack) Pop() (ht *HandTracker, err error) {
	//x, a = a[len(a)-1], a[:len(a)-1]
	l := len(*hts) - 1
	if l < 0 {
		//memstats := new(runtime.MemStats)
		//runtime.ReadMemStats(memstats)
		//Log(4, "MemStats = %#v", memstats)
		ht = new(HandTracker)
		ht.Trick = new(Trick)
		return
	}
	ht, *hts = (*hts)[l], (*hts)[:l]
	return
}

type HandTracker struct {
	Cards [4]CardMap
	// 0 = know nothing = Unknown
	// 3 = does not have any of this card = None
	// 1 = has this card
	// 2 = has two of these cards
	PlayedCards CardMap
	Owner       uint8 // the playerid of the "owning" player
	Trick       *Trick
	PlayCount   uint8
}

func (ht *HandTracker) sum(cardIndex Card) (sum uint8) {
	sum = ht.PlayedCards[cardIndex]
	for x := 0; x < len(ht.Cards); x++ {
		if ht.Cards[x][cardIndex] != Unknown {
			sum += ht.Cards[x][cardIndex]
		}
	}
	if sum > 2 {
		Log(ht.Owner, "Summing card %s, sum = %d", cardIndex, sum)
		ht.Debug()
		panic("sumthing is wrong, get it?!?!")
	}
	return
}

func (oldht *HandTracker) Copy() (newht *HandTracker, err error) {
	newht, err = getHT(oldht.Owner)
	for x := uint8(0); x < uint8(len(oldht.Cards)); x++ {
		newht.Cards[x] = oldht.Cards[x]
	}
	newht.PlayedCards = oldht.PlayedCards
	newht.PlayCount = oldht.PlayCount
	*newht.Trick = *oldht.Trick
	return
}

func (ht *HandTracker) Debug() {
	Log(ht.Owner, "ht.PlayedCards = %v", ht.PlayedCards)
	for x := 0; x < 4; x++ {
		Log(ht.Owner, "Player%d - %s", x, ht.Cards[x])
	}
	Log(ht.Owner, "PlayCount = %d, Next=%d", ht.PlayCount, ht.Trick.Next)
	panic("don't call debug")
}

func (ht *HandTracker) PlayCard(card Card, trump Suit) {
	//ht.Debug()
	playerid := ht.Trick.Next
	//Log(ht.Owner, "In ht.PlayCard for %d-%s on player %d", playerid, card, ht.Owner)
	val := ht.Cards[playerid][card]
	if val == None {
		ht.Debug()
		Log(ht.Owner, "Player %d does not have card %s, panicking", playerid, card)
		panic("panic")
	}
	ht.PlayedCards.inc(card)
	ht.Cards[playerid].dec(card)
	if ht.sum(card) > 2 {
		panic("Cannot play this card, something is wrong")
	}
	//if card == KS && playerid == 0 {
	//	Log(ht.Owner, "Decremented %s for player %d from %d to %d", card, playerid, val, ht.Cards[playerid][card])
	//}
	if val == 1 && ht.PlayedCards[card] == 1 && playerid != ht.Owner {
		// Other player could have only shown one in meld, but has two - now we don't know who has the last one
		ht.Cards[playerid][card] = Unknown
		//if oright == ht && card == JD && ht.Owner == 0 && playerid == 1 {
		//	Log(ht.Owner, "htcardset - deleted card %s for player %d, setting to Unknown", card, playerid)
		//	ht.Debug()
		//}
		//} else if oright == ht && card == JD && ht.Owner == 0 && playerid == 1 {
		//	Log(ht.Owner, "Not setting %s to unknown, val=%d, played=%d, playerid=%d", card, val, ht.PlayedCards[card], playerid)
		//	ht.Debug()
	}
	//if oright == ht && playerid == 1 && card == JD {
	//	Log(ht.Owner, "Before Calculate")
	//	ht.Debug()
	//}
	ht.calculateCard(card)
	//if oright == ht && playerid == 1 && card == JD {
	//	Log(ht.Owner, "After Calculate")
	//	ht.Debug()
	//}
	ht.Trick.PlayCard(card, trump)
	switch {
	case ht.Trick.leadSuit() == NASuit || trump == NASuit:
		// do nothing, start of the trick, everything is legal
	case card.Suit() != ht.Trick.leadSuit() && card.Suit() != trump: // couldn't follow suit, couldn't lay trump
		ht.noSuit(playerid, trump)
		//if oright == ht && playerid == 1 {
		//	Log(ht.Owner, "Setting all %s to None for playerid=%d", trump, playerid)
		//}
		fallthrough
	case card.Suit() != ht.Trick.leadSuit(): // couldn't follow suit
		ht.noSuit(playerid, ht.Trick.leadSuit())
		//if oright == ht && playerid == 1 {
		//	Log(ht.Owner, "Setting all %s to None for playerid=%d", trick.leadSuit(), playerid)
		//}
	}
	if playerid != ht.Trick.WinningPlayer { // did not win
		for _, f := range Faces {
			tempCard := CreateCard(card.Suit(), f)
			if tempCard.Beats(ht.Trick.winningCard(), trump) {
				//if oright == ht && playerid == 1 {
				//Log(ht.Owner, "Setting %s to None for playerid=%d because it could have won", tempCard, playerid)
				//}
				ht.Cards[playerid][tempCard] = None
				ht.calculateCard(tempCard)
			} else {
				break
			}
		}
	}
	ht.PlayCount++
	//Log(ht.Owner, "Player %d played card %s, PlayCount=%d", playerid, card, ht.PlayCount)
}

func (cm CardMap) String() string {
	output := "CardMap={"
	for x := AS; int8(x) <= AllCards; x++ {
		if cm[x] == Unknown {
			continue
		}
		if cm[x] == None {
			output += fmt.Sprintf("%s:%d ", x, 0)
		} else {
			output += fmt.Sprintf("%s:%d ", x, cm[x])
		}
	}
	return output + "}"
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
	RealHand   *Hand
	Trump      Suit
	BidAmount  uint8
	HighBid    uint8
	HighBidder uint8
	NumBidders uint8
	PlayerImpl
	HT *HandTracker
}

func (ai *AI) MarshalJSON() ([]byte, error) {
	return json.Marshal("AI")
}

func (a *AI) reset() {
	var err error
	if a.HT == nil {
		a.HT, err = getHT(a.Playerid)
		if err != nil {
			panic("not going to run out of memory here right?!")
		}
	}
	a.HT.reset(a.Playerid)
}

func createAI() (a *AI) {
	a = new(AI)
	a.reset()
	return a
}

func (ai AI) powerBid(suit Suit) (count uint8) {
	count = 5 // your partner's good for at least this right?!?
	suitMap := make(map[Suit]int)
	for _, card := range *ai.RealHand {
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

func (ai AI) calculateBid() (amount uint8, trump Suit, show Hand) {
	bids := make(map[Suit]uint8)
	for _, suit := range Suits {
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
	bids[trump] += uint8(rand.Intn(3)) // adds 0, 1, or 2 for a little spontanaeity
	return bids[trump], trump, show
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

func (trick *Trick) leadSuit() Suit {
	if trick.Plays == 0 {
		return NASuit
	}
	return trick.Played[trick.Lead].Suit()
}

func (trick *Trick) winningCard() Card {
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
	return card.Beats(trick.winningCard(), trump)
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

func playHandWithCard(ht *HandTracker, trump Suit) (Card, uint) {
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
	end := time.Now().Add(time.Millisecond * 2500)
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
				return decisionMap[0], 0
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
	return tierSlice[0][0].Children[bestChild].Card, count
}

func (ai *AI) findCardToPlay(action *Action) (Card, uint) {
	ai.HT.Trick.Next = action.Playerid
	card, amount := playHandWithCard(ai.HT, action.Trump)
	runtime.GC() // since we created so much garbage, we need to have the GC mark it as unlinked/unused so next round it can be reused
	//Log(ai.Playerid, "PlayHandWithCard returned %s for %d points.", card, points)
	return card, amount
}

func (pw *PlayWalker) potentialCards(trick *Trick, trump Suit) Hand {
	//Log(ht.Owner, "PotentialCards called with %d,winning=%s,lead=%s,trump=%s", playerid, winning, lead, trump)
	//Log(ht.Owner, "PotentialCards Player%d - %s", playerid, ht.Cards[playerid])
	validHand := getHand()
	handStatus := Nothing
	winning := NACard
	lead := NASuit
	if trick.Plays != 4 {
		winning = trick.winningCard()
		lead = trick.leadSuit()
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

func (ai *AI) Tell(g *goon.Goon, c appengine.Context, game *Game, action *Action) *Action {
	//Log(ai.Playerid, "Action received - %+v", action)
	switch action.Type {
	case "Bid":
		if action.Playerid == ai.Playerid {
			//Log(ai.Playerid, "------------------Player %d asked to bid against player %d", ai.Playerid, ai.HighBidder)
			ai.BidAmount, ai.Trump, _ = ai.calculateBid()
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
			ai.NumBidders++
		}
	case "Play":
		fallthrough
	case "PlayRequest":
		//Log(ai.Playerid, "Trick = %s", ai.Trick)
		var response *Action
		if action.Playerid == ai.Playerid {
			var start = time.Now()
			card, amount := ai.findCardToPlay(action)
			response = CreatePlay(card, ai.Playerid)
			if c != nil {
				c.Debugf("Logged %d unique paths in %s", amount, time.Now().Sub(start))
			}
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

func (a *AI) SetHand(g *goon.Goon, c appengine.Context, game *Game, h Hand, dealer, playerid uint8) {
	a.Playerid = playerid
	hand := make(Hand, len(h))
	copy(hand, h)
	a.Tell(g, c, game, CreateDeal(hand, playerid, dealer))
}

type Human struct {
	RealHand *Hand
	Client   *Client
	PlayerImpl
}

func (h *Human) MarshalJSON() ([]byte, error) {
	log.Printf("Logging from MarshalJSON in Human on %s\n", h.Client.Name)
	return json.Marshal(h.Client.Name)
}

func (h *Human) Tell(g *goon.Goon, c appengine.Context, game *Game, action *Action) *Action {
	return h.Client.Tell(g, c, game, action)
}

func (h Human) Hand() *Hand {
	return h.RealHand
}

func (a *Human) SetHand(g *goon.Goon, c appengine.Context, game *Game, h Hand, dealer, playerid uint8) {
	hand := make(Hand, len(h))
	copy(hand, h)
	a.RealHand = &hand
	a.Playerid = playerid
	a.Tell(g, c, game, CreateDeal(hand, a.Playerid, dealer))
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
	Players     []Player `datastore:"-"`
	Dealer      uint8    `datastore:"-" json:"-"`
	Score       []int16  `datastore:"-"`
	Meld        []uint8  `datastore:"-"`
	CountMeld   []bool   `datastore:"-" json:"-"`
	Counters    []uint8  `datastore:"-" json:"-"`
	HighBid     uint8    `datastore:"-"`
	HighPlayer  uint8    `datastore:"-"`
	Trump       Suit     `datastore:"-"`
	State       string
	Next        uint8     `datastore:"-"`
	Hands       []Hand    `datastore:"-" json:"-"`
	HandsPlayed uint8     `datastore:"-" json:"-"`
	Updated     time.Time `json:"-"`
}

func (x *Game) Load(c <-chan datastore.Property) (err error) {
	gobbed := false
	for {
		prop := <-c
		if prop.Name == "" {
			if !gobbed {
				panic("Loaded Game without a GameGob!")
			}
			fmt.Fprintf(&logBuffer, "Loaded Game - %#v", x)
			return
		}
		switch prop.Name {
		case "GameGob":
			err = gob.NewDecoder(bytes.NewReader(prop.Value.([]byte))).Decode(&x)
			gobbed = true
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
	for x := range game.Players {
		game.Players[x] = createAI()
	}
	game.Score = make([]int16, players/2)
	game.Meld = make([]uint8, players/2)
	game.State = StateNew
	return game
}

// PRE : Players are already created and set
func (game *Game) NextHand(g *goon.Goon, c appengine.Context) (*Game, error) {
	game.Meld = make([]uint8, len(game.Players)/2)
	game.Trick = Trick{}
	game.CountMeld = make([]bool, len(game.Players)/2)
	game.Counters = make([]uint8, len(game.Players)/2)
	game.HighBid = 20
	game.HighPlayer = game.Dealer
	game.State = StateBid
	game.Next = game.Dealer
	//Log(4, "Dealer is %d", game.Dealer)
	deck := CreateDeck()
	deck.Shuffle()
	hands := deck.Deal()
	for x := uint8(0); x < uint8(len(game.Players)); x++ {
		game.Next = game.inc()
		sort.Sort(hands[x])
		game.Players[game.Next].SetHand(g, c, game, hands[x], game.Dealer, game.Next)
		//Log(4, "Dealing player %d hand %s", game.Next, game.Players[game.Next].Hand())
	}
	game.Next = game.inc() // increment so that Dealer + 1 is asked to bid first
	return game.processAction(g, c, nil, game.Players[game.Next].Tell(g, c, game, CreateBid(0, game.Next)))
	// processAction will write the game to the datastore when it's done processing the action(s)
}

func (game *Game) inc() uint8 {
	return (game.Next + 1) % uint8(len(game.Players))
}

func (game *Game) Broadcast(g *goon.Goon, c appengine.Context, a *Action, p uint8) {
	for x, player := range game.Players {
		if p != uint8(x) {
			player.Tell(g, c, game, a)
		}
	}
}

func (game *Game) BroadcastAll(g *goon.Goon, c appengine.Context, a *Action) {
	game.Broadcast(g, c, a, uint8(len(game.Players)))
}

func (game *Game) retell(g *goon.Goon, c appengine.Context) {
	switch game.State {
	case StateNew:
		// do nothing, we're not waiting on anyone in particular
	case StateBid:
		game.Players[game.Next].Tell(g, c, game, CreateDeal(*game.Players[game.Next].Hand(), game.Next, game.Dealer))
		game.Players[game.Next].Tell(g, c, game, CreateBid(game.HighBid, game.Next))
	case StateTrump:
		game.Players[game.Next].Tell(g, c, game, CreateDeal(*game.Players[game.Next].Hand(), game.Next, game.Dealer))
		game.Players[game.Next].Tell(g, c, game, CreateTrump(NASuit, game.Next))
	case StateMeld:
		// never going to be stuck here on a user action
	case StatePlay:
		if game.Trick.Plays != 0 {
			x := game.Trick.Lead
			for y := uint8(0); y < game.Trick.Plays; y++ {
				game.Players[game.Next].Tell(g, c, game, CreatePlay(game.Trick.Played[x], x))
				x = (x + 1) % uint8(len(game.Trick.Played))
			}
		}
		game.Players[game.Next].Tell(g, c, game, CreatePlayRequest(game.Trick.winningCard(), game.Trick.leadSuit(), game.Trump, game.Next, game.Players[game.Next].Hand()))
	}
}

// client parameter only required for actions that modify the client, like sitting at a table, setting your name, etc
func (game *Game) processAction(g *goon.Goon, c appengine.Context, client *Client, action *Action) (*Game, error) {
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
				if _, ok := player.(*AI); ok {
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
				game.Broadcast(g, c, CreateBid(game.HighBid, game.Dealer), game.Dealer)
				game.Next = game.inc()
			}
			if game.Next == game.Dealer { // the bidding is done
				game.State = StateTrump
				game.Next = game.HighPlayer
				action = game.Players[game.HighPlayer].Tell(g, c, game, CreateTrump(NASuit, game.HighPlayer))
				continue
			}
			game.Next = game.inc()
			action = game.Players[game.Next].Tell(g, c, game, CreateBid(0, game.Next))
			continue
		case game.State == StateTrump:
			switch action.Type {
			case "Throwin":
				game.Broadcast(g, c, action, action.Playerid)
				game.Score[game.HighPlayer%2] -= int16(game.HighBid)
				game.BroadcastAll(g, c, CreateMessage(fmt.Sprintf("Player %d threw in! Scores are now Team0 = %d to Team1 = %d, played %d hands", action.Playerid, game.Score[0], game.Score[1], game.HandsPlayed)))
				//Log(4, "Scores are now Team0 = %d to Team1 = %d, played %d hands", game.Score[0], game.Score[1], game.HandsPlayed)
				game.BroadcastAll(g, c, CreateScore(game.Score, false, false))
				game.Dealer = (game.Dealer + 1) % 4
				//Log(4, "-----------------------------------------------------------------------------")
				return game.NextHand(g, c)
			case "Trump":
				game.Trump = action.Trump
				//Log(4, "Trump is set to %s", game.Trump)
				game.Broadcast(g, c, action, game.HighPlayer)
				for x := uint8(0); x < uint8(len(game.Players)); x++ {
					meld, meldHand := game.Players[x].Hand().Meld(game.Trump)
					meldAction := CreateMeld(meldHand, meld, x)
					game.BroadcastAll(g, c, meldAction)
					game.Meld[x%2] += meld
				}
				game.Next = game.HighPlayer
				game.Counters = make([]uint8, 2)
				game.State = StatePlay
				action = game.Players[game.Next].Tell(g, c, game, CreatePlayRequest(game.Trick.winningCard(), game.Trick.leadSuit(), game.Trump, game.Next, game.Players[game.Next].Hand()))
				continue
			}
		case game.State == StatePlay:
			// TODO: check for throw in
			if ValidPlay(action.PlayedCard, game.Trick.winningCard(), game.Trick.leadSuit(), game.Players[game.Next].Hand(), game.Trump) &&
				game.Players[game.Next].Hand().Remove(action.PlayedCard) {
				game.Broadcast(g, c, action, game.Next)
				game.Trick.Next = game.Next
				game.Trick.PlayCard(action.PlayedCard, game.Trump)
			} else {
				action = game.Players[game.Next].Tell(g, c, game, CreatePlayRequest(game.Trick.winningCard(), game.Trick.leadSuit(), game.Trump, game.Next, game.Players[game.Next].Hand()))
				continue
			}
			if game.Trick.Plays == uint8(len(game.Players)) {
				game.Counters[game.Trick.WinningPlayer%2] += game.Trick.counters()
				game.CountMeld[game.Trick.WinningPlayer%2] = true
				game.Next = game.Trick.WinningPlayer
				game.BroadcastAll(g, c, CreateMessage(fmt.Sprintf("Player %d wins trick with %s", game.Trick.WinningPlayer, game.Trick.winningCard())))
				game.BroadcastAll(g, c, CreateTrick(game.Trick.WinningPlayer))
				c.Debugf("Player %d wins trick with %s", game.Trick.WinningPlayer, game.Trick.winningCard())
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
					game.BroadcastAll(g, c, CreateMessage(fmt.Sprintf("Scores are now Team0 = %d to Team1 = %d, played %d hands", game.Score[0], game.Score[1], game.HandsPlayed)))
					//Log(4, "Scores are now Team0 = %d to Team1 = %d, played %d hands", game.Score[0], game.Score[1], game.HandsPlayed)
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
						game.Players[x].Tell(g, c, game, CreateScore(game.Score, gameOver, win[x%2]))
					}
					if gameOver {
						g := goon.FromContext(c)
						for _, player := range game.Players {
							if human, ok := player.(*Human); ok {
								logError(c, g.Get(human.Client))
								human.Client.TableId = 0
								_, err := g.Put(human.Client)
								logError(c, err)
							} else {
								htstack.Push(player.(*AI).HT)
							}
						}
						key := g.Key(game)
						if key != nil {
							logError(c, g.Delete(key))
						}
						return nil, nil // game over
					}
					game.Dealer = (game.Dealer + 1) % 4
					//Log(4, "-----------------------------------------------------------------------------")
					return game.NextHand(g, c)
				}
				game.Trick.reset()
				action = game.Players[game.Next].Tell(g, c, game, CreatePlayRequest(game.Trick.winningCard(), game.Trick.leadSuit(), game.Trump, game.Next, game.Players[game.Next].Hand()))
				continue
			}
			game.Next = game.inc()
			action = game.Players[game.Next].Tell(g, c, game, CreatePlayRequest(game.Trick.winningCard(), game.Trick.leadSuit(), game.Trump, game.Next, game.Players[game.Next].Hand()))
			continue
		}
	}
}

type Client struct {
	Id        int64 `datastore:"-" goon:"id"`
	Connected bool
	Name      string
	TableId   int64
	Token     string
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
		client.Tell(g, c, game, CreateName())
	}
	hostname, err := appengine.ModuleHostname(c, "default", "", "")
	logError(c, err)
	if game == nil && client.TableId == 0 {
		var tables []*Game
		query := datastore.NewQuery("Game").Filter("State = ", "new").Limit(30)
		_, err := g.GetAll(query, &tables)
		if err != datastore.ErrNoSuchEntity && logError(c, err) {
			return
		}
		tables = append(tables, NewGame(4))
		c.Debugf("Sending first table to %d - %s %#v", client.Id, client.Name, tables[0])
		myTableString, err := json.Marshal(struct{ Type, Tables interface{} }{Type: "Tables", Tables: tables})
		logError(c, err)
		//_, err = taskqueue.Add(c, taskqueue.NewPOSTTask("/tell", url.Values{"Client": []string{fmt.Sprintf("%d", client.Id)}, "JSON": []string{string(myTableString)}}), "frontend")
		_, err = urlfetch.Client(c).PostForm("http://"+hostname+"/tell", url.Values{"Client": []string{fmt.Sprintf("%d", client.Id)}, "JSON": []string{string(myTableString)}})
		logError(c, err)
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
		myTableString, err := json.Marshal(struct{ Type, MyTable, Playerid interface{} }{Type: "MyTable", MyTable: game, Playerid: me})
		if logError(c, err) {
			return
		}
		c.Debugf("Sending MyTable to %d-%s %#v", client.Id, client.Name, game)
		//_, err = taskqueue.Add(c, taskqueue.NewPOSTTask("/tell", url.Values{"Client": []string{fmt.Sprintf("%d", client.Id)}, "JSON": []string{string(myTableString)}}), "frontend")
		_, err = urlfetch.Client(c).PostForm("http://"+hostname+"/tell", url.Values{"Client": []string{fmt.Sprintf("%d", client.Id)}, "JSON": []string{string(myTableString)}})
		//logError(c, err)
		//c.Debugf("Tell urlfetch task added to %d of action %s", client.Id, myTableString)
		logError(c, err)
		game.retell(g, c)
	}
}

func (client *Client) Tell(g *goon.Goon, c appengine.Context, game *Game, action *Action) *Action {
	if !client.Connected {
		// client is not connected, can't tell them
		return nil
	}
	actionJson, err := action.MarshalJSON()
	if logError(c, err) {
		return nil
	}
	hostname, err := appengine.ModuleHostname(c, "default", "", "")
	logError(c, err)

	//_, err = taskqueue.Add(c, taskqueue.NewPOSTTask("/tell", url.Values{"Client": []string{fmt.Sprintf("%d", client.Id)}, "JSON": []string{string(actionJson)}}), "frontend")
	_, err = urlfetch.Client(c).PostForm("http://"+hostname+"/tell", url.Values{"Client": []string{fmt.Sprintf("%d", client.Id)}, "JSON": []string{string(actionJson)}})
	logError(c, err)
	return nil
	//err := channel.SendJSON(c, fmt.Sprintf("%d", client.Id), action)
	//if err != nil {
	//	Log(4, "Error in Send - %v", err)
	//	client.Connected = false
	//	_, err = g.Put(client)
	//	logError(c, err)
	//	if game != nil {
	//		me := uint8(0)
	//		for x, player := range game.Players {
	//			if human, ok := player.(*Human); ok {
	//				if human.Client.Id == client.Id {
	//					me = uint8(x)
	//					human.Client.Connected = false // update the client in the game as well
	//					break
	//				}
	//			}
	//		}
	//		game.Broadcast(g, c, CreateDisconnect(me), me)
	//	}
	//	return nil
	//}
	//c.Debugf("Sent %s", action)
	//return nil
}

type Player interface {
	Tell(*goon.Goon, appengine.Context, *Game, *Action) *Action // returns the response if known immediately
	Hand() *Hand
	SetHand(*goon.Goon, appengine.Context, *Game, Hand, uint8, uint8)
	PlayerID() uint8
	Team() uint8
	MarshalJSON() ([]byte, error)
}
