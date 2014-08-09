package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"

	"github.com/Sirupsen/logrus"

	"github.com/hivetech/iron-app/log"
)

var Log *log.IronLogger

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

	Log = log.NewIronLogger(args[1], verbose)
	if err := Log.SetupHook(loghookArg); err != nil {
		Log.Error(err)
	}
	Log.WithFields(logrus.Fields{
		"verbose":   verbose,
		"namespace": args[1],
		"formatter": "Text",
	}).Debug("Logger ready (obviously !)")

	services := strings.Split(linkServices, ",")
	config := IronConfig{
		Namespace:    "iron-app",
		App:          args[0],
		ConsulAddr:   consulAddr,
		ConsulDC:     consulDC,
		Sanitize:     sanitize,
		Upcase:       upcase,
		PollInterval: 5,
	}
	result, err := watchAndExec(args[1:], &config, services, reload, errExit)

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

  Exoskeleton for applications.

  Load env from Consul's K/V store, discover provided services, route
  application output and store performances and application metadata.

Options:
`
