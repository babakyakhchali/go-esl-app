package fs

//IEvent is fs event
type IEvent interface {
	GetHeader(name string) string
	GetBody() []byte
	GetType() string
}

//IEsl common interface for freeswitch esl
type IEsl interface {
	Send(cmd string) error
	SendMsg(cmd map[string]string, uuid string, data string) error
	BgAPI(cmd string, uuid string) error
	ReadMessage() (IEvent, error)
}

// EventHandlerFunc a function to receive an event
type EventHandlerFunc func(IEvent)

//ISession is fs call interface
type ISession interface {
	Set(name string, value string) (IEvent, error)
	Unset(name string) (IEvent, error)
	MultiSet(variables map[string]string) (IEvent, error)
	MultiUnset(variables map[string]string) (IEvent, error)
	Answer() (IEvent, error)
	PreAnswer() (IEvent, error)
	Hangup(cause ...string) (IEvent, error)
	Playback(path string) (IEvent, error)
	PlayAndGetDigits(min uint, max uint, tries uint, timeout uint,
		terminators string, file string, invalidFile string, varName string, regexp string, digitTimeout uint,
		transferOnFailure string) (IEvent, error)
	PlayAndGetOneDigit(path string) (uint64, error)
	Bridge(bstr string) (IEvent, error)
	Voicemail(settingsProfile string, domain string, username string) (IEvent, error)
	//SendEvent fires event using channel execute
	SendEvent(headers map[string]string) (IEvent, error)

	ExecBgAPI(cmd string) (IEvent, error)
	ExecAPI(cmd string) error
	AddEventHandler(eventName string, handler EventHandlerFunc)
}
