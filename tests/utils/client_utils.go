package utils

import (
	"fmt"
)

type AddMetricOneResp struct {
	NodeID string `json:"node_id"`
}

func ComposeInitMetricOneQuery(nodeId string, payload string) string {

	query := `mutation {
                    initMetricOne(node_id: "%s", payload: "%s", signature: "%s") {
                         node_id
                    }
                  }`
	fmtString := fmt.Sprintf(query, nodeId, payload, "123455")
	return fmtString
}

func ComposeGetNodesQuery() string {
	query := `query {
                    nodes(network: "testnet")
                  }`
	return query
}

func ComposeGetMetricOneQuery(nodeID string) string {
	query := `query{
                     getMetricOne(node_id: "%s") {
                        node_id,
                        metric_name,
                        color
                     }
                  }`
	return fmt.Sprintf(query, nodeID)
}
