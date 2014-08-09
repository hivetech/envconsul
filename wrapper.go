package main

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/armon/consul-api"

	"github.com/hivetech/iron-app/log"
)

type IronConfig struct {
	Namespace    string
	App          string
	ConsulAddr   string
	ConsulDC     string
	Sanitize     bool
	Upcase       bool
	PollInterval int
}

// Compose the application components
type IronWrapper struct {
	// NOTE May be consulapi.KV() is enough
	config  *IronConfig
	network *ConsulNetwork
	db      *InfluxDB
	monitor *Monitor
	cmd     *exec.Cmd
	// An internal env state is kept to check for changes
	env map[string]string
}

func NewIronWrapper(config *IronConfig) (*IronWrapper, error) {
	// Setup Network
	network, err := NewConsulNetwork(config.App, config.ConsulAddr, config.ConsulDC)
	if err != nil {
		return nil, err
	}

	// Setup database
	user, password := "root", "root"
	db, err := NewInfluxDB(fmt.Sprintf("%s.%s", config.Namespace, config.App), user, password)
	if err != nil {
		return nil, err
	}

	return &IronWrapper{
		config:  config,
		network: network,
		db:      db,
		monitor: &Monitor{},
	}, nil
}

func (self *IronWrapper) loadEnv(pairs consulapi.KVPairs) []string {
	newEnv := make(map[string]string)
	for _, pair := range pairs {
		k := strings.TrimPrefix(pair.Key, fmt.Sprintf("%s/%s", self.config.Namespace, self.config.App))
		k = strings.TrimLeft(k, "/")
		if self.config.Sanitize {
			k = InvalidRegexp.ReplaceAllString(k, "_")
		}
		if self.config.Upcase {
			k = strings.ToUpper(k)
		}

		Log.WithFields(logrus.Fields{
			"key":   k,
			"value": string(pair.Value),
		}).Info("Fetched environment variable.")
		newEnv[k] = string(pair.Value)
	}

	// If the environmental variables didn't actually change,
	// then don't do anything.
	if reflect.DeepEqual(self.env, newEnv) {
		Log.Debug("Nothing new in KV store.")
		return nil
	}

	Log.Info("Loading variables into process env.")
	processEnv := os.Environ()
	cmdEnv := make(
		[]string, len(processEnv), len(newEnv)+len(processEnv))
	copy(cmdEnv, processEnv)
	for k, v := range newEnv {
		cmdEnv = append(cmdEnv, fmt.Sprintf("%s=%s", k, v))
	}

	// Replace the env so we can detect future changes
	self.env = newEnv
	return cmdEnv
}

func (self *IronWrapper) reloadProcess() {
	Log.Info("Sending SIGTERM to the process.")
	exited := false
	if err := self.cmd.Process.Signal(syscall.SIGTERM); err == nil {
		// Wait a few seconds for it to exit
		killCh := make(chan struct{})
		go func() {
			defer close(killCh)
			self.cmd.Process.Wait()
		}()

		select {
		case <-killCh:
			exited = true
		case <-time.After(3 * time.Second):
		}
	}

	// If we still haven't exited from a SIGKILL
	if !exited {
		Log.Warn("That wasn't enough, sending SIGKILL.")
		self.cmd.Process.Kill()
	}
}

func (self *IronWrapper) Fork(rawCmd, cmdEnv []string, exitCh chan<- int) (int, error) {
	self.cmd = exec.Command(rawCmd[0], rawCmd[1:]...)
	// TODO Change rawCmd[0] by self.config.App
	self.cmd.Stdout = log.NewLogstream(Log, "stdout", rawCmd[0])
	self.cmd.Stderr = log.NewLogstream(Log, "stderr", rawCmd[0])

	self.cmd.Env = cmdEnv
	err := self.cmd.Start()
	if err != nil {
		return 111, err
	}
	Log.WithFields(logrus.Fields{
		"bin":  self.cmd.Path,
		"args": self.cmd.Args,
		"pid":  self.cmd.Process.Pid,
	}).Info("Started process")

	go func(cmd *exec.Cmd, exitCh chan<- int) {
		err := cmd.Wait()
		if err == nil {
			exitCh <- 0
			return
		}

		if exiterr, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				exitCh <- status.ExitStatus()
				return
			}
		}

		exitCh <- 111
	}(self.cmd, exitCh)

	return 0, nil
}

