package db

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
	"github.com/LNOpenMetrics/lnmetrics.utils/db/leveldb"
	"github.com/LNOpenMetrics/lnmetrics.utils/log"
)

type NoSQLDatabase struct {
	metricsKey map[uint]string
	// the db in memory cache it is valid?
	validCache bool
	// the last version of the index cache
	indexCache map[string][]uint
	lock       *sync.Mutex
	dbVersion  int
}

// Create a new instance of the NOSql database
func NewNoSQLDB(options map[string]interface{}) (*NoSQLDatabase, error) {
	path, found := options["path"]
	if !found {
		return nil, fmt.Errorf("DB Path not specified in the options conf")
	}
	log.GetInstance().Info(fmt.Sprintf("Creating local db at %s", path))
	if err := db.GetInstance().InitDB(path.(string)); err != nil {
		return nil, err
	}

	//keys, _ := db.GetInstance().ListOfKeys()

	instance := &NoSQLDatabase{
		map[uint]string{1: "metric_one"},
		false,
		make(map[string][]uint),
		new(sync.Mutex),
		1,
	}
	if err := instance.createIndexDBIfMissin(); err != nil {
		return nil, err
	}
	//FIXME: Create a debug procedure
	/*
		for _, key := range keys {
			fmt.Println(*key)
			tokens := strings.Split(*key, "/")
			nodeId := tokens[0]
			if strings.Contains(nodeId, "data_version") || strings.Contains(nodeId, "node_index") {
				continue
			}
			_ = instance.indexingInDB(nodeId)
		}
	*/
	return instance, nil
}

// Get access to the raw data contained with the specified key
func (instance NoSQLDatabase) GetRawValue(key string) ([]byte, error) {
	return db.GetInstance().GetValueInBytes(key)
}

// Put a raw value with the specified key in the db
func (instance NoSQLDatabase) PutRawValue(key string, value []byte) error {
	return db.GetInstance().PutValueInBytes(key, value)
}

// In the NO sql database, at list for the moment we don't need to
// make a schema. The data are the schema it self.
func (instance NoSQLDatabase) CreateMetricOne(options *map[string]interface{}) error {
	return nil
}

// Init the metric in the database.
// TODO: Migrate to the new data model.
func (instance NoSQLDatabase) InsertMetricOne(toInsert *model.MetricOne) error {

	// we need to index the node in the nodes_index
	if err := instance.indexingInDB(toInsert.NodeID); err != nil {
		return err
	}
	return instance.UpdateMetricOne(toInsert)
}

// Adding new metric  for the node,
//TODO: Support this operation
func (instance NoSQLDatabase) UpdateMetricOne(toInsert *model.MetricOne) error {
	//FIXME: I can run this operation in parallel
	baseKey, err := instance.ItemID(toInsert)
	if err != nil {
		return err
	}
	// It insert information with key {baseKey}/metadata
	if err := instance.extractMetadata(baseKey, toInsert); err != nil {
		return err
	}

	// Store information with key {baseKey}/{timestamp}/{metric_name}
	// and also index the timestamp in the following space
	// {baseKey}/index
	if err := instance.extractNodeMetric(baseKey, toInsert); err != nil {
		return err
	}

	return nil
}

// get the metrics metadata
func (instance *NoSQLDatabase) GetNodes(network string) ([]*model.NodeMetadata, error) {
	//TODO: Ignoring the network for now, different network stay
	// in different db
	nodesID, err := instance.getIndexDB()
	if err != nil {
		return nil, err
	}

	nodesMetadata := make([]*model.NodeMetadata, 0)
	for _, nodeID := range nodesID {
		if strings.Contains(nodeID, "node_index") ||
			strings.Contains(nodeID, "data_version") ||
			strings.Contains(nodeID, "abc") {
			continue
		}

		log.GetInstance().Info(fmt.Sprintf("Finding node with id %s", nodeID))

		nodeMetadata, err := instance.GetNode(network, nodeID, "metric_one")
		if err != nil {
			return nil, err
		}
		nodesMetadata = append(nodesMetadata, nodeMetadata)
	}

	return nodesMetadata, nil

}

func (instance *NoSQLDatabase) GetNode(network string, nodeID string, metricName string) (*model.NodeMetadata, error) {
	// FIXME(vincenzopalazzo) Need to remove the network from the API?
	// in this case different network need to stay in different db
	metadataIndex := strings.Join([]string{nodeID, metricName, "metadata"}, "/")
	jsonNodeMeta, err := db.GetInstance().GetValue(metadataIndex)
	if err != nil {
		return nil, err
	}
	var modelMetadata model.NodeMetadata
	if err := json.Unmarshal([]byte(jsonNodeMeta), &modelMetadata); err != nil {
		return nil, err
	}
	return &modelMetadata, nil
}

