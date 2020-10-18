package adapters

import (
	fs "github.com/babakyakhchali/go-esl-wrapper/fs"
	goesl "github.com/babakyakhchali/go-esl-wrapper/goesl"
)

//EslWrapper wrapper goesl Client to abstract goesl out of eslsession
type EslWrapper struct {
	*goesl.Client
}

//ReadEvent wrapper
func (c *EslWrapper) ReadEvent() (fs.IEvent, error) {
	msg, err := c.Client.ReadEvent()
	return &MessageWrapper{Message: msg}, err
}

//MessageWrapper wrapper around goesl message
type MessageWrapper struct {
	*goesl.Message
}

//GetBody return event body if present
func (m *MessageWrapper) GetBody() []byte {
	return m.Message.Body
}

//SendMsg wrapper
func (c *EslWrapper) SendMsg(cmd map[string]string, uuid string, data string) (fs.IEvent, error) {
	msg, err := c.Client.SendMsg(cmd, uuid, data)
	return &MessageWrapper{Message: msg}, err
}

//Send wrapper
func (c *EslWrapper) Send(cmd string) (fs.IEvent, error) {
	msg, err := c.Client.Send(cmd)
	return &MessageWrapper{Message: msg}, err
}
