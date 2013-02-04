// pinochle.go
package sdzpinochle

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"time"
)

func Log(m string, v ...interface{}) {
	fmt.Printf(m+"\n", v...)
}

const (
	debug       = false
	Ace         = Face("A")
	Ten         = Face("T")
	King        = Face("K")
	Queen       = Face("Q")
	Jack        = Face("J")
	Nine        = Face("9")
	Spades      = Suit("S")
	Hearts      = Suit("H")
	Clubs       = Suit("C")
	Diamonds    = Suit("D")
	acearound   = 10
	kingaround  = 8
	queenaround = 6
	jackaround  = 4
)

type Card string // two chars Face + String
type Suit string // one char
type Face string // one char

func Faces() [6]Face {
	return [6]Face{Ace, Ten, King, Queen, Jack, Nine}
}

func Suits() [4]Suit {
	return [4]Suit{Spades, Hearts, Clubs, Diamonds}
}

type Deck [48]Card
type Hand []Card

func CreateCard(suit Suit, face Face) Card {
	return Card(string(face) + string(suit))
}

func (a Card) Beats(b Card, trump Suit) bool {
	// a is the challenging card
	switch {
	case a.Suit() == b.Suit():
		return a.Face().Less(b.Face())
	case a.Suit() == trump:
		return true
	}
	return false
}

func (c Card) Suit() Suit {
	return Suit(c[1])
}

func (c Card) Face() Face {
	return Face(c[0])
}

func (d *Deck) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func (d *Deck) Shuffle() {
	//	http://en.wikipedia.org/wiki/Fisher%E2%80%93Yates_shuffle#The_modern_algorithm
	rand.Seed(time.Now().UnixNano())
	for i := len(d) - 1; i >= 1; i-- {
		if j := rand.Intn(i); i != j {
			d.Swap(i, j)
		}
	}
}

func (h Hand) String() {
	cards := ""
	for x := 0; x < len(h); x++ {
		cards += string(h[x]) + " "
	}
}

func (h Hand) Len() int {
	return len(h)
}

func (h Hand) Less(i, j int) bool {
	if h[i].Suit() == h[j].Suit() {
		return h[i].Face().Less(h[j].Face())
	}
	return h[i].Suit().Less(h[j].Suit())
}

func (a Face) Less(b Face) bool {
	switch {
	case b == Ace:
		return false
	case a == Ace:
		return true
	case b == Ten:
		return false
	case a == Ten:
		return true
	case b == King:
		return false
	case a == King:
		return true
	case b == Queen:
		return false
	case a == Queen:
		return true
	case b == Jack:
		return false
	case a == Jack:
		return true
	}
	return false
}

func (a Suit) Less(b Suit) bool { // only for sorting the suits for display in the hand
	switch {
	case a == Spades:
		return false
	case b == Spades:
		return true
	case a == Hearts:
		return false
	case b == Hearts:
		return true
	case a == Clubs:
		return false
	case b == Clubs:
		return true
	}
	return false
}

func (h Hand) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (d Deck) Deal() (hands []Hand) {
	hands = make([]Hand, 4)
	for x := 0; x < 4; x++ {
		hands[x] = make([]Card, 12)
	}
	for y := 0; y < 12; y++ {
		for x := 0; x < 4; x++ {
			hands[x][y] = d[y*4+x]
		}
	}
	return
}

func CreateDeck() (deck Deck) {
	index := 0
	for _, face := range Faces() {
		for _, suit := range Suits() {
			for z := 0; z < 2; z++ {
				deck[index] = Card(string(face) + string(suit))
				index++
			}
		}
	}
	return
}

type Action struct {
	Type                    string
	Playerid                int
	Bid                     int
	PlayedCard, WinningCard Card
	Lead, Trump             Suit
	Amount                  int
	Message                 string
	Hand                    Hand
	Option                  int
	GameOver, Win           bool
	Score                   []int
	Dealer                  int
}

func (action *Action) MarshalJSON() ([]byte, error) {
	data := make(map[string]interface{})
	typ := reflect.TypeOf(*action)
	val := reflect.ValueOf(*action)
	count := typ.NumField()
	for x := 0; x < count; x++ {
		switch {
		case typ.Field(x).Name == "Playerid":
			if action.Type == "Hello" || action.Type == "Score" || action.Type == "Message" || action.Type == "Game" {
				// don't include playerid', it's not relevant'
			} else {
				data["Playerid"] = action.Playerid
			}
		case typ.Field(x).Name == "Win" && action.GameOver:
			data["Win"] = action.Win
		case typ.Field(x).Name == "GameOver" && action.Type == "Score":
			data["GameOver"] = action.GameOver
		case typ.Field(x).Name == "Dealer" && action.Type == "Deal":
			data["Dealer"] = action.Dealer
		case reflect.DeepEqual(val.Field(x).Interface(), reflect.New(typ.Field(x).Type).Elem().Interface()):
			continue
		default:
			data[typ.Field(x).Name] = val.Field(x).Interface()
		}
	}
	return json.Marshal(data)
}

