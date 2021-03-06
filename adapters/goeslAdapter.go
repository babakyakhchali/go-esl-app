package adapters

import (
	fs "github.com/babakyakhchali/go-esl-wrapper/fs"
	goesl "github.com/babakyakhchali/go-esl-wrapper/goesl"
)

//EslWrapper wrapper goesl Client to abstract goesl out of eslsession
type EslWrapper struct {
	*goesl.Client
}

//ReadMessage wrapper
func (c *EslWrapper) ReadMessage() (fs.IEvent, error) {
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
