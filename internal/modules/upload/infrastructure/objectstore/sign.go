package objectstore

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

func signingKey(secret, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), dateStamp)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	return hmacSHA256(kService, "aws4_request")
}

func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

func signRequest(req *http.Request, accessKey, secret, region string, payloadHash string, now time.Time) {
	if region == "" {
		region = "auto"
	}
	amzDate := now.UTC().Format("20060102T150405Z")
	dateStamp := now.UTC().Format("20060102")
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", payloadHash)

	headers := map[string]string{
		"host":                 strings.TrimSpace(req.URL.Host),
		"x-amz-content-sha256": payloadHash,
		"x-amz-date":           amzDate,
	}
	if contentType := strings.TrimSpace(req.Header.Get("Content-Type")); contentType != "" {
		headers["content-type"] = contentType
	}
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)
	canonicalHeaders := strings.Builder{}
	for _, name := range names {
		canonicalHeaders.WriteString(name)
		canonicalHeaders.WriteByte(':')
		canonicalHeaders.WriteString(headers[name])
		canonicalHeaders.WriteByte('\n')
	}
	signedHeaders := strings.Join(names, ";")
	canonical := strings.Join([]string{
		req.Method,
		canonicalPath(req.URL.EscapedPath()),
		req.URL.RawQuery,
		canonicalHeaders.String(),
		signedHeaders,
		payloadHash,
	}, "\n")
	scope := dateStamp + "/" + region + "/s3/aws4_request"
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		hex.EncodeToString(hashBytes([]byte(canonical))),
	}, "\n")
	signature := hex.EncodeToString(hmacSHA256(signingKey(secret, dateStamp, region, "s3"), stringToSign))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+accessKey+"/"+scope+", SignedHeaders="+signedHeaders+", Signature="+signature)
}

func canonicalPath(escaped string) string {
	if escaped == "" {
		return "/"
	}
	parts := strings.Split(escaped, "/")
	for i, part := range parts {
		unescaped, err := url.PathUnescape(part)
		if err != nil {
			unescaped = part
		}
		parts[i] = url.PathEscape(unescaped)
	}
	return strings.Join(parts, "/")
}

func hashBytes(payload []byte) []byte {
	sum := sha256.Sum256(payload)
	return sum[:]
}

func payloadHash(payload []byte) string {
	return hex.EncodeToString(hashBytes(payload))
}
