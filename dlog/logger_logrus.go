package dlog

import (
	"io"
	"log"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type logrusLogger interface {
	WithField(key string, value interface{}) *logrus.Entry
	WriterLevel(level logrus.Level) *io.PipeWriter
	Log(level logrus.Level, args ...interface{})
	Logln(level logrus.Level, args ...interface{})
	Logf(level logrus.Level, format string, args ...interface{})
}

type logrusWrapper struct {
	logrusLogger
}

var _ OptimizedLogger = logrusWrapper{}

// Helper does nothing--we use a Logrus Hook instead (see below).
func (l logrusWrapper) Helper() {}

func (l logrusWrapper) WithField(key string, value interface{}) Logger {
	return logrusWrapper{l.logrusLogger.WithField(key, value)}
}

var dlogLevel2logrusLevel = [5]logrus.Level{
	logrus.ErrorLevel,
	logrus.WarnLevel,
	logrus.InfoLevel,
	logrus.DebugLevel,
	logrus.TraceLevel,
}

func (l logrusWrapper) StdLogger(level LogLevel) *log.Logger {
	if level > LogLevelTrace {
		panic(errors.Errorf("invalid LogLevel: %d", level))
	}
	return log.New(l.logrusLogger.WriterLevel(dlogLevel2logrusLevel[level]), "", 0)
}

func (l logrusWrapper) Log(level LogLevel, msg string) {
	if level > LogLevelTrace {
		panic(errors.Errorf("invalid LogLevel: %d", level))
	}
	l.logrusLogger.Log(dlogLevel2logrusLevel[level], msg)
}

func (l logrusWrapper) MaxLevel() LogLevel {
	ll := l.logrusLogger
	if le, ok := ll.(*logrus.Entry); ok {
		ll = le.Logger
	}
	logrusLevel := ll.(*logrus.Logger).GetLevel()
	for i, l := range dlogLevel2logrusLevel {
		if l == logrusLevel {
			return LogLevel(i)
		}
	}
	panic(errors.Errorf("invalid logrus LogLevel: %d", logrusLevel))
}

func (l logrusWrapper) SetMaxLevel(level LogLevel) {
	ll := l.logrusLogger
	if le, ok := ll.(*logrus.Entry); ok {
		ll = le.Logger
	}
	logrusLevel := dlogLevel2logrusLevel[level]
	ll.(*logrus.Logger).SetLevel(logrusLevel)
}

func (l logrusWrapper) UnformattedLog(level LogLevel, args ...interface{}) {
	if level > LogLevelTrace {
		panic(errors.Errorf("invalid LogLevel: %d", level))
	}
	l.logrusLogger.Log(dlogLevel2logrusLevel[level], args...)
}

func (l logrusWrapper) UnformattedLogln(level LogLevel, args ...interface{}) {
	if level > LogLevelTrace {
		panic(errors.Errorf("invalid LogLevel: %d", level))
	}
	l.logrusLogger.Logln(dlogLevel2logrusLevel[level], args...)
}

func (l logrusWrapper) UnformattedLogf(level LogLevel, format string, args ...interface{}) {
	if level > LogLevelTrace {
		panic(errors.Errorf("invalid LogLevel: %d", level))
	}
	l.logrusLogger.Logf(dlogLevel2logrusLevel[level], format, args...)
}

// WrapLogrus converts a logrus *Logger into a generic Logger.
//
// You should only really ever call WrapLogrus from the initial
// process set up (i.e. directly inside your 'main()' function), and
// you should pass the result directly to WithLogger.
func WrapLogrus(in *logrus.Logger) Logger {
	in.AddHook(logrusFixCallerHook{})
	return logrusWrapper{in}
}

type logrusFixCallerHook struct{}

func (logrusFixCallerHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (logrusFixCallerHook) Fire(entry *logrus.Entry) error {
	if entry.Caller != nil && strings.HasPrefix(entry.Caller.Function, dlogPackage+".") {
		entry.Caller = getCaller()
	}
	return nil
}

const (
	dlogPackage            = "github.com/datawire/dlib/dlog"
	logrusPackage          = "github.com/sirupsen/logrus"
	maximumCallerDepth int = 25
	minimumCallerDepth int = 2 // runtime.Callers + getCaller
)

// Duplicate of logrus.getCaller() because Logrus doesn't have the
// kind if skip/.Helper() functionality that testing.TB has.
//
// https://github.com/sirupsen/logrus/issues/972
func getCaller() *runtime.Frame {
	// Restrict the lookback frames to avoid runaway lookups
	pcs := make([]uintptr, maximumCallerDepth)
	depth := runtime.Callers(minimumCallerDepth, pcs)
	frames := runtime.CallersFrames(pcs[:depth])

	for f, again := frames.Next(); again; f, again = frames.Next() {
		// If the caller isn't part of this package, we're done
		if strings.HasPrefix(f.Function, logrusPackage+".") {
			continue
		}
		if strings.HasPrefix(f.Function, dlogPackage+".") {
			continue
		}
		return &f //nolint:scopelint
	}

	// if we got here, we failed to find the caller's context
	return nil
}
