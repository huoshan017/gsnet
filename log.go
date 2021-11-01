package gsnet

import (
	"io"
	"log"
	"os"

	"github.com/pkg/errors"
)

type Logger interface {
	SetOutput(output io.Writer)
	WithStack(err interface{})
	Fatalf(format string, args ...interface{})
	Fatal(args ...interface{})
	Infof(format string, args ...interface{})
	Info(args ...interface{})
}

var gslog Logger

func getLogger() Logger {
	if gslog == nil {
		SetLogger(newDefaultLogger())
	}
	return gslog
}

func SetLogger(logger Logger) {
	gslog = logger
}

func SetLoggerOutput(output io.Writer) {
	gslog.SetOutput(output)
}

type defaultLog struct {
	log *log.Logger
}

func newDefaultLogger() *defaultLog {
	return &defaultLog{log: log.New(os.Stderr, "gsnet: ", log.LstdFlags|log.Lshortfile)}
}

func (l *defaultLog) SetOutput(output io.Writer) {
	l.log.SetOutput(output)
}

func (l *defaultLog) WithStack(err interface{}) {
	er := errors.Errorf("%v", err)
	l.log.Fatalf("\n%+v", er)
}

func (l *defaultLog) Fatalf(format string, args ...interface{}) {
	l.log.Fatalf(format, args...)
}

func (l *defaultLog) Fatal(args ...interface{}) {
	l.log.Fatal(args...)
}

func (l *defaultLog) Infof(format string, args ...interface{}) {
	l.log.Printf(format, args...)
}

func (l *defaultLog) Info(args ...interface{}) {
	l.log.Print(args...)
}
