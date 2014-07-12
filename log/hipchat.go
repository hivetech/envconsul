package log

import (
  "fmt"

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

func NewHipchatHook(app, roomId, apikey string) *HipchatHook {
  return &HipchatHook{
    Client: &hipchat.Client{AuthToken: apikey},
    App:    app,
    Room:   roomId,
    From:   "Envconsul",
  }
}

func (hook *HipchatHook) Fire(entry *logrus.Entry) error {
  entry.Logger.WithFields(logrus.Fields{
    "room": hook.Room,
    "data": entry.Data,
  }).Debug("Pushing notification to Hipchat.")

  title := fmt.Sprintf("[%s - %s] Error trapped !\n", entry.Data["time"], hook.App)
  message := title + entry.Data["msg"].(string)

  req := hipchat.MessageRequest{
    RoomId:        hook.Room,
    From:          hook.From,
    Message:       message,
    Color:         hipchat.ColorPurple,
    MessageFormat: hipchat.FormatText,
    Notify:        true,
  }

  if err := hook.Client.PostMessage(req); err != nil {
    entry.Logger.WithFields(logrus.Fields{
      "source": "hipchat",
      "room":   hook.Room,
    }).Warn("Failed to send error to Hipchat")
  }

  return nil
}

func (hook *HipchatHook) Levels() []logrus.Level {
  return []logrus.Level{
    logrus.Error,
    logrus.Fatal,
    logrus.Panic,
  }
}
