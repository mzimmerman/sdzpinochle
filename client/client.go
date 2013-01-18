// sdzpinochle-client project main.go
package main

import (
	"encoding/json"
	"fmt"
	sdz "github.com/mzimmerman/sdzpinochle"
	"net"
)

func send(enc *json.Encoder, data interface{}) {
	err := enc.Encode(data)
	if err != nil {
		sdz.Log("Error sending - %v", err)
	}
}

func main() {
	conn, err := net.Dial("tcp", "localhost:1201")
	var playerid int
	var hand *sdz.Hand
	var bidAmount int
	if err != nil {
		sdz.Log("Error - %v", err)
		return
	}
	defer conn.Close()
	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)
	for {
		var action sdz.Action
		err := dec.Decode(&action)
		if err != nil {
			sdz.Log("Error decoding - %v", err)
			return
		}
		switch action.Type {
		case "Bid":
			if action.Playerid == playerid {
				sdz.Log("How much would you like to bid?:")
				fmt.Scan(&bidAmount)
				send(enc, sdz.CreateBid(bidAmount, playerid))
			} else {
				// received someone else's bid value'
				sdz.Log("Player #%d bid %d", action.Playerid, action.Bid)
			}
		case "Play":
			if action.Playerid == playerid {
				var card sdz.Card
				sdz.Log("Your turn, in your hand is %s - what would you like to play?:", hand)
				fmt.Scan(&card)
				sdz.Log("Received input %s", card)
				send(enc, sdz.CreatePlay(card, playerid))
			} else {
				sdz.Log("Player %d played card %s", action.Playerid, action.PlayedCard)
				// received someone else's play'
			}
		case "Trump":
			if action.Playerid == playerid {
				var trump sdz.Suit
				sdz.Log("What would you like to make trump?")
				fmt.Scan(&trump)
				send(enc, sdz.CreateTrump(trump, playerid))
			} else {
				sdz.Log("Player %d says trump is %s", action.Playerid, action.Trump)
			}
		case "Throwin":
			sdz.Log("Player %d threw in", action.Playerid)
		case "Deal":
			playerid = action.Playerid
			hand = &action.Hand
			sdz.Log("Your hand is - %s", hand)
		case "Meld":
			sdz.Log("Player %d is melding %s for %d points", action.Playerid, action.Hand, action.Amount)
		case "Hello":
			send(enc, sdz.CreateHello("create"))
		default:
			sdz.Log("Received an action I didn't understand - %v", action)
		}

	}
}
