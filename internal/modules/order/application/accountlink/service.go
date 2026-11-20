package accountlink

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	ErrUnavailable = errors.New("api account link unavailable")
	ErrInvalid     = errors.New("api account link invalid")
	ErrExpired     = errors.New("api account link expired")
	ErrConflict    = errors.New("api account link conflict")
	ErrForbidden   = errors.New("api account link forbidden")
)

var safeNextPath = regexp.MustCompile(`^/(|plans|checkout|profile|orders|orders/[A-Za-z0-9_-]{1,64})$`)

type Status struct {
	Enabled       bool       `json:"enabled"`
	Linked        bool       `json:"linked"`
	Status        string     `json:"status"`
	NativeUserID  int        `json:"native_user_id,omitempty"`
	PendingOrders int        `json:"pending_orders"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

type Challenge struct {
	ChallengeID  string    `json:"challenge_id"`
	AuthorizeURL string    `json:"authorize_url"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type CompleteResult struct {
	Linked         bool   `json:"linked"`
	Status         string `json:"status"`
	NativeUserID   int    `json:"native_user_id,omitempty"`
	NextPath       string `json:"next_path,omitempty"`
	AwakenedOrders int    `json:"awakened_orders"`
}

type Bridge interface {
	Status(ctx context.Context, merchantCustomerID string) (Status, error)
	Start(ctx context.Context, merchantCustomerID, nextPath string) (Challenge, error)
	Complete(ctx context.Context, merchantCustomerID, challengeID, approval string) (CompleteResult, error)
}

type Service struct {
	bridge Bridge
}

func New(bridge Bridge) *Service {
	return &Service{bridge: bridge}
}

func merchantCustomerID(userID uint) (string, error) {
	if userID == 0 {
		return "", ErrInvalid
	}
	return "needtoken-storefront:user:" + strconv.FormatUint(uint64(userID), 10), nil
}

func normalizeNextPath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "/plans", nil
	}
	if !safeNextPath.MatchString(trimmed) {
		return "", ErrInvalid
	}
	return trimmed, nil
}

func validAuthorizeURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" && u.User == nil
}

func (s *Service) Status(ctx context.Context, userID uint) (Status, error) {
	ref, err := merchantCustomerID(userID)
	if err != nil {
		return Status{}, err
	}
	if s.bridge == nil {
		return Status{Enabled: false, Linked: false, Status: "disabled"}, nil
	}
	return s.bridge.Status(ctx, ref)
}

func (s *Service) Start(ctx context.Context, userID uint, nextPath string) (Challenge, error) {
	ref, err := merchantCustomerID(userID)
	if err != nil {
		return Challenge{}, err
	}
	safeNext, err := normalizeNextPath(nextPath)
	if err != nil {
		return Challenge{}, err
	}
	if s.bridge == nil {
		return Challenge{}, ErrUnavailable
	}
	ch, err := s.bridge.Start(ctx, ref, safeNext)
	if err != nil {
		return Challenge{}, err
	}
	if strings.TrimSpace(ch.ChallengeID) == "" || !validAuthorizeURL(ch.AuthorizeURL) {
		return Challenge{}, ErrUnavailable
	}
	return ch, nil
}

func (s *Service) Complete(ctx context.Context, userID uint, challengeID, approval string) (CompleteResult, error) {
	ref, err := merchantCustomerID(userID)
	if err != nil {
		return CompleteResult{}, err
	}
	cid := strings.TrimSpace(challengeID)
	tok := strings.TrimSpace(approval)
	if cid == "" || len(cid) > 128 || tok == "" || len(tok) > 2048 {
		return CompleteResult{}, ErrInvalid
	}
	if s.bridge == nil {
		return CompleteResult{}, ErrUnavailable
	}
	res, err := s.bridge.Complete(ctx, ref, cid, tok)
	if err != nil {
		return CompleteResult{}, err
	}
	if res.NextPath != "" && !safeNextPath.MatchString(res.NextPath) {
		res.NextPath = "/orders"
	}
	return res, nil
}
