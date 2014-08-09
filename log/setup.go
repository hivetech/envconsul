package log

import (
	"github.com/Sirupsen/logrus"
	"os"
)

type IronLogger struct {
	logrus.Logger
	namespace string
}

func NewIronLogger(namespace string, verbose bool) *IronLogger {
	var logger = logrus.New()
	if verbose {
		logger.Level = logrus.DebugLevel
	}
	return &IronLogger{
		*logger,
		namespace,
	}
}

func (self *IronLogger) SetupHook(loghook string) error {
	self.WithFields(logrus.Fields{"hook": loghook}).Info("Registering loghook.")
	switch loghook {
	case "pushbullet":
		if pbHook, err := NewPushbulletHook(self.namespace); err != nil {
			return err
		} else {
			self.Hooks.Add(pbHook)
		}
	case "hipchat":
		if hipchatHook, err := NewPushbulletHook(self.namespace); err != nil {
			return err
		} else {
			self.Hooks.Add(hipchatHook)
		}
	default:
		if loghook != "" {
			self.Debug("Assuming provided loghook is a file")
			if fd, err := os.Create(loghook); err != nil {
				return err
			} else {
				//self.Out = fd
				logrus.SetOutput(fd)
			}
			// TODO Close this goddamn fd
		}
	}
	return nil
}
