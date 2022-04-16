package log

import (
	"io"
	"log"
	"os"

	"github.com/pkg/errors"
)

type Logger interface {
	SetOutput(output io.Writer)
	WithStack(err any)
	Fatalf(format string, args ...any)
	Fatal(args ...any)
	Infof(format string, args ...any)
	Info(args ...any)
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

func (l *defaultLog) WithStack(err any) {
	er := errors.Errorf("%v", err)
	l.log.Fatalf("\n%+v", er)
}

func (l *defaultLog) Fatalf(format string, args ...any) {
	l.log.Fatalf(format, args...)
}

func (l *defaultLog) Fatal(args ...any) {
	l.log.Fatal(args...)
}

func (l *defaultLog) Infof(format string, args ...any) {
	l.log.Printf(format, args...)
}

func (l *defaultLog) Info(args ...any) {
	l.log.Print(args...)
}

func SetOutput(output io.Writer) {
	getLogger().SetOutput(output)
}

func WithStack(err any) {
	getLogger().WithStack(err)
}

func Fatalf(format string, args ...any) {
	getLogger().Fatalf(format, args...)
}

func Fatal(args ...any) {
	getLogger().Fatal(args...)
}

func Infof(format string, args ...any) {
	getLogger().Infof(format, args...)
}

func Info(args ...any) {
	getLogger().Info(args...)
}
