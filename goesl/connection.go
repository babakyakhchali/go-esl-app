// Copyright 2015 Nevio Vesic
// Please check out LICENSE file for more information about what you CAN and what you CANNOT do!
// Basically in short this is a free software for you to do whatever you want to do BUT copyright must be included!
// I didn't write all of this code so you could say it's yours.
// MIT License

package goesl

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	l "github.com/babakyakhchali/go-esl-wrapper/logger"
)

var (
	connectionLogger = l.NewLogger("connection")
)

// SocketConnection Main connection against ESL - Gotta add more description here
type SocketConnection struct {
	net.Conn
	//err chan error
	//m       chan *Message
	events     chan *Message
	eventError chan error
	replies    chan *Message
	replyError chan error
	mtx        sync.Mutex
}

// Dial - Will establish timedout dial against specified address. In this case, it will be freeswitch server
func (c *SocketConnection) Dial(network string, addr string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout(network, addr, timeout)
}

// Send - Will send raw message to open net connection
func (c *SocketConnection) Send(cmd string) (*Message, error) {

	if strings.Contains(cmd, "\r\n") {
		return nil, fmt.Errorf(EInvalidCommandProvided, cmd)
	}

	// lock mutex
	c.mtx.Lock()
	defer c.mtx.Unlock()

	_, err := io.WriteString(c, cmd)
	if err != nil {
		return nil, err
	}

	_, err = io.WriteString(c, "\r\n\r\n")
	if err != nil {
		return nil, err
	}

	select {
	case err := <-c.replyError:
		return nil, err
	case m := <-c.replies:
		return m, nil
	}

}

// SendMany - Will loop against passed commands and return 1st error if error happens
func (c *SocketConnection) SendMany(cmds []string) (*Message, error) {
	var msg *Message
	for _, cmd := range cmds {
		if msg, err := c.Send(cmd); err != nil {
			return msg, err
		}
	}

	return msg, nil
}

// SendEvent - Will loop against passed event headers
func (c *SocketConnection) SendEvent(eventHeaders []string) (*Message, error) {
	if len(eventHeaders) <= 0 {
		return nil, fmt.Errorf(ECouldNotSendEvent, len(eventHeaders))
	}

	// lock mutex to prevent event headers from conflicting
	c.mtx.Lock()
	defer c.mtx.Unlock()

	_, err := io.WriteString(c, "sendevent ")
	if err != nil {
		return nil, err
	}

	for _, eventHeader := range eventHeaders {
		_, err := io.WriteString(c, eventHeader)
		if err != nil {
			return nil, err
		}

		_, err = io.WriteString(c, "\r\n")
		if err != nil {
			return nil, err
		}

	}

	_, err = io.WriteString(c, "\r\n")
	if err != nil {
		return nil, err
	}

	select {
	case err := <-c.replyError:
		return nil, err
	case m := <-c.replies:
		return m, nil
	}
}

// Execute - Helper fuck to execute commands with its args and sync/async mode
func (c *SocketConnection) Execute(command, args string, sync bool) (m *Message, err error) {
	return c.SendMsg(map[string]string{
		"call-command":     "execute",
		"execute-app-name": command,
		"execute-app-arg":  args,
		"event-lock":       strconv.FormatBool(sync),
	}, "", "")
}

// ExecuteUUID - Helper fuck to execute uuid specific commands with its args and sync/async mode
func (c *SocketConnection) ExecuteUUID(uuid string, command string, args string, sync bool) (m *Message, err error) {
	return c.SendMsg(map[string]string{
		"call-command":     "execute",
		"execute-app-name": command,
		"execute-app-arg":  args,
		"event-lock":       strconv.FormatBool(sync),
	}, uuid, "")
}

// SendMsg - Basically this func will send message to the opened connection
func (c *SocketConnection) SendMsg(msg map[string]string, uuid, data string) (m *Message, err error) {
	connectionLogger.Debug("SendMsg %s", uuid)
	b := bytes.NewBufferString("sendmsg")

	if uuid != "" {
		if strings.Contains(uuid, "\r\n") {
			return nil, fmt.Errorf(EInvalidCommandProvided, msg)
		}

		b.WriteString(" " + uuid)
	}

	b.WriteString("\n")

	for k, v := range msg {
		if strings.Contains(k, "\r\n") {
			return nil, fmt.Errorf(EInvalidCommandProvided, msg)
		}

		if v != "" {
			if strings.Contains(v, "\r\n") {
				return nil, fmt.Errorf(EInvalidCommandProvided, msg)
			}

			b.WriteString(fmt.Sprintf("%s: %s\n", k, v))
		}
	}

	b.WriteString("\n")

	if msg["content-length"] != "" && data != "" {
		b.WriteString(data)
	}

	// lock mutex
	c.mtx.Lock()
	_, err = b.WriteTo(c)
	if err != nil {
		c.mtx.Unlock()
		return nil, err
	}
	c.mtx.Unlock()

	select {
	case err := <-c.replyError:
		connectionLogger.Debug("SendMsg %s error %s", uuid, err)
		return nil, err
	case m := <-c.replies:
		connectionLogger.Debug("SendMsg %s result %v", uuid, m)
		return m, nil
	}
}

// OriginatorAddr - Will return originator address known as net.RemoteAddr()
// This will actually be a freeswitch address
func (c *SocketConnection) OriginatorAddr() net.Addr {
	return c.RemoteAddr()
}

// ReadEvent - Will read event message from channels and return them back accordingy.
// If error is received, error will be returned. If not, message will be returned back!
func (c *SocketConnection) ReadEvent() (*Message, error) {
	connectionLogger.Debug("Waiting for connection message to be received ...")

	select {
	case err := <-c.eventError:
		return nil, err
	case msg := <-c.events:
		return msg, nil
	}
}

// Handle - Will handle new messages and close connection when there are no messages left to process
func (c *SocketConnection) Handle() {

	done := make(chan bool)

	rbuf := bufio.NewReaderSize(c, ReadBufferSize)

	go func() {
		for {
			msg, err := newMessage(rbuf, true)
			var msgChannel *chan *Message
			var errChannel *chan error
			connectionLogger.Debug("Handle() got message")
			if msg.msgType == "text/event-plain" || msg.msgType == "text/event-json" {
				msgChannel = &c.events
				errChannel = &c.eventError
			} else {
				msgChannel = &c.replies
				errChannel = &c.replyError
			}

			if err != nil {
				connectionLogger.Debug("Handle() got error")
				*errChannel <- err
				done <- true
				break
			}

			*msgChannel <- msg
			connectionLogger.Debug("Handle() passed message")
		}
	}()

	<-done

	// Closing the connection now as there's nothing left to do ...
	c.Close()
	connectionLogger.Debug("!!!Main socket closed!!!")
}

// Close - Will close down net connection and return error if error happen
func (c *SocketConnection) Close() error {
	if err := c.Conn.Close(); err != nil {
		return err
	}

	return nil
}
