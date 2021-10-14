package utils

import (
	"fmt"
)

type AddMetricOneResp struct {
	NodeId string
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
