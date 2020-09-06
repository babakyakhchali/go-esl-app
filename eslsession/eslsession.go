package eslession

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/0x19/goesl"
	"github.com/google/uuid"
)

var (
	sessions = map[string]*Session{}
)

//ISession is fs call interface
type ISession interface {
	Set(name string, value string) (IEvent, error)
	Answer() (IEvent, error)
	Hangup(cause ...string) (IEvent, error)
	Playback(path string) (IEvent, error)
	PlayAndGetDigits(min uint, max uint, tries uint, timeout uint,
		terminators string, file string, invalidFile string, varName string, regexp string, digitTimeout uint,
		transferOnFailure string) (IEvent, error)
	PlayAndGetOneDigit(path string) (uint64, error)
}

//Session main object to interact with a call
type Session struct {
	FsConnector
}

//Set sets a variable on managed channel
func (s *Session) Set(name string, value string) (IEvent, error) {
	return s.exec(name, value)
}

//Answer runs answer application on managed channel
func (s *Session) Answer() (IEvent, error) {
	return s.exec("answer", "")
}

//Hangup runs hangup application on managed channel
func (s *Session) Hangup(cause ...string) (IEvent, error) {
	c := "NORMAL_CLEARING"
	if len(cause) > 0 {
		c = cause[0]
	}
	return s.exec("hangup", c)
}

//Playback runs playback application on managed channel
func (s *Session) Playback(path string) (IEvent, error) {
	return s.exec("playback", path)
}

//PlayAndGetDigits runs play_and_get_digits application on managed channel
func (s *Session) PlayAndGetDigits(min uint, max uint, tries uint, timeout uint,
	terminators string, file string, invalidFile string, varName string, regexp string, digitTimeout uint,
	transferOnFailure string) (IEvent, error) {
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
	uuid   string
	cmds   chan map[string]string
	events chan IEvent
	errors chan error

	result      chan string
	resultError chan string

	closed bool
}

var (
	//EChannelHangup occurs when exec hits hangup
	EChannelHangup = "ChannelHangup"
)

//Application-UUID Event-UUID
func (fs *FsConnector) exec(app string, args string) (IEvent, error) {
	if fs.closed {
		return nil, fmt.Errorf(EChannelHangup)
	}
	headers := make(map[string]string)
	headers["call-command"] = "execute"
	headers["execute-app-name"] = app
	headers["execute-app-arg"] = args
	headers["Event-UUID"] = uuid.New().String()

	fs.cmds <- headers
	for {
		select {
		case event := <-fs.events:
			ename := event.GetHeader("Event-Name")
			euuid := event.GetHeader("Application-UUID")
			if ename == "CHANNEL_EXECUTE_COMPLETE" && euuid == headers["Event-UUID"] {
				return event, nil
			} else if ename == "CHANNEL_HANGUP" || ename == "CHANNEL_HANGUP_COMPLETE" {
				return nil, fmt.Errorf(EChannelHangup)
			}
		case err := <-fs.errors:
			return nil, err
		}
	}

}

//IEvent is fs event
type IEvent interface {
	GetHeader(name string) string
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
type AppFactory func(s ISession) IEslApp

func eslSessionHandler(msg *goesl.Message, esl *goesl.Client, f AppFactory) {
	s := Session{
		FsConnector: FsConnector{
			uuid:   msg.GetHeader("Unique-ID"),
			cmds:   make(chan map[string]string),
			events: make(chan IEvent),
			errors: make(chan error),
			closed: false,
		},
	}
	sessions[s.uuid] = &s
	app := f(&s)
	go app.Run()
	for {
		cmd, more := <-s.cmds
		if !more {
			break
		}
		esl.SendMsg(cmd, s.uuid, "")
	}
	goesl.Debug("session ended:%s", s.uuid)
}

//EslConnectionHandler handles incomming events
func EslConnectionHandler(client *goesl.Client, factory AppFactory) {
	client.Send("events json CHANNEL_HANGUP CHANNEL_EXECUTE CHANNEL_EXECUTE_COMPLETE CHANNEL_PARK CHANNEL_DESTROY")
	for {
		msg, err := client.ReadMessage()
		if err != nil {

			// If it contains EOF, we really dont care...
			if !strings.Contains(err.Error(), "EOF") && err.Error() != "unexpected end of JSON input" {
				goesl.Error("Error while reading Freeswitch message: %s", err)
				continue
			}
			for _, v := range sessions {
				v.errors <- err
			}
			break
		}
		eventName := msg.GetHeader("Event-Name")
		eventSubclass := msg.GetHeader("Event-Subclass")
		channelUUID := msg.GetHeader("Unique-ID")
		goesl.Debug("got event:%s(%s) uuid:%s", eventName, eventSubclass, channelUUID)
		if eventName == "CHANNEL_PARK" {
			go eslSessionHandler(msg, client, factory)
		} else if channelUUID != "" {
			s, r := sessions[channelUUID]
			if r {
				if eventName == "CHANNEL_DESTROY" {
					delete(sessions, channelUUID)
					fmt.Printf("deleted channel %s. remained channels:%d", channelUUID, len(sessions))
					continue
				}
				select {
				case s.events <- msg:
					fmt.Printf("handled event %s for channel %s", msg.GetHeader("Event-Name"), msg.GetHeader("Unique-ID"))
				default:
					fmt.Printf("ignoring event %s for channel %s", msg.GetHeader("Event-Name"), msg.GetHeader("Unique-ID"))
				}

			}
		}
		//goesl.Debug("%v", msg)
	}
	fmt.Printf("Application exitted")
}