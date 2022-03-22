package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"

	"github.com/joho/godotenv"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/LNOpenMetrics/lnmetrics.utils/log"

	"github.com/LNOpenMetrics/lnmetrics.server/graph"
	"github.com/LNOpenMetrics/lnmetrics.server/graph/generated"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/backend"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/db"
)

const DefaultPort = "8080"

func init() {
	if err := godotenv.Load(); err != nil {
		log.GetInstance().Info(fmt.Sprintf("%s", err))
	}
}

// Put in the lnmetrics.utils
func isHttpUrl(path string) bool {
	return strings.HasPrefix(path, "http")
}

func main() {
	port := DefaultPort
	if envPort := os.Getenv("SERVER_PORT"); envPort != "" {
		port = envPort
	}

	// Server options that can pass throw the interface
	// to configure all the type of interface that we need
	options := make(map[string]interface{})

	if path := os.Getenv("DB_PATH"); path != "" {
		options["path"] = path
	}

	var lnBackend backend.Backend
	if path := os.Getenv("BACKEND_PATH"); path != "" {
		if isHttpUrl(path) {
			lnBackend = backend.NewRestBackend(path)
		} else {
			panic("No backend supported")
		}
	} else {
		panic("BACKEND_PATH env prop it is not set")
	}

	dbVal, err := db.NewNoSQLDB(options)
	if err != nil {
		panic(err)
	}

	log.GetInstance().Info("Check if db need to be migrated")
	if err := dbVal.Migrate(); err != nil {
		log.GetInstance().Error(fmt.Sprintf("Error: %s", err))
		panic(err)
	}

	var mb int64 = 1 << 20
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(
		generated.Config{
			Resolvers: graph.NewResolver(dbVal, lnBackend),
		}))
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
