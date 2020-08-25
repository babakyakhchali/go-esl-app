package main

import (
	"fmt"

	"github.com/0x19/goesl"
	"github.com/google/uuid"
)

//ISession is fs call interface
type ISession interface {
	set(name string, value string) (IEvent, error)
	answer() (IEvent, error)
	hangup(cause ...string) (IEvent, error)
	playback(path string) (IEvent, error)
}

//Session main object to interact with a call
type Session struct {
	FsConnector
}

func (s *Session) set(name string, value string) (IEvent, error) {
	return s.exec(name, value)
}

func (s *Session) answer() (IEvent, error) {
	return s.exec("answer", "")
}
func (s *Session) hangup(cause ...string) (IEvent, error) {
	c := "NORMAL_CLEARING"
	if len(cause) > 0 {
		c = cause[0]
	}
	return s.exec("hangup", c)
}
func (s *Session) playback(path string) (IEvent, error) {
	return s.exec("playback", path)
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
	run()
}

func eslSessionHandler(msg *goesl.Message, esl *goesl.Client) {
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
	app := MyApp{
		session: &s,
	}
	go app.run()
	for {
		cmd, more := <-s.cmds
		if !more {
			break
		}
		esl.SendMsg(cmd, s.uuid, "")
	}
	goesl.Debug("session ended:%s", s.uuid)
}
