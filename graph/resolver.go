package graph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import (
	"github.com/OpenLNMetrics/lnmetrics.server/internal/db"
)

type Resolver struct {
	Dbms db.MetricsDatabase
}
