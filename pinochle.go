// pinochle.go
package sdzpinochle

import (
	"bytes"
	"encoding/json"
	"errors"
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
	Ace         Face  = iota
	Ten         Face  = iota
	King        Face  = iota
	Queen       Face  = iota
	Jack        Face  = iota
	Nine        Face  = iota
	NAFace      Face  = -1
	acearound   uint8 = 10
	kingaround  uint8 = 8
	queenaround uint8 = 6
	jackaround  uint8 = 4
	debugLog          = false
	AllCards    int8  = 24
)

const (
	Spades   Suit = iota
	Hearts   Suit = iota
	Clubs    Suit = iota
	Diamonds Suit = iota
	NASuit   Suit = iota
)

const (
	AS     Card = iota
	TS     Card = iota
	KS     Card = iota
	QS     Card = iota
	JS     Card = iota
	NS     Card = iota
	AH     Card = iota
	TH     Card = iota
	KH     Card = iota
	QH     Card = iota
	JH     Card = iota
	NH     Card = iota
	AC     Card = iota
	TC     Card = iota
	KC     Card = iota
	QC     Card = iota
	JC     Card = iota
	NC     Card = iota
	AD     Card = iota
	TD     Card = iota
	KD     Card = iota
	QD     Card = iota
	JD     Card = iota
	ND     Card = iota
	NACard Card = iota
)

var Faces [6]Face
var Suits [4]Suit

func init() {
	rand.Seed(time.Now().UnixNano())
	Faces = [6]Face{Ace, Ten, King, Queen, Jack, Nine}
	Suits = [4]Suit{Spades, Hearts, Clubs, Diamonds}
}

type Card int8 // an integer representation of the card
type Suit int8
type Face int8

type Deck [48]Card
type Hand []Card
type SmallHand [6]byte

func (c Card) GetBitInfo() (bitnum uint, sliceIndex int8) {
	bitnum = uint(c%4) * 2
	sliceIndex = int8(c / 4)
	return
}

func CreateCard(suit Suit, face Face) Card {
	return Card(int(suit)*6 + int(face))
}

func (c Card) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

func (s *Suit) UnmarshalJSON(data []byte) error {
	var str string
	err := json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	if len(str) != 1 {
		return errors.New(fmt.Sprintf("Data %s not a suit", data))
	}
	suit := NASuit
	switch str[0] {
	case 'D':
		suit = Diamonds
	case 'S':
		suit = Spades
	case 'H':
		suit = Hearts
	case 'C':
		suit = Clubs
	default:
		return errors.New(fmt.Sprintf("Data %s not a suit", data))
	}
	*s = Suit(suit)
	return nil
}

func (c *Card) UnmarshalJSON(data []byte) error {
	var str string
	err := json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	if len(str) != 2 {
		return errors.New(fmt.Sprintf("Data %s not a card", data))
	}
	face := NAFace
	switch str[0] {
	case 'A':
		face = Ace
	case 'T':
		face = Ten
	case 'K':
		face = King
	case 'Q':
		face = Queen
	case 'J':
		face = Jack
	case '9':
		face = Nine
	default:
		return errors.New(fmt.Sprintf("Data %s not a card", data))
	}
	suit := NASuit
	switch str[1] {
	case 'D':
		suit = Diamonds
	case 'S':
		suit = Spades
	case 'H':
		suit = Hearts
	case 'C':
		suit = Clubs
	default:
		return errors.New(fmt.Sprintf("Data %s not a card", data))
	}
	*c = Card(int(suit)*6 + int(face))
	return nil
}

func (c Card) String() string {
	if c == NACard {
		return "NA"
	}
	return c.Face().String() + c.Suit().String()
}

func (a Card) Beats(b Card, trump Suit) bool {
	// a is the challenging card
	if b == NACard {
		return true
	}
	switch {
	case a.Suit() == b.Suit():
		return a < b
	case a.Suit() == trump:
		return true
	}
	return false
}

func (c Card) Counter() bool {
	return c.Face() == Ace || c.Face() == Ten || c.Face() == King
}

func (c Card) Suit() Suit {
	if c == NACard {
		return NASuit
	}
	return Suit(int(c) / 6)
}

func (c Card) Face() Face {
	if c == NACard {
		return NAFace
	}
	return Face(int(c) % 6)
}

func (d *Deck) Swap(i, j uint8) {
	d[i], d[j] = d[j], d[i]
}

func (d *Deck) Shuffle() {
	//	http://en.wikipedia.org/wiki/Fisher%E2%80%93Yates_shuffle#The_modern_algorithm
	for i := len(d) - 1; i >= 1; i-- {
		if j := rand.Intn(i); i != j {
			d.Swap(uint8(i), uint8(j))
		}
	}
}

