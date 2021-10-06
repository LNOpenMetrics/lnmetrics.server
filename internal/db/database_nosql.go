package db

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/OpenLNMetrics/ln-metrics-server/graph/model"
	"github.com/OpenLNMetrics/ln-metrics-server/pkg/db"
)

type NoSQLDatabase struct {
	metricsKey map[uint]string
}

// Create a new instance of the NOSql database
func NewNoSQLDB(options map[string]interface{}) (*NoSQLDatabase, error) {
	path, found := options["path"]
	if !found {
		return nil, fmt.Errorf("DB Path not specified in the options conf")
	}

	if err := db.GetInstance().InitDB(path.(string)); err != nil {
		return nil, err
	}

	return &NoSQLDatabase{
		map[uint]string{1: "metric_one"},
	}, nil
}

// In the NO sql database, at list for the moment we don't need to
// make a schema. The data are the schema it self.
func (instance NoSQLDatabase) CreateMetricOne(options *map[string]interface{}) error {
	return nil
}

// Insert the metricModel in the db
func (instance NoSQLDatabase) InsertMetricOne(toInsert *model.MetricOne) error {
	key := strings.Join([]string{toInsert.NodeID, instance.metricsKey[1]}, "_")
	jsonVal, err := json.Marshal(toInsert)
	if err != nil {
		return err
	}

	if err := db.GetInstance().PutValue(key, string(jsonVal)); err != nil {
		return err
	}
	return nil
}

// Get all the node ids that have some metrics stored in the server
func (instance NoSQLDatabase) GetIDs() ([]*string, error) {
	// we need to implement another method inside the leveldb interface
	return make([]*string, 0), nil
}

// Get all the metric of the node with a specified id
func (instance NoSQLDatabase) GetMetricOne(withId string) (*model.MetricOne, error) {
	metricOne, err := db.GetInstance().GetValue(withId)
	if err != nil {
		return nil, err
	}
	var model model.MetricOne
	if err := json.Unmarshal([]byte(metricOne), &model); err != nil {
		return nil, err
	}
	return &model, nil
}
