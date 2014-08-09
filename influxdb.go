package main

import (
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	influxClient "github.com/influxdb/influxdb/client"
)

type InfluxDB struct {
	influxClient.Client
}

func NewInfluxDB(dbName, user, password string) (*InfluxDB, error) {
	// TODO Default values
	influxHost := os.Getenv("INFLUXDB_HOST")
	influxPort := os.Getenv("INFLUXDB_PORT")
	client, _ := influxClient.NewClient(&influxClient.ClientConfig{
		Host:     fmt.Sprintf("%s:%s", influxHost, influxPort),
		Database: dbName,
		Username: user,
		Password: password,
	})
	dbs, _ := client.GetDatabaseList()
	for i, db := range dbs {
		if db["name"] == dbName {
			Log.Info("Found existing database")
			break
		} else if i == len(dbs)-1 {
			Log.Warn("Database not found, creating it")
			if err := client.CreateDatabase(dbName); err != nil {
				return nil, err
			}
		}
	}
	// Make sure everything is ok
	if err := client.Ping(); err != nil {
		return nil, err
	} else {
		Log.WithFields(logrus.Fields{
			"db":       dbName,
			"user":     user,
			"password": password,
		}).Info("Connected to influx database")
	}
	return &InfluxDB{*client}, nil
}

func (self *InfluxDB) saveMetrics(data map[string]interface{}) error {
	// Creating influxdb series
	series := []*influxClient.Series{}
	for table, value := range data {
		series = append(series, &influxClient.Series{
			Name:    table,
			Columns: []string{"time", "value"},
			Points: [][]interface{}{
				{Now(), value},
			},
		})
	}

	if err := self.WriteSeries(series); err != nil {
		// FIXME Sometimes fails (but we go on anyway)
		return err
	}
	return nil
}
