package log

import (
  "fmt"
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

func NewPushbulletHook(app, device, apikey string) *PushbulletHook {
  pb := pushbullet.New(apikey)
  // TODO What to do with the error ?
  // TODO Is it good practice to risk errors here ?
  devs, _ := pb.Devices()

  return &PushbulletHook{
    Client:  pb,
    Devices: devs,
    Device:  device,
    App:     app,
  }
}

func (hook *PushbulletHook) Fire(entry *logrus.Entry) error {
  entry.Logger.WithFields(logrus.Fields{
    "device": hook.Device,
    "data":   entry.Data,
  }).Debug("Pushing notification to Pushbullet.")

  for i := 0; i < len(hook.Devices); i++ {
    if strings.Contains(hook.Devices[i].Extras.Model, hook.Device) || strings.Contains(hook.Devices[i].Extras.Nickname, hook.Device) {
      // TODO Add a security to prevent spamming pushbullet
      hook.Push(entry, hook.Devices[i])
      break
    }
  }

  return nil
}

func (hook *PushbulletHook) Push(entry *logrus.Entry, device *pushbullet.Device) {
  entry.Logger.WithFields(logrus.Fields{
    "id":     device.Id,
    "owner":  device.OwnerName,
    "extras": device.Extras,
  }).Debug("Found device.")

  title := fmt.Sprintf("[%s - %s] Error trapped !", entry.Data["time"], hook.App)
  message := entry.Data["msg"].(string)

  if err := hook.Client.PushNote(device.Id, title, message); err != nil {
    entry.Logger.WithFields(logrus.Fields{
      "source": "pushbullet",
      "device": device,
    }).Warn("Failed to send error to Pushbullet")
  }
}

func (hook *PushbulletHook) Levels() []logrus.Level {
  return []logrus.Level{
    logrus.ErrorLevel,
    logrus.FatalLevel,
    logrus.PanicLevel,
  }
}
