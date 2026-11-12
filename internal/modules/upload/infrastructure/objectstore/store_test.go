package objectstore

import (
	"bytes"
	"context"
	"crypto/hmac"
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
	"time"

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

func TestPutRejectsBodyOverTheMemoryBoundWithoutCallingStorage(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests++
	}))
	defer server.Close()
	store, err := NewWithClient(Config{
		Endpoint:        server.URL,
		Bucket:          "unit-bucket",
		AccessKeyID:     "a",
		SecretAccessKey: "b",
		MaxBytes:        8,
	}, server.Client())
	require.NoError(t, err)
	err = store.Put(context.Background(), "/uploads/common/2026/07/asset.txt", bytes.NewReader([]byte("123456789")))
	require.ErrorContains(t, err, "exceeds 8 bytes")
	require.Zero(t, requests)
}

func TestPutSignatureMatchesAnIndependentSigV4Check(t *testing.T) {
	const (
		access = "AKIDEXAMPLE"
		secret = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	)
	var got *http.Request
	var body []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		got = r.Clone(r.Context())
		got.Body = io.NopCloser(bytes.NewReader(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	store, err := NewWithClient(Config{
		Endpoint:        server.URL,
		Bucket:          "unit-bucket",
		Region:          "auto",
		AccessKeyID:     access,
		SecretAccessKey: secret,
		MaxBytes:        32,
	}, server.Client())
	require.NoError(t, err)
	store.now = func() time.Time { return time.Date(2013, 5, 24, 0, 0, 0, 0, time.UTC) }
	payload := []byte("hello-upload")
	require.NoError(t, store.Put(context.Background(), "/uploads/common/2026/07/asset.txt", bytes.NewReader(payload)))
	require.Equal(t, payload, body)
	gotSignature, ok := signatureValue(got.Header.Get("Authorization"))
	require.True(t, ok)
	require.Equal(t, independentSignature(got, body, secret, "auto"), gotSignature)
}

func signatureValue(header string) (string, bool) {
	_, value, ok := strings.Cut(header, "Signature=")
	return value, ok
}

func independentSignature(r *http.Request, payload []byte, secret, region string) string {
	sum := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(sum[:])
	amzDate := r.Header.Get("x-amz-date")
	dateStamp := amzDate[:8]
	names := []string{"host", "x-amz-content-sha256", "x-amz-date"}
	if r.Header.Get("Content-Type") != "" {
		names = append(names, "content-type")
	}
	sortStrings(names)
	canonicalHeaders := ""
	for _, name := range names {
		value := r.Host
		if name == "content-type" {
			value = r.Header.Get("Content-Type")
		} else if name != "host" {
			value = r.Header.Get(name)
		}
		canonicalHeaders += name + ":" + strings.TrimSpace(value) + "\n"
	}
	canonical := strings.Join([]string{
		r.Method,
		r.URL.EscapedPath(),
		r.URL.RawQuery,
		canonicalHeaders,
		strings.Join(names, ";"),
		payloadHash,
	}, "\n")
	scope := dateStamp + "/" + region + "/s3/aws4_request"
	canonicalSum := sha256.Sum256([]byte(canonical))
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		hex.EncodeToString(canonicalSum[:]),
	}, "\n")
	key := independentHMAC([]byte("AWS4"+secret), dateStamp)
	key = independentHMAC(key, region)
	key = independentHMAC(key, "s3")
	key = independentHMAC(key, "aws4_request")
	return hex.EncodeToString(independentHMAC(key, stringToSign))
}

func independentHMAC(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
