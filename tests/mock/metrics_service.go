package mock

import (
	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"

	"github.com/stretchr/testify/mock"
)

type MockMetricsServices struct {
	mock.Mock
}

func (instance *MockMetricsServices) InitMetricOne(nodeID string, payload *string, signature string) (*model.MetricOne, error) {
	args := instance.Called(nodeID, *payload, signature)
	return args.Get(0).(*model.MetricOne), nil
}

func (instance *MockMetricsServices) UpdateMetricOne(nodeID string, payload *string, signature string) error {
	_ = instance.Called(nodeID, payload, signature)
	return nil
}

// Return all the node information that are pushing the data.
func (instance *MockMetricsServices) GetNodes(network string) ([]*model.NodeMetadata, error) {
	args := instance.Called(network)
	return args.Get(0).([]*model.NodeMetadata), nil
}

func (instance *MockMetricsServices) GetNode(network string, nodeID string) (*model.NodeMetadata, error) {
	args := instance.Called(network)
	return args.Get(0).(*model.NodeMetadata), nil
}

// Get the metric one of one node and add a filtering option by period
func (instance *MockMetricsServices) GetMetricOne(nodeID string, startPeriod int, endPeriod int) (*model.MetricOne, error) {
	args := instance.Called(nodeID, startPeriod, endPeriod)
	return args.Get(0).(*model.MetricOne), nil
}

// Get the metric one of one node and add a filtering option by period
func (instance *MockMetricsServices) GetFullMetricOne(nodeID string) (*model.MetricOne, error) {
	args := instance.Called(nodeID)
	return args.Get(0).(*model.MetricOne), nil
}

// all deprecated function
func (instance *MockMetricsServices) Nodes() ([]*string, error) {
	return make([]*string, 0), nil
}

func (instance *MockMetricsServices) AddNodeMetrics(nodeID string, payload *string) (*model.MetricOne, error) {
	return nil, nil
}
