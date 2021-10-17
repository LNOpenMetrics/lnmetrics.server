package main

import (
	"testing"

	"github.com/OpenLNMetrics/lnmetrics.server/graph"
	"github.com/OpenLNMetrics/lnmetrics.server/graph/generated"
	"github.com/OpenLNMetrics/lnmetrics.server/graph/model"
	"github.com/OpenLNMetrics/lnmetrics.server/internal/db"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	lnmock "github.com/OpenLNMetrics/lnmetrics.server/tests/mock"
	"github.com/OpenLNMetrics/lnmetrics.server/tests/utils"
	"github.com/stretchr/testify/require"
)

var TEST_DB *db.NoSQLDatabase

func init() {
	db, err := db.NewNoSQLDB(map[string]interface{}{"path": "dbtest"})
	if err != nil {
		panic(err)
	}
	TEST_DB = db
}

// TODO: Working here: https://stackoverflow.com/a/63735464/10854225
// FIX: About  key issue https://github.com/99designs/gqlgen/issues/1376
func TestPushMetricWithNodeId(t *testing.T) {
	t.Run("handle the push operation of the new metrics", func(t *testing.T) {
		mockMetricsService := new(lnmock.MockMetricsServices)
		resolvers := graph.Resolver{MetricsService: mockMetricsService}
		cli := client.New(handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &resolvers})))
		if cli == nil {
			panic(cli)
		}

		modelObj := model.MetricOne{Name: "mock", NodeID: "fake_id", Color: "0000"}
		query := utils.ComposeAddMetricOneQuery(modelObj.NodeID, `{name:\"mock\",node_id:\"fake_id\",color:\"0000\"}`)

		mockMetricsService.On("AddMetricOne", modelObj.NodeID, `{name:"mock",node_id:"fake_id",color:"0000"}`).Return(&modelObj)

		var resp struct {
			AddNodeMetrics utils.AddMetricOneResp
		}
		cli.MustPost(query, &resp)

		mockMetricsService.AssertExpectations(t)
		require.Equal(t, modelObj.NodeID, resp.AddNodeMetrics.NodeID)
	})
}
