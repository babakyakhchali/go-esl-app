package main

import "fmt"

//MyApp simple test call handler application
type MyApp struct {
	session ISession
}

func (app *MyApp) run() {
	app.session.answer()
	app.session.playback("conference\\8000\\conf-alone.wav")
	i, e := app.session.playAndGetOneDigit("phrase:demo_ivr_main_menu")
	if e != nil {
		fmt.Printf("error is:%v\n", e)
	} else {
		fmt.Printf("input is:%d\n", i)
	}
	app.session.hangup()
}
