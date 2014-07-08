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
  var logfile string
  var verbose bool
  // Services available
  var webService string
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
    &logfile, "logfile", "",
    "If provided, redirect logs to this file")
  // Services available
  flag.StringVar(
    &webService, "web", "",
    "Comma separated web services on the network to discover")
  // Hooks available
  flag.StringVar(
    &loghookArg, "loghook", "",
    "A service where to send logs, like <service>:<info>")

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
  loghook := strings.Split(loghookArg, ":")
  if loghook[0] == "pushbullet" && len(loghook) == 2 {
    Log.Hooks.Add(log.NewPushbulletHook(args[0], os.Getenv("PUSHBULLET_API_KEY"), loghook[1]))
  } else {
    Log.Warn("Loghook not implemented, skipping.")
  }

  if logfile != "" {
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
  if webService != "" {
    webServices := strings.Split(webService, ",")
    for i := 0; i < len(webServices); i++ {
      Log.WithFields(logrus.Fields{
        "type": "web",
        "tag":  webServices[i],
      }).Info("Service required.")
      if err := nodesNetwork.discoverAndRemember("web", webServices[i], args[0]); err != nil {
        Log.WithFields(logrus.Fields{
          "service": "web",
          "tag":     webServices[i],
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
