package mock

import (
	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"

	"github.com/stretchr/testify/mock"
)

type MockMetricsServices struct {
	mock.Mock
}

func (instance *MockMetricsServices) AddMetricOne(nodeId string, payload string) (*model.MetricOne, error) {
	args := instance.Called(nodeId, payload)
	return args.Get(0).(*model.MetricOne), nil
}

func (instance *MockMetricsServices) GetNodes() ([]*string, error) {
	args := instance.Called()
	return args.Get(0).([]*string), nil
}

func (instance *MockMetricsServices) GetMetricOne(nodeID string) (*model.MetricOne, error) {
	args := instance.Called(nodeID)
	return args.Get(0).(*model.MetricOne), nil
}
