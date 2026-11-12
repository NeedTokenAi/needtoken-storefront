package objectstore

import (
	"encoding/hex"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSigningKeyMatchesAWSVector(t *testing.T) {
	key := signingKey("wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY", "20120215", "us-east-1", "iam")
	require.Equal(t, "f4780e2d9f65fa895f9c67b32ce1baf0b0d8a43505a000a1a9e090d414db404d", hex.EncodeToString(key))
}

// The expected signature was computed outside this package for a GET with an
// empty payload, region auto, and no Content-Type. This does not call R2.
func TestSigV4ProtocolVector(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://example.r2.cloudflarestorage.com/unit-bucket/uploads/common/2026/07/asset.txt", nil)
	require.NoError(t, err)
	now := time.Date(2013, 5, 24, 0, 0, 0, 0, time.UTC)
	const payload = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	signRequest(req, "AKIDEXAMPLE", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", "auto", payload, now)
	require.Equal(t, "20130524T000000Z", req.Header.Get("x-amz-date"))
	require.Equal(t, payload, req.Header.Get("x-amz-content-sha256"))
	require.Contains(t, req.Header.Get("Authorization"), "SignedHeaders=host;x-amz-content-sha256;x-amz-date")
	require.Contains(t, req.Header.Get("Authorization"), "Signature=3ec2159dd9c3db84f95a70cddbb4d0c058cb0189c6b2b1301698b1248a4b8671")
}