func CreateGame(option int) *Action {
	return &Action{Type: "Game", Option: option}
}

func CreateHello(m string) *Action {
	return &Action{Type: "Hello", Message: m}
}

func CreateMessage(m string) *Action {
	return &Action{Type: "Message", Message: m}
}

func CreateBid(bid, playerid int) *Action {
	return &Action{Type: "Bid", Bid: bid, Playerid: playerid}
}

func CreatePlayRequest(winning Card, lead, trump Suit, playerid int) *Action {
	return &Action{Type: "Play", WinningCard: winning, Lead: lead, Trump: trump, Playerid: playerid}
}

func CreatePlay(card Card, playerid int) *Action {
	return &Action{Type: "Play", PlayedCard: card, Playerid: playerid}
}

func CreateTrump(trump Suit, playerid int) *Action {
	return &Action{Type: "Trump", Trump: trump, Playerid: playerid}
}

func CreateTrick(winningPlayer int) *Action {
	return &Action{Type: "Trick", Playerid: winningPlayer}
}

func CreateThrowin(playerid int) *Action {
	return &Action{Type: "Throwin", Playerid: playerid}
}

func CreateMeld(hand Hand, amount, playerid int) *Action {
	return &Action{Type: "Meld", Hand: hand, Amount: amount, Playerid: playerid}
}

func CreateDeal(hand Hand, playerid, dealer int) *Action {
	return &Action{Type: "Deal", Hand: hand, Playerid: playerid, Dealer: dealer}
}

func CreateScore(playerid int, score []int, gameOver, win bool) *Action {
	return &Action{Type: "Score", Playerid: playerid, Score: score, Win: win, GameOver: gameOver}
}

type Player interface {
	Tell(*Action)
	Listen() (*Action, bool)
	Hand() *Hand
	SetHand(Hand, int, int)
	Go()
	Close()
	Playerid() int
	Team() int
}

type PlayerImpl struct {
	Id int
}

func (p PlayerImpl) Playerid() int {
	return p.Id
}

func (p PlayerImpl) Team() int {
	return p.Playerid() % 2
}

func (p PlayerImpl) IsPartner(player int) bool {
	return p.Playerid()%2 == player%2
}

type Game struct {
	Deck       Deck
	Players    []Player
	Dealer     int
	Score      []int
	Meld       []int
	Counters   []int
	MeldHands  []Hand
	HighBid    int
	HighPlayer int
	Trump      Suit
}

// Used to determine if the leader of the trick made a valid play
func IsCardInHand(card Card, hand Hand) bool {
	for _, hc := range hand {
		if hc == card {
			return true
		}
	}
	return false
}

// playedCard, winningCard Card, leadSuit Suit, hand Hand, trump Suit
func ValidPlay(playedCard, winningCard Card, leadSuit Suit, hand *Hand, trump Suit) bool {
	// hand is sorted
	// 1 - Have to follow suit
	// 2 - Can't follow suit, play trump
	// 3 - Have to win
	canFollow := false
	hasTrump := false
	canWin := false
	hasCard := false
	for _, card := range *hand {
		if card.Suit() == leadSuit {
			canFollow = true
		}
		if card.Suit() == trump {
			hasTrump = true
		}
		if card == playedCard {
			hasCard = true
		}
	}
	// have to loop again because we can't set canWin to true if we're playing trump but we can follow a non-trump suit
	for _, card := range *hand {
		if canFollow && leadSuit != trump && card.Suit() == trump {
			continue
		}
		if card.Beats(winningCard, trump) {
			canWin = true
			break
		}
	}
	if !hasCard { // you don't have the card in your hand, not allowed to play it, cheater!
		return false
	}
	if canFollow {
		if playedCard.Suit() != leadSuit {
			return false
		} else if canWin { // we're following suit
			return playedCard.Beats(winningCard, trump)
		} else { // we're following suit and we can't win'
			return true
		}
	} else if hasTrump {
		if playedCard.Suit() != trump {
			return false
		} else if canWin { // we're playing trump
			return playedCard.Beats(winningCard, trump)
		} else { // we're playing trump but we can't win
			return true
		}
	} // else { // we can't follow suit and we don't have trump - anything's legal
	return true
}