func (h *SmallHand) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("SmallHand{")
	for card := AS; int8(card) < AllCards; card++ {
		count := h.Count(card)
		for {
			if count == 0 {
				break
			}
			count--
			buffer.WriteString(card.String())
			buffer.WriteString(", ")
		}
	}
	buffer.WriteString("}")
	return buffer.String()
}

func (h Hand) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("Hand{")
	for x := range h {
		buffer.WriteString(h[x].String())
		buffer.WriteString(", ")
	}
	buffer.WriteString("}")
	return buffer.String()
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
	return a < b
}

func (s Suit) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (a Suit) String() string {
	switch a {
	case NASuit:
		return "~"
	case Diamonds:
		return "D"
	case Spades:
		return "S"
	case Hearts:
		return "H"
	case Clubs:
		return "C"
	}
	panic(fmt.Sprintf("Error finding suit for %d", a))
}

func (a Face) String() string {
	switch a {
	case Nine:
		return "9"
	case Jack:
		return "J"
	case Queen:
		return "Q"
	case King:
		return "K"
	case Ten:
		return "T"
	case Ace:
		return "A"
	}
	panic(fmt.Sprintf("Error finding face for %d", int(a)))
}

func (a Suit) Less(b Suit) bool { // only for sorting the suits for display in the hand
	return a > b
}

func (h Hand) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *Hand) Shuffle() {
	//	http://en.wikipedia.org/wiki/Fisher%E2%80%93Yates_shuffle#The_modern_algorithm
	for i := len(*h) - 1; i >= 1; i-- {
		if j := rand.Intn(i); i != j {
			h.Swap(i, j)
		}
	}
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
	for x := int8(0); x < int8(len(deck)); x++ {
		deck[x] = Card(x % AllCards)
	}
	return
}

type Action struct {
	Type                    string
	Playerid                uint8
	Bid                     uint8
	PlayedCard, WinningCard Card
	Lead, Trump             Suit
	Amount                  uint8
	Message                 string
	Hand                    Hand
	TableId                 int64
	GameOver, Win           bool
	Score                   []int16
	Dealer                  uint8
	WinningPlayer           uint8
}

func (action *Action) String() string {
	data, _ := action.MarshalJSON()
	return string(data)
}

func (action *Action) UnmarshalJSON(data []byte) error {
	victim := new(JSONAction)
	err := json.Unmarshal(data, &victim)
	if err != nil {
		return err
	}
	action.Amount = victim.Amount
	action.Bid = victim.Bid
	action.Message = victim.Message
	//action.PlayedCard =
	action.Playerid = victim.Playerid
	action.TableId = victim.TableId
	//action.Trump =
	action.Type = victim.Type
	var buf bytes.Buffer
	if victim.PlayedCard != "" {
		buf.WriteString("\"")
		buf.WriteString(victim.PlayedCard)
		buf.WriteString("\"")
		err = json.Unmarshal(buf.Bytes(), &action.PlayedCard)
		if err != nil {
			return err
		}
	}
	if victim.Trump != "" {
		buf.Reset()
		buf.WriteString("\"")
		buf.WriteString(victim.Trump)
		buf.WriteString("\"")
		err = json.Unmarshal(buf.Bytes(), &action.Trump)
		if err != nil {
			return err
		}
	}
	return nil
}

type JSONAction struct {
	Type       string
	Playerid   uint8
	Bid        uint8
	PlayedCard string
	Trump      string
	Amount     uint8
	Message    string
	TableId    int64
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
		case typ.Field(x).Name == "WinningPlayer" && action.Type == "Play":
			data["WinningPlayer"] = action.WinningPlayer
		case typ.Field(x).Name == "Amount" && action.Type == "Bid":
			data["Amount"] = action.Amount
		case typ.Field(x).Name == "Win" && action.GameOver:
			data["Win"] = action.Win
		case typ.Field(x).Name == "GameOver" && action.Type == "Score":
			data["GameOver"] = action.GameOver
		case typ.Field(x).Name == "Dealer" && action.Type == "Deal":
			data["Dealer"] = action.Dealer
		case (typ.Field(x).Name == "PlayedCard" || typ.Field(x).Name == "WinningCard") && val.Field(x).Interface() != NACard:
			data[typ.Field(x).Name] = fmt.Sprintf("%s", val.Field(x).Interface())
		case reflect.DeepEqual(val.Field(x).Interface(), reflect.New(typ.Field(x).Type).Elem().Interface()):
		default:
			data[typ.Field(x).Name] = val.Field(x).Interface()
		}
	}
	return json.Marshal(data)
}

