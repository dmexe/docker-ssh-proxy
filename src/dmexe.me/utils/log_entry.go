package utils

import "github.com/Sirupsen/logrus"

// NewLogEntry creates named logger
func NewLogEntry(name string) *logrus.Entry {
	return logrus.WithField("logger", name)
}
