package main

import (
  "fmt"
  "strconv"
  "strings"

  "github.com/Sirupsen/logrus"
  "github.com/armon/consul-api"
)

type ConsulNetwork struct {
  client *consulapi.Client
}

func NewConsulNetwork(consulAddr, consulDC string) *ConsulNetwork {
  kvConfig := consulapi.DefaultConfig()
  kvConfig.Address = consulAddr
  kvConfig.Datacenter = consulDC
  client, _ := consulapi.NewClient(kvConfig)

  return &ConsulNetwork{
    client: client,
  }
}

func (c *ConsulNetwork) discoverAndRemember(serviceName, tag, prefix string) {
  service, err := c.searchService(serviceName, tag)
  if err != nil {
    Log.WithFields(logrus.Fields{
      "msg":     err.Error(),
      "service": serviceName,
      "tag":     tag,
    }).Error("Service not found.")
    return
  }

  // FIXME Change "!", this is a current workaround
  if !c.isServiceHealthy(serviceName, service.Checks) && err == nil {
    Log.WithFields(logrus.Fields{
      "node":    service.Node,
      "service": service.Service,
    }).Info("Identified service as healthy.")

    c.injectIntoEnv(
      fmt.Sprintf("%s/%s_HOST", prefix, strings.ToUpper(tag)),
      service.Node.Address,
    )

    c.injectIntoEnv(
      fmt.Sprintf("%s/%s_PORT", prefix, strings.ToUpper(tag)),
      strconv.Itoa(service.Service.Port),
    )
  }
}

func (c *ConsulNetwork) searchService(name, tag string) (*consulapi.ServiceEntry, error) {
  health := c.client.Health()
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

func (c *ConsulNetwork) isServiceHealthy(serviceName string, healthCatalog []*consulapi.HealthCheck) bool {
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

func (c *ConsulNetwork) injectIntoEnv(key, value string) {
  storage := c.client.KV()
  pair := &consulapi.KVPair{Key: key, Value: []byte(value)}
  meta, _ := storage.Put(pair, nil)
  Log.WithFields(logrus.Fields{
    "pair":    pair,
    "elapsed": meta.RequestTime,
  }).Info("Successfully stored pair.")
}
