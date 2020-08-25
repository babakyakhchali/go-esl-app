package main

import (
	"fmt"
	"strings"

	"github.com/0x19/goesl"
)

var (
	sessions = map[string]*Session{}
)

func main() {
	client, err := goesl.NewClient("127.0.0.1", 8021, "ClueCon", 3)

	if err != nil {
		goesl.Error("Error while creating new client: %s", err)
		return
	}

	goesl.Debug("Yuhu! New client: %q", client)
	go client.Handle()

	client.Send("events json CHANNEL_HANGUP CHANNEL_EXECUTE CHANNEL_EXECUTE_COMPLETE CHANNEL_PARK CHANNEL_DESTROY")
	for {
		msg, err := client.ReadMessage()
		if err != nil {

			// If it contains EOF, we really dont care...
			if !strings.Contains(err.Error(), "EOF") && err.Error() != "unexpected end of JSON input" {
				goesl.Error("Error while reading Freeswitch message: %s", err)
			}
			for _, v := range sessions {
				v.errors <- err
			}
			break
		}
		eventName := msg.GetHeader("Event-Name")
		eventSubclass := msg.GetHeader("Event-Subclass")
		channelUUID := msg.GetHeader("Unique-ID")
		goesl.Debug("got event:%s(%s) uuid:%s", eventName, eventSubclass, channelUUID)
		if eventName == "CHANNEL_PARK" {
			go eslSessionHandler(msg, client)
		} else if channelUUID != "" {
			s, r := sessions[channelUUID]
			if r {
				if eventName == "CHANNEL_DESTROY" {
					delete(sessions, channelUUID)
					fmt.Printf("deleted channel %s. remained channels:%d", channelUUID, len(sessions))
					continue
				}
				select {
				case s.events <- msg:
					fmt.Printf("handled event %s for channel %s", msg.GetHeader("Event-Name"), msg.GetHeader("Unique-ID"))
				default:
					fmt.Printf("ignoring event %s for channel %s", msg.GetHeader("Event-Name"), msg.GetHeader("Unique-ID"))
				}

			}
		}
		//goesl.Debug("%v", msg)
	}
	fmt.Printf("Application exitted")
}
