package sdzpinochleserver

import (
	"appengine"
	"appengine/channel"
	"appengine/datastore"
	"bytes"
	"encoding/gob"
	"github.com/gorilla/sessions"
	"log"
	//"runtime"
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
	PWNew        = iota // PlayWalker is "blank"
	PWPopulated         // PlayWalker has children
	PWExtended          // PlayWalker has grandchildren from all children
	PWCalculated        // PlayWalker picked his best child :)
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
	None    = 0
	Unknown = 3
)

var store = sessions.NewCookieStore([]byte("sdzpinochle"))

//var sem = make(chan bool, runtime.NumCPU())

var Hands = make(chan sdz.Hand, 1000)

var htstack *HTStack

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
	//for x := 0; x < runtime.NumCPU(); x++ {
	//	sem <- true
	//}
	hts := make(HTStack, 0, 1000)
	htstack = &hts
}

func getHand() sdz.Hand {
	var h sdz.Hand
	select {
	case h = <-Hands:
		h = h[:0] // empty the slice
	default:
		h = make(sdz.Hand, 0, 24)
	}
	return h
}

func getHT(owner int) *HandTracker {
	ht := htstack.Pop()
	return ht
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
	//return
	if playerid == 4 {
		fmt.Printf("NP - "+m+"\n", v...)
	} else {
		fmt.Printf("P"+strconv.Itoa(playerid)+" - "+m+"\n", v...)
	}
}

type CardMap [24]int

func (cm *CardMap) inc(x sdz.Card) {
	if cm[x] == 2 {

	}
	if cm[x] == Unknown {
		cm[x] = 1
	} else {
		cm[x]++
	}
}

