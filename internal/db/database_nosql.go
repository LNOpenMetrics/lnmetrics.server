package db

import (
	"encoding/json"
	"fmt"

	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
	"github.com/LNOpenMetrics/lnmetrics.utils/db/leveldb"
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
	key := toInsert.NodeID
	metricOne := make(map[string]interface{})
	metricOne[instance.metricsKey[1]] = toInsert
	jsonVal, err := json.Marshal(metricOne)
	if err != nil {
		return err
	}

	if err := db.GetInstance().PutValue(key, string(jsonVal)); err != nil {
		return err
	}
	return nil
}

// Get all the node ids that have some metrics stored in the server
func (instance NoSQLDatabase) GetNodesID() ([]*string, error) {
	return db.GetInstance().ListOfKeys()
}

// Get all the metric of the node with a specified id
func (instance NoSQLDatabase) GetMetricOne(withId string) (*model.MetricOne, error) {
	// FIXME: In this case the node info can grow when it contains different
	// metric type, a solution could be adding a indirection level.
	nodeInfo, err := db.GetInstance().GetValue(withId)
	if err != nil {
		return nil, fmt.Errorf("Research by node id %s return the following error: %s", withId, err)
	}
	var modelMap map[string]interface{}
	if err := json.Unmarshal([]byte(nodeInfo), &modelMap); err != nil {
		return nil, err
	}
	metricOneMap, found := modelMap[instance.metricsKey[1]]
	if !found {
		return nil, fmt.Errorf("Node %s doesn't contains metric one info", withId)
	}
	metricOneJson, err := json.Marshal(metricOneMap)
	if err != nil {
		return nil, err
	}
	var model model.MetricOne
	if err := json.Unmarshal(metricOneJson, &model); err != nil {
		return nil, err
	}
	return &model, nil
}

// close the connection with database
func (instance NoSQLDatabase) CloseDatabase() error {
	return db.GetInstance().CloseDatabase()
}

// Erase database
func (instance NoSQLDatabase) EraseDatabase() error {
	return db.GetInstance().EraseDatabase()
}

// Close and aftert erase the connection with the database
func (instance NoSQLDatabase) EraseAfterCloseDatabase() error {
	return db.GetInstance().EraseAfterCloseDatabse()
}
