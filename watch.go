package main

import (
	"regexp"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/armon/consul-api"
)

var (
	// Regexp for invalid characters in keys
	InvalidRegexp = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

// Connects to Consul and watches a given K/V prefix and uses that to
// execute a child process.
func watchAndExec(command []string, config *IronConfig, services []string, reload, errExit bool) (int, error) {
	wrapper, _ := NewIronWrapper(config)

	// Service discovery
	if len(services) > 1 {
		for i := 0; i < len(services); i++ {
			linkDetails := strings.Split(services[i], ":")
			// TODO Handle missing tag
			// TODO Optional tag or node name instead (a query dsl ?)
			Log.WithFields(logrus.Fields{"service": linkDetails[0], "tag": linkDetails[1]}).Info("Discovering service.")
			if err := wrapper.network.discoverAndRemember(linkDetails[0], linkDetails[1]); err != nil {
				Log.Error(err)
				// TODO Different strategies (stop, retry, ...)
				if errExit {
					Log.Warn("Forcing exit, as required.")
					return 1, err
				}
			}
		}
	}

	// Start the watcher goroutine that watches for changes in the
	// K/V and notifies us on a channel.
	errCh := make(chan error, 1)
	pairCh := make(chan consulapi.KVPairs)
	quitCh := make(chan struct{})
	// This channel is what is sent to when a process exits that we
	// are running. We start it out as `nil` since we have no process.
	var exitCh chan int
	defer close(quitCh)

	go wrapper.Watch(
		reload, errExit, pairCh, quitCh, errCh,
	)

	for {
		var pairs consulapi.KVPairs

		// Wait for new pairs to come on our channel or an error
		// to occur.
		select {
		case exit := <-exitCh:
			return exit, nil
		case pairs = <-pairCh:
		case err := <-errCh:
			return 0, err
		}

		cmdEnv := wrapper.loadEnv(pairs)
		if cmdEnv == nil {
			// Nothing changed
			continue
		}

		Log.Info("Configuration changed, reload the process.")
		if wrapper.cmd != nil {
			if !reload {
				Log.Info("We don't want to reload the process... just ignore.")
				continue
			}
			wrapper.reloadProcess()
			wrapper.cmd = nil
		}

		// Keeping in touch Consul
		wrapper.storeMetadata(command)

		// Create a new exitCh so that previously invoked commands
		// (if any) don't cause us to exit, and start a goroutine
		// to wait for that process to end.
		exitCh = make(chan int, 1)
		wrapper.Fork(command, cmdEnv, exitCh)
		// Acquire process metrics and store it in influx db
		go wrapper.watchMetrics(quitCh)
	}

	return 0, nil
}
