package diagnosticscontract

import (
	"context"
	"errors"
	"time"
)

type ServiceStatusProvider string

const (
	ServiceStatusOpenAI ServiceStatusProvider = "openai"
	ServiceStatusClaude ServiceStatusProvider = "claude"
)

var (
	ErrServiceStatusUnavailable = errors.New("service status: official source unavailable")
	ErrServiceStatusProvider    = errors.New("service status: unsupported provider")
)

func ValidServiceStatusProvider(provider ServiceStatusProvider) bool {
	return provider == ServiceStatusOpenAI || provider == ServiceStatusClaude
}

type ServiceStatusPage struct {
	Name      string     `json:"name"`
	URL       string     `json:"url"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

type ServiceStatusState struct {
	Indicator   string `json:"indicator"`
	Description string `json:"description"`
}

// IDs and group relationships are preserved: names such as Login are shared by
// unrelated products and must not be used alone to infer consumer-app health.
type ServiceStatusComponent struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Status     string   `json:"status"`
	Group      bool     `json:"group,omitempty"`
	GroupID    string   `json:"group_id,omitempty"`
	Components []string `json:"components,omitempty"`
}

type ServiceStatusSummary struct {
	Page       ServiceStatusPage        `json:"page"`
	Status     ServiceStatusState       `json:"status"`
	Components []ServiceStatusComponent `json:"components"`
}

type ServiceStatusSnapshot struct {
	Provider  ServiceStatusProvider `json:"provider"`
	SourceURL string                `json:"source_url"`
	// FetchedAt is this site's successful retrieval time. Page.UpdatedAt remains
	// the provider's record-update time and is never relabeled as a health check.
	FetchedAt time.Time            `json:"fetched_at"`
	Stale     bool                 `json:"stale"`
	Summary   ServiceStatusSummary `json:"summary"`
}

type ServiceStatusSource interface {
	Fetch(context.Context, ServiceStatusProvider) (ServiceStatusSummary, error)
}

type ServiceStatusReader interface {
	Latest(context.Context, ServiceStatusProvider) (ServiceStatusSnapshot, error)
}
