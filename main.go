package main

import (
	"encoding/json"
	"fmt"
	"time"

	adapters "github.com/babakyakhchali/go-esl-wrapper/adapters"
	eslession "github.com/babakyakhchali/go-esl-wrapper/eslsession"
	fs "github.com/babakyakhchali/go-esl-wrapper/fs"
	goesl "github.com/babakyakhchali/go-esl-wrapper/goesl"
	l "github.com/babakyakhchali/go-esl-wrapper/logger"
)

var (
	appLogger = l.NewLogger("main")
)

func prettyPrint(o interface{}) {
	b, _ := json.MarshalIndent(o, "", "  ")

	fmt.Print(string(b))
}

func appFactory(s fs.ISession) eslession.IEslApp {
	return &MyApp{
		session: s,
	}
}
func testBgAPI() {
	t := time.NewTicker(5 * time.Second)
	apistr := "show calls"
	go func() {
		for range t.C {
			r, e := eslession.BgAPI(apistr, 3)
			if e != nil {
				appLogger.Error("Error issuing bgapi: %s", e)
			} else {
				appLogger.Info("%s: %s", apistr, r)
			}
		}
	}()

}
func main() {
	goesl.SetLogLevel(l.ERROR)

	for i := 0; i < 600; i++ {
		client, err := goesl.NewClient("127.0.0.1", 8021, "ClueCon", 3)
		w := &adapters.EslWrapper{Client: client}

		if err != nil {
			appLogger.Error("Error while creating new client: %s", err)
			break
		}

		go client.Handle()

		go testBgAPI()

		//client.Send("events json CHANNEL_HANGUP CHANNEL_EXECUTE CHANNEL_EXECUTE_COMPLETE CHANNEL_PARK CHANNEL_DESTROY")
		eslession.EslConnectionHandler(w, appFactory)

		appLogger.Info("Socket closed retrying %d", i)
		time.Sleep(10 * time.Millisecond)
	}
	appLogger.Info("App exitted")
}
