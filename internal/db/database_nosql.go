package db

import (
	"encoding/json"
	"fmt"
	"github.com/LNOpenMetrics/lnmetrics.utils/utime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/config"

	db "github.com/LNOpenMetrics/lnmetrics.utils/db/leveldb"
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
	// btcDB instance of the database where to
	// store the bitcon mainet information.
	btcDB db.Database
	// testBtcDB instance of the database where to
	// store the infomation about the bitcon testnet.
	testBtcDB db.Database
}

// NewNoSQLDB Create a new instance of the NOSql database
func NewNoSQLDB(options map[string]interface{}) (*NoSQLDatabase, error) {
	path, found := options["path"]
	if !found {
		return nil, fmt.Errorf("DB Path not specified in the options conf")
	}
	rootPath := path.(string)
	testBtcPath := strings.Join([]string{rootPath, "testnet"}, "/")
	log.GetInstance().Info(fmt.Sprintf("Creating local db for Bitcoin network at %s", rootPath))
	log.GetInstance().Info(fmt.Sprintf("Creating local db for Bitcoin testnet network at %s", testBtcPath))

	testBtcDB, err := db.NewInstance(testBtcPath)
	if err != nil {
		return nil, err
	}

	bitcoinDB, err := db.NewInstance(rootPath)
	if err != nil {
		return nil, err
	}

	instance := &NoSQLDatabase{
		metricsKey: map[uint]string{1: "metric_one"},
		validCache: false,
		indexCache: make(map[string][]uint),
		lock:       new(sync.Mutex),
		dbVersion:  2,
		btcDB:      *bitcoinDB,
		testBtcDB:  *testBtcDB,
	}

	// FIXME: give the possibility to select only some network
	for _, network := range []string{"bitcoin", "testnet"} {
		if err := instance.createIndexDBIfMissin(network); err != nil {
			return nil, err
		}
	}
	return instance, nil
}

// GetRawValue Get access to the raw data contained with the specified key
func (self NoSQLDatabase) GetRawValue(network string, key string) ([]byte, error) {
	switch network {
	case "bitcoin":
		return self.btcDB.GetValueInBytes(key)
	case "testnet":
		return self.testBtcDB.GetValueInBytes(key)
	default:
		return nil, fmt.Errorf("db not found for the network %s", network)
	}
}

// DeleteRawValue Get access to the raw data contained with the specified key
func (self NoSQLDatabase) DeleteRawValue(network string, key string) error {
	switch network {
	case "bitcoin":
		return self.btcDB.DeleteValue(key)
	case "testnet":
		return self.testBtcDB.DeleteValue(key)
	default:
		return fmt.Errorf("db not found for the network %s", network)
	}
}

// PutRawValue Put a raw value with the specified key in the db
func (self NoSQLDatabase) PutRawValue(network string, key string, value []byte) error {
	switch network {
	case "bitcoin":
		return self.btcDB.PutValueInBytes(key, value)
	case "testnet":
		return self.testBtcDB.PutValueInBytes(key, value)
	default:
		return fmt.Errorf("db not found for the network %s", network)
	}
}

func (self NoSQLDatabase) RawIterateThrough(network string, start string, end string, callback func(string) error) error {
	switch network {
	case "bitcoin":
		return self.btcDB.IterateThrough(start, end, callback)
	case "testnet":
		return self.testBtcDB.IterateThrough(start, end, callback)
	default:
		return fmt.Errorf("db not found for the network %s", network)
	}
}

// CreateMetricOne In the NO sql database, at list for the moment we don't need to
// make a schema. The data are the schema it self.
func (instance NoSQLDatabase) CreateMetricOne(network string, options *map[string]interface{}) error {
	return nil
}

// InsertMetricOne Init the metric in the database.
func (instance NoSQLDatabase) InsertMetricOne(network string, toInsert *model.MetricOne) error {
	// we need to index the node in the nodes_index
	if err := instance.indexingInDB(network, toInsert.NodeID); err != nil {
		return err
	}
	return instance.UpdateMetricOne(network, toInsert)
}