func CreateName() *Action {
	return &Action{Type: "Name"}
}

func CreateSit(tableid int64) *Action {
	return &Action{Type: "Sit", TableId: tableid}
}

func CreateMessage(m string) *Action {
	return &Action{Type: "Message", Message: m}
}

func CreateBid(bid, playerid uint8) *Action {
	return &Action{Type: "Bid", Bid: bid, Playerid: playerid}
}

func CreatePlayRequest(winning Card, lead, trump Suit, playerid uint8, hand *Hand) *Action {
	return &Action{Type: "Play", WinningCard: winning, Lead: lead, Trump: trump, Playerid: playerid, Hand: *hand}
}

func CreatePlay(card Card, playerid uint8) *Action {
	return &Action{Type: "Play", PlayedCard: card, Playerid: playerid}
}

func CreateTrump(trump Suit, playerid uint8) *Action {
	return &Action{Type: "Trump", Trump: trump, Playerid: playerid}
}

func CreateTrick(winningPlayer uint8) *Action {
	return &Action{Type: "Trick", Playerid: winningPlayer}
}

func CreateThrowin(playerid uint8) *Action {
	return &Action{Type: "Throwin", Playerid: playerid}
}

func CreateMeld(hand Hand, amount, playerid uint8) *Action {
	return &Action{Type: "Meld", Hand: hand, Amount: amount, Playerid: playerid}
}

func CreateDisconnect(playerid uint8) *Action {
	return &Action{Type: "Disconnect", Playerid: playerid}
}

func CreateDeal(hand Hand, playerid, dealer uint8) *Action {
	return &Action{Type: "Deal", Hand: hand, Playerid: playerid, Dealer: dealer}
}

func CreateScore(score []int16, gameOver, win bool) *Action {
	return &Action{Type: "Score", Score: score, Win: win, GameOver: gameOver}
}

type PlayerImpl struct {
	Playerid uint8
}

func (p PlayerImpl) PlayerID() uint8 {
	return p.Playerid
}

func (p PlayerImpl) Team() uint8 {
	return p.Playerid % 2
}

