package eslsession

import (
	"runtime"
	"strings"

	fs "github.com/babakyakhchali/go-esl-wrapper/fs"
	l "github.com/babakyakhchali/go-esl-wrapper/logger"
)

var (
	sessions      = map[string]*Session{}
	jobs          = make(map[string]string) //relates background job events to sessions
	sessionLogger = l.NewLogger("eslsession")
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
	SetParkData(fs.IEvent)
}

//EslAppFactory signature for applications using this module
type EslAppFactory func(s fs.ISession) IEslApp

func eslSessionHandler(msg fs.IEvent, esl fs.IEsl, f EslAppFactory) {
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
	app.SetParkData(msg)
	go s.dispatch()
	go app.Run()
	//TODO: clean this shiiiit
	for {

		cmd, more := <-s.cmds
		if !more {
			break
		}
		if bgapi, isapi := cmd["bgapi"]; isapi {
			jobs[cmd["Job-UUID"]] = s.uuid
			err := esl.BgAPI(bgapi, cmd["Job-UUID"])
			if err != nil {
				s.jobError <- err
			}
		} else {
			err := esl.SendMsg(cmd, s.uuid, "")
			if err != nil {
				s.execError <- err
			}
		}

	}
	s.logger.Info("session ended:%s", s.uuid)
}

//EslConnectionHandler listens for channel events. On receiving a park event creates a Session and runs
//the app created by factory in a new go routine
func EslConnectionHandler(client fs.IEsl, factory EslAppFactory) {

	client.Send("events json HEARTBEAT CHANNEL_HANGUP CHANNEL_EXECUTE CHANNEL_EXECUTE_COMPLETE CHANNEL_PARK CHANNEL_DESTROY CHANNEL_ANSWER CHANNEL_BRIDGE CHANNEL_UNBRIDGE BACKGROUND_JOB")
	for {
		sessionLogger.Debug("Ready for event status: %d routines, %s", runtime.NumGoroutine(), getMemStats())
		msg, err := client.ReadMessage()
		if err != nil {
			sessionLogger.Error("Error %s", err)
			// If it contains EOF, we really dont care...
			if !strings.Contains(err.Error(), "EOF") && err.Error() != "unexpected end of JSON input" {
				sessionLogger.Error("Error while reading Freeswitch message: %s", err)
			}
			for _, v := range sessions {
				v.errors <- err
			}
			break
		}
		eventName := msg.GetHeader("Event-Name")
		eventSubclass := msg.GetHeader("Event-Subclass")
		channelUUID := msg.GetHeader("Unique-ID")
		if eventName == "BACKGROUND_JOB" { //try to find session which created the job
			jobUUID := msg.GetHeader("Job-UUID")
			if jobSessionUUID, found := jobs[jobUUID]; found {
				channelUUID = jobSessionUUID
				delete(jobs, jobUUID) //job finished so remove it
			}
		}
		if msg.GetType() != "text/event-json" {
			sessionLogger.Debug("got %s: reply:%s body:%s ", msg.GetType(), msg.GetHeader("Reply-Text"), msg.GetBody())
		} else {
			sessionLogger.Debug("got event:%s(%s) uuid:%s", eventName, eventSubclass, channelUUID)
		}

		if eventName == "CHANNEL_PARK" {
			if _, isAlreadyHandled := sessions[channelUUID]; isAlreadyHandled == false {
				go eslSessionHandler(msg, client, factory)
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
	sessionLogger.Info("Application exitted")
}
