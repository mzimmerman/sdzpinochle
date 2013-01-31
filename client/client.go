// sdzpinochle-client project main.go
package main

import (
	"code.google.com/p/go.net/websocket"
	//"encoding/json"
	"fmt"
	sdz "github.com/mzimmerman/sdzpinochle"
	"os"
	"sort"
)

func send(conn *websocket.Conn, action *sdz.Action) {
	err := websocket.JSON.Send(conn, action)
	if err != nil {
		sdz.Log("Error sending - %v", err)
	}
}

func main() {
	autoClient := false
	for _, s := range os.Args {
		fmt.Println(s)
		if s == "auto" {
			autoClient = true
		}
	}
	conn, err := websocket.Dial("ws://localhost:10080/connect", "", "http://localhost:10080/")
	var playerid int
	var hand *sdz.Hand
	var bidAmount int
	var trump sdz.Suit
	if err != nil {
		sdz.Log("Error - %v", err)
		return
	}
	defer conn.Close()
	var previousPlay *sdz.Action
	var previousCard sdz.Card
	for {
		var action sdz.Action
		err := websocket.JSON.Receive(conn, &action)
		if err != nil {
			sdz.Log("Error decoding - %v", err)
			return
		}
		switch action.Type {
		case "Bid":
			if action.Playerid == playerid {
				sdz.Log("How much would you like to bid?:")
				fmt.Scan(&bidAmount)
				send(conn, sdz.CreateBid(bidAmount, playerid))
			} else {
				// received someone else's bid value'
				sdz.Log("Player #%d bid %d", action.Playerid, action.Bid)
			}
		case "Play":
			if action.Playerid == playerid {
				if previousPlay == nil || action.Lead == "" {
					previousPlay = &action
				} else {
					sdz.Log("Server rejected the play of %s, invalid play", previousCard)
					*hand = append(*hand, previousCard)
					sort.Sort(hand)
				}
				var card sdz.Card
				if action.Lead == "" {
					card = (*hand)[0]
				} else {
					for _, c := range *hand {
						if sdz.ValidPlay(c, action.WinningCard, action.Lead, hand, trump) {
							card = c
						}
					}
				}
				sdz.Log("Your turn, in your hand is %s - what would you like to play? Trump is %s - valid play is %s:", hand, trump, card)
				for {
					if !autoClient {
						fmt.Scan(&card)
					}
					//sdz.Log("Received input %s", card)
					if hand.Remove(card) {
						send(conn, sdz.CreatePlay(card, playerid))
						previousCard = card
						break
					} else {
						sdz.Log("Invalid play, card does not exist in your hand")
					}
				}
			} else {
				sdz.Log("Player %d played card %s", action.Playerid, action.PlayedCard)
				previousPlay = nil // not going to ask us to replay since the next response was another player's play
				// received someone else's play'
			}
		case "Trump":
			if action.Playerid == playerid {
				sdz.Log("What would you like to make trump?")
				fmt.Scan(&trump)
				send(conn, sdz.CreateTrump(trump, playerid))
			} else {
				sdz.Log("Player %d says trump is %s", action.Playerid, action.Trump)
				trump = action.Trump
			}
		case "Throwin":
			sdz.Log("Player %d threw in", action.Playerid)
		case "Deal":
			playerid = action.Playerid
			hand = &action.Hand
			sdz.Log("Your hand is - %s", hand)
		case "Meld":
			sdz.Log("Player %d is melding %s for %d points", action.Playerid, action.Hand, action.Amount)
		case "Score": // this client does not have to implement this type as it's already told through Message actions
		case "Message":
			sdz.Log(action.Message)
		case "Hello":
			var response string
			fmt.Scan(&response)
			send(conn, sdz.CreateHello(response))
		case "Game":
			var option int
			fmt.Scan(&option)
			send(conn, sdz.CreateGame(option))
		default:
			sdz.Log("Received an action I didn't understand - %v", action)
		}

	}
}
