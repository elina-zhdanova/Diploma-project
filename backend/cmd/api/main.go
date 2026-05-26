package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/itshop/api/internal/config"
	"github.com/itshop/api/internal/dbschema"
	"github.com/itshop/api/internal/httpserver"
	"github.com/itshop/api/internal/objectstore"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	pc, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db config: %v", err)
	}
	if pc.ConnConfig.RuntimeParams == nil {
		pc.ConnConfig.RuntimeParams = map[string]string{}
	}
	pc.ConnConfig.RuntimeParams["client_encoding"] = "UTF8"
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	if err := dbschema.Ensure(ctx, pool); err != nil {
		log.Fatalf("db schema ensure: %v", err)
	}

	mc, err := objectstore.NewClient(cfg)
	if err != nil {
		log.Fatalf("minio client: %v", err)
	}
	if err := objectstore.EnsureBucket(ctx, mc, cfg.S3Bucket); err != nil {
		log.Fatalf("minio bucket: %v", err)
	}

	mux := httpserver.New(cfg, pool, mc)
	log.Printf("itshop-api listening on :%s", cfg.HTTPPort)
	if err := http.ListenAndServe(":"+cfg.HTTPPort, withCORS(mux)); err != nil {
		log.Fatal(err)
	}
}

func withCORS(next http.Handler) http.Handler {
	origins := os.Getenv("CORS_ALLOW_ORIGIN")
	if origins == "" {
		origins = "*"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origins)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
