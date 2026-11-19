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

type ServiceStatusIncidentComponent struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status,omitempty"`
}

type ServiceStatusAffectedComponent struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	OldStatus string `json:"old_status,omitempty"`
	NewStatus string `json:"new_status,omitempty"`
}

type ServiceStatusIncidentUpdate struct {
	ID                 string                           `json:"id"`
	Status             string                           `json:"status"`
	Body               string                           `json:"body"`
	DisplayAt          *time.Time                       `json:"display_at,omitempty"`
	CreatedAt          *time.Time                       `json:"created_at,omitempty"`
	UpdatedAt          *time.Time                       `json:"updated_at,omitempty"`
	AffectedComponents []ServiceStatusAffectedComponent `json:"affected_components"`
}

// Associations come only from the official component IDs. An empty list does
// not imply that a provider-wide incident affects a particular product.
type ServiceStatusIncident struct {
	ID              string                           `json:"id"`
	Name            string                           `json:"name"`
	Status          string                           `json:"status"`
	Impact          string                           `json:"impact"`
	URL             string                           `json:"url"`
	CreatedAt       *time.Time                       `json:"created_at,omitempty"`
	StartedAt       *time.Time                       `json:"started_at,omitempty"`
	UpdatedAt       *time.Time                       `json:"updated_at,omitempty"`
	ResolvedAt      *time.Time                       `json:"resolved_at,omitempty"`
	Components      []ServiceStatusIncidentComponent `json:"components"`
	IncidentUpdates []ServiceStatusIncidentUpdate    `json:"incident_updates"`
}

// A recent incident feed is partial event history, not an uptime measurement or
// a guarantee that days without returned incidents were healthy.
type ServiceStatusHistory struct {
	State     string                  `json:"state"`
	Coverage  string                  `json:"coverage"`
	SourceURL string                  `json:"source_url"`
	Incidents []ServiceStatusIncident `json:"incidents"`
}

type ServiceStatusSummary struct {
	Page       ServiceStatusPage        `json:"page"`
	Status     ServiceStatusState       `json:"status"`
	Components []ServiceStatusComponent `json:"components"`
	Incidents  []ServiceStatusIncident  `json:"incidents"`
}

type ServiceStatusSeriesEvent struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

type ServiceStatusSeriesDay struct {
	Date   string                     `json:"date"`
	Status string                     `json:"status"`
	Events []ServiceStatusSeriesEvent `json:"events,omitempty"`
}

// ServiceStatusSeries carries a 90-day daily availability timeline for a
// specific official product group (e.g. ChatGPT/Codex or Claude/Claude Code/Claude API).
type ServiceStatusSeries struct {
	Key            string                   `json:"key"`
	Label          string                   `json:"label"`
	ComponentCount int                      `json:"component_count"`
	Status         string                   `json:"status"`
	UptimePercent  *float64                 `json:"uptime_percent,omitempty"`
	Days           []ServiceStatusSeriesDay `json:"days"`
}

type ServiceStatusSnapshot struct {
	Provider  ServiceStatusProvider `json:"provider"`
	SourceURL string                `json:"source_url"`
	// FetchedAt is this site's successful retrieval time. Page.UpdatedAt remains
	// the provider's record-update time and is never relabeled as a health check.
	FetchedAt time.Time             `json:"fetched_at"`
	Stale     bool                  `json:"stale"`
	Summary   ServiceStatusSummary  `json:"summary"`
	History   ServiceStatusHistory  `json:"history"`
	Series    []ServiceStatusSeries `json:"series,omitempty"`
}

type ServiceStatusSource interface {
	Fetch(context.Context, ServiceStatusProvider) (ServiceStatusSummary, error)
}

// History is optional so an unavailable recent feed cannot invalidate a
// successfully retrieved current-status summary.
type ServiceStatusHistorySource interface {
	FetchHistory(context.Context, ServiceStatusProvider) (ServiceStatusHistory, error)
}

// Series is optional so an unavailable 90-day uptime feed cannot invalidate a
// successfully retrieved current-status summary.
type ServiceStatusSeriesSource interface {
	FetchSeries(context.Context, ServiceStatusProvider, ServiceStatusSummary) ([]ServiceStatusSeries, error)
}

type ServiceStatusReader interface {
	Latest(context.Context, ServiceStatusProvider) (ServiceStatusSnapshot, error)
}
