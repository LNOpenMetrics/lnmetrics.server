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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var TEST_DB *db.NoSQLDatabase

func init() {
	db, err := db.NewNoSQLDB(map[string]interface{}{"path": "test"})
	if err != nil {
		panic(err)
	}
	TEST_DB = db
}

// TODO: Working here: https://stackoverflow.com/a/63735464/10854225
// FIX: About  key issue https://github.com/99designs/gqlgen/issues/1376
func TestPushMetricWithNodeId(t *testing.T) {
	t.Run("handle the push operation of the new metrics", func(t *testing.T) {
		resolvers := graph.Resolver{Dbms: TEST_DB}
		cli := client.New(handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &resolvers})))
		if cli == nil {
			panic(cli)
		}

		mockMetricsService := new(lnmock.MockMetricsServices)
		query := utils.ComposeAddMetricOneQuery("1234..", `{\"node_id\": \"1234..\"}`)

		metricObj := model.MetricOne{NodeID: "1234.."}
		mockMetricsService.On("AddNewMetricOne",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string"),
		).Return(&metricObj)

		var resp struct {
			MetricOne utils.AddMetricOneResp
		}
		cli.MustPost(query, &resp)
		mockMetricsService.AssertExpectations(t)
		require.Equal(t, "1234..", resp.MetricOne.NodeId)
	})
}
