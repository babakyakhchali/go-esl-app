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
	sessions      = map[string]*Session{}
	jobs          = make(map[string]string) //relates background job events to sessions
	sessionLogger = l.NewLogger("eslsession")
)

//SetLogLevel set loglevel for eslsession logger
func SetLogLevel(l int) {
	sessionLogger.SetLevel(l)
}

//Session main object to interact with a call
type Session struct {
	FsConnector
}

//Set sets a variable on managed channel
func (s *Session) Set(name string, value string) (fs.IEvent, error) {
	return s.exec("set", name+"="+value)
}

//Unset sets a variable on managed channel
func (s *Session) Unset(name string) (fs.IEvent, error) {
	return s.exec("unseset", name)
}

//MultiSet sets multiple variable on managed channel
func (s *Session) MultiSet(vars map[string]string) (fs.IEvent, error) {
	c := "^^:"
	for k, v := range vars {
		c += k + "=" + v + ":"
	}
	return s.exec("multiset", c)
}

//MultiUnset sets multiple variable on managed channel
func (s *Session) MultiUnset(vars map[string]string) (fs.IEvent, error) {
	c := "^^:"
	for k, v := range vars {
		c += k + "=" + v + ":"
	}
	return s.exec("multiunset", c)
}

//Answer runs answer application on managed channel
func (s *Session) Answer() (fs.IEvent, error) {
	return s.exec("answer", "")
}

