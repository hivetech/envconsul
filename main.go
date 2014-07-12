package main

import (
  "flag"
  "os"
  "path/filepath"
  "strings"

  "github.com/Sirupsen/logrus"

  "github.com/hivetech/envconsul/log"
)

var Log = logrus.New()

func main() {
  os.Exit(realMain())
}

func realMain() int {
  var errExit bool
  var reload bool
  var consulAddr string
  var consulDC string
  var sanitize bool
  var upcase bool
  var verbose bool
  var linkServices string
  // Hooks available
  var loghookArg string

  flag.Usage = usage
  flag.BoolVar(
    &errExit, "errexit", false,
    "exit if there is an error watching config keys")
  flag.BoolVar(
    &reload, "reload", false,
    "if set, restarts the process when config changes")
  flag.StringVar(
    &consulAddr, "addr", "127.0.0.1:8500",
    "consul HTTP API address with port")
  flag.StringVar(
    &consulDC, "dc", "",
    "consul datacenter, uses local if blank")
  flag.BoolVar(
    &sanitize, "sanitize", true,
    "turn invalid characters in the key into underscores")
  flag.BoolVar(
    &upcase, "upcase", true,
    "make all environmental variable keys uppercase")
  flag.BoolVar(
    &verbose, "verbose", false,
    "Extend log output to debug level")
  flag.StringVar(
    &linkServices, "discover", "",
    "Comma separated <service:tag> on the network to discover")
  // Hooks available
  flag.StringVar(
    &loghookArg, "loghook", "",
    "An app where to send logs [pushbullet]")

  flag.Parse()
  if flag.NArg() < 2 {
    flag.Usage()
    return 1
  }

  args := flag.Args()

  // Initialize logger
  if verbose {
    Log.Level = logrus.Debug
  }
  Log.Formatter = new(logrus.TextFormatter)
  if loghookArg == "pushbullet" {
    apiKey := os.Getenv("PUSHBULLET_API_KEY")
    device := os.Getenv("PUSHBULLET_DEVICE")
    if apiKey != "" && device != "" {
      Log.WithFields(logrus.Fields{
        "device": device,
      }).Info("Registering pushbullet loghook.")
      Log.Hooks.Add(log.NewPushbulletHook(args[1], device, apiKey))
    } else {
      Log.Warn("Missing pushbullet informations.")
    }
  } else if loghookArg == "hipchat" {
    apiKey := os.Getenv("HIPCHAT_API_KEY")
    roomId := os.Getenv("HIPCHAT_ROOM")
    if apiKey != "" && roomId != "" {
      Log.WithFields(logrus.Fields{
        "room": roomId,
      }).Info("Registering hipchat loghook.")
      Log.Hooks.Add(log.NewHipchatHook(args[1], roomId, apiKey))
    } else {
      Log.Warn("Missing Hipchat informations.")
    }
  } else if loghookArg != "" {
    logfile := loghookArg
    Log.Info("Using File hook.")
    // open output file
    fd, err := os.Create(logfile)
    if err != nil {
      Log.WithFields(logrus.Fields{
        "error": err.Error(),
        "file":  logfile,
      }).Error("Unable to create file.")
    }
    // close fo on exit and check for its returned error
    defer func() {
      if err := fd.Close(); err != nil {
        Log.WithFields(logrus.Fields{
          "error": err.Error(),
          "file":  logfile,
        }).Error("Unable to close file descriptor.")
        panic(err)
      }
    }()
    if err == nil {
      Log.Out = fd
    }
  }

  Log.WithFields(logrus.Fields{
    "verbose":   verbose,
    "formatter": "Text",
  }).Debug("Logger ready (obviously !)")

  var nodesNetwork = NewConsulNetwork(consulAddr, consulDC)
  if linkServices != "" {
    links := strings.Split(linkServices, ",")
    for i := 0; i < len(links); i++ {
      linkDetails := strings.Split(links[i], ":")
      // TODO Handle missing tag
      // TODO Optional tag or node name instead (a query dsl ?)
      Log.WithFields(logrus.Fields{
        "service": linkDetails[0],
        "tag":     linkDetails[1],
      }).Info("Service required.")
      if err := nodesNetwork.discoverAndRemember(linkDetails[0], linkDetails[1], args[0]); err != nil {
        Log.WithFields(logrus.Fields{
          "service": linkDetails[0],
          "tag":     linkDetails[1],
        }).Error(err.Error())
        // TODO Different strategies (stop, retry, ...)
        if errExit {
          Log.WithFields(logrus.Fields{
            "reason":  err.Error(),
            "errExit": errExit,
          }).Warn("Forcing exit ...")
          os.Exit(1)
        }
      }
    }
  }

  config := WatchConfig{
    ConsulAddr: consulAddr,
    ConsulDC:   consulDC,
    Cmd:        args[1:],
    ErrExit:    errExit,
    Prefix:     args[0],
    Reload:     reload,
    Sanitize:   sanitize,
    Upcase:     upcase,
  }
  result, err := watchAndExec(&config)
  Log.WithFields(logrus.Fields{
    "result": result,
    "error":  err,
  }).Info("Done")
  if err != nil {
    Log.Error("Error: %s\n", err)
    return 111
  }

  return result
}

func usage() {
  cmd := filepath.Base(os.Args[0])
  Log.Errorf(strings.TrimSpace(helpText)+"\n\n", cmd)
  flag.PrintDefaults()
}

const helpText = `
Usage: %s [options] prefix child...

  Sets environmental variables for the child process by reading
  K/V from Consul's K/V store with the given prefix.

Options:
`