// UpdateMetricOne Adding new metric  for the node,
func (instance NoSQLDatabase) UpdateMetricOne(network string, toInsert *model.MetricOne) error {
	//FIXME: I can run this operation in parallel
	baseKey, err := instance.ItemID(toInsert)
	if err != nil {
		return err
	}
	// It insert information with key {baseKey}/metadata
	if err := instance.extractMetadata(network, baseKey, toInsert); err != nil {
		return err
	}

	// Store information with key {baseKey}/{timestamp}/{metric_name}
	// and also index the timestamp in the following space
	// {baseKey}/index
	if err := instance.extractNodeMetric(network, baseKey, toInsert); err != nil {
		return err
	}

	return nil
}

// GetNodes get the metrics metadata
func (instance *NoSQLDatabase) GetNodes(network string) ([]*model.NodeMetadata, error) {
	//TODO: Ignoring the network for now, different network stay
	// in different db
	nodesID, err := instance.getIndexDB(network)
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

		log.GetInstance().Infof("Finding node with id %s", nodeID)

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
	jsonNodeMeta, err := instance.GetRawValue(network, metadataIndex)
	if err != nil {
		return nil, err
	}

	var modelMetadata model.NodeMetadata
	if err := json.Unmarshal(jsonNodeMeta, &modelMetadata); err != nil {
		return nil, err
	}
	return &modelMetadata, nil
}

func (self *NoSQLDatabase) DeleteNode(network string, metricName string, nodeID string) error {
	metadataIndex := strings.Join([]string{nodeID, metricName, "metadata"}, "/")
	if err := self.DeleteRawValue(network, metadataIndex); err != nil {
		return err
	}

	if err := self.DeleteMetricOne(network, nodeID); err != nil {
		return err
	}
	return self.deindexingInDB(network, nodeID)
}

// GetMetricOne Get all the metric of the node with a specified id
func (instance NoSQLDatabase) GetMetricOne(network string, nodeID string, startPeriod int, endPeriod int) (*model.MetricOne, error) {

	// 1. Take the medatata
	// 2. take the node index of timestamp, and filter by period
	// 3. fill the metric model
	baseKey := strings.Join([]string{nodeID, "metric_one"}, "/")

	metadataNode, err := instance.retreivalMetadata(network, baseKey)
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

	nodeMetric, err := instance.retrievalNodesMetric(network, baseKey, "metric_one", startPeriod, endPeriod)
	if err != nil {
		return nil, err
	}

	modelMetricOne.UpTime = nodeMetric.UpTime
	modelMetricOne.ChannelsInfo = nodeMetric.ChannelsInfo

	return modelMetricOne, nil
}

func (self *NoSQLDatabase) DeleteMetricOne(network string, nodeID string) error {
	baseKey := strings.Join([]string{nodeID, config.MetricOneInputSuffic}, "/")
	return self.deleteNodesMetric(network, baseKey)
}

func (instance *NoSQLDatabase) GetMetricOneInfo(network string, nodeID string, first int, last int) (*model.MetricOneInfo, error) {
	// 1. take the node index of timestamp, and filter by period
	// 1.1 check the period used
	// 2. fill the metric model and return it
	baseKey := strings.Join([]string{nodeID, "metric_one"}, "/")
	nodeMetric, err := instance.retrievalNodesMetric(network, baseKey, "metric_one", first, last)
	if err != nil {
		return nil, err
	}

	index := instance.getIndex(network, baseKey)
	lastTimestamp := index[len(index)-1]
	last = int(utime.AddToTimestamp(int64(last), 10*time.Minute))
	hasNext := lastTimestamp >= uint(last)

	modelMetricOne := model.MetricOneInfo{
		UpTime:       nodeMetric.UpTime,
		ChannelsInfo: nodeMetric.ChannelsInfo,
		PageInfo: &model.PageInfo{
			// The last is the database is included, so we advance by one minute
			StartCursor: int(utime.AddToTimestamp(int64(last), 1*time.Minute)),
			// FIXME: make the period to work with timestamp fixed by some config
			EndCursor: int(utime.AddToTimestamp(int64(last), 6*30*time.Minute)),
			// if we found something's we can continue!
			HasNext: hasNext,
		},
	}
	jsonStr, err := json.Marshal(modelMetricOne)
	if err != nil {
		log.GetInstance().Errorf("Error during debug operation %s", err)
	} else {
		log.GetInstance().Infof("Metric Info generated is %s", string(jsonStr))
	}
	return &modelMetricOne, nil
}

