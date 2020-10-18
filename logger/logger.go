// Copyright 2015 Nevio Vesic
// Please check out LICENSE file for more information about what you CAN and what you CANNOT do!
// Basically in short this is a free software for you to do whatever you want to do BUT copyright must be included!
// I didn't write all of this code so you could say it's yours.
// MIT License

package logger

import (
	"log"
)

func Debug(message string, args ...interface{}) {
	log.Printf(message, args...)
}

func Error(message string, args ...interface{}) {
	log.Printf(message, args...)
}

func Notice(message string, args ...interface{}) {
	log.Printf(message, args...)
}

func Info(message string, args ...interface{}) {
	log.Printf(message, args...)
}

func Warning(message string, args ...interface{}) {
	log.Printf(message, args...)
}

type NsLogger struct {
	ns string
}

func (l *NsLogger) Debug(message string, args ...interface{}) {
	Debug("["+l.ns+"] "+message, args...)
}

func (l *NsLogger) Error(message string, args ...interface{}) {
	Error("["+l.ns+"] "+message, args...)
}

func (l *NsLogger) Notice(message string, args ...interface{}) {
	Notice("["+l.ns+"] "+message, args...)
}

func (l *NsLogger) Info(message string, args ...interface{}) {
	Info("["+l.ns+"] "+message, args...)
}
func (l *NsLogger) Warning(message string, args ...interface{}) {
	Warning("["+l.ns+"] "+message, args...)
}

func NewLogger(ns string) *NsLogger {
	return &NsLogger{ns: ns}
}
