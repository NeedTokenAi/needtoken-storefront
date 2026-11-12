package objectstore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dujiao-next/internal/modules/upload/contract"
	"github.com/stretchr/testify/require"
)

func TestNewFailsClosedWithoutTarget(t *testing.T) {
	_, err := New(Config{})
	require.Error(t, err)
	_, err = New(Config{Endpoint: "https://example.r2.cloudflarestorage.com", Bucket: "uploads"})
	require.Error(t, err)
}

func TestSaveDoesNotUseLocalDisk(t *testing.T) {
	var gotMethod, gotPath, gotHash string
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotHash = r.Header.Get("x-amz-content-sha256")
		body, _ := io.ReadAll(r.Body)
		gotBody = body
		if !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
			http.Error(w, "missing signature", http.StatusUnauthorized)
			return
		}
		switch r.Method {
		case http.MethodPut:
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			_, _ = w.Write([]byte("png-bytes"))
		case http.MethodHead:
			w.Header().Set("Content-Length", "9")
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(wd) })

	store, err := NewWithClient(Config{
		Endpoint:        server.URL,
		Bucket:          "unit-bucket",
		Region:          "auto",
		AccessKeyID:     "test-access",
		SecretAccessKey: "test-secret",
	}, server.Client())
	require.NoError(t, err)

	publicURL, err := store.Save(contract.StoreInput{
		Source:   strings.NewReader("hello-upload"),
		Scene:    "common",
		Year:     "2026",
		Month:    "07",
		Filename: "asset.txt",
	})
	require.NoError(t, err)
	require.Equal(t, "/uploads/common/2026/07/asset.txt", publicURL)
	require.Equal(t, http.MethodPut, gotMethod)
	require.Equal(t, "/unit-bucket/uploads/common/2026/07/asset.txt", gotPath)
	require.Equal(t, "hello-upload", string(gotBody))
	sum := sha256.Sum256([]byte("hello-upload"))
	require.Equal(t, hex.EncodeToString(sum[:]), gotHash)
	_, statErr := os.Stat(filepath.Join(dir, "uploads"))
	require.ErrorIs(t, statErr, fs.ErrNotExist)

	body, err := store.Open(publicURL)
	require.NoError(t, err)
	defer body.Close()
	streamed, err := io.ReadAll(body)
	require.NoError(t, err)
	require.Equal(t, "png-bytes", string(streamed))

	info, err := store.Stat("uploads/common/2026/07/asset.txt")
	require.NoError(t, err)
	require.Equal(t, int64(9), info.Size())
	require.NoError(t, store.Remove(publicURL))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://store.example"+publicURL, nil)
	store.ServePublic(recorder, request, publicURL)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "png-bytes", recorder.Body.String())
}

func TestObjectKeyRejectsEscape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("escaped path reached storage: %s", r.URL.Path)
	}))
	defer server.Close()
	store, err := NewWithClient(Config{
		Endpoint: server.URL, Bucket: "unit-bucket", AccessKeyID: "a", SecretAccessKey: "b",
	}, server.Client())
	require.NoError(t, err)
	err = store.Put(context.Background(), "/uploads/../secrets", bytes.NewReader([]byte("x")))
	require.Error(t, err)
	_, err = store.Open("/uploads/%2e%2e/secrets")
	require.Error(t, err)
}
