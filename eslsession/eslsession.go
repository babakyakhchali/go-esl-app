package eslsession

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	fs "github.com/babakyakhchali/go-esl-wrapper/fs"
	l "github.com/babakyakhchali/go-esl-wrapper/logger"
	"github.com/google/uuid"
)

var (
	sessions = map[string]*Session{}
	logger   = l.NewLogger("eslsession")
)

//Session main object to interact with a call
type Session struct {
	FsConnector
}

//Set sets a variable on managed channel
func (s *Session) Set(name string, value string) (fs.IEvent, error) {
	return s.exec(name, value)
}

//Answer runs answer application on managed channel
func (s *Session) Answer() (fs.IEvent, error) {
	return s.exec("answer", "")
}

//Hangup runs hangup application on managed channel
func (s *Session) Hangup(cause ...string) (fs.IEvent, error) {
	c := "NORMAL_CLEARING"
	if len(cause) > 0 {
		c = cause[0]
	}
	return s.exec("hangup", c)
}

//Playback runs playback application on managed channel
func (s *Session) Playback(path string) (fs.IEvent, error) {
	return s.exec("playback", path)
}

//PlayAndGetDigits runs play_and_get_digits application on managed channel
func (s *Session) PlayAndGetDigits(min uint, max uint, tries uint, timeout uint,
	terminators string, file string, invalidFile string, varName string, regexp string, digitTimeout uint,
	transferOnFailure string) (fs.IEvent, error) {
	args := fmt.Sprintf("%d %d %d %d %s %s %s %s %s %d %s",
		min, max, tries, timeout, terminators, file, invalidFile, varName, regexp, digitTimeout, transferOnFailure)
	return s.exec("play_and_get_digits", strings.TrimSpace(args))
}

//PlayAndGetOneDigit a wrapper around freeswitch play_and_get_digits to get just one digit
func (s *Session) PlayAndGetOneDigit(path string) (uint64, error) {
	varname := "pagd-" + strconv.FormatInt(time.Now().Unix(), 10)
	r, e := s.PlayAndGetDigits(1, 1, 3, 5000, "#", path, "''", varname, "\\d", 5000, "''")
	if e != nil {
		return 0, e
	}
	return strconv.ParseUint(r.GetHeader("variable_"+varname), 10, 32)
}

//FsConnector acts as a channel between fs and session
type FsConnector struct {
	uuid           string
	cmds           chan map[string]string
	events         chan fs.IEvent
	appEvent       chan fs.IEvent
	appError       chan error
	errors         chan error
	currentAppUUID string
	closed         bool
}

var (
	//EChannelHangup occurs when exec hits hangup
	EChannelHangup = "ChannelHangup"
)

func (fs *FsConnector) dispatch() {
	for {
		select {
		case event := <-fs.events:
			ename := event.GetHeader("Event-Name")
			logger.Debug("dispatch(): got event %s:%s", ename, fs.uuid)
			euuid := event.GetHeader("Application-UUID")
			if ename == "CHANNEL_EXECUTE_COMPLETE" && euuid == fs.currentAppUUID {
				fs.appEvent <- event
			}
		case err := <-fs.errors:
			fs.appError <- err
		}
	}

}

//Application-UUID Event-UUID
func (fs *FsConnector) exec(app string, args string) (fs.IEvent, error) {
	if fs.closed {
		return nil, fmt.Errorf(EChannelHangup)
	}
	headers := make(map[string]string)
	headers["call-command"] = "execute"
	headers["execute-app-name"] = app
	headers["execute-app-arg"] = args
	headers["Event-UUID"] = uuid.New().String()
	fs.currentAppUUID = headers["Event-UUID"]
	fs.cmds <- headers

	select {
	case event := <-fs.appEvent:
		return event, nil
	case err := <-fs.appError:
		return nil, err
	}
}

//SessionManager manages sessions
type SessionManager struct {
	sessions map[string]*Session
}

//IEslApp all call handler apps must implement this
type IEslApp interface {
	Run()
}

//AppFactory signature for applications using this module
type AppFactory func(s fs.ISession) IEslApp

func eslSessionHandler(msg fs.IEvent, esl fs.IEsl, f AppFactory) {
	s := Session{
		FsConnector: FsConnector{
			uuid:     msg.GetHeader("Unique-ID"),
			cmds:     make(chan map[string]string),
			appError: make(chan error),
			appEvent: make(chan fs.IEvent),
			events:   make(chan fs.IEvent),
			errors:   make(chan error),
			closed:   false,
		},
	}
	sessions[s.uuid] = &s
	app := f(&s)
	go s.dispatch()
	go app.Run()
	for {
		cmd, more := <-s.cmds
		if !more {
			break
		}
		esl.SendMsg(cmd, s.uuid, "")
	}
	fmt.Printf("session ended:%s", s.uuid)
}

//EslConnectionHandler listens for channel events. On receiving a park event creates a Session and runs
//the app created by factory in a new go routine
func EslConnectionHandler(client fs.IEsl, factory AppFactory) {
	client.Send("events json CHANNEL_HANGUP CHANNEL_EXECUTE CHANNEL_EXECUTE_COMPLETE CHANNEL_PARK CHANNEL_DESTROY")
	for {
		msg, err := client.ReadMessage()
		if err != nil {

			// If it contains EOF, we really dont care...
			if !strings.Contains(err.Error(), "EOF") && err.Error() != "unexpected end of JSON input" {
				logger.Error("Error while reading Freeswitch message: %s", err)
			}
			for _, v := range sessions {
				v.errors <- err
			}
			break
		}
		eventName := msg.GetHeader("Event-Name")
		eventSubclass := msg.GetHeader("Event-Subclass")
		channelUUID := msg.GetHeader("Unique-ID")
		logger.Debug("got event:%s(%s) uuid:%s", eventName, eventSubclass, channelUUID)
		if eventName == "CHANNEL_PARK" {
			go eslSessionHandler(msg, client, factory)
		} else if channelUUID != "" {
			s, r := sessions[channelUUID]
			if r {
				if eventName == "CHANNEL_DESTROY" {
					delete(sessions, channelUUID)
					logger.Debug("deleted channel %s. remained channels:%d", channelUUID, len(sessions))
					continue
				}
				select {
				case s.events <- msg:
					logger.Debug("handled event %s for channel %s", msg.GetHeader("Event-Name"), msg.GetHeader("Unique-ID"))
				default:
					logger.Debug("ignoring event %s for channel %s", msg.GetHeader("Event-Name"), msg.GetHeader("Unique-ID"))
				}

			}
		}
		//goesl.Debug("%v", msg)
	}
	logger.Info("Application exitted")
}
