package db

import (
	"github.com/OpenLNMetrics/lnmetrics.server/graph/model"
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
	GetNodesID() ([]*string, error)

	// Get all the node data by id
	GetMetricOne(withId string) (*model.MetricOne, error)

	// Close the connection with the database
	CloseDatabase() error

	// Erase the content of the database
	EraseDatabase() error

	// Close the connection and erase the database
	EraseAfterCloseDatabase() error
}
