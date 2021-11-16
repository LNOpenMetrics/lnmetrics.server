package services

import (
	"encoding/json"
	"fmt"

	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/backend"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/db"
)

type IMetricsService interface {
	// Init method it is called only the first time from the node
	// when it is not init in the server, but it has some metrics collected
	InitMetricOne(nodeID string, payload string, signature string) (*model.MetricOne, error)
	//  Append other metrics collected in a range of period by the node
	UpdateMetricOne(nodeID string, payload string, signature string) error
	// Return the list of nodes available on the server
	GetNodes(network string) ([]*model.NodeMetadata, error)
	// Return the node metadata available on the server (utils for the init method)
	GetNode(network string, nodeID string) (*model.NodeMetadata, error)
	// Return the node metrics with a nodeID, and option range, from start to an end period
	GetMetricOne(nodeID string, startPeriod uint, endPeriod uint) (*model.MetricOne, error)
}

type MetricsService struct {
	Storage db.MetricsDatabase
	Backend backend.Backend
}

// Constructor method.
func NewMetricsService(db db.MetricsDatabase, lnBackend backend.Backend) *MetricsService {
	return &MetricsService{Storage: db, Backend: lnBackend}
}

// Verify and Store metrics one in the internal storage.
// Deprecated: Used to test the first version of the server, not deprecated in favor or UpdateMetricOne, InitMetricOne, AppendMetricOne.
func (instance *MetricsService) AddMetricOne(nodeID string, payload string) (*model.MetricOne, error) {
	// TODO: make a function that check if the value are correct value.
	// TODO: Verify the payload as raw content, with the following steps:
	// 1: Make the sha256 of the payload in bytes
	// 2: verify the signature with the backend

	var model model.MetricOne
	if err := json.Unmarshal([]byte(payload), &model); err != nil {
		return nil, fmt.Errorf("Error during reading JSON %s. It is a valid JSON?", err)
	}

	// TODO: Check the actual version supported by the server, and check the version of the paylad
	// that the node has
	if err := instance.Storage.InsertMetricOne(&model); err != nil {
		return nil, err
	}

	return &model, nil
}

func (instance *MetricsService) InitMetricOne(nodeID string, payload string, signature string) (*model.MetricOne, error) {
	return nil, fmt.Errorf("Not implemented yet")
}

func (instance *MetricsService) UpdateMetricOne(nodeID string, payload string, signature string) error {
	return fmt.Errorf("Not implemented yet")
}

// Return all the node information that are pushing the data.
func (instance *MetricsService) GetNodes(network string) ([]*model.NodeMetadata, error) {
	return nil, fmt.Errorf("Not implemented yet")
}

func (instance *MetricsService) GetNode(network string, nodeID string) (*model.NodeMetadata, error) {
	return nil, fmt.Errorf("Not implemented yet")
}

// Get the metric one of one node and add a filtering option by period
func (instance *MetricsService) GetMetricOne(nodeID string, startPeriod uint, endPeriod uint) (*model.MetricOne, error) {
	metricNodeInfo, err := instance.Storage.GetMetricOne(nodeID)
	if err != nil {
		return nil, err
	}
	return metricNodeInfo, nil
}
