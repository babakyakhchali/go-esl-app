package main

//MyApp simple test call handler application
type MyApp struct {
	session ISession
}

func (app *MyApp) run() {
	app.session.answer()
	app.session.playback("conference\\8000\\conf-alone.wav")
	app.session.hangup()
}
