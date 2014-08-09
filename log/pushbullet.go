package log

import (
	"fmt"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/xconstruct/go-pushbullet"
)

// PushbulletHook to send exceptions to a connected device with the Pushbullet
// API. You must provide:
// * pushbullet.Device
// * pushbullet.ApiKey
//
// Before using this hook, to send an error. Entries that trigger an Error,
// Fatal or Panic should now include an "error" field to send to Pushbullet.
type PushbulletHook struct {
	Client  *pushbullet.Client
	Devices []*pushbullet.Device
	Device  string
	App     string
}

func NewPushbulletHook(app string) (*PushbulletHook, error) {
	apiKey := os.Getenv("PUSHBULLET_API_KEY")
	device := os.Getenv("PUSHBULLET_DEVICE")
	if apiKey == "" || device == "" {
		return nil, fmt.Errorf("Missing informations to setup pushbullet loghook")
	}
	pb := pushbullet.New(apiKey)
	// TODO What to do with the error ?
	devs, err := pb.Devices()
	if err != nil {
		return nil, err
	}

	return &PushbulletHook{
		Client:  pb,
		Devices: devs,
		Device:  device,
		App:     app,
	}, nil
}

func (self *PushbulletHook) Fire(entry *logrus.Entry) error {
	entry.Logger.WithFields(logrus.Fields{
		"device": self.Device,
		"data":   entry.Data,
	}).Debug("Pushing notification to Pushbullet.")

	for i := 0; i < len(self.Devices); i++ {
		if strings.Contains(self.Devices[i].Extras.Model, self.Device) || strings.Contains(self.Devices[i].Extras.Nickname, self.Device) {
			// TODO Add a security to prevent spamming pushbullet
			self.Push(entry, self.Devices[i])
			break
		}
	}

	return nil
}

func (self *PushbulletHook) Push(entry *logrus.Entry, device *pushbullet.Device) {
	entry.Logger.WithFields(logrus.Fields{
		"id":     device.Id,
		"owner":  device.OwnerName,
		"extras": device.Extras,
	}).Debug("Found device.")

	title := fmt.Sprintf("[%s - %s] Error trapped !", entry.Data["time"], self.App)
	message := entry.Data["msg"].(string)

	if err := self.Client.PushNote(device.Id, title, message); err != nil {
		entry.Logger.WithFields(logrus.Fields{
			"source": "pushbullet",
			"device": device,
		}).Warn("Failed to send error to Pushbullet")
	}
}

func (self *PushbulletHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.ErrorLevel,
		logrus.FatalLevel,
		logrus.PanicLevel,
	}
}
