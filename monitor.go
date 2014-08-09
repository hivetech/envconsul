package main

import (
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/armon/consul-api"
	"github.com/shirou/gopsutil"
)

type Monitor struct {
	namespace string
	app       string
	kv        *consulapi.KV
	db        *InfluxDB
}

func NewMonitor(app string, consul *consulapi.Client) (*Monitor, error) {
	user, password := "root", "root"
	db, err := NewInfluxDB(fmt.Sprintf("iron-app.%s", app), user, password)
	if err != nil {
		return nil, err
	}

	return &Monitor{
		namespace: "iron-app",
		app:       app,
		kv:        consul.KV(),
		db:        db,
	}, nil
}

func (self *Monitor) metadataPath() string {
	return fmt.Sprintf("%s/%s/metadata", self.namespace, self.app)
}

func (self *Monitor) consulPut(key, value string) (*consulapi.WriteMeta, error) {
	return self.kv.Put(&consulapi.KVPair{
		Key:   key,
		Value: []byte(value),
	}, nil)
}

// TODO error handling
func (self *Monitor) storeMetadata(appCmd []string) error {
	metadata_path := self.metadataPath()
	// Acquire metadatas
	hostData, _ := gopsutil.HostInfo()

	Log.WithFields(logrus.Fields{
		"process": appCmd[0],
		"path":    metadata_path,
	}).Info("Storing process metadata")

	// Store it
	self.consulPut(metadata_path+"/application", self.app)
	self.consulPut(metadata_path+"/command", strings.Join(appCmd, " "))
	self.consulPut(metadata_path+"/host/hostname", hostData.Hostname)
	self.consulPut(metadata_path+"/host/os", hostData.OS)
	self.consulPut(metadata_path+"/host/platform/name", hostData.Platform)
	self.consulPut(metadata_path+"/host/platform/family", hostData.PlatformFamily)
	self.consulPut(metadata_path+"/host/platform/version", hostData.PlatformVersion)

	return nil
}

func (self *Monitor) watchMetrics(pid int, quitCh <-chan struct{}, pollInterval int) error {
	var pidi32 interface{}
	pidi32 = int32(pid)
	process, _ := gopsutil.NewProcess(pidi32.(int32))

	Log.WithFields(logrus.Fields{
		"pid":  pid,
		"poll": pollInterval,
	}).Info("Monitor process.")
	for {
		select {
		case <-quitCh:
			return nil
		default:
		}

		Wait(pollInterval)

		// Acquire data
		io_count, _ := process.IOCounters()
		data := make(map[string]interface{})
		data["io.read.count"] = io_count.ReadCount
		data["io.write.count"] = io_count.WriteCount

		Log.WithFields(logrus.Fields{
			"data": data,
		}).Debug("Save metrics")
		if err := self.db.saveMetrics(data); err != nil {
			Log.Error(err)
		}
	}

	return fmt.Errorf("Why did we quit the loop ?")
}
