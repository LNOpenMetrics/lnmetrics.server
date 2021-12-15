package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/backend"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/db"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/metric"

	"github.com/LNOpenMetrics/lnmetrics.utils/log"
)

type IMetricsService interface {
	// Deprecated
	AddNodeMetrics(nodeID string, payload *string) (*model.MetricOne, error)

	// Init method it is called only the first time from the node
	// when it is not init in the server, but it has some metrics collected
	InitMetricOne(nodeID string, payload *string, signature string) (*model.MetricOne, error)

	//  Append other metrics collected in a range of period by the node
	UpdateMetricOne(nodeID string, payload *string, signature string) error

	// Deprecated: Use GetNode isteand
	// Return the list of node IF available on the server.
	Nodes() ([]*string, error)

	// Return the list of nodes available on the server
	GetNodes(network string) ([]*model.NodeMetadata, error)

	// Return the node metadata available on the server (utils for the init method)
	GetNode(network string, nodeID string) (*model.NodeMetadata, error)

	// Return the node metrics with a nodeID, and option range, from start to an end period
	GetMetricOne(nodeID string, startPeriod int, endPeriod int) (*model.MetricOne, error)

	// Return the metric one output
	GetMetricOneOutput(network string, nodeID string) (*model.MetricOneOutput, error)
}

type MetricsService struct {
	Storage db.MetricsDatabase
	Backend backend.Backend
}

// Constructor method.
func NewMetricsService(db db.MetricsDatabase, lnBackend backend.Backend) *MetricsService {
	return &MetricsService{Storage: db, Backend: lnBackend}
}

func (instance *MetricsService) AddNodeMetrics(nodeID string, payload *string) (*model.MetricOne, error) {
	var metricModel model.MetricOne
	if err := json.Unmarshal([]byte(*payload), &metricModel); err != nil {
		return nil, err
	}

	if metricModel.Network == nil || *metricModel.Network != "bitcoin" {
		return nil, fmt.Errorf("Unsupported network, or old client version")
	}

	if err := instance.Storage.InsertMetricOne(&metricModel); err != nil {
		return nil, err
	}

	return &metricModel, nil
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
		return nil, fmt.Errorf("Metrics Already initialized, please call get metric one API to get your last update here")
	}

	var metricModel model.MetricOne
	if err := json.Unmarshal([]byte(*payload), &metricModel); err != nil {
		return nil, err
	}

	if metricModel.Network == nil || *metricModel.Network != "bitcoin" {
		return nil, fmt.Errorf("Unsupported network, or an old client version is sending this data")
	}

	if metricModel.Version != nil || *metricModel.Version < 4 {
		return nil, fmt.Errorf("The commit payload it is made from an old client, please considered to update the client (plugin)")
	}

	if err := instance.Storage.InsertMetricOne(&metricModel); err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	log.GetInstance().Info(fmt.Sprintf("New node in the lnmetric services ad %d with node id %s", now, nodeID))
	if err := metric.CalculateMetricOneOutput(instance.Storage, &metricModel); err != nil {
		return nil, err
	}
	return &metricModel, nil
}

func (instance *MetricsService) UpdateMetricOne(nodeID string, payload *string, signature string) error {

	if !instance.Storage.ContainsIndex(nodeID, "metric_one") {
		return fmt.Errorf("Metrics One not initialized, please call initialize the metric.")
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
		return fmt.Errorf("Unsupported network, or an old client version is sending this data")
	}

	if metricModel.Version == nil || *metricModel.Version < 4 {
		return fmt.Errorf("The commit payload it is made from an old client, please considered to update the client (plugin)")
	}

	if err := instance.Storage.UpdateMetricOne(&metricModel); err != nil {
		return err
	}

	now := time.Now().Format(time.RFC850)
	log.GetInstance().Info(fmt.Sprintf("Update for the node %s with new metrics lnmetric in date %s", nodeID, now))

	if err := metric.CalculateMetricOneOutput(instance.Storage, &metricModel); err != nil {
		return err
	}

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

// Return all the node information that are pushing the data.
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

// Get the metric one of one node and add a filtering option by period
func (instance *MetricsService) GetMetricOne(nodeID string, startPeriod int, endPeriod int) (*model.MetricOne, error) {
	metricNodeInfo, err := instance.Storage.GetMetricOne(nodeID, startPeriod, endPeriod)
	if err != nil {
		return nil, err
	}
	return metricNodeInfo, nil
}

func (instance *MetricsService) GetMetricOneOutput(network string, nodeID string) (*model.MetricOneOutput, error) {
	return instance.Storage.GetMetricOneOutput(nodeID)
}
