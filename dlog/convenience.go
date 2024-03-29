// Code generated by "./convenience.go.gen". DO NOT EDIT.

package dlog

import (
	"context"
	"fmt"
)

func Error(ctx context.Context, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLog(LogLevelError, args...)
	} else {
		l.Log(LogLevelError, fmt.Sprint(args...))
	}
}
func Errorln(ctx context.Context, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLogln(LogLevelError, args...)
	} else {
		l.Log(LogLevelError, sprintln(args...))
	}
}
func Errorf(ctx context.Context, format string, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLogf(LogLevelError, format, args...)
	} else {
		l.Log(LogLevelError, fmt.Sprintf(format, args...))
	}
}
func Warn(ctx context.Context, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLog(LogLevelWarn, args...)
	} else {
		l.Log(LogLevelWarn, fmt.Sprint(args...))
	}
}
func Warnln(ctx context.Context, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLogln(LogLevelWarn, args...)
	} else {
		l.Log(LogLevelWarn, sprintln(args...))
	}
}
func Warnf(ctx context.Context, format string, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLogf(LogLevelWarn, format, args...)
	} else {
		l.Log(LogLevelWarn, fmt.Sprintf(format, args...))
	}
}
func Info(ctx context.Context, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLog(LogLevelInfo, args...)
	} else {
		l.Log(LogLevelInfo, fmt.Sprint(args...))
	}
}
func Infoln(ctx context.Context, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLogln(LogLevelInfo, args...)
	} else {
		l.Log(LogLevelInfo, sprintln(args...))
	}
}
func Infof(ctx context.Context, format string, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLogf(LogLevelInfo, format, args...)
	} else {
		l.Log(LogLevelInfo, fmt.Sprintf(format, args...))
	}
}
func Debug(ctx context.Context, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLog(LogLevelDebug, args...)
	} else {
		l.Log(LogLevelDebug, fmt.Sprint(args...))
	}
}
func Debugln(ctx context.Context, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLogln(LogLevelDebug, args...)
	} else {
		l.Log(LogLevelDebug, sprintln(args...))
	}
}
func Debugf(ctx context.Context, format string, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLogf(LogLevelDebug, format, args...)
	} else {
		l.Log(LogLevelDebug, fmt.Sprintf(format, args...))
	}
}
func Trace(ctx context.Context, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLog(LogLevelTrace, args...)
	} else {
		l.Log(LogLevelTrace, fmt.Sprint(args...))
	}
}
func Traceln(ctx context.Context, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLogln(LogLevelTrace, args...)
	} else {
		l.Log(LogLevelTrace, sprintln(args...))
	}
}
func Tracef(ctx context.Context, format string, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLogf(LogLevelTrace, format, args...)
	} else {
		l.Log(LogLevelTrace, fmt.Sprintf(format, args...))
	}
}
func Print(ctx context.Context, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLog(LogLevelInfo, args...)
	} else {
		l.Log(LogLevelInfo, fmt.Sprint(args...))
	}
}
func Println(ctx context.Context, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLogln(LogLevelInfo, args...)
	} else {
		l.Log(LogLevelInfo, sprintln(args...))
	}
}
func Printf(ctx context.Context, format string, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLogf(LogLevelInfo, format, args...)
	} else {
		l.Log(LogLevelInfo, fmt.Sprintf(format, args...))
	}
}
func Warning(ctx context.Context, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLog(LogLevelWarn, args...)
	} else {
		l.Log(LogLevelWarn, fmt.Sprint(args...))
	}
}
func Warningln(ctx context.Context, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLogln(LogLevelWarn, args...)
	} else {
		l.Log(LogLevelWarn, sprintln(args...))
	}
}
func Warningf(ctx context.Context, format string, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLogf(LogLevelWarn, format, args...)
	} else {
		l.Log(LogLevelWarn, fmt.Sprintf(format, args...))
	}
}
