package app

import (
	"testing"

	"github.com/dujiao-next/internal/config"
	"github.com/stretchr/testify/require"
)

func TestCloudflareProfileRejectsIncompleteStartupBeforeDatabase(t *testing.T) {
	cfg := &config.Config{
		Cloudflare: config.CloudflareConfig{
			Enabled:     true,
			WriterFence: "1",
			ObjectStorage: config.ObjectStorageConfig{
				Endpoint:        "https://example.r2.cloudflarestorage.com",
				Bucket:          "storefront-uploads",
				AccessKeyID:     "test-access",
				SecretAccessKey: "test-secret",
			},
		},
		Database: config.DatabaseConfig{Driver: "postgres", DSN: "postgres://store@db.example/storefront"},
		Queue:    config.QueueConfig{Enabled: true, Host: "redis.example"},
		Redis:    config.RedisConfig{Enabled: true, Host: "redis.example", Prefix: "dj"},
	}

	_, err := BuildRunner(cfg, ModeAPI)
	require.ErrorContains(t, err, "Asynq")

	cfg.Database.Driver = "sqlite"
	cfg.Database.DSN = "./db/dujiao.db"
	_, err = BuildRunner(cfg, ModeAll)
	require.ErrorContains(t, err, "Postgres")

	cfg.Database.Driver = "postgres"
	cfg.Database.DSN = "postgres://store@db.example/storefront"
	cfg.Cloudflare.ObjectStorage.Bucket = ""
	_, err = BuildRunner(cfg, "api+worker")
	require.ErrorContains(t, err, "durable upload target")
}
