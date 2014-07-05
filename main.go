package main

import (
  "flag"
  "os"
  "path/filepath"
  "strings"

  "github.com/Sirupsen/logrus"
)

var Log = logrus.New()

func init() {
  // Initialize logger
  Log.Level = logrus.Debug
  Log.Formatter = new(logrus.TextFormatter)
  Log.WithFields(logrus.Fields{
    "verbose":   "Debug",
    "formatter": "Text",
  }).Debug("Logger ready (obviously !)")
}

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
  // Services available
  var webService string

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
  // Services available
  flag.StringVar(
    &webService, "web", "",
    "Comma separated web services on the network to discover")

  flag.Parse()
  if flag.NArg() < 2 {
    flag.Usage()
    return 1
  }

  args := flag.Args()

  var nodesNetwork = NewConsulNetwork(consulAddr, consulDC)
  webServices := strings.Split(webService, ",")
  for i := 0; i < len(webServices); i++ {
    Log.WithFields(logrus.Fields{
      "type": "web",
      "tag":  webServices[i],
    }).Info("Service required.")
    nodesNetwork.discoverAndRemember("web", webServices[i], args[0])
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
  Log.Error(strings.TrimSpace(helpText)+"\n\n", cmd)
  flag.PrintDefaults()
}

const helpText = `
Usage: %s [options] prefix child...

  Sets environmental variables for the child process by reading
  K/V from Consul's K/V store with the given prefix.

Options:
`
