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
	return event.GetHeader("variable_esl_manage") != ""
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
	//prettyPrint(event)
	r, e := app.session.ExecBgAPI("sched_hangup +7 " + app.data.GetHeader("Unique-ID") + " ALLOTTED_TIMEOUT")
	if e != nil {
		fmt.Printf("scheduling error:%s", e)
	} else {
		prettyPrint(r)
	}

}

//Run is called to control a channel like freeswitch xml extension does
func (app *MyApp) Run() {
	//app.session.AddEventHandler("CHANNEL_ANSWER", app.answerHandler)
	app.session.Answer()

	app.session.Set("hangup_after_bridge", "true")
	app.session.Set("continue_on_fail", "true")
	app.session.Set("call_timeout", "20")

	vars := map[string]string{
		"go_ession_var1": "false",
		"go_ession_var2": "true",
		"go_ession_var3": "20",
		"go_ession_var4": "hoooooa",
	}
	app.session.MultiSet(vars)
	app.session.Playback("ivr-asterisk_like_syphilis.wav")
	username := app.data.GetHeader("Caller-Destination-Number")
	r, err := app.session.Bridge("user/" + username + "@" + app.data.GetHeader("variable_domain_name"))
	if err != nil {
		fmt.Printf("bridge error:%s\n", err)
	} else if failCause := r.GetHeader("variable_originate_failed_cause"); failCause != "" {
		fmt.Printf("call failed with cause:%s\n", failCause)
		r, _ = app.session.Voicemail("default", "$${domain}", username)
	}
	//prettyPrint(r)
	app.session.Hangup("NORMAL_CLEARING")
	fmt.Printf("call end\n")
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
