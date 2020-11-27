package log

import (
	glog "log"
	"os"
)

// Verbose is set by the CLI flag to enable debug logging
var Verbose bool

var infoLogger, warningLogger, errorLogger, debugLogger *glog.Logger

func init() {
	infoLogger = glog.New(os.Stderr, "INFO: ", glog.Ldate|glog.Ltime)
	warningLogger = glog.New(os.Stderr, "WARNING: ", glog.Ldate|glog.Ltime)
	errorLogger = glog.New(os.Stderr, "ERROR: ", glog.Ldate|glog.Ltime)
	debugLogger = glog.New(os.Stderr, "DEBUG: ", glog.Ldate|glog.Ltime)
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
