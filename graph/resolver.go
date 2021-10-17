package graph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

//go:generate go run github.com/99designs/gqlgen generate

import (
	"github.com/OpenLNMetrics/lnmetrics.server/internal/db"
	"github.com/OpenLNMetrics/lnmetrics.server/internal/services"
)

type Resolver struct {
	MetricsService services.IMetricsService
}

func NewResolver(db db.MetricsDatabase) *Resolver {
	return &Resolver{
		MetricsService: services.NewMetricsService(db),
	}
}
