package diagnosticscontract

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidInput    = errors.New("account diagnostic: invalid input")
	ErrUnauthorized    = errors.New("account diagnostic: credential rejected")
	ErrChallenge       = errors.New("account diagnostic: upstream interaction required")
	ErrUnavailable     = errors.New("account diagnostic: upstream unavailable")
	ErrInvalidResponse = errors.New("account diagnostic: invalid upstream response")
	ErrAccountNotFound = errors.New("account diagnostic: selected account not found")
)

// Input contains the user's short-lived credential only for this request. It is
// deliberately not a database model and must never be logged or persisted.
type Input struct {
	AccessToken string `json:"access_token"`
	AccountID   string `json:"account_id,omitempty"`
}

type Account struct {
	ID        string     `json:"id"`
	Plan      string     `json:"plan"`
	IsDefault bool       `json:"is_default"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type UsageWindow struct {
	UsedPercent   float64    `json:"used_percent"`
	WindowSeconds int64      `json:"window_seconds"`
	ResetAt       *time.Time `json:"reset_at,omitempty"`
}

type Usage struct {
	Primary   *UsageWindow `json:"primary,omitempty"`
	Secondary *UsageWindow `json:"secondary,omitempty"`
}

type Result struct {
	CheckedAt         time.Time `json:"checked_at"`
	Accounts          []Account `json:"accounts"`
	SelectedAccountID string    `json:"selected_account_id,omitempty"`
	UsageStatus       string    `json:"usage_status"`
	Usage             *Usage    `json:"usage,omitempty"`
}

type Inspector interface {
	Inspect(context.Context, Input) (Result, error)
}
