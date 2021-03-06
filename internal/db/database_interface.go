package db

import (
	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
)

// Interface to abstract from an db implementation the
// the logic to store and make analysis over the data.
type MetricsDatabase interface {
	// GetRawValue Get access to the raw data contained with the specified key
	GetRawValue(key string) ([]byte, error)

	// PutRawValue Put a raw value with the specified key in the db
	PutRawValue(key string, value []byte) error

	RawIterateThrough(start string, end string, callback func(string) error) error

	// CreateMetricOne Prepare the database for the metric one data model
	// Takes a interface, if the db implementation required some
	// custom propieties
	CreateMetricOne(options *map[string]interface{}) error

	// InsertMetricOne Insert a metric in the data model
	InsertMetricOne(toInset *model.MetricOne) error

	// GetNodes Get all the node id stored in the database
	GetNodes(network string) ([]*model.NodeMetadata, error)

	// GetNode Get metric metadata of a specific node
	GetNode(network string, nodeID string, metriName string) (*model.NodeMetadata, error)

	// GetMetricOne Get all the node data by id
	GetMetricOne(withId string, startPeriod int, endPeriod int) (*model.MetricOne, error)

	// GetMetricOneInfo return the core info of the metrics one payload without metadata
	// this is useful when we are using the paginator patter, or we are just interested to
	// the metrics data.
	GetMetricOneInfo(nodeID string, first int, last int) (*model.MetricOneInfo, error)

	// GetMetricOneOutput Get the metric result calculated by the server
	GetMetricOneOutput(nodeID string) (*model.MetricOneOutput, error)

	// UpdateMetricOne Update the metric of the node, with new one.
	UpdateMetricOne(toInser *model.MetricOne) error

	GetMetricOneIndex(nodeID string) ([]int64, error)

	// CloseDatabase Close the connection with the database
	CloseDatabase() error

	// EraseDatabase Erase the content of the database
	EraseDatabase() error

	// EraseAfterCloseDatabase Close the connection and erase the database
	EraseAfterCloseDatabase() error

	// GetVersionData Return the version of the data in the database
	GetVersionData() (uint, error)

	// Migrate procedure to convert a more from aversion to another
	Migrate() error

	// ItemID From the metrics payload return the id of the node
	ItemID(toInsert *model.MetricOne) (string, error)

	// ContainsIndex Check if the node it is indexed for a specific metrics
	ContainsIndex(nodeID string, metricName string) bool
}
