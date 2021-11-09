package graph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

//go:generate go run ../scripts/gqlgen.go

import (
	"github.com/LNOpenMetrics/lnmetrics.server/internal/db"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/services"
)

type Resolver struct {
	MetricsService services.IMetricsService
}

func NewResolver(db db.MetricsDatabase) *Resolver {
	return &Resolver{
		MetricsService: services.NewMetricsService(db),
	}
}
