package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestConfigureTrustedProxiesRejectsSpoofedForwardedIPFromUntrustedPeer(t *testing.T) {
	engine := gin.New()
	if err := configureTrustedProxies(engine, nil); err != nil {
		t.Fatal(err)
	}
	engine.GET("/ip", func(c *gin.Context) { c.String(http.StatusOK, c.ClientIP()) })

	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.RemoteAddr = "203.0.113.9:1234"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)
	if resp.Body.String() != "203.0.113.9" {
		t.Fatalf("untrusted peer spoofed client IP: %q", resp.Body.String())
	}
}

func TestConfigureTrustedProxiesAcceptsForwardedIPFromConfiguredProxy(t *testing.T) {
	engine := gin.New()
	if err := configureTrustedProxies(engine, []string{"127.0.0.1/32"}); err != nil {
		t.Fatal(err)
	}
	engine.GET("/ip", func(c *gin.Context) { c.String(http.StatusOK, c.ClientIP()) })

	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)
	if resp.Body.String() != "1.2.3.4" {
		t.Fatalf("trusted proxy client IP=%q, want 1.2.3.4", resp.Body.String())
	}
}

func TestConfigureTrustedProxiesRejectsInvalidCIDR(t *testing.T) {
	if err := configureTrustedProxies(gin.New(), []string{"not-a-cidr"}); err == nil {
		t.Fatal("expected invalid trusted proxy configuration to fail")
	}
}

func TestConfigureTrustedProxiesRejectsTrustAllNetworks(t *testing.T) {
	for _, network := range []string{"0.0.0.0/0", "::/0"} {
		t.Run(network, func(t *testing.T) {
			if err := configureTrustedProxies(gin.New(), []string{network}); err == nil {
				t.Fatalf("expected %s to be rejected", network)
			}
		})
	}
}

func TestConfigureTrustedProxiesPrefersCFConnectingIPFromTrustedProxyOnly(t *testing.T) {
	engine := gin.New()
	if err := configureTrustedProxies(engine, []string{"10.0.0.0/8"}); err != nil {
		t.Fatal(err)
	}
	engine.GET("/ip", func(c *gin.Context) { c.String(http.StatusOK, c.ClientIP()) })

	// 来自受信任容器网桥的请求：优先读取 CF-Connecting-IP，忽略可能伪造的 X-Forwarded-For
	trustedReq := httptest.NewRequest(http.MethodGet, "/ip", nil)
	trustedReq.RemoteAddr = "10.0.1.5:8080"
	trustedReq.Header.Set("CF-Connecting-IP", "198.51.100.77")
	trustedReq.Header.Set("X-Forwarded-For", "1.2.3.4")
	trustedResp := httptest.NewRecorder()
	engine.ServeHTTP(trustedResp, trustedReq)
	if trustedResp.Body.String() != "198.51.100.77" {
		t.Fatalf("trusted proxy CF-Connecting-IP=%q, want 198.51.100.77", trustedResp.Body.String())
	}

	// 来自非受信任公网直连的请求：忽略伪造的 CF-Connecting-IP
	untrustedReq := httptest.NewRequest(http.MethodGet, "/ip", nil)
	untrustedReq.RemoteAddr = "203.0.113.9:1234"
	untrustedReq.Header.Set("CF-Connecting-IP", "198.51.100.77")
	untrustedResp := httptest.NewRecorder()
	engine.ServeHTTP(untrustedResp, untrustedReq)
	if untrustedResp.Body.String() != "203.0.113.9" {
		t.Fatalf("untrusted peer spoofed CF-Connecting-IP: %q", untrustedResp.Body.String())
	}
}
