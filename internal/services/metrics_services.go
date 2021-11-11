package services

import (
	"encoding/json"
	"fmt"

	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/backend"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/db"
)

type IMetricsService interface {
	AddMetricOne(nodeID string, payload string) (*model.MetricOne, error)
	GetNodes() ([]*string, error)
	GetMetricOne(nodeID string) (*model.MetricOne, error)
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

// FIXME: This method is slow because get all the key in the database,
// and this don't scale well, we need also to indexing in a thread safe way the
// node that are putting information in the db.
func (instance *MetricsService) GetNodes() ([]*string, error) {
	return instance.Storage.GetNodesID()
}

// Get the the metric one information by node id
func (instance *MetricsService) GetMetricOne(nodeID string) (*model.MetricOne, error) {
	metricNodeInfo, err := instance.Storage.GetMetricOne(nodeID)
	if err != nil {
		return nil, err
	}
	return metricNodeInfo, nil
}
