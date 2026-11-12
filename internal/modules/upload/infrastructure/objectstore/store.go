package objectstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/dujiao-next/internal/modules/upload/contract"
)

// Config is an R2-compatible object storage target. It is used only when the
// Cloudflare profile is enabled.
type Config struct {
	Endpoint        string
	Bucket          string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Prefix          string
}

// Store writes and reads upload objects. It does not use the local disk.
type Store struct {
	endpoint string
	bucket   string
	region   string
	prefix   string
	access   string
	secret   string
	client   *http.Client
	now      func() time.Time
}

var _ contract.Store = (*Store)(nil)

// New validates the durable target and returns an object store.
func New(cfg Config) (*Store, error) {
	return NewWithClient(cfg, nil)
}

// NewWithClient is New with an HTTP client, used by tests.
func NewWithClient(cfg Config, client *http.Client) (*Store, error) {
	endpoint := strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed == nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return nil, errors.New("cloudflare profile: durable upload target is missing")
	}
	bucket := strings.TrimSpace(cfg.Bucket)
	if bucket == "" || strings.Contains(bucket, "/") {
		return nil, errors.New("cloudflare profile: durable upload target is missing")
	}
	if strings.TrimSpace(cfg.AccessKeyID) == "" || strings.TrimSpace(cfg.SecretAccessKey) == "" {
		return nil, errors.New("cloudflare profile: durable upload target is missing")
	}
	if client == nil {
		client = &http.Client{Transport: defaultTransport()}
	}
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "auto"
	}
	prefix := strings.Trim(strings.TrimSpace(cfg.Prefix), "/")
	return &Store{
		endpoint: endpoint,
		bucket:   bucket,
		region:   region,
		prefix:   prefix,
		access:   strings.TrimSpace(cfg.AccessKeyID),
		secret:   strings.TrimSpace(cfg.SecretAccessKey),
		client:   client,
		now:      time.Now,
	}, nil
}

func defaultTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// Save stores an upload and returns its public /uploads URL.
func (s *Store) Save(input contract.StoreInput) (string, error) {
	publicPath := fmt.Sprintf("/uploads/%s/%s/%s/%s", input.Scene, input.Year, input.Month, input.Filename)
	if err := s.Put(context.Background(), publicPath, input.Source); err != nil {
		return "", err
	}
	return publicPath, nil
}

// Put writes an upload object. publicPath must stay under /uploads.
func (s *Store) Put(ctx context.Context, publicPath string, body io.Reader) error {
	if s == nil {
		return errors.New("cloudflare profile: durable upload target is missing")
	}
	key, err := objectKey(publicPath)
	if err != nil {
		return err
	}
	payload, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	return s.do(ctx, http.MethodPut, key, payload, contentTypeFor(key))
}

// Open reads an upload. The caller closes the reader.
func (s *Store) Open(name string) (io.ReadCloser, error) {
	body, _, err := s.open(context.Background(), name)
	return body, err
}

// Stat returns object size. A missing object is fs.ErrNotExist.
func (s *Store) Stat(name string) (fs.FileInfo, error) {
	key, err := objectKey(name)
	if err != nil {
		return nil, err
	}
	response, err := s.request(context.Background(), http.MethodHead, key, nil, "")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return nil, fs.ErrNotExist
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("object stat: status %d", response.StatusCode)
	}
	return objectInfo{name: path.Base(key), size: response.ContentLength}, nil
}

// Remove deletes an upload. A missing object is fs.ErrNotExist.
func (s *Store) Remove(name string) error {
	if s == nil {
		return errors.New("cloudflare profile: durable upload target is missing")
	}
	key, err := objectKey(name)
	if err != nil {
		return err
	}
	response, err := s.request(context.Background(), http.MethodDelete, key, nil, "")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return fs.ErrNotExist
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("object delete: status %d", response.StatusCode)
	}
	return nil
}

// ServePublic streams an upload to the response without buffering the body.
func (s *Store) ServePublic(w http.ResponseWriter, r *http.Request, publicPath string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if r.Method == http.MethodHead {
		info, err := s.Stat(publicPath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if contentType := contentTypeFor(publicPath); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
		w.WriteHeader(http.StatusOK)
		return
	}
	body, contentType, err := s.open(r.Context(), publicPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer body.Close()
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, body)
}

func (s *Store) open(ctx context.Context, name string) (io.ReadCloser, string, error) {
	if s == nil {
		return nil, "", errors.New("cloudflare profile: durable upload target is missing")
	}
	key, err := objectKey(name)
	if err != nil {
		return nil, "", err
	}
	response, err := s.request(ctx, http.MethodGet, key, nil, "")
	if err != nil {
		return nil, "", err
	}
	if response.StatusCode == http.StatusNotFound {
		_ = response.Body.Close()
		return nil, "", fs.ErrNotExist
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_ = response.Body.Close()
		return nil, "", fmt.Errorf("object get: status %d", response.StatusCode)
	}
	contentType := strings.TrimSpace(response.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = contentTypeFor(key)
	}
	return response.Body, contentType, nil
}

func (s *Store) do(ctx context.Context, method, key string, payload []byte, contentType string) error {
	response, err := s.request(ctx, method, key, payload, contentType)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		return fmt.Errorf("object %s: status %d: %s", method, response.StatusCode, strings.TrimSpace(string(snippet)))
	}
	return nil
}

func (s *Store) request(ctx context.Context, method, key string, payload []byte, contentType string) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	objectURL := s.endpoint + "/" + s.bucket + "/" + escapeKey(s.storageKey(key))
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, objectURL, body)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.ContentLength = int64(len(payload))
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	now := time.Now
	if s.now != nil {
		now = s.now
	}
	signRequest(req, s.access, s.secret, s.region, payloadHash(payload), now())
	return s.client.Do(req)
}

func (s *Store) storageKey(key string) string {
	if s.prefix == "" {
		return key
	}
	return s.prefix + "/" + key
}

func objectKey(publicOrRelative string) (string, error) {
	value := strings.TrimSpace(publicOrRelative)
	value = strings.TrimPrefix(value, "/")
	if value == "" || strings.Contains(value, "\\") || strings.Contains(value, "\x00") {
		return "", errors.New("upload path is invalid")
	}
	if strings.Contains(value, "%") {
		unescaped, err := url.PathUnescape(value)
		if err != nil || strings.Contains(unescaped, "..") {
			return "", errors.New("upload path is invalid")
		}
		value = unescaped
	}
	cleaned := path.Clean(value)
	if cleaned != value || cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.Contains(cleaned, "/../") {
		return "", errors.New("upload path is invalid")
	}
	if cleaned != "uploads" && !strings.HasPrefix(cleaned, "uploads/") {
		return "", errors.New("upload path is invalid")
	}
	if cleaned == "uploads" || strings.HasSuffix(cleaned, "/") {
		return "", errors.New("upload path is invalid")
	}
	return cleaned, nil
}

func escapeKey(key string) string {
	parts := strings.Split(key, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func contentTypeFor(key string) string {
	ext := path.Ext(key)
	if ext == "" {
		return "application/octet-stream"
	}
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		return "application/octet-stream"
	}
	return contentType
}

type objectInfo struct {
	name string
	size int64
}

func (o objectInfo) Name() string       { return o.name }
func (o objectInfo) Size() int64        { return o.size }
func (o objectInfo) Mode() fs.FileMode  { return 0o644 }
func (o objectInfo) ModTime() time.Time { return time.Time{} }
func (o objectInfo) IsDir() bool        { return false }
func (o objectInfo) Sys() any           { return nil }
