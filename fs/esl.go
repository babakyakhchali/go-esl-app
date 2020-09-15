package fs

//IEvent is fs event
type IEvent interface {
	GetHeader(name string) string
	GetBody() []byte
}

//IEsl common interface for freeswitch esl
type IEsl interface {
	Send(cmd string) error
	SendMsg(cmd map[string]string, uuid string, data string) (IEvent, error)
	ReadMessage() (IEvent, error)
}

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
