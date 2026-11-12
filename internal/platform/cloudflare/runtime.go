package cloudflare

import (
	"encoding/json"
	"net/http"
	"sync"
)

// Runtime reports whether the Asynq dispatcher inside this process is running.
// It is nil unless the Cloudflare profile is enabled.
type Runtime struct {
	mu         sync.RWMutex
	fence      string
	dispatcher bool
}

func NewRuntime(fence string) *Runtime {
	return &Runtime{fence: fence}
}

// SetDispatcher is called by the worker service when it starts and stops.
func (r *Runtime) SetDispatcher(running bool) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.dispatcher = running
	r.mu.Unlock()
}

func (r *Runtime) Dispatcher() bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.dispatcher
}

// ServeHealth is the worker heartbeat. It does not include the fence value.
func (r *Runtime) ServeHealth(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	payload := map[string]any{"status": "unavailable", "dispatcher": false}
	status := http.StatusServiceUnavailable
	if r != nil && r.Dispatcher() {
		payload["status"] = "ok"
		payload["dispatcher"] = true
		status = http.StatusOK
	}
	body, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "health unavailable", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
