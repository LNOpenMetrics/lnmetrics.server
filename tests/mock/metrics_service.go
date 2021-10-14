package mock

import (
	"github.com/OpenLNMetrics/lnmetrics.server/graph/model"

	"github.com/stretchr/testify/mock"
)

type MockMetricsServices struct {
	mock.Mock
}

func (instance *MockMetricsServices) AddNewMetricOne(nodeId string, payload *string) *model.MetricOne {
	args := instance.Called(nodeId, payload)
	return args.Get(0).(*model.MetricOne)
}
