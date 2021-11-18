package services

import (
	"encoding/json"
	"fmt"

	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/backend"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/db"
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
	// Return the node metric with a nodeID that contains the all period of the collected metric
	GetFullMetricOne(nodeID string) (*model.MetricOne, error)
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
		//FIXME: The node can put a modify version of the payload, maybe the plugin and the server
		// need to share some secret.
		return nil, fmt.Errorf("Metrics Already initialized, please call get metric one API to get your last update here")
	}

	var metricModel model.MetricOne
	if err := json.Unmarshal([]byte(*payload), &metricModel); err != nil {
		return nil, err
	}

	if err := instance.Storage.InsertMetricOne(&metricModel); err != nil {
		return nil, err
	}

	return &metricModel, nil
}

func (instance *MetricsService) UpdateMetricOne(nodeID string, payload *string, signature string) error {
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

	if err := instance.Storage.UpdateMetricOne(&metricModel); err != nil {
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

func (instance *MetricsService) GetFullMetricOne(nodeID string) (*model.MetricOne, error) {
	return nil, nil
}