func (instance *NoSQLDatabase) GetMetricOneOutput(network string, nodeID string) (*model.MetricOneOutput, error) {
	metricKey := strings.Join([]string{nodeID, config.MetricOneOutputSuffix}, "/")
	rawOutput, err := instance.GetRawValue(network, metricKey)
	if err != nil {
		return nil, err
	}
	var metricOneModel model.MetricOneOutput
	if err := json.Unmarshal(rawOutput, &metricOneModel); err != nil {
		return nil, err
	}

	return &metricOneModel, nil
}

func (instance *NoSQLDatabase) GetMetricOneIndex(network string, nodeID string) ([]int64, error) {
	indexKey := strings.Join([]string{nodeID, "metric_one", "index"}, "/")
	indexRaw, err := instance.GetRawValue(network, indexKey)
	if err != nil {
		return nil, err
	}
	var index []int64
	if err := json.Unmarshal(indexRaw, &index); err != nil {
		return nil, err
	}
	return index, nil
}

// CloseDatabase close the connection with database
func (self NoSQLDatabase) CloseDatabase(network string) error {
	switch network {
	case "bitcoin":
		return self.btcDB.CloseDatabase()
	case "testnet":
		return self.testBtcDB.CloseDatabase()
	default:
		return fmt.Errorf("db not found for the network %s", network)
	}
}

// EraseDatabase Erase database
func (self NoSQLDatabase) EraseDatabase(network string) error {
	switch network {
	case "bitcoin":
		return self.btcDB.EraseDatabase()
	case "testnet":
		return self.testBtcDB.EraseDatabase()
	default:
		return fmt.Errorf("db not found for the network %s", network)
	}
}

// EraseAfterCloseDatabase Close and aftert erase the connection with the database
func (self NoSQLDatabase) EraseAfterCloseDatabase(network string) error {
	switch network {
	case "bitcoin":
		return self.btcDB.EraseAfterCloseDatabse()
	case "testnet":
		return self.testBtcDB.EraseAfterCloseDatabse()
	default:
		return fmt.Errorf("db not found for the network %s", network)
	}
}

// Migrate procedure
func (instance *NoSQLDatabase) Migrate(network string) error {
	versionData, err := instance.GetVersionData(network)
	if versionData <= 1 {
		log.GetInstance().Info("Migration process started")
		if err := instance.migrateFromBlobToTimestamp(network); err != nil {
			return err
		}
		log.GetInstance().Info("Migration process Ended with success")
		return instance.SetVersionData(network)
	}
	log.GetInstance().Info(fmt.Sprintf("No db migration needed (db version = %d)", versionData))
	return err
}

// GetVersionData Get the version of data stored in the db
func (self *NoSQLDatabase) GetVersionData(network string) (uint, error) {
	versionStr, err := self.GetRawValue(network, "data_version")
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseUint(string(versionStr), 10, 32)
	return uint(value), err
}

func (self *NoSQLDatabase) SetVersionData(network string) error {
	versionStr := fmt.Sprint(self.dbVersion)
	if err := self.PutRawValue(network, "data_version", []byte(versionStr)); err != nil {
		return err
	}
	return nil
}

// ItemID Generate the id of an item from a Metrics Model
func (instance *NoSQLDatabase) ItemID(toInsert *model.MetricOne) (string, error) {
	nodeIdentifier := strings.Join([]string{
		toInsert.NodeID,
		toInsert.Name,
	}, "/")
	return nodeIdentifier, nil
}

func (instance *NoSQLDatabase) ContainsIndex(network string, nodeID string, metricName string) bool {
	indexID := strings.Join([]string{nodeID, metricName, "index"}, "/")
	_, err := instance.GetRawValue(network, indexID)
	if err != nil {
		log.GetInstance().Debug(fmt.Sprintf("Ignoring db error: %s", err))
		return false
	}
	return true
}

func (instance *NoSQLDatabase) createIndexDBIfMissin(network string) error {
	_, err := instance.GetRawValue(network, "node_index")
	if err != nil {
		jsonFakeIndex, err := json.Marshal(instance.indexCache)
		if err != nil {
			return err
		}

		if err := instance.PutRawValue(network, "node_index", jsonFakeIndex); err != nil {
			return err
		}
	}
	return nil
}

