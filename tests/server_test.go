package main

import (
	"testing"

	"github.com/OpenLNMetrics/ln-metrics-server/graph"
	"github.com/OpenLNMetrics/ln-metrics-server/graph/model"
	"github.com/OpenLNMetrics/ln-metrics-server/graph/generated"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TODO: Working here: https://stackoverflow.com/a/63735464/10854225
func TestPushMetricWithNodeId(t *testing.T) {
	t.Run("handle the push operation of the new metrics", func(t *testing.T) {
		resolvers := graph.Resolver{Nodes: make([]*model.NodeInfo, 0)}
		cli := client.New(handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &resolvers})))

	})
}
