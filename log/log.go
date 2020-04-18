package log

import (
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	instance *logrus.Logger
	once     sync.Once
)

func GetLog() *logrus.Logger {

	once.Do(func() {
		instance = logrus.New()
		instance.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	})

	return instance
}