//PreAnswer runs pre_answer application on managed channel
func (s *Session) PreAnswer() (fs.IEvent, error) {
	return s.exec("pre_answer", "")
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

//Bridge runs bridge application on managed channel
func (s *Session) Bridge(bstr string) (fs.IEvent, error) {
	return s.exec("bridge", bstr)
}

//Voicemail runs voicemail application on managed channel
func (s *Session) Voicemail(settingsProfile string, domain string, username string) (fs.IEvent, error) {
	return s.exec("voicemail", fmt.Sprintf("%s %s %s", settingsProfile, domain, username))
}

//SendEvent runs event application on managed channel
func (s *Session) SendEvent(headers map[string]string) (fs.IEvent, error) {
	data := ""
	i := 0
	for k, v := range headers {
		data += k + "=" + v
		i++
		if i < len(headers) {
			data += ","
		}
	}
	return s.exec("event", data)
	//<action application="event" data="Event-Subclass=VoiceWorks.pl::ACDnotify,Event-Name=CUSTOM,state=Intro,condition=IntroPlayed"/>
}

//ExecAPI exectue freeswitch apis in blocking mode
func (s *Session) ExecAPI(cmd string) error {
	return nil
}

//ExecBgAPI exectue freeswitch apis in non blocking mode
func (s *Session) ExecBgAPI(cmd string) (fs.IEvent, error) {
	return s.bgapi(cmd)
}

//AddEventHandler used to set handlers for different events by event name
func (s *Session) AddEventHandler(eventName string, handler fs.EventHandlerFunc) {
	s.EventHandlers[eventName] = handler
}

//FsConnector acts as a channel between fs and session
type FsConnector struct {
	uuid string
	//used to send api and execute to freeswitch
	cmds chan map[string]string
	/*used to recieve events by session dispatcher.
	this will receive both exec result events and other channel events by dispatcher*/
	events chan fs.IEvent
	//receives errors from fs connection
	errors chan error

	/*used by dispatcher to notify the exec() when execution completes*/
	execEvent chan fs.IEvent
	execError chan error

	jobEvent       chan fs.IEvent
	jobError       chan error
	currentAppUUID string
	closed         bool
	logger         *l.NsLogger
	EventHandlers  map[string]fs.EventHandlerFunc

	currentJobUUID string
}

var (
	//EChannelClosed occurs when exec is called on a channel which already is destroyed by hangup
	EChannelClosed = "ChannelHangup"
)

//sits between event channel and session and receives all events and replies for the session
func (fs *FsConnector) dispatch() {
	for {
		select {
		case event := <-fs.events:
			ename := event.GetHeader("Event-Name")
			fs.logger.Debug("dispatch(): got event %s:%s", ename, fs.uuid)
			euuid := event.GetHeader("Application-UUID")
			if ename == "CHANNEL_EXECUTE_COMPLETE" && euuid == fs.currentAppUUID {
				select { //this must be nonblocking
				case fs.execEvent <- event:
				default:
				}
			}
			if ename == "CHANNEL_DESTROY" {
				fs.closed = true
				select { //this must be nonblocking
				case fs.execError <- fmt.Errorf(EChannelClosed):
				default:
				}
			}
			if h, e := fs.EventHandlers[ename]; e {
				go h(event)
			}
		case err := <-fs.errors: //inform blocked execs and bgapis
			select {
			case fs.execError <- err:
			default:
			}
			select {
			case fs.jobError <- err:
			default:
			}

		}
	}

}

//Application-UUID Event-UUID
//
//this method handles complex logic because of the event based nature of the module
//channel may be in 3 states when this method is called on session:
//
// * already hangged up
// * in the middle of hangup
// * up and running
func (fs *FsConnector) exec(app string, args string) (fs.IEvent, error) {
	if fs.closed {
		return nil, fmt.Errorf(EChannelClosed)
	}
	headers := make(map[string]string)
	headers["call-command"] = "execute"
	headers["execute-app-name"] = app
	headers["execute-app-arg"] = args
	headers["Event-UUID"] = uuid.New().String()
	fs.currentAppUUID = headers["Event-UUID"]

	defer func() {
		fs.currentAppUUID = ""
	}()

	fs.cmds <- headers

	select {
	case event := <-fs.execEvent:
		return event, nil
	case err := <-fs.execError:
		fs.logger.Debug("exec(%s,%s)(%s) error: %s", app, args, fs.currentAppUUID, err)
		return nil, err
	}
}

func (fs *FsConnector) bgapi(cmd string) (fs.IEvent, error) {
	if fs.closed {
		return nil, fmt.Errorf(EChannelClosed)
	}
	headers := make(map[string]string)
	headers["bgapi"] = cmd
	headers["Job-UUID"] = uuid.New().String()
	fs.currentJobUUID = headers["Job-UUID"]

	defer func() {
		fs.currentJobUUID = ""
	}()

	fs.cmds <- headers

	select {
	case event := <-fs.jobEvent:
		fs.logger.Debug("bgapi(%s) => %s", cmd, event.GetBody())
		return event, nil
	case err := <-fs.jobError:
		fs.logger.Debug("bgapi(%s) error: %s", cmd, err)
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
	IsApplicable(fs.IEvent) bool
	SetParkData(fs.IEvent)
}

//AppFactory signature for applications using this module
type AppFactory func(s fs.ISession) IEslApp

func eslSessionHandler(msg fs.IEvent, esl fs.IEsl, f AppFactory) {
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
func EslConnectionHandler(client fs.IEsl, factory AppFactory) {

	client.Send("events json CHANNEL_HANGUP CHANNEL_EXECUTE CHANNEL_EXECUTE_COMPLETE CHANNEL_PARK CHANNEL_DESTROY CHANNEL_ANSWER CHANNEL_BRIDGE CHANNEL_UNBRIDGE BACKGROUND_JOB")
	for {
		sessionLogger.Debug("Ready for event")
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
			sessionLogger.Debug("got %s: %s", msg.GetType(), msg.GetBody())
		} else {
			sessionLogger.Debug("got event:%s(%s) uuid:%s", eventName, eventSubclass, channelUUID)
		}

		if eventName == "CHANNEL_PARK" {
			go eslSessionHandler(msg, client, factory)
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
		//goesl.Debug("%v", msg)
	}
	sessionLogger.Info("Application exitted")
}
