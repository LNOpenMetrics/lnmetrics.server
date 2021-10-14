package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"

	"github.com/OpenLNMetrics/lnmetrics.server/graph"
	"github.com/OpenLNMetrics/lnmetrics.server/graph/generated"
	"github.com/OpenLNMetrics/lnmetrics.server/internal/db"
)

const DEFAULT_PORT = "8080"

func init() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}
}

func main() {
	port := DEFAULT_PORT
	if envPort := os.Getenv("SERVER_PORT"); envPort != "" {
		port = envPort
	}

	// Server option that can pass throw the interface
	// to configure all the type of interface that we need
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
