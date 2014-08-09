package log

import (
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/andybons/hipchat"
)

// HipchatHook to send exceptions to a connected device with the Hipchat
// API. You must set:
// * hipchat.Device
// * hipchat.ApiKey
//
// Before using this hook, to send an error. Entries that trigger an Error,
// Fatal or Panic should now include an "error" field to send to Hipchat.
type HipchatHook struct {
	Client *hipchat.Client
	App    string
	Room   string
	From   string
}

func NewHipchatHook(app string) (*HipchatHook, error) {
	apiKey := os.Getenv("HIPCHAT_API_KEY")
	roomId := os.Getenv("HIPCHAT_ROOM")
	if apiKey == "" || roomId == "" {
		return nil, fmt.Errorf("Missing informations to setup hipchat loghook")
	}
	return &HipchatHook{
		Client: &hipchat.Client{AuthToken: apiKey},
		App:    app,
		Room:   roomId,
		From:   "Envconsul",
	}, nil
}

func (self *HipchatHook) Fire(entry *logrus.Entry) error {
	entry.Logger.WithFields(logrus.Fields{
		"room": self.Room,
		"data": entry.Data,
	}).Debug("Pushing notification to Hipchat.")

	title := fmt.Sprintf("[%s - %s] Error trapped !\n", entry.Data["time"], self.App)
	message := title + entry.Data["msg"].(string)

	req := hipchat.MessageRequest{
		RoomId:        self.Room,
		From:          self.From,
		Message:       message,
		Color:         hipchat.ColorPurple,
		MessageFormat: hipchat.FormatText,
		Notify:        true,
	}

	if err := self.Client.PostMessage(req); err != nil {
		entry.Logger.WithFields(logrus.Fields{
			"source": "hipchat",
			"room":   self.Room,
		}).Warn("Failed to send error to Hipchat")
	}

	return nil
}

func (self *HipchatHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.ErrorLevel,
		logrus.FatalLevel,
		logrus.PanicLevel,
	}
}