func (p PlayerImpl) IsPartner(player uint8) bool {
	return p.Playerid%2 == player%2
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
	if winningCard == NACard || leadSuit == NASuit {
		return true
	}
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
	if !hasCard { // you don't have the card in your hand, not allowed to play it, cheater!
		return false
	}
	if winningCard == NACard { // nothing to follow so far, so you win!
		return true
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

func (h *SmallHand) Contains(card Card) bool {
	if h == nil {
		return false
	}
	bitnum, sliceIndex := card.GetBitInfo()
	resp := ((*h)[sliceIndex]>>bitnum)&1 == 1
	return resp
}

func (h *SmallHand) Count(card Card) int8 {
	if h == nil {
		return 0
	}
	bitnum, sliceIndex := card.GetBitInfo()
	if ((*h)[sliceIndex]>>(bitnum+1))&1 == 1 {
		return 2
	} else if ((*h)[sliceIndex]>>bitnum)&1 == 1 {
		return 1
	}
	return 0
}

func (h *Hand) Contains(card Card) bool {
	for _, c := range *h {
		if c == card {
			return true
		}
	}
	return false
}

func (h *SmallHand) CopySmallHand() *SmallHand {
	sh := new(SmallHand)
	*sh = *h
	return sh
}

func NewSmallHand() *SmallHand {
	return new(SmallHand)
}

func (sh *SmallHand) Append(cards ...Card) {
	for x := range cards {
		bitnum, sliceIndex := cards[x].GetBitInfo()
		if sh.Contains(cards[x]) {
			(*sh)[sliceIndex] = (*sh)[sliceIndex] | (1 << (bitnum + 1))
		} else {
			(*sh)[sliceIndex] = (*sh)[sliceIndex] | (1 << bitnum)
		}
	}
}

func (sh *SmallHand) Remove(card Card) bool {
	bitnum, sliceIndex := card.GetBitInfo()
	count := sh.Count(card)
	if count == 2 {
		//x &^ (1 << i)
		(*sh)[sliceIndex] = (*sh)[sliceIndex] & ^(1 << (bitnum + 1))
		return true
	}
	if count == 1 {
		(*sh)[sliceIndex] = (*sh)[sliceIndex] & ^(1 << bitnum)
		return true
	}
	return false
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

func (h Hand) CountSuit(suit Suit) (count int) {
	for _, card := range h {
		if card.Suit() == suit {
			count++
		}
	}
	return
}

func (h Hand) Count() (cards map[Card]uint8) {
	cards = make(map[Card]uint8)
	for _, face := range Faces {
		for _, suit := range Suits {
			cards[CreateCard(suit, face)] = 0
		}
	}
	for x := 0; x < len(h); x++ {
		cards[h[x]]++
	}
	return
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

func (h Hand) Meld(trump Suit) (meld uint8, result Hand) {
	// hand does not have to be sorted
	count := h.Count()
	if debugLog {
		fmt.Printf("Count is %v\n", count)
	}
	show := make(map[Card]uint8)
	around := make(map[Face]uint8)
	for _, value := range Faces {
		around[value] = 2
	}
	//	fmt.Printf("AroundBefore = %v\n", around)
	for _, suit := range Suits { // look through each suit
		switch { // straights & marriages
		case trump == suit:
			if debugLog {
				fmt.Printf("Scoring %d nine(s) in trump %s\n", count[CreateCard(suit, Nine)], trump)
			}
			meld += count[CreateCard(suit, Nine)] // 9s in trump
			show[CreateCard(suit, Nine)] = count[CreateCard(suit, Nine)]
			switch {
			// double straight
			case count[CreateCard(suit, Ace)] == 2 && count[CreateCard(suit, Ten)] == 2 && count[CreateCard(suit, King)] == 2 && count[CreateCard(suit, Queen)] == 2 && count[CreateCard(suit, Jack)] == 2:
				meld += 150
				for _, face := range Faces {
					show[CreateCard(suit, face)] = 2
				}
				if debugLog {
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
					if debugLog {
						fmt.Println("SingleStraightWithExtraMarriage")
					}
				} else {
					if debugLog {
						fmt.Println("SingleStraight")
					}
					meld += 15
				}
			case count[CreateCard(suit, King)] == 2 && count[CreateCard(suit, Queen)] == 2:
				meld += 8
				show[CreateCard(suit, King)] = 2
				show[CreateCard(suit, Queen)] = 2
				if debugLog {
					fmt.Println("DoubleMarriageInTrump")
				}
			case count[CreateCard(suit, King)] >= 1 && count[CreateCard(suit, Queen)] >= 1:
				meld += 4
				show[CreateCard(suit, King)] = max(show[CreateCard(suit, King)], 1)
				show[CreateCard(suit, Queen)] = max(show[CreateCard(suit, Queen)], 1)
				if debugLog {
					fmt.Println("SingleMarriageInTrump")
				}
			}
		case count[CreateCard(suit, King)] == 2 && count[CreateCard(suit, Queen)] == 2:
			show[CreateCard(suit, King)] = 2
			show[CreateCard(suit, Queen)] = 2
			meld += 4
			if debugLog {
				fmt.Println("DoubleMarriage")
			}
		case count[CreateCard(suit, King)] >= 1 && count[CreateCard(suit, Queen)] >= 1:
			show[CreateCard(suit, King)] = max(show[CreateCard(suit, King)], 1)
			show[CreateCard(suit, Queen)] = max(show[CreateCard(suit, Queen)], 1)
			if debugLog {
				fmt.Println("SingleMarriage")
			}
			meld += 2
		}
		for _, face := range Faces { // looking for "around" meld
			//						fmt.Printf("Looking for %d in suit %d\n", value, suit)
			around[face] = min(count[CreateCard(suit, face)], around[face])
		}
	}
	for _, face := range []Face{Ace, King, Queen, Jack} {
		if around[face] > 0 {
			var worth uint8
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
			for _, suit := range Suits {
				show[CreateCard(suit, face)] = max(show[CreateCard(suit, face)], around[face])
			}
			meld += worth
			if debugLog {
				fmt.Printf("Around-%d\n", worth)
			}
		}
	}
	switch { // pinochle
	case count[CreateCard(Diamonds, Jack)] == 2 && count[CreateCard(Spades, Queen)] == 2:
		meld += 30
		show[CreateCard(Spades, Queen)] = 2
		show[CreateCard(Diamonds, Jack)] = 2
		if debugLog {
			fmt.Println("DoubleNochle")
		}
	case count[CreateCard(Diamonds, Jack)] >= 1 && count[CreateCard(Spades, Queen)] >= 1:
		meld += 4
		show[CreateCard(Diamonds, Jack)] = max(show[CreateCard(Diamonds, Jack)], 1)
		show[CreateCard(Spades, Queen)] = max(show[CreateCard(Spades, Queen)], 1)
		if debugLog {
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
