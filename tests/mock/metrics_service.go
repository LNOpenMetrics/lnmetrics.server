package mock

import (
	"fmt"

	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"

	"github.com/stretchr/testify/mock"
)

type MockMetricsServices struct {
	mock.Mock
}

func (instance *MockMetricsServices) InitMetricOne(nodeID string, payload string, signature string) (*model.MetricOne, error) {
	return nil, fmt.Errorf("Not implemented yet")
}

func (instance *MockMetricsServices) UpdateMetricOne(nodeID string, payload string, signature string) error {
	return fmt.Errorf("Not implemented yet")
}

// Return all the node information that are pushing the data.
func (instance *MockMetricsServices) GetNodes() ([]*model.NodeMetadata, error) {
	return nil, fmt.Errorf("Not implemented yet")
}

func (instance *MockMetricsServices) GetNode(nodeID string) (*model.NodeMetadata, error) {
	return nil, fmt.Errorf("Not implemented yet")
}

// Get the metric one of one node and add a filtering option by period
func (instance *MockMetricsServices) GetMetricOne(nodeID string, startPeriod uint, endPeriod uint) (*model.MetricOne, error) {
	return nil, fmt.Errorf("Not implemented yet")
}
