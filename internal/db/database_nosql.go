package db

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
	"github.com/LNOpenMetrics/lnmetrics.utils/db/leveldb"
	"github.com/LNOpenMetrics/lnmetrics.utils/log"
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

// Migrate procedure
func (instance *NoSQLDatabase) Migrate() error {
	versionData, err := instance.GetVersionData()
	if versionData < 1 {
		return instance.migrateFromBlobToTimestamp()
	}
	log.GetInstance().Info(fmt.Sprintf("No db migration needed (db version = %d)", versionData))
	return err
}

// Get the version of data stored in the db
func (instance *NoSQLDatabase) GetVersionData() (uint, error) {
	versionStr, err := db.GetInstance().GetValue("data_version")
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseUint(versionStr, 10, 32)
	return uint(value), err
}

// Generate the id of an item from a Metrics Model
func (instance *NoSQLDatabase) ItemID(toInsert *model.MetricOne) (string, error) {
	nodeIdentifier := strings.Join([]string{
		toInsert.NodeID,
		fmt.Sprint(toInsert.LastUpdate),
	}, "/")
	return nodeIdentifier, nil
}

// Private function to migrate the nosql data model from a view to another view
func (instance *NoSQLDatabase) migrateFromBlobToTimestamp() error {
	listNodes, err := db.GetInstance().ListOfKeys()
	if err != nil {
		return err
	}

	for _, nodeId := range listNodes {
		log.GetInstance().Info(fmt.Sprintf("Migrating Node %s", *nodeId))
		metricOneBlob, err := db.GetInstance().GetValue(*nodeId)
		if err != nil {
			return err
		}
		var metricOne model.MetricOne
		if err := json.Unmarshal([]byte(metricOneBlob), &metricOne); err != nil {
			return err
		}
		newNodeID, err := instance.ItemID(&metricOne)
		if err != nil {
			return err
		}

		if err := instance.extractMetadata(&metricOne); err != nil {
			return err
		}

		if err := instance.extractNodeMetric(newNodeID, &metricOne); err != nil {
			return err
		}

	}

	return nil
}

// Private function that a metric on store only the meta information of the node
// the key to store this information it is nodeID/metadata
func (instance *NoSQLDatabase) extractMetadata(metricOne *model.MetricOne) error {
	nodeInfo := model.NodeImpInfo{
		Implementation: "unknown",
		Version:        "unknown",
	}

	metadata := model.NodeMetadata{
		NodeID:   metricOne.NodeID,
		Alias:    metricOne.NodeAlias,
		Color:    metricOne.Color,
		OSInfo:   metricOne.OSInfo,
		NodeInfo: &nodeInfo,
		Address:  make([]*model.NodeAddress, 0),
	}

	metaJson, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	metadataID := strings.Join([]string{metadata.NodeID, "metadata"}, "/")

	if err := db.GetInstance().PutValue(metadataID, string(metaJson)); err != nil {
		return err
	}
	log.GetInstance().Info(fmt.Sprintf("Insert Node (%s) medatata with id %s", metadata.NodeID, metadataID))
	return nil
}

func (instance *NoSQLDatabase) extractNodeMetric(itemID string, metricOne *model.MetricOne) error {
	// TODO iterate over timestamp and channels
	sizeUpdates := len(metricOne.UpTime)
	listUpTime := make([]*model.Status, 0)
	listChannelsInfo := make([]*model.StatusChannel, 0)
	timestamp := -1
	for i := 0; i < sizeUpdates; i++ {
		uptime := metricOne.UpTime[i]
		channelsInfo := metricOne.ChannelsInfo[i]
		listUpTime = append(listUpTime, uptime)
		listChannelsInfo = append(listChannelsInfo, channelsInfo)
		if timestamp == -1 {
			timestamp = uptime.Timestamp
		}
		if uptime.Event == "on_update" {
			metricKey := strings.Join([]string{
				itemID,
				fmt.Sprint(timestamp)}, "/")
			nodeMetric := model.NodeMetric{
				Timestamp:    timestamp,
				UpTime:       listUpTime,
				ChannelsInfo: listChannelsInfo,
			}

			jsonMetric, err := json.Marshal(nodeMetric)
			if err != nil {
				return err
			}

			// TODO store somewhere this index, or keep in memory.
			if err := db.GetInstance().PutValue(metricKey, string(jsonMetric)); err != nil {
				return err
			}

			listUpTime = nil
			listChannelsInfo = nil
			timestamp = -1

			log.GetInstance().Info(fmt.Sprintf("Insert metric with id %s", metricKey))
			continue
		}
	}
	return nil
}
