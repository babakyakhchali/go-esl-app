package eslsession

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	fs "github.com/babakyakhchali/go-esl-wrapper/fs"
	l "github.com/babakyakhchali/go-esl-wrapper/logger"
	"github.com/google/uuid"
)

var (
	sessions      = map[string]*Session{}
	bgapi2Session = make(map[string]string) //relates background job events to sessions

	sessionLogger = l.NewLogger("eslsession")
	client        fs.IEsl
	bgApiJobs     = make(map[string]bgAPICtx)
)

//SetLogLevel set loglevel for eslsession logger
func SetLogLevel(l int) {
	sessionLogger.SetLevel(l)
}

var (
	//EChannelClosed occurs when exec is called on a channel which already is destroyed by hangup
	EChannelClosed = "ChannelHangup"
)

//SessionManager manages sessions
type SessionManager struct {
	sessions map[string]*Session
}

//IEslApp all call handler apps must implement this
type IEslApp interface {
	Run()
	IsApplicable(fs.IEvent) bool
	Setup(fs.IEvent)
}

//EslAppFactory signature for applications using this module
type EslAppFactory func(s fs.ISession) IEslApp

func eslSessionHandler(msg fs.IEvent, f EslAppFactory) {
	s := Session{
		FsConnector: FsConnector{
			uuid:          msg.GetHeader("Unique-ID"),
			cmds:          make(chan map[string]string),
			execError:     make(chan error),
			execEvent:     make(chan fs.IEvent),
			jobEvent:      make(chan fs.IEvent),
			jobError:      make(chan error),
			events:        make(chan fs.IEvent),
			errors:        make(chan error),
			closed:        false,
			EventHandlers: make(map[string]fs.EventHandlerFunc),
		},
	}
	s.logger = sessionLogger.CreateChild(msg.GetHeader("Unique-ID"))
	sessions[s.uuid] = &s
	app := f(&s)
	if !app.IsApplicable((msg)) {
		s.logger.Error("session not applicable:%s", s.uuid)
		return
	}
	app.Setup(msg)
	go s.dispatch()
	go app.Run()
	//TODO: clean this shiiiit
	for {

		cmd, more := <-s.cmds
		if !more {
			break
		}
		if bgapi, isapi := cmd["bgapi"]; isapi {
			bgapi2Session[cmd["Job-UUID"]] = s.uuid
			err := client.BgAPI(bgapi, cmd["Job-UUID"])
			if err != nil {
				s.jobError <- err
			}
		} else {
			err := client.SendMsg(cmd, s.uuid, "")
			if err != nil {
				s.execError <- err
			}
		}

	}
	s.logger.Info("session ended:%s", s.uuid)
}

type bgAPICtx struct {
	result        string
	errorChannel  chan error
	resultChannel chan string
	jobUUID       string
}

//BgAPI run an api using bgapi and wait for result
func BgAPI(api string, timout int) (string, error) {
	ctx := bgAPICtx{
		result:        "",
		errorChannel:  make(chan error, 1),
		resultChannel: make(chan string, 1),
		jobUUID:       uuid.New().String(),
	}
	to := make(chan bool, 1)
	go func() {
		time.Sleep(time.Duration(timout) * time.Second)
		to <- true
	}()

	defer delete(bgApiJobs, ctx.jobUUID)

	bgApiJobs[ctx.jobUUID] = ctx
	client.BgAPI(api, ctx.jobUUID)
	select {
	case r := <-ctx.resultChannel:
		return r, nil
	case err := <-ctx.errorChannel:
		return "", err
	case _ = <-to:
		return "", fmt.Errorf("timeout")
	}
}

//EslPropagateError sends error to waiting sessions or bgapi
func EslPropagateError(e error) {
	for _, v := range sessions {
		v.errors <- e
	}
	for _, v := range bgApiJobs {
		v.errorChannel <- e
	}
}

//EslConnectionHandler listens for channel events. On receiving a park event creates a Session and runs
//the app created by factory in a new go routine
func EslConnectionHandler(c fs.IEsl, factory EslAppFactory) error {
	client = c
	client.Send("events json HEARTBEAT CHANNEL_HANGUP CHANNEL_EXECUTE CHANNEL_EXECUTE_COMPLETE CHANNEL_PARK CHANNEL_DESTROY CHANNEL_ANSWER CHANNEL_BRIDGE CHANNEL_UNBRIDGE BACKGROUND_JOB")
	for {
		sessionLogger.Debug("Ready for event session:%d status: %d routines, %s", len(sessions), runtime.NumGoroutine(), getMemStats())
		msg, err := client.ReadMessage()
		if err != nil {
			sessionLogger.Error("Error %s", err)
			//TODO: handle reconnects, if reconnect succeeds may be channels can continue
			// If it contains EOF, we really dont care...
			if !strings.Contains(err.Error(), "EOF") && err.Error() != "unexpected end of JSON input" {
				sessionLogger.Error("Error while reading Freeswitch message: %s", err)
			}
			return err
		}
		eventName := msg.GetHeader("Event-Name")
		eventSubclass := msg.GetHeader("Event-Subclass")
		channelUUID := msg.GetHeader("Unique-ID")
		if eventName == "BACKGROUND_JOB" { //try to find session which created the job
			jobUUID := msg.GetHeader("Job-UUID")
			if jobSessionUUID, found := bgapi2Session[jobUUID]; found {
				channelUUID = jobSessionUUID
				delete(bgapi2Session, jobUUID) //job finished so remove it
			}
			if jobCTX, found := bgApiJobs[jobUUID]; found {
				jobCTX.resultChannel <- string(msg.GetBody())
			}
		}
		if msg.GetType() != "text/event-json" {
			sessionLogger.Debug("got %s: reply:%s body:%s ", msg.GetType(), msg.GetHeader("Reply-Text"), msg.GetBody())
		} else {
			sessionLogger.Debug("got event:%s(%s) uuid:%s", eventName, eventSubclass, channelUUID)
		}

		if eventName == "CHANNEL_PARK" {
			if _, isAlreadyHandled := sessions[channelUUID]; isAlreadyHandled == false {
				go eslSessionHandler(msg, factory)
				continue
			}
		}
		if eventName == "HEARTBEAT" {
			sessionLogger.Debug("HEARTBEAT: cps:%s , %s", msg.GetHeader("Session-Per-Sec"), msg.GetHeader("Up-Time"))
		} else if channelUUID != "" {
			s, r := sessions[channelUUID]
			if r {
				select {
				case s.events <- msg:
					sessionLogger.Debug("handled event %s for channel %s", msg.GetHeader("Event-Name"), msg.GetHeader("Unique-ID"))
				default:
					sessionLogger.Debug("ignoring event %s for channel %s", msg.GetHeader("Event-Name"), msg.GetHeader("Unique-ID"))
				}
				if eventName == "CHANNEL_DESTROY" {
					delete(sessions, channelUUID)
					sessionLogger.Debug("deleted channel %s. remained channels:%d", channelUUID, len(sessions))
				}
			}
		}
	}

	/*uncomment following lines to debug active goroutines on exit
	sessionLogger.Debug("all routines stackl %s", dumpAllRoutines())
	*/
}
