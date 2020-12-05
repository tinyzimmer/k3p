package log

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Verbose is set by the CLI flag to enable debug logging
var Verbose bool

// LogWriter can be overwritten by tests to suppress log output
var LogWriter io.Writer = os.Stdout

var infoLogger, warningLogger, errorLogger, debugLogger *logger

const (
	boldColor    = "\033[1m%s\033[0m"
	infoColor    = "\033[0;34m%s\033[0m"
	noticeColor  = "\033[0;36m%s\033[0m"
	warningColor = "\033[0;33m%s\033[0m"
	errorColor   = "\033[0;31m%s\033[0m"
	debugColor   = "\033[0;36m%s\033[0m"
)

func init() {
	infoLogger = &logger{"INFO", infoColor}
	warningLogger = &logger{"WARNING", warningColor}
	errorLogger = &logger{"ERROR", errorColor}
	debugLogger = &logger{"DEBUG", debugColor}
}

type logger struct {
	prefix, color string
}

func (l *logger) getPrefix() string {
	return fmt.Sprintf(l.color, fmt.Sprintf("[%s]", l.prefix))
}

const timeFormat = "2006/01/02 15:04:05"

func (l *logger) getTime() string {
	return fmt.Sprintf(noticeColor, time.Now().Local().Format(timeFormat))
}

func (l *logger) seedLine() {
	fmt.Fprint(LogWriter, l.getTime(), "  ", l.getPrefix(), "\t")
}

func (l *logger) Println(args ...interface{}) {
	l.seedLine()
	line := fmt.Sprintln(args...)
	fmt.Fprintf(LogWriter, boldColor, line)
}

func (l *logger) Printf(fstr string, args ...interface{}) {
	l.seedLine()
	line := fmt.Sprintf(fstr, args...)
	fmt.Fprintf(LogWriter, boldColor, line)
}

// TailReader will follow the given reader and send its contents
// to a dedicated logger configured with the given prefix.
// TODO: Make color configurable
func TailReader(prefix string, rdr io.Reader) {
	l := &logger{prefix, infoColor}
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
