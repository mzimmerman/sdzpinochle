package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	sdz "github.com/mzimmerman/sdzpinochle"
)

type BiddingStrategy func(h *sdz.Hand, bids []uint8, score [2]uint8) (uint8, sdz.Suit)
type PlayingStrategy func(ht *HandTracker, t sdz.Suit) sdz.Card

var biddingStrategies = map[string]BiddingStrategy{
	/*
		"NeverBid": func(hand *sdz.Hand, bids []uint8, score [2]uint8) (uint8, sdz.Suit) {
			return 20, sdz.Hearts
			// TODO: make this choose the best suit in case it gets stuck
		},
		"MostMeldPlus7": func(h *sdz.Hand, b []uint8, score [2]uint8) (uint8, sdz.Suit) {
			meld, suit := chooseSuitWithMostMeld(h, b, score)
			bid := meld + 7
			if bid < 20 {
				bid = 20
			}
			return bid, suit
		},
	*/
	"ChooseSuitWithMostMeld": chooseSuitWithMostMeld,
	"BasicBid":               basicBid,
	constMattBid: func(realHand *sdz.Hand, prevBids []uint8, score [2]uint8) (amount uint8, trump sdz.Suit) {
		bids := make(map[sdz.Suit]uint8)
		for _, suit := range sdz.Suits {
			bids[suit], _ = realHand.Meld(suit)
			bids[suit] = bids[suit] + powerBid(realHand, suit)
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
		return bids[trump], trump
	},
}

func countSuit(hand *sdz.Hand, suit sdz.Suit) uint8 {
	var count uint8 = 0
	for _, card := range *hand {
		if card.Suit() == suit {
			count++
		}
	}
	return count
}

func countAces(hand *sdz.Hand) uint8 {
	var count uint8 = 0
	for _, card := range *hand {
		if card.Face() == sdz.Ace {
			count++
		}
	}
	return count
}

func maximum(nums []uint8) uint8 {
	var largest uint8 = 0
	for _, num := range nums {
		if num > largest {
			largest = num
		}
	}
	return largest
}

func dontOverbid(bid uint8, bids []uint8, trump sdz.Suit) (uint8, sdz.Suit) {
	var highestBid uint8 = bid
	maxBid := maximum(bids)
	if len(bids) == 3 && maxBid >= 20 && highestBid > maxBid {
		highestBid = maxBid + 1
	}
	return highestBid, trump
}

func basicBid(hand *sdz.Hand, bids []uint8, score [2]uint8) (uint8, sdz.Suit) {
	estimatedCounters := func(hand *sdz.Hand, s sdz.Suit) uint8 {
		return countSuit(hand, s) + countAces(hand)
	}
	var highestBid, partnerHelp uint8 = 0, 4
	var trump sdz.Suit
	for _, suit := range sdz.Suits {
		meld, _ := hand.Meld(suit)
		bid := meld + estimatedCounters(hand, suit) + partnerHelp
		if bid > highestBid {
			highestBid = bid
			trump = suit
		}
	}
	if highestBid < 20 && len(bids) == 3 {
		// Be a little more aggressive if we're stuck
		highestBid = highestBid + 3
	}
	return dontOverbid(highestBid, bids, trump)
}

func chooseSuitWithMostMeld(hand *sdz.Hand, bids []uint8, score [2]uint8) (uint8, sdz.Suit) {
	var highestMeld uint8 = 0
	var trump sdz.Suit
	for _, suit := range sdz.Suits {
		meld, _ := hand.Meld(suit)
		if meld > highestMeld {
			highestMeld = meld
			trump = suit
		}
	}
	if highestMeld < 20 {
		highestMeld = 20
	}
	return highestMeld, trump
}

var playingStrategies = map[string]PlayingStrategy{
	/*
		"PlayHighest": func(ht *HandTracker, t sdz.Suit) sdz.Card {
			hand := handFromHT(ht)
			for _, face := range sdz.Faces {
				for _, card := range *hand {
					if card.Face() == face && sdz.ValidPlay(card, ht.Trick.WinningCard(), ht.Trick.LeadSuit(), hand, t) {
						return card
					}
				}
			}
			return (*hand)[0]
		},
		"PlayLowest": func(ht *HandTracker, t sdz.Suit) sdz.Card {
			hand := handFromHT(ht)
			for _, face := range [6]sdz.Face{sdz.Nine, sdz.Jack, sdz.Queen, sdz.King, sdz.Ten, sdz.Ace} {
				for _, card := range *hand {
					if card.Face() == face && sdz.ValidPlay(card, ht.Trick.WinningCard(), ht.Trick.LeadSuit(), hand, t) {
						return card
					}
				}
			}
			return (*hand)[0]
		},
		"PlayRandom": func(ht *HandTracker, t sdz.Suit) sdz.Card {
			hand := handFromHT(ht)
			for _, v := range rand.Perm(len(*hand)) {
				if sdz.ValidPlay((*hand)[v], ht.Trick.WinningCard(), ht.Trick.LeadSuit(), hand, t) {
					return (*hand)[v]
				}
			}
			return (*hand)[0]
		},
	*/
	"BasicPlay": basicPlay,
	//constMattSimulation: PlayHandWithCardDuration(time.Second * 5),
	//	"Simulation2":       PlayHandWithCardDuration(time.Second * 2),
	//	"Simulation5":       PlayHandWithCardDuration(time.Second * 5),
	"MattSimulation1Second": PlayHandWithCardDuration(time.Second * 1),
}

func findBareAce(hand *sdz.Hand) sdz.Card {
	for _, suit := range sdz.Suits {
		if countSuit(hand, suit) == 1 {
			for _, card := range *hand {
				if card.Suit() == suit && card.Face() == sdz.Ace {
					return card
				}
			}
		}
	}
	return sdz.NACard
}

func basicPlay(ht *HandTracker, trump sdz.Suit) sdz.Card {
	hand := handFromHT(ht)
	findPartner := map[uint8]uint8{0: 2, 1: 3, 2: 0, 3: 1}
	// We're first to play
	if ht.Trick.Plays == 0 {
		// Play bare aces first
		card := findBareAce(hand)
		if card != sdz.NACard {
			return card
		}
		// Play aces first and counters last
		for _, face := range [6]sdz.Face{sdz.Ace, sdz.Queen, sdz.Nine, sdz.Jack, sdz.King, sdz.Ten} {
			for _, card := range *hand {
				if card.Face() == face && card.Suit() != trump {
					return card
				}
			}
		}
	} else if ht.Trick.Plays > 1 && // Our partner has played
		ht.Trick.WinningPlayer == findPartner[ht.Trick.Next] && // and is winning
		(ht.Trick.Played[ht.Trick.Plays-2].Face() == sdz.Ace || // with an Ace
			(ht.Trick.Played[ht.Trick.Plays-2].Suit() == trump &&
				ht.Trick.Played[0].Suit() != trump)) { // Or has trumped in
		// Play counters first and aces last
		for _, face := range [6]sdz.Face{sdz.King, sdz.Ten, sdz.Nine, sdz.Jack, sdz.Queen, sdz.Ace} {
			for _, card := range *hand {
				if card.Face() == face && sdz.ValidPlay(card, ht.Trick.WinningCard(), ht.Trick.LeadSuit(), hand, trump) {
					return card
				}
			}
		}

	}
	// Play aces if we can win with them
	for _, card := range *hand {
		if card.Face() == sdz.Ace &&
			sdz.ValidPlay(card, ht.Trick.WinningCard(), ht.Trick.LeadSuit(), hand, trump) &&
			card.Beats(ht.Trick.WinningCard(), trump) &&
			card.Suit() != trump {
			return card
		}
	}
	// We can't play an Ace so play the lowest thing we can
	for _, face := range [6]sdz.Face{sdz.Nine, sdz.Jack, sdz.Queen, sdz.King, sdz.Ten, sdz.Ace} {
		for _, card := range *hand {
			if card.Face() == face && sdz.ValidPlay(card, ht.Trick.WinningCard(), ht.Trick.LeadSuit(), hand, trump) {
				return card
			}
		}
	}
	return sdz.NACard
}

func handFromHT(ht *HandTracker) *sdz.Hand {
	hand := make(sdz.Hand, 0)
	for x := 1; x < 25; x++ {
		if ht.Cards[ht.Trick.Next][x] > 0 {
			hand = append(hand, sdz.Card(x))
		}
	}
	return &hand
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

var HTStack = sync.Pool{
	New: func() interface{} {
		return &HandTracker{
			Trick: &Trick{},
		}
	},
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

func (ht *HandTracker) sum(cardIndex sdz.Card) (sum uint8) {
	sum = ht.PlayedCards[cardIndex]
	for x := 0; x < len(ht.Cards); x++ {
		if ht.Cards[x][cardIndex] != Unknown {
			sum += ht.Cards[x][cardIndex]
		}
	}
	if sum > 2 {
		sdz.Log(ht.Owner, "Summing card %s, sum = %d", cardIndex, sum)
		ht.Debug()
		panic("sumthing is wrong, get it?!?!")
	}
	return
}

func (oldht *HandTracker) Copy() (newht *HandTracker, err error) {
	newht = getHT()
	for x := uint8(0); x < uint8(len(oldht.Cards)); x++ {
		newht.Cards[x] = oldht.Cards[x]
	}
	newht.PlayedCards = oldht.PlayedCards
	newht.PlayCount = oldht.PlayCount
	*newht.Trick = *oldht.Trick
	newht.Owner = oldht.Owner
	return
}

func (ht *HandTracker) Debug() {
	sdz.Log(ht.Owner, "ht.PlayedCards = %v", ht.PlayedCards)
	for x := 0; x < 4; x++ {
		sdz.Log(ht.Owner, "Player%d - %s", x, ht.Cards[x])
	}
	sdz.Log(ht.Owner, "PlayCount = %d, Next=%d", ht.PlayCount, ht.Trick.Next)
	panic("don't call debug")
}

func (ht *HandTracker) PlayCard(card sdz.Card, trump sdz.Suit) {
	//ht.Debug()
	playerid := ht.Trick.Next
	//Log(ht.Owner, "In ht.PlayCard for %d-%s on player %d", playerid, card, ht.Owner)
	val := ht.Cards[playerid][card]
	if val == None {
		fmt.Println("\n\n\nCard", card)
		sdz.Log(ht.Owner, "Player %d does not have card %s, panicking", playerid, card)
		ht.Debug()
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
	case ht.Trick.LeadSuit() == sdz.NASuit || trump == sdz.NASuit:
		// do nothing, start of the trick, everything is legal
	case card.Suit() != ht.Trick.LeadSuit() && card.Suit() != trump: // couldn't follow suit, couldn't lay trump
		ht.noSuit(playerid, trump)
		//if oright == ht && playerid == 1 {
		//	Log(ht.Owner, "Setting all %s to None for playerid=%d", trump, playerid)
		//}
		fallthrough
	case card.Suit() != ht.Trick.LeadSuit(): // couldn't follow suit
		ht.noSuit(playerid, ht.Trick.LeadSuit())
		//if oright == ht && playerid == 1 {
		//	Log(ht.Owner, "Setting all %s to None for playerid=%d", trick.LeadSuit(), playerid)
		//}
	}
	if playerid != ht.Trick.WinningPlayer { // did not win
		for _, f := range sdz.Faces {
			tempCard := sdz.CreateCard(card.Suit(), f)
			if tempCard.Beats(ht.Trick.WinningCard(), trump) {
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

type CardMap [25]uint8

func (cm *CardMap) inc(x sdz.Card) {
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

func (cm CardMap) String() string {
	output := "CardMap={"
	for x := sdz.AS; int8(x) <= sdz.AllCards; x++ {
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
