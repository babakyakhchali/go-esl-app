// Copyright 2015 Nevio Vesic
// Please check out LICENSE file for more information about what you CAN and what you CANNOT do!
// Basically in short this is a free software for you to do whatever you want to do BUT copyright must be included!
// I didn't write all of this code so you could say it's yours.
// MIT License

package logger

import (
	"fmt"
)

//logging colors
const (
	InfoColor    = "\033[1;34m%s\033[0m"
	NoticeColor  = "\033[1;36m%s\033[0m"
	WarningColor = "\033[1;33m%s\033[0m"
	ErrorColor   = "\033[1;31m%s\033[0m"
	DebugColor   = "\033[0;36m%s\033[0m"
	PrintColor   = "\033[1;37m%s\033[0m"
)

//different levels of logging
const (
	DEBUG = iota
	INFO
	NOTICE
	WARNING
	ERROR
)

//NsLogger namespaced logger
type NsLogger struct {
	ns    string
	level *int
}

//Debug print a debug log
func (l *NsLogger) Debug(message string, args ...interface{}) {
	l.doLog(DEBUG, message, args...)
}

//Error print a error log
func (l *NsLogger) Error(message string, args ...interface{}) {
	l.doLog(ERROR, message, args...)
}

//Notice print a notice log
func (l *NsLogger) Notice(message string, args ...interface{}) {
	l.doLog(NOTICE, message, args...)
}

//Info print a info log
func (l *NsLogger) Info(message string, args ...interface{}) {
	l.doLog(INFO, message, args...)
}

//Warning print a warning log
func (l *NsLogger) Warning(message string, args ...interface{}) {
	l.doLog(WARNING, message, args...)
}

func (l *NsLogger) doLog(level int, message string, args ...interface{}) {
	lstr := ""
	if *l.level > level {
		return
	}
	switch level {
	case DEBUG:
		lstr = fmt.Sprintf(DebugColor, "[DEBUG]")
	case INFO:
		lstr = fmt.Sprintf(InfoColor, "[INFO]")
	case NOTICE:
		lstr = fmt.Sprintf(NoticeColor, "[NOTICE]")
	case WARNING:
		lstr = fmt.Sprintf(WarningColor, "[WARNING]")
	case ERROR:
		lstr = fmt.Sprintf(ErrorColor, "[ERROR]")
	default:
		lstr = fmt.Sprintf(PrintColor, "[CONSOLE]")
	}
	fmt.Printf(lstr+" "+l.ns+" "+message+"\n", args...)
}

//CreateChild create a child logger
func (l *NsLogger) CreateChild(ns string) *NsLogger {
	nl := NewLogger(ns)
	nl.level = l.level
	nl.ns = l.ns + " [" + ns + "]"
	return nl
}

//SetLevel set log level for this and all child loggers
func (l *NsLogger) SetLevel(level int) {
	*l.level = level
}

//NewLogger create a parent logger
func NewLogger(ns string) *NsLogger {
	l := NsLogger{
		ns:    "[" + ns + "]",
		level: new(int),
	}
	*l.level = DEBUG
	return &l
}
