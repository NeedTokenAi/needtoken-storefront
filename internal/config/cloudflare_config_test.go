package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestCloudflareEnvBindsNamesOnly(t *testing.T) {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	v.SetDefault("cloudflare.enabled", false)
	v.SetDefault("cloudflare.isolation", "")
	v.SetDefault("cloudflare.writer_fence", "")
	v.SetDefault("cloudflare.background_consumers", "")
	v.SetDefault("cloudflare.object_storage.endpoint", "")
	v.SetDefault("cloudflare.object_storage.bucket", "")
	v.SetDefault("cloudflare.object_storage.access_key_id", "")
	v.SetDefault("cloudflare.object_storage.secret_access_key", "")
	t.Setenv("CLOUDFLARE_ENABLED", "true")
	t.Setenv("CLOUDFLARE_ISOLATION", "isolated")
	t.Setenv("CLOUDFLARE_WRITER_FENCE", "4")
	t.Setenv("CLOUDFLARE_BACKGROUND_CONSUMERS", "off")
	t.Setenv("CLOUDFLARE_OBJECT_STORAGE_BUCKET", "unit-bucket")
	t.Setenv("CLOUDFLARE_OBJECT_STORAGE_ENDPOINT", "https://example.r2.cloudflarestorage.com")

	var cfg Config
	require.NoError(t, v.Unmarshal(&cfg))
	require.True(t, cfg.Cloudflare.Enabled)
	require.Equal(t, "isolated", cfg.Cloudflare.Isolation)
	require.Equal(t, "4", cfg.Cloudflare.WriterFence)
	require.Equal(t, "off", cfg.Cloudflare.BackgroundConsumers)
	prefixed := Config{Redis: RedisConfig{Prefix: "dj"}}
	ApplyRedisKeyPrefix(&prefixed, "staging:")
	require.Equal(t, "staging", prefixed.Redis.Prefix)
	ApplyRedisKeyPrefix(&prefixed, "")
	require.Equal(t, "staging", prefixed.Redis.Prefix)
	require.Equal(t, "unit-bucket", cfg.Cloudflare.ObjectStorage.Bucket)
	require.Equal(t, "https://example.r2.cloudflarestorage.com", cfg.Cloudflare.ObjectStorage.Endpoint)
	require.Empty(t, cfg.Cloudflare.ObjectStorage.SecretAccessKey)
}
