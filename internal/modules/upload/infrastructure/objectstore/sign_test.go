package objectstore

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSigningKeyMatchesAWSVector(t *testing.T) {
	key := signingKey("wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY", "20120215", "us-east-1", "iam")
	require.Equal(t, "f4780e2d9f65fa895f9c67b32ce1baf0b0d8a43505a000a1a9e090d414db404d", hex.EncodeToString(key))
}
