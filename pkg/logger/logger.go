package logger

import (
	"log"
	"os"
)

type Logger interface {
	Info(format string, v ...interface{})
	Error(format string, v ...interface{})
}

type StdLogger struct {
	info  *log.Logger
	error *log.Logger
}

func New() *StdLogger {
	return &StdLogger{
		info:  log.New(os.Stdout, "INFO: ", log.LstdFlags|log.Lshortfile),
		error: log.New(os.Stderr, "ERROR: ", log.LstdFlags|log.Lshortfile),
	}
}

func (l *StdLogger) Info(format string, v ...interface{}) {
	l.info.Printf(format, v...)
}

func (l *StdLogger) Error(format string, v ...interface{}) {
	l.error.Printf(format, v...)
}