// Adding the node GetMetricOneInfoid to the node_index.
func (instance *NoSQLDatabase) indexingInDB(network string, nodeID string) error {
	// TODO: use cache
	// TODO: during the megrationg create the index too.
	// we can check this with a bool filter to speed up this code
	instance.lock.Lock()
	dbIndex, err := instance.GetRawValue(network, "node_index")
	if err != nil {
		return err
	}
	// FIXME(vincenzopalazzo): We can use the indexCache
	var dbIndexModel map[string][]uint

	if err := json.Unmarshal(dbIndex, &dbIndexModel); err != nil {
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
		if err := instance.PutRawValue(network, "node_index", jsonNewIndex); err != nil {
			return err
		}
	}
	instance.lock.Unlock()
	return nil
}

func (self *NoSQLDatabase) deindexingInDB(network string, nodeID string) error {
	self.lock.Lock()

	indexDB, err := self.getRawIndexDB(network)
	if err != nil {
		return err
	}
	delete(indexDB, nodeID)

	if err := self.setRawIndexDB(network, indexDB); err != nil {
		return err
	}
	self.lock.Unlock()
	return nil
}

func (self *NoSQLDatabase) getRawIndexDB(network string) (map[string][]uint, error) {
	dbIndex, err := self.GetRawValue(network, "node_index")
	if err != nil {
		return nil, err
	}

	var dbIndexModel map[string][]uint
	if err := json.Unmarshal(dbIndex, &dbIndexModel); err != nil {
		return nil, err
	}
	return dbIndexModel, nil
}

func (self *NoSQLDatabase) setRawIndexDB(network string, idx map[string][]uint) error {
	jsonStr, err := json.Marshal(idx)
	if err != nil {
		return err
	}
	return self.PutRawValue(network, "node_index", jsonStr)
}

