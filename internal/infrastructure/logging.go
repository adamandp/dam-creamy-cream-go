// Package infrastructure initializes shared application dependencies.
package infrastructure

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

func New(env string) *logrus.Entry {
	log := logrus.New()

	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})

	if env == "development" {
		log.SetLevel(logrus.DebugLevel)
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	return log.WithField("service", "dam-creamy-cream-go")
}
