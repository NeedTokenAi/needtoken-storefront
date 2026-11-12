package cloudflare

import (
	"testing"

	"github.com/dujiao-next/internal/config"
	"github.com/stretchr/testify/require"
)

func TestDisabledProfileKeepsCurrentStartup(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite", DSN: "./db/dujiao.db"},
	}
	require.NoError(t, ValidateStartup(cfg, ModeAPI))
	mode, err := ResolveMode(false, ModeAPI)
	require.NoError(t, err)
	require.Equal(t, ModeAPI, mode)
	mode, err = ResolveMode(false, "")
	require.NoError(t, err)
	require.Empty(t, mode)
}

func TestEnabledProfileRequiresAllOrPair(t *testing.T) {
	for _, mode := range []string{"", ModeAll, ModeAPIWorker} {
		resolved, err := ResolveMode(true, mode)
		require.NoError(t, err, mode)
		require.Equal(t, ModeAll, resolved)
	}
	_, err := ResolveMode(true, ModeAPI)
	require.ErrorContains(t, err, "Asynq")
	_, err = ResolveMode(true, ModeWorker)
	require.ErrorContains(t, err, "complete API")
}

func TestEnabledProfileFailsClosedWithoutDurableTarget(t *testing.T) {
	cfg := enabledConfig()
	cfg.Cloudflare.ObjectStorage.Bucket = ""
	err := Validate(cfg)
	require.ErrorIs(t, err, ErrDurableTargetMissing)

	cfg = enabledConfig()
	cfg.Database.Driver = "sqlite"
	cfg.Database.DSN = "./db/dujiao.db"
	err = ValidateStartup(cfg, ModeAll)
	require.ErrorContains(t, err, "Postgres")
	require.NotErrorIs(t, err, ErrDurableTargetMissing)

	cfg = enabledConfig()
	cfg.Database.DSN = "file:dujiao.db"
	require.ErrorContains(t, Validate(cfg), "Postgres")

	cfg = enabledConfig()
	cfg.Queue.Enabled = false
	require.ErrorContains(t, Validate(cfg), "Asynq")

	cfg = enabledConfig()
	cfg.Cloudflare.WriterFence = ""
	require.ErrorContains(t, Validate(cfg), "writer fence")

	cfg = enabledConfig()
	cfg.Cloudflare.ObjectStorage.AccessKeyID = ""
	require.ErrorIs(t, Validate(cfg), ErrDurableTargetMissing)

	cfg = enabledConfig()
	cfg.Cloudflare.Isolation = ""
	err = Validate(cfg)
	require.ErrorIs(t, err, ErrIsolationRequired)
	require.ErrorContains(t, err, "does not stop the existing Cloud Run writer")
}

func TestEnabledProfileAcceptsPostgresAndObjectStorage(t *testing.T) {
	require.NoError(t, ValidateStartup(enabledConfig(), ModeAPIWorker))
	keyword := enabledConfig()
	keyword.Database.DSN = "host=db.example user=store dbname=storefront sslmode=require"
	require.NoError(t, Validate(keyword))
}

func TestModeFromArgs(t *testing.T) {
	mode, ok := ModeFromArgs([]string{"-mode", "api"})
	require.True(t, ok)
	require.Equal(t, "api", mode)
	mode, ok = ModeFromArgs([]string{"--mode=api+worker"})
	require.True(t, ok)
	require.Equal(t, ModeAPIWorker, mode)
	_, ok = ModeFromArgs([]string{"-config", "config.yml"})
	require.False(t, ok)
	mode, ok = ModeFromArgs([]string{"-mode"})
	require.True(t, ok)
	require.Empty(t, mode)
}

func enabledConfig() *config.Config {
	return &config.Config{
		Cloudflare: config.CloudflareConfig{
			Enabled:     true,
			Isolation:   IsolationIsolated,
			WriterFence: "1",
			ObjectStorage: config.ObjectStorageConfig{
				Endpoint:        "https://example.r2.cloudflarestorage.com",
				Bucket:          "storefront-uploads",
				AccessKeyID:     "test-access-key",
				SecretAccessKey: "test-secret-key",
			},
		},
		Database: config.DatabaseConfig{
			Driver: "postgres",
			DSN:    "postgres://store@db.example/storefront?sslmode=require",
		},
		Queue: config.QueueConfig{Enabled: true, Host: "redis.example"},
		Redis: config.RedisConfig{Enabled: true, Host: "redis.example", Prefix: "dj"},
	}
}
