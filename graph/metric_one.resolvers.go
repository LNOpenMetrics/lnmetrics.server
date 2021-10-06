package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/OpenLNMetrics/ln-metrics-server/graph/generated"
	"github.com/OpenLNMetrics/ln-metrics-server/graph/model"
)

// mutation {
//    addNodeMetrics(input: {node_id: "abc", payload_metric_one: "bcd"}) {
//       node_id
//	 name
//    }
// }
func (instance *mutationResolver) AddNodeMetrics(ctx context.Context, input model.NodeMetrics) (*model.MetricOne, error) {
	// This method need to include the following feature
	// 1. Verify the source of these data, we can't put without any check this data in the db.
	// 2. Deflate the result of the payload, for the moment we send compressed string payload.
	// this is stupid because we can simple append metrics, but for the moment this is the
	// architecture of the demo.
	// 3. Store this data in the db with the node as key.
	// 4. Return a result

	// Implementing point 3
	//
	// Inflate the JSON into a MetricOne Object
	var model model.MetricOne

	if err := json.Unmarshal([]byte(input.PayloadMetricOne), &model); err != nil {
		return nil, fmt.Errorf("Error during reading JSON %s. It is a valid JSON?", err)
	}

	if err := instance.Dbms.InsertMetricOne(&model); err != nil {
		return nil, err
	}

	return &model, nil
}

func (r *queryResolver) Nodes(ctx context.Context) ([]*model.NodeInfo, error) {
	panic(fmt.Errorf("not implemented"))
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
