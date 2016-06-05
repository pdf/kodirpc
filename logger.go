package kodirpc

import (
	"fmt"
	"os"
	"sync"
)

// LevelledLogger represents a minimal levelled logger
type LevelledLogger interface {
	// Debugf handles debug level messages
	Debugf(format string, args ...interface{})
	// Infof handles info level messages
	Infof(format string, args ...interface{})
	// Warnf handles warn level messages
	Warnf(format string, args ...interface{})
	// Errorf handles error level messages
	Errorf(format string, args ...interface{})
	// Fatalf handles fatal level messages, and must exit the application
	Fatalf(format string, args ...interface{})
	// Panicf handles debug level messages, and must panic the application
	Panicf(format string, args ...interface{})
}

// stubLogger satisfies the Logger interface, and simply does nothing with
// received messages
type stubLogger struct{}

// Debugf handles debug level messages
func (l *stubLogger) Debugf(format string, args ...interface{}) {}

// Infof handles info level messages
func (l *stubLogger) Infof(format string, args ...interface{}) {}

// Warnf handles warn level messages
func (l *stubLogger) Warnf(format string, args ...interface{}) {}

// Errorf handles error level messages
func (l *stubLogger) Errorf(format string, args ...interface{}) {}

// Fatalf handles fatal level messages, exits the application
func (l *stubLogger) Fatalf(format string, args ...interface{}) {
	os.Exit(1)
}

// Panicf handles debug level messages, and panics the application
func (l *stubLogger) Panicf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}

type logPrefixer struct {
	log LevelledLogger
	sync.Mutex
}

// Debugf handles debug level messages, prefixing them for golifx
func (l *logPrefixer) Debugf(format string, args ...interface{}) {
	l.Lock()
	l.log.Debugf(l.prefix(format), args...)
	l.Unlock()
}

// Infof handles info level messages, prefixing them for golifx
func (l *logPrefixer) Infof(format string, args ...interface{}) {
	l.Lock()
	l.log.Infof(l.prefix(format), args...)
	l.Unlock()
}

// Warnf handles warn level messages, prefixing them for golifx
func (l *logPrefixer) Warnf(format string, args ...interface{}) {
	l.Lock()
	l.log.Warnf(l.prefix(format), args...)
	l.Unlock()
}

// Errorf handles error level messages, prefixing them for golifx
func (l *logPrefixer) Errorf(format string, args ...interface{}) {
	l.Lock()
	l.log.Errorf(l.prefix(format), args...)
	l.Unlock()
}

// Fatalf handles fatal level messages, prefixing them for golifx
func (l *logPrefixer) Fatalf(format string, args ...interface{}) {
	l.Lock()
	l.log.Fatalf(l.prefix(format), args...)
	l.Unlock()
}

// Panicf handles debug level messages, prefixing them for golifx
func (l *logPrefixer) Panicf(format string, args ...interface{}) {
	l.Lock()
	l.log.Panicf(l.prefix(format), args...)
	l.Unlock()
}

func (l *logPrefixer) prefix(format string) string {
	return `[kodirpc] ` + format
}

var (
	// Log holds the global logger used by golifx, can be set via SetLogger() in
	// the golifx package
	logger LevelledLogger
)

func init() {
	SetLogger(&stubLogger{})
}

// SetLogger wraps the supplied logger with a logPrefixer to denote golifx logs
func SetLogger(l LevelledLogger) {
	logger = &logPrefixer{log: l}
}
