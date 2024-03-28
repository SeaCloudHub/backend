package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/SeaCloudHub/backend/adapters/httpserver"
	"github.com/SeaCloudHub/backend/adapters/postgrestore"
	redisstore "github.com/SeaCloudHub/backend/adapters/redis_store"

	"github.com/SeaCloudHub/backend/pkg/config"
	"github.com/SeaCloudHub/backend/pkg/logger"
	"github.com/SeaCloudHub/backend/pkg/sentry"
	sentrygo "github.com/getsentry/sentry-go"
	_ "github.com/lib/pq"
)

func main() {
	applog, err := logger.NewAppLogger()
	if err != nil {
		log.Fatalf("cannot load config: %v\n", err)
	}
	defer logger.Sync(applog)

	cfg, err := config.LoadConfig()
	if err != nil {
		applog.Fatal(err)
	}

	err = sentrygo.Init(sentrygo.ClientOptions{
		Dsn:              cfg.SentryDSN,
		Environment:      cfg.AppEnv,
		AttachStacktrace: true,
	})
	if err != nil {
		applog.Fatalf("cannot init sentry: %v", err)
	}
	defer sentrygo.Flush(sentry.FlushTime)

	db, err := postgrestore.NewConnection(postgrestore.ParseFromConfig(cfg))
	if err != nil {
		applog.Fatal(err)
	}

	server, err := httpserver.New(cfg, applog)
	if err != nil {
		applog.Fatal(err)
	}

	server.BookStore = postgrestore.NewBookStore(db)

	redisSvc, err := redisstore.NewRedisStorage(cfg)
	if err != nil {
		applog.Fatal(err)
	}
	server.RedisSvc = redisSvc
	addr := fmt.Sprintf(":%d", cfg.Port)
	applog.Info("server started!")
	applog.Fatal(http.ListenAndServe(addr, server))
}
