package services

import (
	"encoding/json"
	"fmt"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/metric/metric_one"
	"github.com/LNOpenMetrics/lnmetrics.utils/utime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/backend"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/config"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/db"
	"github.com/LNOpenMetrics/lnmetrics.utils/log"
)

type IMetricsService interface {
	// InitMetricOne Init method it is called only the first time from the node
	// when it is not init in the server, but it has some metrics collected
	InitMetricOne(nodeID string, payload *string, signature string) (*model.MetricOne, error)

	// UpdateMetricOne Append other metrics collected in a range of period by the node
	UpdateMetricOne(nodeID string, payload *string, signature string) error

	// Deprecated: Use GetNode instead
	// Return the list of node IF available on the server.
	Nodes() ([]*string, error)

	// GetNodes Return the list of nodes available on the server
	GetNodes(network string) ([]*model.NodeMetadata, error)

	// GetNode Return the node metadata available on the server (utils for the init method)
	GetNode(network string, nodeID string) (*model.NodeMetadata, error)

	// GetMetricOne Return the node metrics with a nodeID, and option range, from start to an end period
	GetMetricOne(network string, nodeID string, startPeriod int, endPeriod int) (*model.MetricOne, error)

	// GetMetricOneOutput Return the metric one output
	GetMetricOneOutput(network string, nodeID string) (*model.MetricOneOutput, error)

	// GetMetricOnePaginator / use the paginator patter to return the metric one, and the paginator info
	GetMetricOnePaginator(network string, nodeID string, first int, last *int) (*model.MetricOneInfo, error)
}

type MetricsService struct {
	Storage db.MetricsDatabase
	Backend backend.Backend
}

// NewMetricsService Constructor method
func NewMetricsService(db db.MetricsDatabase, lnBackend backend.Backend) *MetricsService {
	instance := &MetricsService{Storage: db, Backend: lnBackend}
	if err := instance.initMetricOneOutput(); err != nil {
		log.GetInstance().Infof("Error: %s", err)
	}
	return instance
}

// Function to help to init the metric one output on the server
// for the old data that have no metric one supported
// This function it is used only for the server update
func (instance *MetricsService) initMetricOneOutput() error {
	nodes, err := instance.GetNodes("bitcoin")
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	wg.Add(len(nodes))
	for _, node := range nodes {
		go instance.initMetricOneOutputOnNode(node, &wg)
	}
	wg.Wait()
	return nil
}

func (instance *MetricsService) initMetricOneOutputOnNode(node *model.NodeMetadata, wg *sync.WaitGroup) {
	defer wg.Done()
	if !instance.containsMetricOneOutput(node) {
		// DONE 1. Get the index list of the node
		// DONE 2. Iterate over the timestamp and load the metric model
		// TODO 3. Skip the if the timestamp difference it is less of 20 minutes and
		// it is made from the same event
		log.GetInstance().Infof("Calculate Metric One Output for the node %s", node.NodeID)
		timestampIndex, err := instance.Storage.GetMetricOneIndex(node.Network, node.NodeID)
		if err != nil {
			log.GetInstance().Infof("Error: %s", err)
			return
		}
		sort.Slice(timestampIndex, func(i, j int) bool { return timestampIndex[i] < timestampIndex[j] })
		start := timestampIndex[0]
		end := timestampIndex[len(timestampIndex)-1]
		keyPrefix := strings.Join([]string{node.NodeID, "metric_one"}, "/")
		startIndex := strings.Join([]string{keyPrefix, fmt.Sprint(start), "metric"}, "/")
		endIndex := strings.Join([]string{keyPrefix, fmt.Sprint(end), "metric"}, "/")
		err = instance.Storage.RawIterateThrough(node.Network, startIndex, endIndex, func(metricOne string) error {
			var model model.MetricOne
			if err := json.Unmarshal([]byte(metricOne), &model); err != nil {
				return err
			}

			return metric_one.CalculateMetricOneOutputSync(instance.Storage, &model)
		})
		if err != nil {
			log.GetInstance().Infof("Error: %s", err)
			return
		}
	}
}

func (instance *MetricsService) containsMetricOneOutput(node *model.NodeMetadata) bool {
	metricKey := strings.Join([]string{node.NodeID, config.RawMetricOnePrefix}, "/")
	_, err := instance.Storage.GetRawValue(node.Network, metricKey)
	if err != nil {
		log.GetInstance().Errorf("Error: %s", err)
		return false
	}
	return true
}

func calculateMetricOneOutput(metricsServices *MetricsService, metricModel *model.MetricOne, startTime int64) {
	if err := metric_one.CalculateMetricOneOutputSync(metricsServices.Storage, metricModel); err != nil {
		log.GetInstance().Errorf("Calculate metric one output return an error %s", err)
	} else {
		log.GetInstance().Debugf("Calculate metric one for node %s at %d", metricModel.NodeID, startTime)
	}
}