func (self *IronWrapper) metadataPath() string {
	return fmt.Sprintf("%s/%s/metadata", self.config.Namespace, self.config.App)
}

func (self *IronWrapper) envPath() string {
	return fmt.Sprintf("%s/%s/env", self.config.Namespace, self.config.App)
}

func (self *IronWrapper) Watch(watch, errExit bool, pairCh chan<- consulapi.KVPairs, quitCh <-chan struct{}, errCh chan<- error) {
	// Get the initial list of k/v pairs. We don't do a retryableList
	// here because we want a fast fail if the initial request fails.
	pairs, meta, err := self.network.consul.KV().List(self.envPath(), nil)
	if err != nil {
		errCh <- err
		return
	}

	Log.Debug("Send the initial list out right away.")
	pairCh <- pairs

	// If we're not watching, just return right away
	if !watch {
		return
	}

	curIndex := meta.LastIndex
	Log.WithFields(logrus.Fields{
		"index": curIndex,
	}).Info("Loop forever and watch the keys for changes.")
	for {
		select {
		case <-quitCh:
			return
		default:
		}

		pairs, meta, err = retryableList(
			func() (consulapi.KVPairs, *consulapi.QueryMeta, error) {
				opts := &consulapi.QueryOptions{WaitIndex: curIndex}
				return self.network.consul.KV().List(fmt.Sprintf("%s/%s", self.config.Namespace, self.config.App), opts)
			})
		if err != nil {
			if errExit {
				errCh <- err
				return
			}
		}

		pairCh <- pairs
		curIndex = meta.LastIndex
	}
}

func (self *IronWrapper) consulPut(key, value string) (*consulapi.WriteMeta, error) {
	return self.network.consul.KV().Put(&consulapi.KVPair{
		Key:   key,
		Value: []byte(value),
	}, nil)
}

// TODO error handling
func (self *IronWrapper) storeMetadata(appCmd []string) error {
	metadata_path := self.metadataPath()
	// Acquire metadata
	hostData, _ := self.monitor.MetaData()

	Log.WithFields(logrus.Fields{
		"process": appCmd[0],
		"path":    metadata_path,
	}).Info("Storing process metadata")

	// Store it
	self.consulPut(metadata_path+"/application", self.config.App)
	self.consulPut(metadata_path+"/command", strings.Join(appCmd, " "))
	self.consulPut(metadata_path+"/host/hostname", hostData.Hostname)
	self.consulPut(metadata_path+"/host/os", hostData.OS)
	self.consulPut(metadata_path+"/host/platform/name", hostData.Platform)
	self.consulPut(metadata_path+"/host/platform/family", hostData.PlatformFamily)
	self.consulPut(metadata_path+"/host/platform/version", hostData.PlatformVersion)

	return nil
}

func (self *IronWrapper) watchMetrics(quitCh <-chan struct{}) {
	self.monitor.RegisterProcess(self.cmd.Process.Pid)

	Log.WithFields(logrus.Fields{
		"pid":  self.cmd.Process.Pid,
		"poll": self.config.PollInterval,
	}).Info("Monitor process.")
	for {
		select {
		case <-quitCh:
			return
		default:
		}

		Wait(self.config.PollInterval * 1000)

		// Acquire data
		data, _ := self.monitor.Measure()

		Log.WithFields(logrus.Fields{
			"data": data,
		}).Debug("Save metrics")
		if err := self.db.saveMetrics(data); err != nil {
			Log.Error(err)
		}
	}
}
