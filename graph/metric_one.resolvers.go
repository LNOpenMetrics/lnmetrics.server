package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/LNOpenMetrics/lnmetrics.server/graph/generated"
	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"
)

func (r *mutationResolver) InitMetricOne(ctx context.Context, nodeID string, payload string, signature string) (*model.MetricOne, error) {
	return r.MetricsService.InitMetricOne(nodeID, &payload, signature)
}

func (r *mutationResolver) UpdateMetricOne(ctx context.Context, nodeID string, payload string, signature string) (bool, error) {
	if err := r.MetricsService.UpdateMetricOne(nodeID, &payload, signature); err != nil {
		return false, err
	}
	return true, nil
}

func (r *queryResolver) Nodes(ctx context.Context) ([]string, error) {
	nodes, err := r.MetricsService.GetNodes("bitcoin")
	if err != nil {
		return nil, err
	}
	listNodes := make([]string, 0)
	for _, node := range nodes {
		listNodes = append(listNodes, node.NodeID)
	}
	return listNodes, nil
}

func (r *queryResolver) GetNodes(ctx context.Context, network string) ([]*model.NodeMetadata, error) {
	return r.MetricsService.GetNodes(network)
}

func (r *queryResolver) GetNode(ctx context.Context, network string, nodeID string) (*model.NodeMetadata, error) {
	return r.MetricsService.GetNode(network, nodeID)
}

func (r *queryResolver) GetMetricOne(ctx context.Context, nodeID string, startPeriod int, endPeriod int) (*model.MetricOne, error) {
	return r.MetricsService.GetMetricOne(nodeID, startPeriod, endPeriod)
}

func (r *queryResolver) GetMetricOneResult(ctx context.Context, network string, nodeID string) (*model.MetricOneOutput, error) {
	return r.MetricsService.GetMetricOneOutput(network, nodeID)
}

func (r *queryResolver) MetricOne(ctx context.Context, nodeID string, first int, last *int) (*model.MetricOneInfo, error) {
	return r.MetricsService.GetMetricOnePaginator(nodeID, first, last)
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
func (r *queryResolver) MetricsOne(ctx context.Context, nodeID string, first int, last *int) (*model.MetricOneInfo, error) {
	return r.MetricsService.GetMetricOnePaginator(nodeID, first, last)
}
