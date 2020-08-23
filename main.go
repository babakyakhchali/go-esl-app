package main

import (
	"strings"

	"github.com/0x19/goesl"
)

func sessionHandler(msg *goesl.Message, esl *goesl.Client) {
	s := Session{
		FsConnector: FsConnector{
			uuid:   msg.GetHeader("Unique-ID"),
			cmds:   make(chan map[string]string),
			events: make(chan IEvent),
			errors: make(chan error),
			closed: false,
		},
	}
	sessions[s.uuid] = &s
	app := MyApp{
		session: &s,
	}
	go app.run()
	for {
		cmd, more := <-s.cmds
		if !more {
			break
		}
		esl.SendMsg(cmd, s.uuid, "")
	}
	goesl.Debug("session ended:%s", s.uuid)
}

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

	client.Send("events json ALL")
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
		goesl.Debug("got event:%s(%s) uuid:%s", msg.GetHeader("Event-Name"), msg.GetHeader("Event-Subclass"), msg.GetHeader("Unique-ID"))
		if msg.GetHeader("Event-Name") == "CHANNEL_PARK" {
			go sessionHandler(msg, client)
		} else if msg.GetHeader("Unique-ID") != "" {
			s, r := sessions[msg.GetHeader("Unique-ID")]
			if r {
				s.events <- msg
			}
		}
		goesl.Debug("%s", msg)
	}
}
