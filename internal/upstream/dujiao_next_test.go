package upstream

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
)

// 上游返回的图片地址只允许与上游站点同主机，且大小受限，防止 SSRF 把内部响应落到公开目录。
func TestDownloadImageRestrictsHostAndSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/huge.png" {
			_, _ = w.Write([]byte(strings.Repeat("x", maxUpstreamImageBytes+1)))
			return
		}
		_, _ = w.Write([]byte("png"))
	}))
	defer server.Close()
	adapter := NewDujiaoNextAdapter(&siteconnectiondomain.Connection{BaseURL: server.URL}, t.TempDir())

	if local, err := adapter.DownloadImage(context.Background(), "/uploads/a.png"); err != nil || !strings.HasSuffix(local, ".png") {
		t.Fatalf("same-host image should download: local=%q err=%v", local, err)
	}
	if _, err := adapter.DownloadImage(context.Background(), "http://169.254.169.254/latest/meta-data"); err == nil {
		t.Fatal("foreign host must be rejected")
	}
	if _, err := adapter.DownloadImage(context.Background(), "/huge.png"); err == nil {
		t.Fatal("oversized image must be rejected")
	}
}

func TestDownloadImageUsesDurableSink(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("png"))
	}))
	defer server.Close()
	dir := t.TempDir()
	adapter := NewDujiaoNextAdapter(&siteconnectiondomain.Connection{BaseURL: server.URL}, dir)
	if err := adapter.RequireDurableImages(nil); err == nil {
		t.Fatal("missing durable target must fail closed")
	}
	sink := &memoryImageSink{}
	if err := adapter.RequireDurableImages(sink); err != nil {
		t.Fatal(err)
	}
	local, err := adapter.DownloadImage(context.Background(), "/uploads/a.png")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(local, "/uploads/upstream/") || string(sink.body) != "png" {
		t.Fatalf("durable image = %s body %q", local, sink.body)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("durable download wrote local files: %v", entries)
	}
}

type memoryImageSink struct {
	body []byte
}

func (s *memoryImageSink) Put(_ context.Context, _ string, body io.Reader) error {
	payload, err := io.ReadAll(body)
	s.body = payload
	return err
}
