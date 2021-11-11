package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/LNOpenMetrics/lnmetrics.server/graph/generated"
	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
)

func (r *mutationResolver) AddNodeMetrics(ctx context.Context, input model.NodeMetrics) (*model.MetricOne, error) {
	return r.MetricsService.AddMetricOne(input.NodeID, input.PayloadMetricOne)
}

func (r *mutationResolver) InitMetricOne(ctx context.Context, nodeID string, payload string, signature string) (*model.MetricOne, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdateMetricOne(ctx context.Context, nodeID string, payload string, signature string) (bool, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetNodes(ctx context.Context) ([]*model.NodeMetadata, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetMetricOne(ctx context.Context, nodeID string, startPeriod int, endPeriod int) (*model.MetricOne, error) {
	return r.MetricsService.GetMetricOne(nodeID)
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//  - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//    it when you're done.
//  - You have helper methods in this file. Move them out to keep these resolver files clean.
func (r *queryResolver) Nodes(ctx context.Context) ([]string, error) {
	nodes, err := r.MetricsService.GetNodes()
	if err != nil {
		return make([]string, 0), err
	}
	// TODO: Change in the generation code, the return type, from array of string
	// to slice of string.
	result := make([]string, 0)
	for _, nodeId := range nodes {
		result = append(result, *nodeId)
	}
	return result, nil
}
