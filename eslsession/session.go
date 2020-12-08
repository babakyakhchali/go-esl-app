package eslsession

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	fs "github.com/babakyakhchali/go-esl-wrapper/fs"
)

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
