package cloudflare

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/dujiao-next/internal/config"
)

const (
	ModeAll       = "all"
	ModeAPI       = "api"
	ModeWorker    = "worker"
	ModeAPIWorker = "api+worker"

	// This phase keeps the existing Postgres and Upstash. Only the R2 bucket is a
	// separate isolated resource.
	isolatedObjectBucket   = "needtoken-staging"
	isolatedRedisKeyPrefix = "staging"
)

// ErrDurableTargetMissing means the Cloudflare profile is on and no durable
// upload target was configured. Callers must fail closed.
var ErrDurableTargetMissing = errors.New("cloudflare profile: durable upload target is missing")

// ValidateStartup rejects a Cloudflare profile that would drop the Asynq
// dispatcher, open SQLite, or start without a durable upload target.
// A disabled profile returns nil and does not inspect the database.
func ValidateStartup(cfg *config.Config, mode string) error {
	if cfg == nil || !cfg.Cloudflare.Enabled {
		return nil
	}
	if _, err := ResolveMode(true, mode); err != nil {
		return err
	}
	return Validate(cfg)
}

// ResolveMode keeps the requested mode when the profile is off.
// When the profile is on, the process must be the existing all mode or the
// api+worker pair, both of which run the API and the Asynq dispatcher.
func ResolveMode(enabled bool, mode string) (string, error) {
	mode = strings.TrimSpace(mode)
	if !enabled {
		return mode, nil
	}
	switch mode {
	case "", ModeAll, ModeAPIWorker:
		return ModeAll, nil
	case ModeAPI:
		return "", errors.New("cloudflare profile: mode api does not start the Asynq dispatcher")
	case ModeWorker:
		return "", errors.New("cloudflare profile: mode worker is not a complete API plus worker pair")
	default:
		return "", fmt.Errorf("cloudflare profile: mode %q is not all or api+worker", mode)
	}
}

// Validate checks durable dependencies. It does not open a database and does
// not substitute SQLite, a memory store, or a local upload directory.
func Validate(cfg *config.Config) error {
	if cfg == nil || !cfg.Cloudflare.Enabled {
		return nil
	}
	driver := strings.ToLower(strings.TrimSpace(cfg.Database.Driver))
	if driver != "postgres" || !postgresDSN(cfg.Database.DSN) {
		return errors.New("cloudflare profile: database must be an existing Postgres DSN")
	}
	if !cfg.Queue.Enabled || strings.TrimSpace(cfg.Queue.Host) == "" {
		return errors.New("cloudflare profile: Asynq queue must stay enabled")
	}
	if !cfg.Redis.Enabled || strings.TrimSpace(cfg.Redis.Host) == "" || strings.TrimSpace(cfg.Redis.Prefix) == "" {
		return errors.New("cloudflare profile: Redis must stay enabled for the queue and writer fence")
	}
	if strings.TrimSpace(cfg.Cloudflare.WriterFence) == "" {
		return errors.New("cloudflare profile: writer fence is missing")
	}
	if strings.TrimSpace(cfg.Cloudflare.Isolation) != IsolationIsolated {
		return ErrIsolationRequired
	}
	if err := validateObjectStorage(cfg.Cloudflare.ObjectStorage); err != nil {
		return err
	}
	if bucket := strings.TrimSpace(cfg.Cloudflare.ObjectStorage.Bucket); bucket != isolatedObjectBucket {
		return fmt.Errorf("cloudflare profile: object bucket %q is outside the isolated allowlist", bucket)
	}
	if normalizeRedisKeyPrefix(cfg.Redis.Prefix) != isolatedRedisKeyPrefix {
		return errors.New("cloudflare profile: REDIS_KEY_PREFIX must be staging: so this process does not use production keys")
	}
	if !strings.EqualFold(strings.TrimSpace(cfg.Cloudflare.BackgroundConsumers), "off") {
		return errors.New("cloudflare profile: CLOUDFLARE_BACKGROUND_CONSUMERS must be off so this process does not consume production queues or schedules")
	}
	return nil
}

func normalizeRedisKeyPrefix(prefix string) string {
	return strings.TrimSuffix(strings.TrimSpace(prefix), ":")
}

func validateObjectStorage(cfg config.ObjectStorageConfig) error {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed == nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return fmt.Errorf("%w: object storage endpoint is missing", ErrDurableTargetMissing)
	}
	bucket := strings.TrimSpace(cfg.Bucket)
	if bucket == "" || strings.Contains(bucket, "/") || strings.Contains(bucket, "\\") {
		return fmt.Errorf("%w: object storage bucket is missing", ErrDurableTargetMissing)
	}
	if strings.TrimSpace(cfg.AccessKeyID) == "" || strings.TrimSpace(cfg.SecretAccessKey) == "" {
		return fmt.Errorf("%w: object storage credentials are missing", ErrDurableTargetMissing)
	}
	return nil
}

func postgresDSN(dsn string) bool {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return false
	}
	lower := strings.ToLower(dsn)
	if strings.Contains(lower, "sqlite") || strings.HasSuffix(lower, ".db") || strings.HasPrefix(lower, "file:") {
		return false
	}
	if strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://") {
		return true
	}
	return strings.Contains(lower, "host=") && strings.Contains(lower, "dbname=")
}

// ModeFromArgs reads the process mode without parsing the global flag set.
// The boolean is false when -mode is absent.
func ModeFromArgs(args []string) (string, bool) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		if value, ok := strings.CutPrefix(arg, "--mode="); ok {
			return value, true
		}
		if value, ok := strings.CutPrefix(arg, "-mode="); ok {
			return value, true
		}
		if arg == "--mode" || arg == "-mode" {
			if i+1 < len(args) {
				return args[i+1], true
			}
			return "", true
		}
	}
	return "", false
}
