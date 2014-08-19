package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"

	"github.com/hivetech/iron-app/log"
)

var Log *log.IronLogger

func main() {
	app := cli.NewApp()
	app.Name = "iron-app"
	app.Version = Version
	app.Usage = `Exoskeleton for applications.

    Load env from Consul's K/V store, discover provided services, route
    application output and store performances and application metadata.
  `

	app.Flags = []cli.Flag{
		cli.BoolFlag{Name: "verbose", Usage: "Extends log output to debug level"},
		cli.BoolFlag{Name: "errexit", Usage: "exit if there is an error occurs"},
		cli.BoolFlag{Name: "reload", Usage: "restarts the process when config changes"},
		cli.BoolFlag{Name: "monitor", Usage: "Measure and store process perfs"},
		// TODO Make it true by default (cli.IsSet("sanitize") ?)
		cli.BoolFlag{Name: "sanitize, s", Usage: "turn invalid charactersin the key into underscores"},
		cli.BoolFlag{Name: "upcase, u", Usage: "make all environment variable keys uppercase"},
		cli.StringFlag{Name: "app", Value: "app", Usage: "application's name"},
		// NOTE Use EnvVar when available
		cli.StringFlag{Name: "addr", Value: "127.0.0.1:8500", Usage: "consul HTTP API address with port"},
		cli.StringFlag{Name: "dc", Usage: "consul datacenter, default uses local"},
		cli.StringSliceFlag{Name: "discover, d", Value: &cli.StringSlice{}, Usage: "<service:tag>, known by consul, to discover"},
		cli.StringFlag{Name: "loghook, l", Usage: "endpoint to forward logs"},
		// TODO cli.DurationFlag doesn't work
		cli.IntFlag{Name: "pollinterval, p", Value: 5, Usage: "Metrics poll interval"},
	}

	// TODO Exit with process number
	app.Action = func(c *cli.Context) {
		Log = log.NewIronLogger(c.String("app"), c.Bool("verbose"))
		if err := Log.SetupHook(c.String("loghook")); err != nil {
			Log.Error(err)
			if c.Bool("errexit") {
				return
			}
		}

		config := IronConfig{
			Namespace:    "iron-app",
			App:          c.String("app"),
			ConsulAddr:   c.String("addr"),
			ConsulDC:     c.String("dc"),
			Sanitize:     c.Bool("sanitize"),
			Upcase:       c.Bool("upcase"),
			PollInterval: c.Int("pollinterval"),
		}
		result, err := watchAndExec(c.Args(), &config, c.StringSlice("discover"), c.Bool("monitor"), c.Bool("reload"), c.Bool("errexit"))

		Log.WithFields(logrus.Fields{"result": result, "error": err}).Info("Done")
		if err != nil {
			Log.Errorf("Error: %v\n", err)
			return
		}
	}

	app.RunAndExitOnError()
}
