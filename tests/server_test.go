package main

import (
	"testing"

	"github.com/LNOpenMetrics/lnmetrics.server/graph"
	"github.com/LNOpenMetrics/lnmetrics.server/graph/generated"
	"github.com/LNOpenMetrics/lnmetrics.server/graph/model"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	lnmock "github.com/LNOpenMetrics/lnmetrics.server/tests/mock"
	"github.com/LNOpenMetrics/lnmetrics.server/tests/utils"
	"github.com/stretchr/testify/require"
)

func TestInitMetricWithNodeId(t *testing.T) {
	t.Run("handle the push operation of the new metrics", func(t *testing.T) {
		mockMetricsService := new(lnmock.MockMetricsServices)
		resolvers := graph.Resolver{MetricsService: mockMetricsService}
		cli := client.New(handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &resolvers})))
		if cli == nil {
			panic(cli)
		}

		modelObj := model.MetricOne{Name: "mock", NodeID: "fake_id", Color: "0000"}
		query := utils.ComposeInitMetricOneQuery(modelObj.NodeID, `{name:\"mock\",node_id:\"fake_id\",color:\"0000\"}`)

		mockMetricsService.On("InitMetricOne", modelObj.NodeID, `{name:"mock",node_id:"fake_id",color:"0000"}`).Return(&modelObj)

		var resp struct {
			AddNodeMetrics utils.AddMetricOneResp
		}
		cli.MustPost(query, &resp)

		mockMetricsService.AssertExpectations(t)
		require.Equal(t, modelObj.NodeID, resp.AddNodeMetrics.NodeID)
	})
}

func TestGetNodesId(t *testing.T) {
	t.Run("Retrieval all the node id that push some data", func(t *testing.T) {
		mockMetricsService := new(lnmock.MockMetricsServices)
		resolvers := graph.Resolver{MetricsService: mockMetricsService}
		cli := client.New(handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &resolvers})))
		if cli == nil {
			panic(cli)
		}
		query := utils.ComposeGetNodesQuery()

		emptyList := make([]*string, 0)
		mockMetricsService.On("GetNodes").Return(emptyList)

		var resp struct {
			Nodes []*string
		}
		cli.MustPost(query, &resp)

		mockMetricsService.AssertExpectations(t)
		require.Equal(t, emptyList, resp.Nodes)
	})
}

func TestGetMetricOneByNodeID(t *testing.T) {
	t.Run("Get Metric one by id", func(t *testing.T) {
		mockMetricsService := new(lnmock.MockMetricsServices)
		resolvers := graph.Resolver{MetricsService: mockMetricsService}
		cli := client.New(handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &resolvers})))
		if cli == nil {
			panic(cli)
		}

		modelObj := model.MetricOne{Name: "mock", NodeID: "fake_id", Color: "0000"}
		mockMetricsService.On("GetMetricOne", modelObj.NodeID).Return(&modelObj)
		query := utils.ComposeGetMetricOneQuery(modelObj.NodeID)

		var resp struct {
			GetMetricOne *model.MetricOne
		}
		cli.MustPost(query, &resp)

		mockMetricsService.AssertExpectations(t)
		require.Equal(t, &modelObj, resp.GetMetricOne)
	})
}
