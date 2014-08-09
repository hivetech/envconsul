package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/armon/consul-api"
)

type ConsulNetwork struct {
	namespace string
	app       string
	consul    *consulapi.Client
}

func NewConsulNetwork(app, consulAddr, consulDC string) (*ConsulNetwork, error) {
	kvConfig := consulapi.DefaultConfig()
	kvConfig.Address = consulAddr
	kvConfig.Datacenter = consulDC
	client, err := consulapi.NewClient(kvConfig)
	if err != nil {
		return nil, err
	}

	return &ConsulNetwork{
		namespace: "iron-app",
		app:       app,
		consul:    client,
	}, nil
}

func (self *ConsulNetwork) discoverAndRemember(serviceName, tag string) error {
	service, err := self.searchService(serviceName, tag)
	if err != nil {
		return err
	}

	if self.isServiceHealthy(serviceName, service.Checks) && err == nil {
		Log.WithFields(logrus.Fields{
			"node":    service.Node,
			"service": service.Service,
		}).Info("Identified service as healthy.")

		self.injectIntoEnv(
			fmt.Sprintf("%s/%s/%s_HOST", self.namespace, self.app, strings.ToUpper(serviceName)),
			service.Node.Address,
		)

		if service.Service.Port != 0 {
			self.injectIntoEnv(
				fmt.Sprintf("%s/%s/%s_PORT", self.namespace, self.app, strings.ToUpper(serviceName)),
				strconv.Itoa(service.Service.Port),
			)
		} else {
			Log.Warn("Service port not found, skipping.")
		}
	} else {
		if err != nil {
			return err
		} else {
			return fmt.Errorf("service not healthy")
		}
	}

	return nil
}

func (self *ConsulNetwork) searchService(name, tag string) (*consulapi.ServiceEntry, error) {
	health := self.consul.Health()
	servicesCheck, _, err := health.Service(name, tag, false, nil)
	if err != nil {
		return nil, err
	}
	if len(servicesCheck) == 0 {
		return nil, fmt.Errorf("service %s:%s not found", name, tag)
	}

	// NOTE Returns the first occurence, but if many, this shoulb based on smarter
	// criterias (like load balancing or proximity)
	return servicesCheck[0], nil
}

func (self *ConsulNetwork) isServiceHealthy(serviceName string, healthCatalog []*consulapi.HealthCheck) bool {
	for i := range healthCatalog {
		Log.Debugf("Inspecting report for %s", healthCatalog[i].Name)
		if healthCatalog[i].ServiceName == serviceName {
			Log.WithFields(logrus.Fields{
				"id":           healthCatalog[i].CheckID,
				"name":         healthCatalog[i].Name,
				"service-name": healthCatalog[i].ServiceName,
				"service-id":   healthCatalog[i].ServiceID,
				"status":       healthCatalog[i].Status,
				"Notes":        healthCatalog[i].Notes,
				"Output":       healthCatalog[i].Output,
			}).Info("Found health report.")
			return healthCatalog[i].Status == "passing"
		}
	}

	Log.Warningf("No check report for service %s.", serviceName)
	return false
}

func (self *ConsulNetwork) injectIntoEnv(key, value string) {
	storage := self.consul.KV()
	pair := &consulapi.KVPair{Key: key, Value: []byte(value)}
	meta, _ := storage.Put(pair, nil)
	Log.WithFields(logrus.Fields{
		"pair":    pair,
		"elapsed": meta.RequestTime,
	}).Info("Successfully stored pair.")
}