func (cm *CardMap) dec(x sdz.Card) {
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

func (ht *HandTracker) reset(owner int) {
	ht.Owner = owner
	for x := 0; x < len(ht.PlayedCards); x++ {
		for y := 0; y < 4; y++ {
			if y == ht.Owner {
				ht.PlayedCards[x] = None
				ht.Cards[y][x] = None
			} else {
				ht.Cards[y][x] = Unknown
			}
		}
	}
	ht.Trick = new(Trick)
	ht.Trick.reset()
}

type HTStack []*HandTracker

func (hts *HTStack) Push(ht *HandTracker) {
	*hts = append(*hts, ht)
}

func (hts *HTStack) Pop() (ht *HandTracker) {
	//x, a = a[len(a)-1], a[:len(a)-1]
	l := len(*hts) - 1
	if l < 0 {
		ht = new(HandTracker)
		ht.Trick = new(Trick)
		return ht
	}
	ht, *hts = (*hts)[l], (*hts)[:l]
	return ht
}

type HandTracker struct {
	Cards [4]CardMap
	// 0 = know nothing = Unknown
	// 3 = does not have any of this card = None
	// 1 = has this card
	// 2 = has two of these cards
	PlayedCards CardMap
	Owner       int // the playerid of the "owning" player
	Trick       *Trick
	PlayCount   int
}

func (ht *HandTracker) sum(cardIndex sdz.Card) int {
	sum := ht.PlayedCards[cardIndex]
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
	return sum
}

func (oldht *HandTracker) Copy() (newht *HandTracker) {
	newht = getHT(oldht.Owner)
	for x := 0; x < len(oldht.Cards); x++ {
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
	Log(ht.Owner, "PlayCount = %d", ht.PlayCount)
}

func (ht *HandTracker) PlayCard(card sdz.Card, trump sdz.Suit) {
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
	case ht.Trick.leadSuit() == sdz.NASuit || trump == sdz.NASuit:
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
		for _, f := range sdz.Faces {
			tempCard := sdz.CreateCard(card.Suit(), f)
			if tempCard.Beats(ht.Trick.winningCard(), trump) {
				//if oright == ht && playerid == 1 {
				//	Log(ht.Owner, "Setting %s to None for playerid=%d because it could have won", tempCard, playerid)
				//}
				ht.Cards[playerid][tempCard] = None
				ht.calculateCard(tempCard)
			} else {
				break
			}
		}
	}
	ht.PlayCount++
	Log(ht.Owner, "Player %d played card %s, PlayCount=%d", playerid, card, ht.PlayCount)
}

func (cm CardMap) String() string {
	output := "CardMap={"
	for x := 0; x < sdz.AllCards; x++ {
		if cm[x] == Unknown {
			continue
		}
		if cm[x] == None {
			output += fmt.Sprintf("%s:%d ", sdz.Card(x), 0)
		} else {
			output += fmt.Sprintf("%s:%d ", sdz.Card(x), cm[x])
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

func (ht *HandTracker) noSuit(playerid int, suit sdz.Suit) {
	//Log(ht.Owner, "No suit start")
	card := sdz.Card(suit * 6)
	for x := 0; x < 6; x++ {
		ht.Cards[playerid][card] = None
		ht.calculateCard(card)
		card++
	}
	//Log(ht.Owner, "No suit end")
}

func (ht *HandTracker) calculateHand(hand int) {
	totalCards := 0
	for x := 0; x < sdz.AllCards; x++ {
		if ht.Cards[hand][x] != Unknown {
			totalCards += ht.Cards[hand][x]
		}
	}
	if totalCards > 12 {
		Log(ht.Owner, "Player %d has more than 12 cards!", hand)
		panic("Player has more than 12 cards")
	}
	if totalCards == 12 {
		for x := 0; x < sdz.AllCards; x++ {
			if ht.Cards[hand][x] == Unknown {
				ht.Cards[hand][x] = None
				ht.calculateCard(sdz.Card(x))
			}
		}
	}
}

func (ht *HandTracker) calculateCard(cardIndex sdz.Card) {
	sum := ht.sum(cardIndex)
	//if cardIndex == TH {
	//	Log(ht.Owner, "htcardset - Sum for %s is %d", cardIndex, sum)
	//	debug.PrintStack()
	//}
	if sum > 2 || sum < 0 {
		sdz.Log("htcardset - Card=%s,sum=%d", cardIndex, sum)
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
	//if cardIndex == JD && ht == oright {
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
	RealHand   *sdz.Hand
	Trump      sdz.Suit
	BidAmount  int
	HighBid    int
	HighBidder int
	NumBidders int
	sdz.PlayerImpl
	HT *HandTracker
}

func (ai *AI) MarshalJSON() ([]byte, error) {
	return json.Marshal("AI")
}

func (a *AI) reset() {
	if a.HT == nil {
		a.HT = getHT(a.Playerid)
	}
	a.HT.reset(a.Playerid)
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
			case sdz.Ace:
				count += 3
			case sdz.Ten:
				count += 2
			case sdz.King:
				fallthrough
			case sdz.Queen:
				fallthrough
			case sdz.Jack:
				fallthrough
			case sdz.Nine:
				count += 1
			}
		} else if card.Face() == sdz.Ace {
			count += 2
		} else if card.Face() == sdz.Jack || card.Face() == sdz.Nine {
			count -= 1
		}
	}
	for _, x := range sdz.Suits {
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
	for _, suit := range sdz.Suits {
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
	Played        [4]sdz.Card
	WinningPlayer int
	Lead          int
	Plays         int
	Next          int // the next player that needs to play
}

func (t *Trick) PlayCard(card sdz.Card, trump sdz.Suit) {
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
	t.Next = (t.Next + 1) % len(t.Played)
	if t.Plays == 4 {
		t.Next = t.WinningPlayer
	}
	//Log(4, "After trick.PlayCard - %s", t)
	//Log(4, "After trick.PlayCard - %#v", t)
}

func (t *Trick) reset() {
	t.Plays = 0
}

func (trick *Trick) counterWorth() int {
	if trick.Plays != 4 {
		Log(4, "Trick %s is not finished", trick)
		panic("Trick")
	}
	playerid := trick.Next - 1
	if playerid == -1 {
		playerid = len(trick.Played) - 1
	}
	if trick.WinningPlayer%2 == playerid%2 {
		return trick.counters()
	} else {
		return -trick.counters()
	}
}

func (trick *Trick) worth(playerid int, trump sdz.Suit) (worth int) {
	if trick.Plays != 4 {
		// return 0 when it's not a complete trick, it's not worth anything as it's not over
		return
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
	return
}

func (t *Trick) String() string {
	var str bytes.Buffer
	str.WriteString("-")
	if t.Plays == 0 {
		return "-----"
	}
	var printme [4]bool
	walker := t.Lead - 1
	for x := 0; x < t.Plays; x++ {
		walker = (walker + 1) % 4
		printme[walker] = true
	}
	for y := 0; y < 4; y++ {
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

func (trick *Trick) leadSuit() sdz.Suit {
	if trick.Plays == 0 {
		return sdz.NASuit
	}
	return trick.Played[trick.Lead].Suit()
}

func (trick *Trick) winningCard() sdz.Card {
	if trick.Plays == 0 {
		return sdz.NACard
	}
	return trick.Played[trick.WinningPlayer]
}

func (trick *Trick) counters() (counters int) {
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

func CardBeatsTrick(card sdz.Card, trick *Trick, trump sdz.Suit) bool {
	return card.Beats(trick.winningCard(), trump)
}

//type Result struct {
//	Card   sdz.Card
//	Points int
//}

type PlayWalker struct {
	Walker   int
	State    int // PlayWalker is (PW)New, Populated, Extended, or Calculated
	Parent   *PlayWalker
	Children []*PlayWalker
	Card     sdz.Card
	HT       *HandTracker
	Result   int // from the perspective of the current player
	CurTrick *Trick
}

func (pw *PlayWalker) PlayTrail() string {
	tricks := make([]*Trick, 0)
	walker := pw
	tricks = append([]*Trick{walker.HT.Trick}, tricks...)
	for {
		if walker.Parent == nil {
			break
		}
		walker = walker.Parent
		if walker.HT.PlayCount%4 == 0 && walker.HT.PlayCount > 0 {
			tricks = append([]*Trick{walker.HT.Trick}, tricks...)
		}
	}
	var str bytes.Buffer
	for x := range tricks {
		str.WriteString(tricks[x].String())
		str.WriteString(" ")
	}
	return str.String()
}

func playHandWithCard(ht *HandTracker, trump sdz.Suit) sdz.Card {
	count := 0
	pw := &PlayWalker{
		HT:   ht,
		Card: sdz.NACard,
	}
	//end := false
	//time.AfterFunc(time.Second*30, func() {
	//	//end = true
	//	panic("Compute time exceeded for play")
	//})
	for {
		//Log(ht.Owner, "PlayCount looping with %d - %s", pw.HT.PlayCount, pw.PlayTrail())
		//pw.HT.Debug()
		if pw.State == PWNew { // load children, all possible cards
			//Log(4, "One")
			trickPlays := pw.HT.Trick.Plays
			if pw.HT.Trick.Plays == 4 {
				pw.HT.Trick.Plays = 0
			}
			decisionMap := pw.HT.potentialCards(pw.HT.Trick.winningCard(), pw.HT.Trick.leadSuit(), trump, pw.HT.Trick.Plays == 3)
			pw.HT.Trick.Plays = trickPlays
			if len(decisionMap) == 0 {
				if pw.HT.PlayCount != 48 {
					pw.State = PWCalculated
					pw.Result = -1000
					pw = pw.Parent
				}
				Log(ht.Owner, "************** Hand is at the end! - %s", pw.PlayTrail())
				pw.State = PWExtended
				pw = pw.Parent
				continue
			}
			pw.Children = make([]*PlayWalker, len(decisionMap))
			for x := range decisionMap {
				pw.Children[x] = &PlayWalker{
					HT:     pw.HT.Copy(),
					Card:   decisionMap[x],
					Parent: pw,
				}
				count++
				if pw.HT.Trick.Plays == 4 {
					pw.Children[x].HT.Trick.reset()
				}
				pw.Children[x].HT.PlayCard(pw.Children[x].Card, trump)
				//Log(ht.Owner, "Created trick %s", pw.Children[x].HT.Trick)
			}
			//Log(ht.Owner, "Visiting child of walker %#v - %s", pw, pw.PlayTrail())
			pw.State = PWPopulated
			pw = pw.Children[pw.Walker]
			pw.Parent.Walker++
		} else if pw.State == PWPopulated { // children populated
			if pw.Walker < len(pw.Children) { // visit the child
				pw = pw.Children[pw.Walker]
				pw.Parent.Walker++
			} else {
				pw.State = PWExtended
				if pw.Parent == nil { // back at the root, end!
					break
				}
				pw = pw.Parent
			}
		}
	}
	// the whole hand is played, now we score it
	bestChild := 0
	for {
		//if pw.Children == nil || pw.Walker != len(pw.Children) { // incomplete PlayWalker tree
		//	// pw.Best = 0 -- already 0 due to initialization
		//	pw = pw.Parent
		//	continue
		//}
		if pw.State <= PWPopulated { // set my result, I am a dead child
			//Log(ht.Owner, "Five")
			//pw.Result = pw.HT.Trick.worth(pw.HT.Trick.Next, trump)
			if pw.HT.Trick.Plays == 4 {
				pw.Result = pw.HT.Trick.counterWorth()
			}
			Log(ht.Owner, "Scoring dead trick %s as value %d", pw.HT.Trick, pw.Result)
			pw.State = PWCalculated
			pw = pw.Parent
		} else if pw.State == PWExtended {
			//Log(ht.Owner, "Six")
			if pw.Walker < len(pw.Children) { // visit the children first
				pw = pw.Children[pw.Walker]
				pw.Parent.Walker++
			} else {
				bestChild = 0
				Log(ht.Owner, "Starting best child loop")
				for x := 0; x < len(pw.Children); x++ {
					if (pw.Children[x].Result > pw.Children[bestChild].Result && ht.Owner%2 == pw.HT.Trick.Next%2) || pw.Children[x].Result < pw.Children[bestChild].Result {
						bestChild = x
					}
					Log(ht.Owner, "Child is %s with result %d", pw.Children[x].PlayTrail(), pw.Children[x].Result)
				}
				Log(ht.Owner, "Best child was %s with result %d", pw.Children[bestChild].PlayTrail(), pw.Children[bestChild].Result)
				if pw.Parent == nil { // I'm the root
					// return the best card
					Log(ht.Owner, "Created %d PlayWalkers, found best card %s with value %d", count, pw.Children[bestChild].Card, pw.Children[bestChild].Result)
					return pw.Children[bestChild].Card
				}
				Log(ht.Owner, "Working on PlayWalker %#v with Card %s, CurTrick %s, and pw.HT.Trick %s", pw, pw.Card, pw.CurTrick, pw.HT.Trick)
				if pw.HT.Trick.Plays == len(pw.HT.Trick.Played) {
					pw.CurTrick = pw.HT.Trick
					pw.Result = pw.CurTrick.counterWorth() + pw.Children[bestChild].Result
					//pw.Result = pw.HT.Trick.worth(pw.HT.Trick.Next, trump) + pw.Children[bestChild].Result
					Log(ht.Owner, "Scoring final trick %s and adding %d for result %d", pw.HT.Trick, pw.Children[bestChild].Result, pw.Result)
				} else { // pass the trick information upstream
					//pw.Children[bestChild].HT.Trick.Next = pw.HT.Trick.Next
					//pw.HT.Trick = pw.Children[bestChild].HT.Trick
					pw.CurTrick = pw.Children[bestChild].CurTrick
					pw.CurTrick.Next = pw.HT.Trick.Next
					pw.Result = pw.CurTrick.counterWorth() + pw.Children[bestChild].Result
					//pw.Result = pw.HT.Trick.worth(pw.HT.Trick.Next, trump)
					Log(ht.Owner, "Scoring non-final trick %s and adding %d for result %d", pw.CurTrick, pw.Children[bestChild].Result, pw.Result)
				}
				pw.State = PWCalculated
				pw = pw.Parent
			}
		} else if pw.State == PWCalculated {
			//Log(ht.Owner, "Seven")
			pw = pw.Parent
		} else {
			//Log(ht.Owner, "Eight - PW = %#v", pw)
			panic("bah")
		}
	}
}

//func inline(ht *HandTracker, trump sdz.Suit, myCard sdz.Card, results chan Result) {
//	newht := ht.Copy()
//	Log(ht.Owner, "Marking card %s as played for %d", myCard, ht.Trick.Next)
//	newht.PlayCard(myCard, trump)
//	//Log(ht.Owner, "Old trick = %s, new trick = %s", ht.Trick, newht.Trick)
//	result := Result{
//		Card: myCard,
//	}
//	if newht.Trick.Plays < 4 {
//		Log(ht.Owner, "Player %d moving to next player in the trick - %s", newht.Trick.Next, newht.Trick)
//		_, result.Points = playHandWithCard(newht, trump)
//	} else { // trick is over, create a new trick
//		//Log(4, "Player %d pretend won the hand with a %s", newtrick.WinningPlayer, newtrick.winningCard())
//		newht.CurTrick++
//		if newht.CurTrick == 12 { // hand over
//			if ht.Owner%2 == newht.Trick.WinningPlayer%2 {
//				result.Points = newht.Trick.counters() + 1
//			} else {
//				result.Points = 0
//			}
//		} else if newht.CurTrick-2 >= ht.StartTrick { // not over but close enough!
//			result.Points = newht.Trick.worth(ht.Owner, trump)
//		} else {
//			Log(ht.Owner, "Fake winner of trick %s is %d", newht.Trick, newht.Trick.WinningPlayer)
//			newht.Trick.reset()
//			_, result.Points = playHandWithCard(newht, trump)
//		}
//	}
//	Log(ht.Owner, "Results for playing %s by playerid %d are %d", result.Card, ht.Trick.Next, result.Points)
//	results <- result
//	htstack.Push(newht)
//}

//func playHandWithCard(ht *HandTracker, trump sdz.Suit) (sdz.Card, int) {
//	Log(ht.Owner, "Calling playHandWithCard on trick #%d - %s for playerid %d", ht.CurTrick, ht.Trick, ht.Trick.Next)
//	if ht.Trick.Plays == 4 {
//		panic("playHandWithCard && trick.Plays == 4")
//	}
//	decisionMap := ht.potentialCards(ht.Trick.winningCard(), ht.Trick.leadSuit(), trump, ht.Trick.Plays == 3)
//	numCards := len(decisionMap)
//	results := make(chan Result, numCards)
//	for _, card := range decisionMap {
//		//select {
//		//case <-sem:
//		//	go func(myCard sdz.Card) {
//		//		inline(playerid, ht, trick, trump, myCard, results)
//		//		sem <- true
//		//	}(card)
//		//default:
//		inline(ht, trump, card, results)
//		//}
//	}
//	Hands <- decisionMap // need to return the Hand object obtained from potentialCards to the pool of resources
//	var bestCard sdz.Card
//	bestPoints := 0
//	partner := ht.Owner%2 == ht.Trick.Next%2
//	for x := 0; x < numCards; x++ {
//		result := <-results
//		if x == 0 || (result.Points >= bestPoints && partner) || (result.Points <= bestPoints && !partner) {
//			bestPoints = result.Points
//			bestCard = result.Card
//			if ht.Trick.Plays == 4 {
//				bestPoints += ht.Trick.worth(ht.Owner, trump)
//			}
//		}
//	}
//	//Log(4, "Best play for player %d is %s worth %d for the winners", playerid, bestCard, bestPoints)
//	return bestCard, bestPoints
//}

func (ai *AI) findCardToPlay(action *sdz.Action) sdz.Card {
	ai.HT.Trick.Next = action.Playerid
	card := playHandWithCard(ai.HT, action.Trump)
	//Log(ai.Playerid, "PlayHandWithCard returned %s for %d points.", card, points)
	return card
}

func (ht *HandTracker) potentialCards(winning sdz.Card, lead sdz.Suit, trump sdz.Suit, lastPlay bool) sdz.Hand {
	//Log(ht.Owner, "PotentialCards called with %d,winning=%s,lead=%s,trump=%s", playerid, winning, lead, trump)
	//Log(ht.Owner, "PotentialCards Player%d - %s", playerid, ht.Cards[playerid])
	validHand := getHand()
	potentialHand := getHand()
	handStatus := Nothing
allCardLoop:
	for x := 0; x < sdz.AllCards; x++ {
		card := sdz.Card(x)
		suit := card.Suit()
		val := ht.Cards[ht.Trick.Next][card]
		if val == Unknown {
			potentialHand = append(potentialHand, card)
		} else if val != None {
			cardStatus := Nothing
			switch {
			case winning == sdz.NACard:
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
					((cardStatus == FollowWin || cardStatus == TrumpWin) && lastPlay) {
					// there should be a maximum of two cards in validHand, counter and non-counter
					for y, vhc := range validHand {
						//Log(4, "Comparing vhc=%s to card=%s", vhc, card)
						if (vhc.Counter() && card.Counter()) || (!vhc.Counter() && !card.Counter()) {
							if card > vhc {
								//Log(4, "Replacing %s with %s", vhc, card)
								validHand[y] = card
							}
							continue allCardLoop
						}
					}
				}
				//Log(4, "Appending card valid normal")
				validHand = append(validHand, card)
			}
		}
	}
	//if len(validHand)+len(potentialHand) >= 4 {
	//	potentialHand.Shuffle()
	//	oldLen := len(potentialHand)
	//	if oldLen != 0 {
	//		potentialHand = potentialHand[:max(1, len(potentialHand)/3)]
	//		//	//Log(ht.Owner, "Reducing potential hand from %d to %d", oldLen, len(potentialHand))
	//	}
	//}
	if handStatus == Nothing {
		validHand = append(validHand, potentialHand...)
	} else {
	potentialLoop:
		for _, card := range potentialHand {
			//Log(4, "Potential card %s", card)
			cardStatus := Nothing
			suit := card.Suit()
			switch {
			case suit == lead && card.Beats(winning, trump):
				cardStatus = FollowWin
			case suit == lead:
				cardStatus = FollowLose
			case suit == trump && card.Beats(winning, trump):
				cardStatus = TrumpWin
			case suit == trump:
				cardStatus = TrumpLose
			}
			if cardStatus >= handStatus {
				if (cardStatus == FollowLose || cardStatus == TrumpLose) ||
					((cardStatus == FollowWin || cardStatus == TrumpWin) && lastPlay) {
					// there should be a maximum of two cards in validHand, counter and non-counter
					for y, vhc := range validHand {
						//Log(4, "Comparing vhc=%s to card=%s", vhc, card)
						if (vhc.Counter() && card.Counter()) || (!vhc.Counter() && !card.Counter()) {
							if card > vhc {
								//Log(4, "Replacing %s with %s", vhc, card)
								validHand[y] = card
							}
							continue potentialLoop
						}
					}
				}
				validHand = append(validHand, card)
				//Log(4, "Adding potential card %s", card)
			}
		}
	}
	Hands <- potentialHand
	//Log(ht.Owner, "Returning %d potential plays of %s for playerid %d on trick %s", len(validHand), validHand, ht.Trick.Next, ht.Trick)
	//if len(validHand) == 0 && ht.PlayCount != 48 {
	//	ht.Debug()
	//	panic("hand is not at the end but still returning 0 potential cards")
	//}
	return validHand
}

func (ai *AI) Tell(g *goon.Goon, c appengine.Context, game *Game, action *sdz.Action) *sdz.Action {
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
		//Log(ai.Playerid, "Trick = %s", ai.Trick)
		var response *sdz.Action
		if action.Playerid == ai.Playerid {
			response = sdz.CreatePlay(ai.findCardToPlay(action), ai.Playerid)
			action.PlayedCard = response.PlayedCard
		}
		ai.HT.Trick.Next = action.Playerid
		ai.HT.PlayCard(action.PlayedCard, ai.Trump)
		return response
	case "Trump":
		if action.Playerid == ai.Playerid {
			//meld, _ := ai.RealHand.Meld(ai.Trump)
			//Log(ai.Playerid, "Player %d being asked to name trump on hand %s and have %d meld", ai.Playerid, ai.RealHand, meld)
			switch {
			// TODO add case for the end of the game like if opponents will coast out
			case ai.BidAmount < 15:
				return sdz.CreateThrowin(ai.Playerid)
			default:
				return sdz.CreateTrump(ai.Trump, ai.Playerid)
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
	game.Trick = Trick{}
	game.CountMeld = make([]bool, len(game.Players)/2)
	game.Counters = make([]int, len(game.Players)/2)
	game.HighBid = 20
	game.HighPlayer = game.Dealer
	game.State = StateBid
	game.Next = game.Dealer
	//Log(4, "Dealer is %d", game.Dealer)
	deck := sdz.CreateDeck()
	deck.Shuffle()
	hands := deck.Deal()
	for x := 0; x < len(game.Players); x++ {
		game.Next = game.inc()
		sort.Sort(hands[x])
		game.Players[game.Next].SetHand(g, c, game, hands[x], game.Dealer, game.Next)
		//Log(4, "Dealing player %d hand %s", game.Next, game.Players[game.Next].Hand())
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
		if game.Trick.Plays != 0 {
			x := game.Trick.Lead
			for y := 0; y < game.Trick.Plays; y++ {
				game.Players[game.Next].Tell(g, c, game, sdz.CreatePlay(game.Trick.Played[x], x))
				x = (x + 1) % len(game.Trick.Played)
			}
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
				game.BroadcastAll(g, c, sdz.CreateMessage(fmt.Sprintf("Player %d threw in! Scores are now Team0 = %d to Team1 = %d, played %d hands", action.Playerid, game.Score[0], game.Score[1], game.HandsPlayed)))
				//Log(4, "Scores are now Team0 = %d to Team1 = %d, played %d hands", game.Score[0], game.Score[1], game.HandsPlayed)
				game.BroadcastAll(g, c, sdz.CreateScore(game.Score, false, false))
				game.Dealer = (game.Dealer + 1) % 4
				//Log(4, "-----------------------------------------------------------------------------")
				return game.NextHand(g, c)
			case "Trump":
				game.Trump = action.Trump
				//Log(4, "Trump is set to %s", game.Trump)
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
				game.Broadcast(g, c, action, game.Next)
				game.Trick.Next = game.Next
				game.Trick.PlayCard(action.PlayedCard, game.Trump)
			} else {
				action = game.Players[game.Next].Tell(g, c, game, sdz.CreatePlayRequest(game.Trick.winningCard(), game.Trick.leadSuit(), game.Trump, game.Next, game.Players[game.Next].Hand()))
				continue
			}
			if game.Trick.Plays == len(game.Players) {
				game.Counters[game.Trick.WinningPlayer%2] += game.Trick.counters()
				game.CountMeld[game.Trick.WinningPlayer%2] = true
				game.Next = game.Trick.WinningPlayer
				game.BroadcastAll(g, c, sdz.CreateMessage(fmt.Sprintf("Player %d wins trick with %s", game.Trick.WinningPlayer, game.Trick.winningCard())))
				game.BroadcastAll(g, c, sdz.CreateTrick(game.Trick.WinningPlayer))
				c.Debugf("Player %d wins trick with %s", game.Trick.WinningPlayer, game.Trick.winningCard())
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
						game.Players[x].Tell(g, c, game, sdz.CreateScore(game.Score, gameOver, win[x%2]))
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
