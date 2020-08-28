package wsserver

import (
	"fmt"
	"io"
	"log"
	"os"
)

// Logger define logging interface
type Logger interface {
	Info(msg string)
	Infof(msg string, args ...interface{})
	Debug(msg string)
	Debugf(msg string, args ...interface{})
	Warn(msg string)
	Warnf(msg string, args ...interface{})
	Error(msg string)
	Errorf(msg string, args ...interface{})
	Fatal(msg string)
	Fatalf(msg string, args ...interface{})
	SetOutput(io.Writer)
}

const (
	colorBlack = iota + 30
	colorRed
	colorGreen
	colorYellow
	colorBlue
	colorMagenta
	colorCyan
	colorWhite

	colorBold     = 1
	colorDarkGray = 90
)

func colorize(s interface{}, c int) string {
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}

func NewDefaultLogger() Logger {
	return &logger{Logger: log.New(os.Stdout, "", log.Ldate|log.Ltime)}
}

type logger struct {
	*log.Logger
}

func (log *logger) SetOutput(writer io.Writer) {
	log.Logger.SetOutput(writer)
}

func (log *logger) Info(msg string) {
	log.Logger.Print(msg)
}
func (log *logger) Infof(msg string, args ...interface{}) {
	log.Logger.Printf(msg, args...)
}
func (log *logger) Debug(msg string) {
	log.Logger.Print(msg)
}
func (log *logger) Debugf(msg string, args ...interface{}) {
	log.Logger.Printf(msg, args...)
}
func (log *logger) Warn(msg string) {
	log.Logger.Print(msg)
}
func (log *logger) Warnf(msg string, args ...interface{}) {
	log.Logger.Printf(msg, args...)
}
func (log *logger) Error(msg string) {
	log.Logger.Fatal(msg)
}
func (log *logger) Errorf(msg string, args ...interface{}) {
	log.Logger.Fatalf(msg, args...)
}
func (log *logger) Fatal(msg string) {
	log.Logger.Panic(msg)
}
func (log *logger) Fatalf(msg string, args ...interface{}) {
	log.Logger.Panicf(msg, args...)
}
