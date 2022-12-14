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
	"github.com/alecthomas/kong"
	cron "github.com/robfig/cron/v3"

	"github.com/LNOpenMetrics/lnmetrics.server/cmd/lnmetrics.server/args"
	"github.com/LNOpenMetrics/lnmetrics.server/graph"
	"github.com/LNOpenMetrics/lnmetrics.server/graph/generated"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/backend"
	"github.com/LNOpenMetrics/lnmetrics.server/internal/db"
)

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
	_ = kong.Parse(&args.CliArgs)
	port := args.CliArgs.Port
	path := args.CliArgs.DbPath

	if envPort := os.Getenv("SERVER_PORT"); envPort != "" && port == "8080" {
		port = envPort
	}

	// Server options that can pass throw the interface
	// to configure all the type of interface that we need
	options := make(map[string]interface{})

	if dbPath := os.Getenv("DB_PATH"); dbPath != "" && path == "" {
		options["path"] = dbPath
	} else {
		options["path"] = path
	}

	var lnBackend backend.Backend
	if path := os.Getenv("BACKEND_PATH"); path != "" {
		if isHttpUrl(path) {
			log.GetInstance().Infof("Backend configuration with rest backend")
			lnBackend = backend.NewRestBackend(path)
		} else {
			panic("BACKEND_PATH env prop need to be a valid http url")
		}
	} else {
		log.GetInstance().Infof("Configuration with native backend")
		lnBackend = backend.NewNativeBackend()
	}

	dbVal, err := db.NewNoSQLDB(options)
	if err != nil {
		panic(err)
	}

	log.GetInstance().Info("Check if db need to be migrated")
	for _, network := range []string{"bitcoin", "testnet"} {
		if err := dbVal.Migrate(network); err != nil {
			log.GetInstance().Error(fmt.Sprintf("Error: %s", err))
			panic(err)
		}
	}

	// set up a timer that will call a clean up function periodically
	c := cron.New()
	_, _ = c.AddFunc("@daily", func() {
		if err := dbVal.PeriodicallyCleanUp(); err != nil {
			log.GetInstance().Infof("error: %s", err)
		}
	})
	c.Start()

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

	fmt.Printf("connect to http://localhost:%s/ for GraphQL playground\n", port)
	fmt.Printf("database root path %s\n", options["path"])
	fmt.Printf("%s", http.ListenAndServe(":"+port, nil))
}