// Get all the metric of the node with a specified id
func (instance NoSQLDatabase) GetMetricOne(nodeID string, startPeriod int, endPeriod int) (*model.MetricOne, error) {

	// 1. Take the medatata
	// 2. take the node index of timestamp, and filter by period
	// 3. fill the metric model
	baseKey := strings.Join([]string{nodeID, "metric_one"}, "/")

	metadataNode, err := instance.retreivalMetadata(baseKey)
	if err != nil {
		return nil, err
	}

	modelMetricOne := &model.MetricOne{
		Version:      &metadataNode.Version,
		Name:         "metric_one",
		NodeID:       metadataNode.NodeID,
		Color:        metadataNode.Color,
		NodeAlias:    metadataNode.Alias,
		Network:      &metadataNode.Network,
		OSInfo:       metadataNode.OSInfo,
		NodeInfo:     metadataNode.NodeInfo,
		Address:      metadataNode.Address,
		Timezone:     metadataNode.Timezone,
		UpTime:       make([]*model.Status, 0),
		ChannelsInfo: make([]*model.StatusChannel, 0),
	}

	nodeMetric, err := instance.retreivalNodesMetric(baseKey, "metric_one", startPeriod, endPeriod)
	if err != nil {
		return nil, err
	}

	modelMetricOne.UpTime = nodeMetric.UpTime
	modelMetricOne.ChannelsInfo = nodeMetric.ChannelsInfo

	return modelMetricOne, nil
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
	if versionData <= 1 {
		log.GetInstance().Info("Migration process started")
		if err := instance.migrateFromBlobToTimestamp(); err != nil {
			return err
		}
		log.GetInstance().Info("Migration process Ended with success")
		return instance.SetVersionData()
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

func (instance *NoSQLDatabase) SetVersionData() error {
	if err := db.GetInstance().PutValue("data_version", fmt.Sprint(instance.dbVersion)); err != nil {
		return err
	}
	return nil
}

// Generate the id of an item from a Metrics Model
func (instance *NoSQLDatabase) ItemID(toInsert *model.MetricOne) (string, error) {
	nodeIdentifier := strings.Join([]string{
		toInsert.NodeID,
		toInsert.Name,
	}, "/")
	return nodeIdentifier, nil
}

func (instance *NoSQLDatabase) ContainsIndex(nodeID string, metricName string) bool {
	indexID := strings.Join([]string{nodeID, metricName, "index"}, "/")
	_, err := db.GetInstance().GetValue(indexID)
	if err != nil {
		log.GetInstance().Debug(fmt.Sprintf("Ignoring db error: %s", err))
		return false
	}
	return true
}

func (instance *NoSQLDatabase) createIndexDBIfMissin() error {
	_, err := db.GetInstance().GetValue("node_index")
	if err != nil {
		jsonFakeIndex, err := json.Marshal(instance.indexCache)
		if err != nil {
			return err
		}

		if err := db.GetInstance().PutValue("node_index", string(jsonFakeIndex)); err != nil {
			return err
		}
	}
	return nil
}

// Adding the nodeid to the node_index.
// TODO: We need to lock this method to avoid concurrency
func (instance *NoSQLDatabase) indexingInDB(nodeID string) error {
	// TODO: use cache
	// TODO: during the megrationg create the index too.
	// we can check this with a bool filter to speed up this code
	instance.lock.Lock()
	dbIndex, err := db.GetInstance().GetValue("node_index")
	if err != nil {
		return err
	}
	// FIXME(vincenzopalazzo): We can use the indexCache
	var dbIndexModel map[string][]uint

	if err := json.Unmarshal([]byte(dbIndex), &dbIndexModel); err != nil {
		return err
	}

	_, found := dbIndexModel[nodeID]
	if !found {
		log.GetInstance().Info(fmt.Sprintf("Indexing node with id %s", nodeID))
		// for now there is only the metric one
		dbIndexModel[nodeID] = []uint{1}
		jsonNewIndex, err := json.Marshal(dbIndexModel)
		if err != nil {
			return err
		}
		if err := db.GetInstance().PutValue("node_index", string(jsonNewIndex)); err != nil {
			return err
		}
	}
	instance.lock.Unlock()

	return nil
}

// Return the list of node that are stored in the index
// the leveldb index is stored with the key node_index
//nolint:golint,unused
func (instance *NoSQLDatabase) getIndexDB() ([]string, error) {
	nodesIndex := make([]string, 0)
	// TODO: use cache
	// TODO: during the megrationg create the index too.
	dbIndex, err := db.GetInstance().GetValue("node_index")
	if err != nil {
		return nil, err
	}

	var dbIndexModel map[string][]uint

	if err := json.Unmarshal([]byte(dbIndex), &dbIndexModel); err != nil {
		return nil, err
	}

	for key := range dbIndexModel {
		log.GetInstance().Info(fmt.Sprintf("Key found %s", key))
		nodesIndex = append(nodesIndex, key)
	}
	log.GetInstance().Info(fmt.Sprintf("Index key set size %d", len(nodesIndex)))
	return nodesIndex, nil
}

// called each time that we need a fresh cache
//nolint:golint,unused
func (instance *NoSQLDatabase) invalidateInMemIndex() error {
	instance.validCache = false
	instance.indexCache = make(map[string][]uint)
	return nil
}

// Private function to migrate the nosql data model from a view to another view
func (instance *NoSQLDatabase) migrateFromBlobToTimestamp() error {
	log.GetInstance().Info("Get list of key in the db")
	listNodes, err := db.GetInstance().ListOfKeys()
	log.GetInstance().Info(fmt.Sprintf("Found %d keys in the db", len(listNodes)))
	if err != nil {
		return err
	}

	// Create an empty index
	// if any error occurs during the migration we don't need to lost all the data
	if err := instance.createIndexDBIfMissin(); err != nil {
		return err
	}
	for _, nodeId := range listNodes {
		log.GetInstance().Info(fmt.Sprintf("DB key under analysis is: %s", *nodeId))
		if strings.Contains(*nodeId, "/") ||
			strings.Contains(*nodeId, "node_index") ||
			strings.Contains(*nodeId, "data_version") {
			continue
		}
		log.GetInstance().Info(fmt.Sprintf("Migrating Node %s", *nodeId))
		if err := instance.indexingInDB(*nodeId); err != nil {
			return err
		}

		// NODEID: {metric_name: full_payload}
		metricOneBlob, err := db.GetInstance().GetValue(*nodeId)
		if err != nil {
			return err
		}

		var modelMap map[string]interface{}
		if err := json.Unmarshal([]byte(metricOneBlob), &modelMap); err != nil {
			return err
		}

		metricOneInMap, found := modelMap[instance.metricsKey[1]]
		if !found {
			return fmt.Errorf("Node %s doesn't contains metric one info", *nodeId)
		}

		metricOneCore, err := json.Marshal(metricOneInMap)
		if err != nil {
			return err
		}

		var metricOne model.MetricOne
		if err := json.Unmarshal(metricOneCore, &metricOne); err != nil {
			return err
		}

		newNodeID, err := instance.ItemID(&metricOne)
		if err != nil {
			return err
		}

		if err := instance.extractMetadata(newNodeID, &metricOne); err != nil {
			return err
		}

		if err := instance.extractNodeMetric(newNodeID, &metricOne); err != nil {
			return err
		}

		if err := db.GetInstance().DeleteValue(*nodeId); err != nil {
			return err
		}

	}

	return nil
}

// Private function that a metric on store only the meta information of the node
// the key to store this information it is nodeID/metadata
func (instance *NoSQLDatabase) extractMetadata(itemID string, metricOne *model.MetricOne) error {
	now := int(time.Now().Unix())

	version := 0
	network := "unknown"

	if metricOne.Version != nil {
		version = *metricOne.Version
	}

	if metricOne.Network != nil {
		network = *metricOne.Network
	}

	metadata := model.NodeMetadata{
		Version:    version,
		Network:    network,
		NodeID:     metricOne.NodeID,
		Alias:      metricOne.NodeAlias,
		Color:      metricOne.Color,
		OSInfo:     metricOne.OSInfo,
		NodeInfo:   metricOne.NodeInfo,
		Address:    metricOne.Address,
		Timezone:   metricOne.Timezone,
		LastUpdate: now,
	}

	metaJson, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	metadataID := strings.Join([]string{itemID, "metadata"}, "/")

	if err := db.GetInstance().PutValue(metadataID, string(metaJson)); err != nil {
		return err
	}
	log.GetInstance().Info(fmt.Sprintf("Insert Node (%s) medatata with id %s", metadata.NodeID, metadataID))
	return nil
}

func (instance *NoSQLDatabase) retreivalMetadata(itemID string) (*model.NodeMetadata, error) {
	metadataKey := strings.Join([]string{itemID, "metadata"}, "/")
	nodeMetadataJson, err := db.GetInstance().GetValue(metadataKey)
	if err != nil {
		return nil, err
	}

	var metaModel model.NodeMetadata
	if err := json.Unmarshal([]byte(nodeMetadataJson), &metaModel); err != nil {
		return nil, err
	}
	return &metaModel, nil
}

func (instance *NoSQLDatabase) extractNodeMetric(itemID string, metricOne *model.MetricOne) error {
	// TODO iterate over timestamp and channels
	sizeUpdates := len(metricOne.UpTime)
	listUpTime := make([]*model.Status, 0)
	listTimestamp := make([]int, 0)
	timestampIndex := strings.Join([]string{
		itemID,
		"index",
	}, "/")

	oldTimestamp, err := db.GetInstance().GetValue(timestampIndex)
	if err == nil {
		if err := json.Unmarshal([]byte(oldTimestamp), &listTimestamp); err != nil {
			log.GetInstance().Error(fmt.Sprintf("Error: %s", err))
		}
	}
	for i := 0; i < sizeUpdates; i++ {
		uptime := metricOne.UpTime[i]
		listChannelsInfo := metricOne.ChannelsInfo
		listUpTime = append(listUpTime, uptime)
		timestamp := uptime.Timestamp
		listTimestamp = append(listTimestamp, timestamp)
		metricKey := strings.Join([]string{
			itemID,
			fmt.Sprint(timestamp),
			"metric",
		}, "/")
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

		log.GetInstance().Info(fmt.Sprintf("Insert metric with id %s", metricKey))

	}

	jsonIndex, err := json.Marshal(listTimestamp)
	if err != nil {
		return err
	}

	return db.GetInstance().PutValue(timestampIndex, string(jsonIndex))
}

// Private function to get a single metric given a specific timestamp
func (instance *NoSQLDatabase) retreivalNodeMetric(nodeKey string, timestamp uint, metricName string) (*model.NodeMetric, error) {
	metricKey := strings.Join([]string{nodeKey, fmt.Sprint(timestamp), "metric"}, "/")
	metricJson, err := db.GetInstance().GetValue(metricKey)
	if err != nil {
		return nil, err
	}

	var modelMetric model.NodeMetric
	if err := json.Unmarshal([]byte(metricJson), &modelMetric); err != nil {
		return nil, err
	}
	return &modelMetric, nil

}

// Private function that it is able to get the collection of metric in a period
// expressed in unix time.
func (instance *NoSQLDatabase) retreivalNodesMetric(nodeKey string, metricName string, startPeriod int, endPeriod int) (*model.NodeMetric, error) {
	timestampsKey := strings.Join([]string{nodeKey, "index"}, "/")
	timestampJson, err := db.GetInstance().GetValue(timestampsKey)
	log.GetInstance().Debug(fmt.Sprintf("index of timestamp: %s", timestampJson))
	if err != nil {
		return nil, err
	}

	var modelTimestamp []uint
	if err := json.Unmarshal([]byte(timestampJson), &modelTimestamp); err != nil {
		return nil, err
	}

	modelMetric := &model.NodeMetric{
		Timestamp:    0,
		UpTime:       make([]*model.Status, 0),
		ChannelsInfo: make([]*model.StatusChannel, 0),
	}

	for _, timestamp := range modelTimestamp {
		if (startPeriod == -1 && endPeriod == -1) ||
			(int(timestamp) >= startPeriod && int(timestamp) <= endPeriod) {
			log.GetInstance().Info(fmt.Sprintf("Get metric %s for %s at time %d", metricName, nodeKey, timestamp))
			tmpModelMetric, err := instance.retreivalNodeMetric(nodeKey, timestamp, metricName)
			if err != nil {
				return nil, err
			}

			//FIXME: It is safe? or it make sense?
			modelMetric.Timestamp = int(timestamp)
			modelMetric.UpTime = append(modelMetric.UpTime,
				tmpModelMetric.UpTime...)
			modelMetric.ChannelsInfo = append(modelMetric.ChannelsInfo,
				tmpModelMetric.ChannelsInfo...)

		}
	}

	return modelMetric, nil
}