func (game *Game) Go(players []Player) {
	game.Deck = CreateDeck()
	game.Players = players
	game.Score = make([]int, 2)
	handsPlayed := 0
	game.Dealer = 0
	for {
		handsPlayed++
		// shuffle & deal
		game.Deck.Shuffle()
		hands := game.Deck.Deal()
		next := game.Dealer
		game.Meld = make([]int, len(game.Players))
		game.MeldHands = make([]Hand, len(game.Players))
		game.Counters = make([]int, len(game.Players))
		for x := 0; x < len(game.Players); x++ {
			next = (next + 1) % 4
			sort.Sort(hands[x])
			game.Players[next].SetHand(hands[x], game.Dealer, next)
			Log("Dealing player %d hand %s", next, game.Players[next].Hand())
		}
		// ask players to bid
		game.HighBid = 20
		game.HighPlayer = game.Dealer
		next = game.Dealer
		for x := 0; x < 4; x++ {
			next = (next + 1) % 4
			game.Players[next].Tell(CreateBid(0, next))
			bidAction, open := game.Players[next].Listen()
			if !open {
				game.Broadcast(CreateMessage("Player disconnected"), next)
				return
			}
			game.Broadcast(bidAction, next)
			if bidAction.Bid > game.HighBid {
				game.HighBid = bidAction.Bid
				game.HighPlayer = next
			}
		}
		// ask trump
		game.Players[game.HighPlayer].Tell(CreateTrump(*new(Suit), game.HighPlayer))
		response, open := game.Players[game.HighPlayer].Listen()
		if !open {
			game.Broadcast(CreateMessage("Player disconnected"), game.HighPlayer)
			return
		}
		switch response.Type {
		case "Throwin":
			game.Broadcast(response, response.Playerid)
			// TODO: adjust the score
		case "Trump":
			game.Trump = response.Trump
			Log("Trump is set to %s", game.Trump)
			game.Broadcast(response, game.HighPlayer)
		default:
			panic("Didn't receive either expected response")
		}
		for x := 0; x < len(game.Players); x++ {
			game.Meld[x], game.MeldHands[x] = game.Players[x].Hand().Meld(game.Trump)
			meldAction := CreateMeld(game.MeldHands[x], game.Meld[x], x)
			game.BroadcastAll(meldAction)
		}
		next = game.HighPlayer
		for trick := 0; trick < 12; trick++ {
			var winningCard Card
			var cardPlayed Card
			var leadSuit Suit
			winningPlayer := next
			counters := 0
			for x := 0; x < 4; x++ {
				// play the hand
				// TODO: handle possible throwin
				var action *Action
				for {
					action = CreatePlayRequest(winningCard, leadSuit, game.Trump, next)
					game.Players[next].Tell(action)
					action, open = game.Players[next].Listen()
					if !open {
						game.Broadcast(CreateMessage("Player disconnected"), next)
						return
					}
					cardPlayed = action.PlayedCard
					//Log("Server received card %s", cardPlayed)
					//Log("Hand length %d", len(*game.Players[next].Hand()))
					if x > 0 {
						if ValidPlay(cardPlayed, winningCard, leadSuit, game.Players[next].Hand(), game.Trump) &&
							game.Players[next].Hand().Remove(cardPlayed) {
							//Log("Hand length %s", len(*game.Players[next].Hand()))
							// playedCard, winningCard Card, leadSuit Suit, hand Hand, trump Suit
							break
						}
					} else if game.Players[next].Hand().Remove(cardPlayed) {
						//Log("Hand length %s", len(*game.Players[next].Hand()))
						break
					}
				}
				switch cardPlayed.Face() {
				case Ace:
					fallthrough
				case Ten:
					fallthrough
				case King:
					counters++
				}
				if x == 0 {
					winningCard = cardPlayed
					leadSuit = cardPlayed.Suit()
				} else {
					if cardPlayed.Beats(winningCard, game.Trump) {
						winningCard = cardPlayed
						winningPlayer = next
					}
				}
				game.Broadcast(action, next)
				next = (next + 1) % 4
			}
			next = winningPlayer
			if trick == 11 {
				counters++
			}
			game.BroadcastAll(CreateMessage(fmt.Sprintf("Player %d wins trick #%d with %s for %d points", winningPlayer, trick+1, winningCard, counters)))
			game.BroadcastAll(CreateTrick(winningPlayer))
			Log("Player %d wins trick #%d with %s for %d points", winningPlayer, trick+1, winningCard, counters)
			game.Counters[game.Players[winningPlayer].Team()] += counters
		}
		game.Meld[0] += game.Meld[2]
		game.Counters[0] += game.Counters[2]
		game.Meld[1] += game.Meld[3]
		game.Counters[1] += game.Counters[3]
		switch game.Players[game.HighPlayer].Team() {
		case 0:
			if game.Meld[0]+game.Counters[0] < game.HighBid {
				game.Score[0] -= game.HighBid
			} else {
				game.Score[0] += game.Meld[0] + game.Counters[0]
			}
			game.Score[1] += game.Meld[1] + game.Counters[1]
		case 1:
			if game.Meld[1]+game.Counters[1] < game.HighBid {
				game.Score[1] -= game.HighBid
			} else {
				game.Score[1] += game.Meld[1] + game.Counters[1]
			}
			game.Score[0] += game.Meld[0] + game.Counters[0]
		}
		// check the score for a winner
		game.BroadcastAll(CreateMessage(fmt.Sprintf("Scores are now Team0 = %d to Team1 = %d, played %d hands", game.Score[0], game.Score[1], handsPlayed)))
		Log("Scores are now Team0 = %d to Team1 = %d, played %d hands", game.Score[0], game.Score[1], handsPlayed)
		gameOver := false
		player0win := false
		if game.Score[game.HighPlayer%2] >= 120 {
			game.BroadcastAll(CreateMessage(fmt.Sprintf("Team%d wins with a score of %d!", game.HighPlayer%2, game.Score[game.HighPlayer%2])))
			Log("Team%d wins with a score of %d!", game.HighPlayer%2, game.Score[game.HighPlayer%2])
			gameOver = true
			player0win = (game.HighPlayer%2 == 0)
		} else if game.Score[0] >= 120 {
			game.BroadcastAll(CreateMessage(fmt.Sprintf("Team0 wins with a score of %d!", game.Score[0])))
			Log("Team0 wins with a score of %d!", game.Score[0])
			gameOver = true
			player0win = true
		} else if game.Score[1] >= 120 {
			game.BroadcastAll(CreateMessage(fmt.Sprintf("Team1 wins with a score of %d!", game.Score[1])))
			Log("Team1 wins with a score of %d!", game.Score[1])
			gameOver = true
			player0win = false
		}
		for x := 0; x < len(game.Players); x++ {
			win := player0win
			if x%2 == 1 && gameOver {
				win = !player0win
			}
			game.Players[x].Tell(CreateScore(x, game.Score, gameOver, win))
		}
		if gameOver {
			return
		}
		game.Dealer = (game.Dealer + 1) % 4
		Log("-----------------------------------------------------------------------------")
	}
	for x := 0; x < 4; x++ {
		game.Dealer = (game.Dealer + 1) % 4
		game.Players[game.Dealer].Close()
	}
}

