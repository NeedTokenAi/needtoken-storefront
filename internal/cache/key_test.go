package cache

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfiguredPrefixIsolatesRiskAndChallengeKeys(t *testing.T) {
	previous := redisPrefix
	t.Cleanup(func() { redisPrefix = previous })

	redisPrefix = "staging"
	require.Equal(t, "staging:risk:order_rate:guest_ip:203.0.113.8", Key("risk:order_rate:guest_ip:203.0.113.8"))
	require.Equal(t, "staging:risk:order_rate:user:7", Key("risk:order_rate:user:7"))
	require.Equal(t, "staging:2fa:user:challenge:jti:fails", Key("2fa:user:challenge:jti:fails"))
	require.Equal(t, "staging:2fa:user:challenge:jti:revoked", Key("2fa:user:challenge:jti:revoked"))
	require.Equal(t, "staging:2fa:challenge:jti:fails", Key("2fa:challenge:jti:fails"))
	require.Equal(t, "staging:2fa:challenge:jti:revoked", Key("2fa:challenge:jti:revoked"))
	require.NotContains(t, Key("risk:order_rate:guest_ip:203.0.113.8"), "dj:risk:")

	redisPrefix = "dj"
	require.Equal(t, "dj:risk:order_rate:guest_ip:203.0.113.8", Key("risk:order_rate:guest_ip:203.0.113.8"))
	require.Equal(t, "dj:risk:order_rate:user:7", Key("risk:order_rate:user:7"))
}