func (instance *MetricsService) InitMetricOne(nodeID string, payload *string, signature string) (*model.MetricOne, error) {
	ok, err := instance.Backend.VerifyMessage(payload, &signature, &nodeID)
	if !ok || err != nil {
		if !ok {
			return nil, fmt.Errorf("The server can not verify the payload")
		}
		if err != nil {
			return nil, err
		}
	}
	log.GetInstance().Info(fmt.Sprintf("Node %s verify check passed with signature %s", nodeID, signature))

	if instance.Storage.ContainsIndex(nodeID, "metric_one") {
		return nil, fmt.Errorf("metrics Already initialized, please call get metric one API to get your last update here")
	}

	var metricModel model.MetricOne
	if err := json.Unmarshal([]byte(*payload), &metricModel); err != nil {
		return nil, err
	}

	if metricModel.Network == nil || *metricModel.Network != "bitcoin" {
		return nil, fmt.Errorf("unsupported network, or an old client version is sending this data")
	}

	if metricModel.Version == nil || *metricModel.Version < 4 {
		return nil, fmt.Errorf("the commit payload it is made from an old client, please considered to update the client (plugin)")
	}

	if err := instance.Storage.InsertMetricOne(*metricModel.Network, &metricModel); err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	log.GetInstance().Info(fmt.Sprintf("New node in the lnmetric services ad %d with node id %s", now, nodeID))
	go calculateMetricOneOutput(instance, &metricModel, time.Now().Unix())
	return &metricModel, nil
}

func (instance *MetricsService) UpdateMetricOne(nodeID string, payload *string, signature string) error {

	if !instance.Storage.ContainsIndex(nodeID, "metric_one") {
		return fmt.Errorf("metrics One not initialized, please call initialize the metric")
	}

	ok, err := instance.Backend.VerifyMessage(payload, &signature, &nodeID)
	if !ok || err != nil {
		if !ok {
			return fmt.Errorf("The server can not verify the payload")
		}
		if err != nil {
			return err
		}
	}
	log.GetInstance().Info(fmt.Sprintf("Node %s verify check passed with signature %s", nodeID, signature))

	var metricModel model.MetricOne
	if err := json.Unmarshal([]byte(*payload), &metricModel); err != nil {
		return err
	}

	if metricModel.Network == nil || *metricModel.Network != "bitcoin" {
		return fmt.Errorf("unsupported network, or an old client version is sending this data")
	}

	if metricModel.Version == nil || *metricModel.Version < 4 {
		return fmt.Errorf("the commit payload it is made from an old client, please considered to update the client (plugin)")
	}

	if err := instance.Storage.UpdateMetricOne(*metricModel.Network, &metricModel); err != nil {
		return err
	}

	now := time.Now().Format(time.RFC850)
	log.GetInstance().Info(fmt.Sprintf("Update for the node %s with new metrics lnmetric in date %s", nodeID, now))

	go calculateMetricOneOutput(instance, &metricModel, time.Now().Unix())
	return nil
}

func (instance *MetricsService) Nodes() ([]*string, error) {
	nodesList := make([]*string, 0)
	nodes, err := instance.GetNodes("bitcoin")
	if err != nil {
		return nil, err
	}
	for _, node := range nodes {
		nodesList = append(nodesList, &node.NodeID)
	}
	return nodesList, nil
}

// GetNodes Return all the node information that are pushing the data.
func (instance *MetricsService) GetNodes(network string) ([]*model.NodeMetadata, error) {
	if network != "bitcoin" {
		return nil, fmt.Errorf("network %s unsupported", network)
	}
	return instance.Storage.GetNodes(network)
}

func (instance *MetricsService) GetNode(network string, nodeID string) (*model.NodeMetadata, error) {
	if network != "bitcoin" {
		return nil, fmt.Errorf("network %s unsupported", network)
	}
	return instance.Storage.GetNode(network, nodeID, "metric_one")
}

// GetMetricOnePaginator Return the metrics one without metadata of the node,
// and with the information to implement the paginator
func (instance *MetricsService) GetMetricOnePaginator(network string, nodeID string, first int, last *int) (*model.MetricOneInfo, error) {
	if last == nil {
		val := int(utime.AddToTimestamp(int64(first), 6*30*time.Minute))
		last = &val
	} else {
		// we check if the last - first is under a limit like 6 hours
		if utime.OccurenceInUnixRange(int64(first), int64(*last), 30*time.Minute) > 12 {
			return nil, fmt.Errorf("distance between the timestamp it is grater than 6 hour")
		}
	}
	metricsInfo, err := instance.Storage.GetMetricOneInfo(network, nodeID, first, *last)
	if err != nil {
		log.GetInstance().Errorf("Error during the call to GetMetricOnePaginator: %s", err)
	}
	return metricsInfo, err
}

// GetMetricOne Get the metric one of one node and add a filtering option by period
func (instance *MetricsService) GetMetricOne(network string, nodeID string, startPeriod int, endPeriod int) (*model.MetricOne, error) {
	metricNodeInfo, err := instance.Storage.GetMetricOne(network, nodeID, startPeriod, endPeriod)
	if err != nil {
		return nil, err
	}
	return metricNodeInfo, nil
}

func (instance *MetricsService) GetMetricOneOutput(network string, nodeID string) (*model.MetricOneOutput, error) {
	return instance.Storage.GetMetricOneOutput(network, nodeID)
}