func (g Game) Broadcast(a *Action, p int) {
	for x, player := range g.Players {
		if p != x {
			player.Tell(a)
		}
	}
}

func (g Game) BroadcastAll(a *Action) {
	g.Broadcast(a, -1)
}

func (h *Hand) Remove(card Card) bool {
	for x := range *h {
		if (*h)[x] == card {
			//temp := append((*h)[:x], (*h)[x+1:]...)
			//h = &temp
			*h = append((*h)[:x], (*h)[x+1:]...)
			return true
		}
	}
	return false
}

func (h Hand) Count() (cards map[Card]int) {
	cards = make(map[Card]int)
	for _, face := range Faces() {
		for _, suit := range Suits() {
			cards[CreateCard(suit, face)] = 0
		}
	}
	for x := 0; x < len(h); x++ {
		cards[h[x]]++
	}
	return
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

func (h Hand) Meld(trump Suit) (meld int, result Hand) {
	// hand does not have to be sorted
	count := h.Count()
	if debug {
		fmt.Printf("Count is %v\n", count)
	}
	show := make(map[Card]int)
	around := make(map[Face]int)
	for _, value := range Faces() {
		around[value] = 2
	}
	//	fmt.Printf("AroundBefore = %v\n", around)
	for _, suit := range Suits() { // look through each suit
		switch { // straights & marriages
		case trump == suit:
			if debug {
				fmt.Printf("Scoring %d nine(s) in trump %s\n", count[CreateCard(suit, Nine)], trump)
			}
			meld += count[CreateCard(suit, Nine)] // 9s in trump
			show[CreateCard(suit, Nine)] = count[CreateCard(suit, Nine)]
			switch {
			// double straight
			case count[CreateCard(suit, Ace)] == 2 && count[CreateCard(suit, Ten)] == 2 && count[CreateCard(suit, King)] == 2 && count[CreateCard(suit, Queen)] == 2 && count[CreateCard(suit, Jack)] == 2:
				meld += 150
				for _, face := range Faces() {
					show[CreateCard(suit, face)] = 2
				}
				if debug {
					fmt.Println("DoubleStraight")
				}
			// single straight
			case count[CreateCard(suit, Ace)] >= 1 && count[CreateCard(suit, Ten)] >= 1 && count[CreateCard(suit, King)] >= 1 && count[CreateCard(suit, Queen)] >= 1 && count[CreateCard(suit, Jack)] >= 1:
				for _, face := range []Face{Ace, Ten, King, Queen, Jack} {
					show[CreateCard(suit, face)] = max(show[CreateCard(suit, face)], 1)
				}
				if count[CreateCard(suit, King)] == 2 && count[CreateCard(suit, Queen)] == 2 {
					show[CreateCard(suit, King)] = 2
					show[CreateCard(suit, Queen)] = 2
					meld += 19
					if debug {
						fmt.Println("SingleStraightWithExtraMarriage")
					}
				} else {
					if debug {
						fmt.Println("SingleStraight")
					}
					meld += 15
				}
			case count[CreateCard(suit, King)] == 2 && count[CreateCard(suit, Queen)] == 2:
				meld += 8
				show[CreateCard(suit, King)] = 2
				show[CreateCard(suit, Queen)] = 2
				if debug {
					fmt.Println("DoubleMarriageInTrump")
				}
			case count[CreateCard(suit, King)] >= 1 && count[CreateCard(suit, Queen)] >= 1:
				meld += 4
				show[CreateCard(suit, King)] = max(show[CreateCard(suit, King)], 1)
				show[CreateCard(suit, Queen)] = max(show[CreateCard(suit, Queen)], 1)
				if debug {
					fmt.Println("SingleMarriageInTrump")
				}
			}
		case count[CreateCard(suit, King)] == 2 && count[CreateCard(suit, Queen)] == 2:
			show[CreateCard(suit, King)] = 2
			show[CreateCard(suit, Queen)] = 2
			meld += 4
			if debug {
				fmt.Println("DoubleMarriage")
			}
		case count[CreateCard(suit, King)] >= 1 && count[CreateCard(suit, Queen)] >= 1:
			show[CreateCard(suit, King)] = max(show[CreateCard(suit, King)], 1)
			show[CreateCard(suit, Queen)] = max(show[CreateCard(suit, Queen)], 1)
			if debug {
				fmt.Println("SingleMarriage")
			}
			meld += 2
		}
		for _, face := range Faces() { // looking for "around" meld
			//						fmt.Printf("Looking for %d in suit %d\n", value, suit)
			around[face] = min(count[CreateCard(suit, face)], around[face])
		}
	}
	for _, face := range []Face{Ace, King, Queen, Jack} {
		if around[face] > 0 {
			var worth int
			switch face {
			case Ace:
				worth = acearound
			case King:
				worth = kingaround
			case Queen:
				worth = queenaround
			case Jack:
				worth = jackaround
			}
			if around[face] == 2 {
				worth *= 10
			}
			for _, suit := range Suits() {
				show[CreateCard(suit, face)] = max(show[CreateCard(suit, face)], around[face])
			}
			meld += worth
			if debug {
				fmt.Printf("Around-%d\n", worth)
			}
		}
	}
	switch { // pinochle
	case count[CreateCard(Diamonds, Jack)] == 2 && count[CreateCard(Spades, Queen)] == 2:
		meld += 30
		show[CreateCard(Spades, Queen)] = 2
		show[CreateCard(Diamonds, Jack)] = 2
		if debug {
			fmt.Println("DoubleNochle")
		}
	case count[CreateCard(Diamonds, Jack)] >= 1 && count[CreateCard(Spades, Queen)] >= 1:
		meld += 4
		show[CreateCard(Diamonds, Jack)] = max(show[CreateCard(Diamonds, Jack)], 1)
		show[CreateCard(Spades, Queen)] = max(show[CreateCard(Spades, Queen)], 1)
		if debug {
			fmt.Println("Nochle")
		}
	}
	result = make([]Card, 0, 12)
	for card, amount := range show {
		for {
			if amount > 0 {
				result = append(result, card)
				amount--
			} else {
				break
			}
		}
	}
	sort.Sort(result)
	return
}
