package main

import (
	"encoding/json"
	"fmt"

	adapters "github.com/babakyakhchali/go-esl-wrapper/adapters"
	eslession "github.com/babakyakhchali/go-esl-wrapper/eslsession"
	fs "github.com/babakyakhchali/go-esl-wrapper/fs"
	goesl "github.com/babakyakhchali/go-esl-wrapper/goesl"
	l "github.com/babakyakhchali/go-esl-wrapper/logger"
)

var (
	appLogger = l.NewLogger("main")
)

//MyApp will act as freeswitch extension xml which wraps an esl session
type MyApp struct {
	session fs.ISession
	data    fs.IEvent
}

func (app *MyApp) SetParkData(event fs.IEvent) {
	app.data = event
}

func (app *MyApp) IsApplicable(event fs.IEvent) bool {
	return event.GetHeader("variable_chakavak_manage") != ""
}

//Run2 is called to control a channel like freeswitch xml extension does
func (app *MyApp) Run2() {
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
func prettyPrint(o interface{}) {
	b, _ := json.MarshalIndent(o, "", "  ")

	fmt.Print(string(b))
}

func (app *MyApp) answerHandler(event fs.IEvent) {
	fmt.Printf("Gooot answer!!! %s", app.data.GetHeader("Unique-ID"))
	prettyPrint(event)
}

//Run is called to control a channel like freeswitch xml extension does
func (app *MyApp) Run() {
	app.session.AddEventHandler("CHANNEL_ANSWER", app.answerHandler)
	app.session.PreAnswer()
	/*
			esl_session.setvar("hangup_after_bridge", "true")
		    esl_session.setvar("continue_on_fail", "true")
		    esl_session.setvar("call_timeout", "20")
		    esl_session.setvar("effective_caller_id_number", route_caller_id)*/
	// vars := map[string]string{
	// 	"hangup_after_bridge":        "false",
	// 	"continue_on_fail":           "true",
	// 	"call_timeout":               "20",
	// 	"effective_caller_id_number": "hoooooa",
	// }
	// app.session.MultiSet(vars)
	username := app.data.GetHeader("Caller-Destination-Number")
	r, _ := app.session.Bridge("user/" + username)
	if failCause := r.GetHeader("variable_originate_failed_cause"); failCause != "" {
		fmt.Printf("call failed with cause:%s", failCause)
		r, _ = app.session.Voicemail("default", "$${domain}", username)
	}
	//prettyPrint(r)
	app.session.Hangup("NORMAL_CLEARING")
}

func appFactory(s fs.ISession) eslession.IEslApp {
	return &MyApp{
		session: s,
	}
}

func main() {
	goesl.SetLogLevel(l.ERROR)
	client, err := goesl.NewClient("127.0.0.1", 8021, "ClueCon", 3)
	w := &adapters.EslWrapper{Client: client}

	if err != nil {
		appLogger.Error("Error while creating new client: %s", err)
		return
	}

	go client.Handle()

	//client.Send("events json CHANNEL_HANGUP CHANNEL_EXECUTE CHANNEL_EXECUTE_COMPLETE CHANNEL_PARK CHANNEL_DESTROY")
	eslession.EslConnectionHandler(w, appFactory)
	appLogger.Info("Application exitted")
}
