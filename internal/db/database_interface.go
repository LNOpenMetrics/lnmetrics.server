package db

import (
	"github.com/OpenLNMetrics/ln-metrics-server/graph/model"
)

// Interface to abstract from an db implementation the
// the logic to store and make analysis over the data.
type MetricsDatabase interface {
	// Prepare the database for the metric one data model
	// Takes a interface, if the db implementation required some
	// custom propieties
	CreateMetricOne(options *map[string]interface{}) error

	// Insert a metric in the data model
	InsertMetricOne(toInset *model.MetricOne) error

	// Get all the node id stored in the database
	GetIDs() ([]*string, error)

	// Get all the node data by id
	GetMetricOne(withId string) (*model.MetricOne, error)
}
