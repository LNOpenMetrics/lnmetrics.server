package utils

import (
	"fmt"
)

type AddMetricOneResp struct {
	NodeID string `json:"node_id"`
}

func ComposeInitMetricOneQuery(nodeId string, payload string, signature string) string {

	query := `mutation {
                    initMetricOne(node_id: "%s", payload: "%s", signature: "%s") {
                         node_id
                    }
                  }`
	fmtString := fmt.Sprintf(query, nodeId, payload, signature)
	return fmtString
}

func ComposeGetNodesQuery() string {
	query := `query {
                    getNodes(network: "bitcoin") {
                         node_id
                    }
                  }`
	return query
}

func ComposeGetMetricOneQuery(network string, nodeID string, startPeriod uint, endPeriod uint) string {
	query := `query{
                     getMetricOne(network: "%s", node_id: "%s", start_period: %d, end_period: %d) {
                        node_id,
                        metric_name,
                        color
                     }
                  }`
	return fmt.Sprintf(query, network, nodeID, startPeriod, endPeriod)
}