// Return the list of node that are stored in the index
// the leveldb index is stored with the key node_index
//
//nolint:golint,unused
func (instance *NoSQLDatabase) getIndexDB(network string) ([]string, error) {
	nodesIndex := make([]string, 0)
	// TODO: use cache
	// TODO: during the migration create the index too.
	dbIndexModel, err := instance.getRawIndexDB(network)
	if err != nil {
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
//
//nolint:golint,unused
func (instance *NoSQLDatabase) invalidateInMemIndex() error {
	instance.validCache = false
	instance.indexCache = make(map[string][]uint)
	return nil
}

// Private function to migrate the nosql data model from a view to another view
func (instance *NoSQLDatabase) migrateFromBlobToTimestamp(network string) error {
	log.GetInstance().Info("Get list of key in the db")
	var listNodes []*string
	var err error

	switch network {
	case "bitcoin":
		listNodes, err = instance.btcDB.ListOfKeys()
	case "testnet":
		listNodes, err = instance.testBtcDB.ListOfKeys()
	default:
		return fmt.Errorf("db not found for the network %s", network)
	}

	if err != nil {
		return err
	}

	// Create an empty index
	// if any error occurs during the migration we don't need to lost all the data
	if err := instance.createIndexDBIfMissin(network); err != nil {
		return err
	}
	for _, nodeId := range listNodes {
		log.GetInstance().Debugf("DB key under analysis is: %s", *nodeId)
		if strings.Contains(*nodeId, "/") ||
			strings.Contains(*nodeId, "node_index") ||
			strings.Contains(*nodeId, "data_version") {
			continue
		}
		log.GetInstance().Debug(fmt.Sprintf("Migrating Node %s", *nodeId))
		if err := instance.indexingInDB(network, *nodeId); err != nil {
			return err
		}

		// NODEID: {metric_name: full_payload}
		metricOneBlob, err := instance.GetRawValue(network, *nodeId)
		if err != nil {
			return err
		}

		var modelMap map[string]interface{}
		if err := json.Unmarshal(metricOneBlob, &modelMap); err != nil {
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

		if err := instance.extractMetadata(network, newNodeID, &metricOne); err != nil {
			return err
		}

		if err := instance.extractNodeMetric(network, newNodeID, &metricOne); err != nil {
			return err
		}

		switch network {
		case "bitcoin":
			return instance.btcDB.DeleteValue(*nodeId)
		case "testnet":
			return instance.testBtcDB.DeleteValue(*nodeId)
		default:
			return fmt.Errorf("db not found for the network %s", network)
		}

	}

	return nil
}

// Private function that a metric on store only the meta information of the node
// the key to store this information it is nodeID/metadata
func (instance *NoSQLDatabase) extractMetadata(network string, itemID string, metricOne *model.MetricOne) error {
	now := int(time.Now().Unix())

	version := 0
	lnnetwork := "unknown"

	if metricOne.Version != nil {
		version = *metricOne.Version
	}

	if metricOne.Network != nil {
		lnnetwork = *metricOne.Network
	}

	metadata := model.NodeMetadata{
		Version:    version,
		Network:    lnnetwork,
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

	if err := instance.PutRawValue(network, metadataID, metaJson); err != nil {
		return err
	}
	log.GetInstance().Debug(fmt.Sprintf("Insert Node (%s) medatata with id %s", metadata.NodeID, metadataID))
	return nil
}

func (instance *NoSQLDatabase) retreivalMetadata(network string, itemID string) (*model.NodeMetadata, error) {
	metadataKey := strings.Join([]string{itemID, "metadata"}, "/")
	nodeMetadataJson, err := instance.GetRawValue(network, metadataKey)
	if err != nil {
		return nil, err
	}

	var metaModel model.NodeMetadata
	if err := json.Unmarshal([]byte(nodeMetadataJson), &metaModel); err != nil {
		return nil, err
	}
	return &metaModel, nil
}

func (instance *NoSQLDatabase) extractNodeMetric(network string, itemID string, metricOne *model.MetricOne) error {
	// TODO iterate over timestamp and channels
	sizeUpdates := len(metricOne.UpTime)
	listUpTime := make([]*model.Status, 0)
	listTimestamp := make([]int, 0)
	timestampIndex := strings.Join([]string{
		itemID,
		"index",
	}, "/")

	oldTimestamp, err := instance.GetRawValue(network, timestampIndex)
	if err == nil {
		if err := json.Unmarshal(oldTimestamp, &listTimestamp); err != nil {
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
		if err := instance.PutRawValue(network, metricKey, jsonMetric); err != nil {
			return err
		}

		log.GetInstance().Info(fmt.Sprintf("Insert metric with id %s", metricKey))
	}

	jsonIndex, err := json.Marshal(listTimestamp)
	if err != nil {
		return err
	}

	return instance.PutRawValue(network, timestampIndex, jsonIndex)
}

// Private function to get a single metric given a specific timestamp
//
//lint:ignore U1000
func (instance *NoSQLDatabase) retreivalNodeMetric(network string, nodeKey string, timestamp uint, metricName string) (*model.NodeMetric, error) {
	metricKey := strings.Join([]string{nodeKey, fmt.Sprint(timestamp), "metric"}, "/")
	metricJson, err := instance.GetRawValue(network, metricKey)
	if err != nil {
		return nil, err
	}

	var modelMetric model.NodeMetric
	if err := json.Unmarshal(metricJson, &modelMetric); err != nil {
		return nil, err
	}
	return &modelMetric, nil

}

// deleteNodeMetric private function to delete a single metric given a specific timestamp
func (self *NoSQLDatabase) deleteNodeMetric(network string, nodeKey string, timestamp uint) error {
	metricKey := strings.Join([]string{nodeKey, fmt.Sprint(timestamp), "metric"}, "/")
	return self.DeleteRawValue(network, metricKey)
}

func (self *NoSQLDatabase) getIndex(network string, nodeKey string) []uint {
	timestampsKey := strings.Join([]string{nodeKey, "index"}, "/")
	timestampJson, err := self.GetRawValue(network, timestampsKey)
	if err != nil {
		log.GetInstance().Errorf("%s", err)
		return nil
	}

	var modelTimestamp []uint
	if err := json.Unmarshal(timestampJson, &modelTimestamp); err != nil {
		log.GetInstance().Errorf("index decoding from raw json: %s", err)
		return nil
	}

	return modelTimestamp
}

// Private function that it is able to get the collection of metric in a period
// expressed in unix time.
func (instance *NoSQLDatabase) retrievalNodesMetric(network string, nodeKey string, metricName string, startPeriod int, endPeriod int) (*model.NodeMetric, error) {
	modelTimestamp := instance.getIndex(network, nodeKey)
	modelMetric := &model.NodeMetric{
		Timestamp:    0,
		UpTime:       make([]*model.Status, 0),
		ChannelsInfo: make([]*model.StatusChannel, 0),
	}

	startTimestamp := uint32(0)
	// find the first usefult timestamp
	for _, timestamp := range modelTimestamp {
		if timestamp >= uint(startPeriod) {
			startTimestamp = uint32(timestamp)
			break
		}
	}

	if startTimestamp == 0 {
		return nil, fmt.Errorf("first usefult timestamp not found, ask for %d but last item is %d", startPeriod, modelTimestamp[len(modelTimestamp)-1])
	} else {
		startPeriod = int(startTimestamp)
	}

	start_key := strings.Join([]string{nodeKey, fmt.Sprint(startPeriod), "metric"}, "/")
	end_key := strings.Join([]string{nodeKey, fmt.Sprint(endPeriod), "metric"}, "/")

	// we should aggregate the channel in a single item and merge
	// all the content metric information
	channelsInfoMap := make(map[string]*model.StatusChannel)

	// FIXME: the Raw Iterator need to pass a raw metrics as sequence of bytes
	err := instance.RawIterateThrough(network, start_key, end_key, func(raw_metric string) error {
		var tmpModelMetric model.NodeMetric
		if err := json.Unmarshal([]byte(raw_metric), &tmpModelMetric); err != nil {
			return err
		}
		modelMetric.Timestamp = tmpModelMetric.Timestamp
		modelMetric.UpTime = append(modelMetric.UpTime,
			tmpModelMetric.UpTime...)

		for _, channelInfo := range tmpModelMetric.ChannelsInfo {
			key := strings.Join([]string{channelInfo.ChannelID, channelInfo.Direction}, "_")
			value, found := channelsInfoMap[key]
			if found {
				value.NodeAlias = channelInfo.NodeAlias
				value.Color = channelInfo.Color
				value.Capacity = channelInfo.Capacity
				value.Forwards = append(value.Forwards, channelInfo.Forwards...)
				value.UpTime = append(value.UpTime, channelInfo.UpTime...)
				value.Online = channelInfo.Online
				value.LastUpdate = channelInfo.LastUpdate
				value.Direction = channelInfo.Direction
				value.Fee = channelInfo.Fee
				value.Limits = channelInfo.Limits
			} else {
				channelsInfoMap[key] = channelInfo
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// append the channel to the channelsInfo list
	// after we aggregate it!
	for _, channel := range channelsInfoMap {
		modelMetric.ChannelsInfo = append(modelMetric.ChannelsInfo, channel)
	}

	return modelMetric, nil
}

// Private function that it is able to delete the collection of metric in a period
// expressed in unix time.
func (self *NoSQLDatabase) deleteNodesMetric(network string, nodeKey string) error {
	timestampsKey := strings.Join([]string{nodeKey, "index"}, "/")
	timestampJson, err := self.GetRawValue(network, timestampsKey)
	if err != nil {
		return err
	}
	var modelTimestamp []uint
	if err := json.Unmarshal(timestampJson, &modelTimestamp); err != nil {
		return err
	}

	for _, timestamp := range modelTimestamp {
		if err := self.deleteNodeMetric(network, nodeKey, timestamp); err != nil {
			log.GetInstance().Errorf("error: %s", err)
		}
	}
	// delete metric index
	return self.DeleteRawValue(network, timestampsKey)
}

func (self *NoSQLDatabase) PeriodicallyCleanUp() error {
	// we ignore any error that occurs because the function
	// will generate the logs if somethings bad happens.
	_ = self.cleanBuggyData()
	// ignore the error also for this function
	// because there is no reason to abort the
	// server.
	// The function will log the error!
	_ = self.deleteInactiveNode()
	return nil
}

// cleanBuggyData while we develop we introduce some fancy bugs
// that can corrupt the data that we are collecting, so
// this procedure collect all the method to delete the buggy data.
func (self *NoSQLDatabase) cleanBuggyData() error {
	return nil
}

// deleteInactiveNode procedure to delete all the not active node
// that there are on server.
//
// We define an inactive node a node that push the data on the server 1 month
// ago. This is also good for privacy when the use want stop to share the data
// these data are gone after 1 month.
func (self *NoSQLDatabase) deleteInactiveNode() error {
	now := time.Now().Unix()
	lastGood := utime.AddToTimestamp(now, -1*utime.Month)
	for _, network := range []string{"bitcoin", "testnet"} {
		nodes, err := self.GetNodes(network)
		if err != nil {
			return nil
		}
		for _, node := range nodes {
			if !utime.InRangeFromUnix(int64(node.LastUpdate), lastGood, 1*utime.Month) {
				if err := self.DeleteNode(network, "metric_one", node.NodeID); err != nil {
					log.GetInstance().Errorf("error: %s", err)
				}
			}
		}
	}
	return nil
}
