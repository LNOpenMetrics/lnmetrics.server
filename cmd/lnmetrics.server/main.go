package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/joho/godotenv"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/OpenLNMetrics/lnmetrics.utils/log"

	"github.com/OpenLNMetrics/lnmetrics.server/graph"
	"github.com/OpenLNMetrics/lnmetrics.server/graph/generated"
	"github.com/OpenLNMetrics/lnmetrics.server/internal/db"
)

const DEFAULT_PORT = "8080"

func init() {
	if err := godotenv.Load(); err != nil {
		log.GetInstance().Info(fmt.Sprintf("%s", err))
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

	var mb int64 = 1 << 20
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: graph.NewResolver(dbVal)}))
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{
		MaxMemory:     100 * mb,
		MaxUploadSize: 100 * mb,
	})
	srv.SetQueryCache(lru.New(1))
	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	fmt.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	fmt.Printf("%s", http.ListenAndServe(":"+port, nil))
}
