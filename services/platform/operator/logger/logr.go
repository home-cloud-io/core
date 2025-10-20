package logger

import (
	"github.com/go-logr/logr"
	"github.com/steady-bytes/draft/pkg/chassis"
)

// sink is a wrapper around chassis.Logger that satisfies the logr.LogSink interface
type sink struct {
	log chassis.Logger
}

func NewLogger(logger chassis.Logger) logr.Logger {
	return logr.New(&sink{
		log: logger.WithCallDepth(4),
	})
}

// LogSink

// Init receives optional information about the logr library for LogSink
// implementations that need it.
func (s *sink) Init(info logr.RuntimeInfo) {
	s.log = s.log.WithCallDepth(info.CallDepth)
}

// Enabled tests whether this LogSink is enabled at the specified V-level.
// For example, commandline flags might be used to set the logging
// verbosity and disable some info logs.
func (s *sink) Enabled(level int) bool {
	// no need to check this as the underlying logger handles filtering
	return true
}

// Info logs a non-error message with the given key/value pairs as context.
// The level argument is provided for optional logging.  This method will
// only be called when Enabled(level) is true. See Logger.Info for more
// details.
func (s *sink) Info(level int, msg string, keysAndValues ...any) {
	// level will always be >=0
	switch level {
	case 0:
		s.log.WithFields(s.handleFields(keysAndValues...)).Info(msg)
	case 1:
		s.log.WithFields(s.handleFields(keysAndValues...)).Debug(msg)
	default:
		s.log.WithFields(s.handleFields(keysAndValues...)).Trace(msg)
	}
}

// Error logs an error, with the given message and key/value pairs as
// context.  See Logger.Error for more details.
func (s *sink) Error(err error, msg string, keysAndValues ...any) {
	s.log.WithFields(s.handleFields(keysAndValues...)).WithError(err).Error(msg)
}

// WithValues returns a new LogSink with additional key/value pairs.  See
// Logger.WithValues for more details.
func (s *sink) WithValues(keysAndValues ...any) logr.LogSink {
	return &sink{
		log: s.log.WithFields(s.handleFields(keysAndValues...)),
	}
}

// WithName returns a new LogSink with the specified name appended.  See
// Logger.WithName for more details.
func (s *sink) WithName(name string) logr.LogSink {
	s.log = s.log.WithField("name", name)
	return s
}

// Helpers

func (s *sink) handleFields(args ...any) chassis.Fields {
	// return if empty or if odd number
	if len(args) == 0 {
		return chassis.Fields{}
	}
	f := chassis.Fields{}
	for i := 0; i < len(args); {
		// protect against odd number of arguments
		if i == len(args)-1 {
			s.log.Error("odd number of arguments passed as key-value pairs for logging")
			break
		}

		key, val := args[i], args[i+1]
		keyStr, isString := key.(string)
		if !isString {
			s.log.Error("non-string key argument passed to logging, ignoring all later arguments")
			break
		}
		f[keyStr] = val
		i += 2
	}
	return f
}
