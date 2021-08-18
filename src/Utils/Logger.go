package Utils

import (
	"bytes"
	"github.com/sirupsen/logrus"
	"sync"
)

var logger *logrus.Logger

func InitLogger(configLogLevel *string) *logrus.Logger {
	var once sync.Once
	once.Do(func() {
		logger = logrus.New()
		configLogLevel := configLogLevel
		var logLevel logrus.Level
		err := logLevel.UnmarshalText(bytes.NewBufferString(*configLogLevel).Bytes())
		if err != nil {
			logger.SetLevel(logrus.InfoLevel)
		}
		logger.SetLevel(logLevel)
		logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	})
	return logger
}

func GetLogger() *logrus.Logger {
	return logger
}
