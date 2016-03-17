// sdzpinochle-client project main.go
package main

import (
	"encoding/json"
	"net"

	//"encoding/json"
	"fmt"
	"os"
	"sort"

	sdz "github.com/mzimmerman/sdzpinochle"
)

func send(conn net.Conn, action *sdz.Action) {
	jsonData, err := json.Marshal(action)
	if err != nil {
		sdz.Log(4, "Error encoding action - %v", err)
	}
	_, err = conn.Write(jsonData)
	if err != nil {
		sdz.Log(4, "Error sending action - %v", err)
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
	conn, err := net.Dial("tcp", ":1201")
	var playerid uint8
	var hand *sdz.Hand
	var bidAmount uint8
	var trump sdz.Suit
	if err != nil {
		sdz.Log(4, "Error - %v", err)
		return
	}
	defer conn.Close()
	decoder := json.NewDecoder(conn)
	var previousPlay *sdz.Action
	var previousCard sdz.Card
	for {
		var action sdz.Action
		err = decoder.Decode(&action)
		if err != nil {
			sdz.Log(playerid, "Error decoding - %v", err)
			return
		}
		switch action.Type {
		case "Bid":
			if action.Playerid == playerid {
				sdz.Log(playerid, "How much would you like to bid?:")
				fmt.Scan(&bidAmount)
				send(conn, sdz.CreateBid(bidAmount, playerid))
			} else {
				// received someone else's bid value'
				sdz.Log(playerid, "Player #%d bid %d", action.Playerid, action.Bid)
			}
		case "Play":
			if action.Playerid == playerid {
				if previousPlay == nil || action.Lead == sdz.NASuit {
					previousPlay = &action
				} else {
					sdz.Log(playerid, "Server rejected the play of %s, invalid play", previousCard)
					*hand = append(*hand, previousCard)
					sort.Sort(hand)
				}
				var card sdz.Card
				if action.Lead == sdz.NASuit {
					card = (*hand)[0]
				} else {
					for _, c := range *hand {
						if sdz.ValidPlay(c, action.WinningCard, action.Lead, hand, trump) {
							card = c
						}
					}
				}
				sdz.Log(playerid, "Your turn, in your hand is %s - what would you like to play? Trump is %s - valid play is %s:", hand, trump, card)
				for {
					if !autoClient {
						fmt.Scan(&card)
					}
					//sdz.Log(playerid,"Received input %s", card)
					if hand.Remove(card) {
						send(conn, sdz.CreatePlay(card, playerid))
						previousCard = card
						break
					} else {
						sdz.Log(playerid, "Invalid play, card does not exist in your hand")
					}
				}
			} else {
				sdz.Log(playerid, "Player %d played card %s", action.Playerid, action.PlayedCard)
				previousPlay = nil // not going to ask us to replay since the next response was another player's play
				// received someone else's play'
			}
		case "Trump":
			if action.Playerid == playerid {
				sdz.Log(playerid, "What would you like to make trump?")
				fmt.Scan(&trump)
				send(conn, sdz.CreateTrump(trump, playerid))
			} else {
				sdz.Log(playerid, "Player %d says trump is %s", action.Playerid, action.Trump)
				trump = action.Trump
			}
		case "Throwin":
			sdz.Log(playerid, "Player %d threw in", action.Playerid)
		case "Deal":
			playerid = action.Playerid
			hand = &action.Hand
			sdz.Log(playerid, "Your hand is - %s", hand)
		case "Meld":
			sdz.Log(playerid, "Player %d is melding %s for %d points", action.Playerid, action.Hand, action.Amount)
		case "Score": // this client does not have to implement this type as it's already told through Message actions
		case "Message":
			sdz.Log(playerid, action.Message)
		case "Hello":
			var response string
			fmt.Scan(&response)
			send(conn, sdz.CreateMessage(response))
		case "Game":
			var option int
			fmt.Scan(&option)
			send(conn, sdz.CreateMessage("option"))
		default:
			sdz.Log(playerid, "Received an action I didn't understand - %v", action)
		}

	}
}
