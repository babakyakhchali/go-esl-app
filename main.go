package main

import (
	"fmt"

	"github.com/0x19/goesl"
	eslession "github.com/babakyakhchali/go-esl-wrapper/eslsession"
)

func h(s eslession.ISession) eslession.IEslApp {
	return &MyApp{
		session: s,
	}
}

func main() {
	client, err := goesl.NewClient("127.0.0.1", 8021, "ClueCon", 3)
	w := &EslWrapper{Client: client}

	if err != nil {
		goesl.Error("Error while creating new client: %s", err)
		return
	}

	goesl.Debug("Yuhu! New client: %q", client)
	go client.Handle()

	client.Send("events json CHANNEL_HANGUP CHANNEL_EXECUTE CHANNEL_EXECUTE_COMPLETE CHANNEL_PARK CHANNEL_DESTROY")
	eslession.EslConnectionHandler(w, h)
	fmt.Printf("Application exitted")
}
