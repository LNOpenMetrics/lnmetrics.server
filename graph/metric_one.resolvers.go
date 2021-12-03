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
	return r.MetricsService.AddNodeMetrics(input.NodeID, &input.PayloadMetricOne)
}

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
	panic(fmt.Errorf("not implemented"))
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
