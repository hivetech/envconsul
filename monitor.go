package main

import (
	"github.com/shirou/gopsutil"
)

type Monitor struct {
	process *gopsutil.Process
}

func (self *Monitor) RegisterProcess(pid int) error {
	var err error
	var pidi32 interface{}
	pidi32 = int32(pid)
	self.process, err = gopsutil.NewProcess(pidi32.(int32))
	return err
}

// TODO More Metadata
func (self *Monitor) MetaData() (*gopsutil.HostInfoStat, error) {
	return gopsutil.HostInfo()
}

// TODO Much more data !!
func (self *Monitor) Measure() (map[string]interface{}, error) {
	io_count, _ := self.process.IOCounters()
	data := make(map[string]interface{})
	data["io.read.count"] = io_count.ReadCount
	data["io.write.count"] = io_count.WriteCount
	return data, nil
}
