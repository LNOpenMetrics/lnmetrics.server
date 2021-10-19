package utils

import (
	"fmt"
)

type AddMetricOneResp struct {
	NodeID string `json:"node_id"`
}

func ComposeAddMetricOneQuery(nodeId string, payload string) string {

	query := `mutation {
                    addNodeMetrics(input: {node_id: "%s", payload_metric_one: "%s"}) {
                         node_id
                    }
                  }`
	fmtString := fmt.Sprintf(query, nodeId, payload)
	return fmtString
}

func ComposeGetNodesQuery() string {
	query := `query {
                    nodes
                  }`
	return query
}

func ComposeGetMetricOneQuery(nodeID string) string {
	query := `query{
                     getMetricOne(node_id: "%s") {
                        node_id,
                        name,
                        color
                     }
                  }`
	return fmt.Sprintf(query, nodeID)
}
