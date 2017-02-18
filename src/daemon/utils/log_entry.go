package utils

import "github.com/Sirupsen/logrus"

// LogEntry is an utility class for log producers
type LogEntry struct {
	Log *logrus.Entry
}

// NewLogEntry creates new LogEntry for log producers
func NewLogEntry(name string) *LogEntry {
	return &LogEntry{logrus.WithField("logger", name)}
}
