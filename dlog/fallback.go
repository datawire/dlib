package dlog

import (
	"sync"

	"github.com/sirupsen/logrus"
)

var globals = struct { //nolint:gochecknoglobals // this is a place where we really do want a global
	fallbackLogger   Logger
	fallbackLoggerMu sync.RWMutex
}{
	fallbackLogger: WrapLogrus(logrus.New()),
}

func getFallbackLogger() Logger {
	globals.fallbackLoggerMu.RLock()
	defer globals.fallbackLoggerMu.RUnlock()
	return globals.fallbackLogger
}

// SetFallbackLogger sets the Logger that is returned for a context
// that doesn't have a Logger associated with it.  A nil fallback
// Logger will cause dlog calls on a context without a Logger to
// panic, which would be good for detecting places where contexts are
// not passed around correctly.  However, the default fallback logger
// is Logrus and will behave reasonably; in order to make using dlog a
// safe "no brainer".
func SetFallbackLogger(l Logger) {
	globals.fallbackLoggerMu.Lock()
	defer globals.fallbackLoggerMu.Unlock()
	globals.fallbackLogger = l
}
