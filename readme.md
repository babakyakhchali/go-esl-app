### About
A xml session emulator over esl for controlling freeswitch channels using something like dialplan.
For example:
``` golang

package main

import (
	"fmt"

	eslession "github.com/babakyakhchali/go-esl-wrapper/eslsession"
	goesl "github.com/babakyakhchali/go-esl-wrapper/goesl"
)

//MyApp will act as freeswitch extension xml which wraps an esl session
type MyApp struct {
	session fs.ISession
	data    fs.IEvent
}

//SetParkData called on recieveing intial park
func (app *MyApp) SetParkData(event fs.IEvent) {
	app.data = event
}

//IsApplicable decides based on channels variables weather the channel should be managed by this app or not
func (app *MyApp) IsApplicable(event fs.IEvent) bool {
	return event.GetHeader("variable_esl_manage") != ""
}

//Run is called to control a channel like freeswitch xml extension does
//every call to dialplan applications like Answer,Playback... returns an event or an error
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

func appFactory(s eslession.ISession) eslession.IEslApp {
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

	go client.Handle()

	client.Send("events json CHANNEL_HANGUP CHANNEL_EXECUTE CHANNEL_EXECUTE_COMPLETE CHANNEL_PARK CHANNEL_DESTROY")
	eslession.EslConnectionHandler(w, appFactory)	//creates instances of MyApp to controll channels
													//appFactory is used on each initial channel park event to create a session instance
	fmt.Printf("Application exitted")
}



```
### Notes
All codes in directory goesl are from https://github.com/0x19/goesl but modified to my needs