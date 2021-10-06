package main

import (
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"

	"github.com/OpenLNMetrics/ln-metrics-server/graph"
	"github.com/OpenLNMetrics/ln-metrics-server/graph/generated"
	"github.com/OpenLNMetrics/ln-metrics-server/internal/db"
)

const DEFAULT_PORT = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = DEFAULT_PORT
	}

	options := make(map[string]interface{})

	if path := os.Getenv("DB_PATH"); path != "" {
		options["path"] = path
	}

	dbVal, err := db.NewNoSQLDB(options)
	if err != nil {
		panic(err)
	}

	srv := handler.NewDefaultServer(generated.NewExecutableSchema(
		generated.Config{Resolvers: &graph.Resolver{
			Dbms: dbVal,
		}}))

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
