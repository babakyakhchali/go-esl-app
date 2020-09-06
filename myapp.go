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
