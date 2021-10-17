package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/OpenLNMetrics/lnmetrics.server/graph/generated"
	"github.com/OpenLNMetrics/lnmetrics.server/graph/model"
)

func (r *mutationResolver) AddNodeMetrics(ctx context.Context, input model.NodeMetrics) (*model.MetricOne, error) {
	return r.MetricsService.AddMetricOne(input.NodeID, input.PayloadMetricOne)
}

func (r *queryResolver) Nodes(ctx context.Context) ([]string, error) {
	panic(fmt.Errorf("not implemented"))
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
