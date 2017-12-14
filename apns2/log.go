package apns2

import (
	"io"
	"log"
	"os"
)

// Logger interface is extracted from log.Logger to aid in configuring
// custom loggers for use in this package.
type Logger interface {
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
	Fatalln(v ...interface{})
	Flags() int
	Output(calldepth int, s string) error
	Panic(v ...interface{})
	Panicf(format string, v ...interface{})
	Panicln(v ...interface{})
	Prefix() string
	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})
	SetFlags(flag int)
	SetOutput(w io.Writer)
	SetPrefix(prefix string)
}

// Log is a runtime-wide logger used by this package. You are allowed to set it
// to a different facility as needed.
var Log Logger = log.New(os.Stderr, "apns2: ", log.LstdFlags)

// Severity represents a log entry severity.
type Severity int

const (
	LogError Severity = iota
	LogWarn
	LogNotice
	LogInfo
)

// LogLevel is a runtime-wide setting that indicates which severity levels
// should be logged.
var LogLevel = LogNotice

var severityStrs = map[Severity]string{
	LogError:    "ERROR ",
	LogWarn:     "WARNING ",
	LogNotice:   "NOTICE ",
	LogInfo:     "INFO ",
	LogInfo + 1: "TRACE ",
}

// Bounds returns a severity value that is clamped between LogError and
// LogInfo + 1, latter being indicative of trace level logging.
func (t Severity) Bound() Severity {
	switch {
	case t < LogError:
		return LogError
	case t > LogInfo:
		return LogInfo + 1
	}
	return t
}

// String returns name associated with given Severity value.
func (t Severity) String() string {
	return severityStrs[t.Bound()]
}

// LogTrace returns a Severity value corresponding to the spcified trace level.
func LogTrace(traceLevel uint) Severity {
	return LogInfo + Severity(traceLevel+1)
}

func logWarn(id string, format string, v ...interface{}) {
	logTag(id, LogWarn, format, v...)
}

func logInfo(id string, format string, v ...interface{}) {
	logTag(id, LogInfo, format, v...)
}

func logTrace(level uint, id string, format string, v ...interface{}) {
	logTag(id, LogInfo+Severity(level+1), format, v...)
}

func logTag(id string, tag Severity, format string, v ...interface{}) {
	if tag > LogLevel {
		return
	}
	format = tag.String() + format
	if len(id) > 0 {
		format = id + ": " + format
	}
	Log.Printf(format, v...)
}
