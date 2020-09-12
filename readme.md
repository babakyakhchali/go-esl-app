### About
A session emulator over esl for controlling freeswitch channels using something like dialplan.
For example:
``` golang
package main

import (
	"fmt"

	eslession "github.com/babakyakhchali/go-esl-wrapper/eslsession"
)

//MyApp simple test call handler application
type MyApp struct {
	session eslession.ISession
}

func (app *MyApp) Run() {
	app.session.Answer()
	app.session.Playback("conference\\8000\\conf-alone.wav")
	i, e := app.session.PlayAndGetOneDigit("phrase:demo_ivr_sub_menu")
	if e != nil {
		fmt.Printf("error is:%v\n", e)
	} else {
		fmt.Printf("input is:%d\n", i)
	}
	app.session.Hangup()
}

```

and

```golang
package main

import (
	"fmt"

	eslession "github.com/babakyakhchali/go-esl-wrapper/eslsession"
	goesl "github.com/babakyakhchali/go-esl-wrapper/goesl"
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

```
### Notes
All codes in directory goesl are from https://github.com/0x19/goesl but modified to my needs