package main

import (
	"github.com/0x19/goesl"
	eslession "github.com/babakyakhchali/go-esl-wrapper/eslsession"
)

//EslWrapper wrapper goesl Client to abstract goesl out of eslsession
type EslWrapper struct {
	*goesl.Client
}

//ReadMessage wrapper
func (c *EslWrapper) ReadMessage() (eslession.IEvent, error) {
	msg, err := c.Client.ReadMessage()
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
func (c *EslWrapper) SendMsg(cmd map[string]string, uuid string, data string) (eslession.IEvent, error) {
	msg, err := c.Client.SendMsg(cmd, uuid, data)
	return &MessageWrapper{Message: msg}, err
}
