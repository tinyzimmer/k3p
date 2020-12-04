package log

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"
)

// Verbose is set by the CLI flag to enable debug logging
var Verbose bool

var infoLogger, warningLogger, errorLogger, debugLogger *logger

func init() {
	infoLogger = &logger{"INFO"}
	warningLogger = &logger{"WARNING"}
	errorLogger = &logger{"ERROR"}
	debugLogger = &logger{"DEBUG"}
}

type logger struct{ level string }

func (l *logger) getLevel() string {
	if l.level != "" {
		return fmt.Sprintf("[%s]", l.level)
	}
	return ""
}

const timeFormat = "2006/01/02 15:04:05"

func (l *logger) getTime() string {
	return time.Now().Local().Format(timeFormat)
}

func (l *logger) seedLine() {
	fmt.Print(l.getTime(), "  ", l.getLevel(), "\t")
}

func (l *logger) Println(args ...interface{}) {
	l.seedLine()
	fmt.Println(args...)
}

func (l *logger) Printf(fstr string, args ...interface{}) {
	l.seedLine()
	fmt.Printf(fstr, args...)
}

// TailReader will follow the given reader and send its contents
// to a dedicated logger configured with the given prefix.
func TailReader(prefix string, rdr io.Reader) {
	l := &logger{prefix}
	scanner := bufio.NewScanner(rdr)
	for scanner.Scan() {
		text := scanner.Text()
		l.Println(strings.TrimSpace(text))
	}
}

// Info is the equivalent of a log.Println on the info logger.
func Info(args ...interface{}) {
	infoLogger.Println(args...)
}

// Infof is the equivalent of a log.Printf on the info logger.
func Infof(fstr string, args ...interface{}) {
	infoLogger.Printf(fstr, args...)
}

// Warning is the equivalent of a log.Println on the warning logger.
func Warning(args ...interface{}) {
	warningLogger.Println(args...)
}

// Warningf is the equivalent of a log.Printf on the warning logger.
func Warningf(fstr string, args ...interface{}) {
	warningLogger.Printf(fstr, args...)
}

// Error is the equivalent of a log.Println on the error logger.
func Error(args ...interface{}) {
	errorLogger.Println(args...)
}

// Errorf is the equivalent of a log.Printf on the error logger.
func Errorf(fstr string, args ...interface{}) {
	errorLogger.Printf(fstr, args...)
}

// Debug is the equivalent of a log.Println on the debug logger.
func Debug(args ...interface{}) {
	if Verbose {
		debugLogger.Println(args...)
	}
}

// Debugf is the equivalent of a log.Printf on the debug logger.
func Debugf(fstr string, args ...interface{}) {
	if Verbose {
		debugLogger.Printf(fstr, args...)
	}
}

// DebugReader is a convenience method for tailing the contents of a reader
// to the debug logger.
func DebugReader(rdr io.Reader) {
	scanner := bufio.NewScanner(rdr)
	for scanner.Scan() {
		text := scanner.Text()
		Debug(strings.TrimSpace(text))
	}
}
