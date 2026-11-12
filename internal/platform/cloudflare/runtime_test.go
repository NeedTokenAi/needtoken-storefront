package cloudflare

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkerHealthStaysDownUntilDispatcherStarts(t *testing.T) {
	runtime := NewRuntime("1")
	recorder := httptest.NewRecorder()
	runtime.ServeHealth(recorder)
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.NotContains(t, recorder.Body.String(), `"fence"`)

	runtime.SetDispatcher(true)
	recorder = httptest.NewRecorder()
	runtime.ServeHealth(recorder)
	require.Equal(t, http.StatusOK, recorder.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, true, payload["dispatcher"])
	_, leaked := payload["fence"]
	require.False(t, leaked)

	runtime.SetDispatcher(false)
	recorder = httptest.NewRecorder()
	runtime.ServeHealth(recorder)
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}
