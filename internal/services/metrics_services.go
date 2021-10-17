package services

import (
	"encoding/json"
	"fmt"

	"github.com/OpenLNMetrics/lnmetrics.server/graph/model"
	"github.com/OpenLNMetrics/lnmetrics.server/internal/db"
)

type IMetricsService interface {
	AddMetricOne(nodeID string, payload string) (*model.MetricOne, error)
	GetNodes() ([]*string, error)
}

type MetricsService struct {
	Storage db.MetricsDatabase
}

// Constructor method.
func NewMetricsService(db db.MetricsDatabase) *MetricsService {
	return &MetricsService{Storage: db}
}

// Verify and Store metrics one in the internal storage.
func (instance *MetricsService) AddMetricOne(nodeID string, payload string) (*model.MetricOne, error) {
	// TODO: Adding utils function to check the parametes here

	var model model.MetricOne

	if err := json.Unmarshal([]byte(payload), &model); err != nil {
		return nil, fmt.Errorf("Error during reading JSON %s. It is a valid JSON?", err)
	}

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
