package accountlink

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var (
	ErrUnavailable = errors.New("api account link unavailable")
	ErrInvalid     = errors.New("api account link invalid")
	ErrExpired     = errors.New("api account link expired")
	ErrConflict    = errors.New("api account link conflict")
	ErrForbidden   = errors.New("api account link forbidden")
)

var safeNextPath = regexp.MustCompile(`^/(|me|plans|api-services|models|checkout|profile|orders|orders/[A-Za-z0-9_-]{1,64})$`)

var allowedConsoleTargetPaths = map[string]struct{}{
	"/dashboard":     {},
	"/keys":          {},
	"/subscriptions": {},
	"/wallet":        {},
	"/usage-logs":    {},
	"/playground":    {},
}

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

type ConsoleSSOInput struct {
	MerchantCustomerID string
	Email              string
	EmailVerified      bool
	DisplayName        string
	TargetPath         string
	NextPath           string
}

type ConsoleSSOResult struct {
	Mode           string `json:"mode"`
	RedirectURL    string `json:"redirect_url"`
	Linked         bool   `json:"linked"`
	NativeUserID   int    `json:"native_user_id,omitempty"`
	AutoCreated    bool   `json:"auto_created"`
	AwakenedOrders int    `json:"awakened_orders"`
}

type MemberIdentity struct {
	UserID        uint
	Email         string
	EmailVerified bool
	DisplayName   string
	Active        bool
}

type IdentityReader interface {
	LookupMember(ctx context.Context, userID uint) (MemberIdentity, error)
}

type Bridge interface {
	Status(ctx context.Context, merchantCustomerID string) (Status, error)
	Start(ctx context.Context, merchantCustomerID, nextPath string) (Challenge, error)
	Complete(ctx context.Context, merchantCustomerID, challengeID, approval string) (CompleteResult, error)
	ConsoleSSO(ctx context.Context, input ConsoleSSOInput) (ConsoleSSOResult, error)
}

type Service struct {
	bridge   Bridge
	identity IdentityReader
}

func New(bridge Bridge) *Service {
	return &Service{bridge: bridge}
}

func (s *Service) WithIdentityReader(reader IdentityReader) *Service {
	s.identity = reader
	return s
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

func normalizeConsoleSSONextPath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "/me", nil
	}
	if !safeNextPath.MatchString(trimmed) {
		return "", ErrInvalid
	}
	return trimmed, nil
}

func normalizeConsoleTargetPath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "/dashboard", nil
	}
	if _, ok := allowedConsoleTargetPaths[trimmed]; !ok {
		return "", ErrInvalid
	}
	return trimmed, nil
}

func validAuthorizeURL(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return false
	}
	for _, r := range trimmed {
		if unicode.IsControl(r) {
			return false
		}
	}
	u, err := url.Parse(trimmed)
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

func (s *Service) ConsoleSSO(ctx context.Context, userID uint, targetPath, nextPath string) (ConsoleSSOResult, error) {
	ref, err := merchantCustomerID(userID)
	if err != nil {
		return ConsoleSSOResult{}, err
	}
	safeTarget, err := normalizeConsoleTargetPath(targetPath)
	if err != nil {
		return ConsoleSSOResult{}, err
	}
	safeNext, err := normalizeConsoleSSONextPath(nextPath)
	if err != nil {
		return ConsoleSSOResult{}, err
	}
	if s.bridge == nil || s.identity == nil {
		return ConsoleSSOResult{}, ErrUnavailable
	}
	member, err := s.identity.LookupMember(ctx, userID)
	if err != nil {
		return ConsoleSSOResult{}, ErrUnavailable
	}
	email := strings.TrimSpace(member.Email)
	if !member.Active || email == "" {
		return ConsoleSSOResult{}, ErrForbidden
	}
	res, err := s.bridge.ConsoleSSO(ctx, ConsoleSSOInput{
		MerchantCustomerID: ref,
		Email:              email,
		EmailVerified:      member.EmailVerified,
		DisplayName:        strings.TrimSpace(member.DisplayName),
		TargetPath:         safeTarget,
		NextPath:           safeNext,
	})
	if err != nil {
		return ConsoleSSOResult{}, err
	}
	if !validAuthorizeURL(res.RedirectURL) {
		return ConsoleSSOResult{}, ErrUnavailable
	}
	return res, nil
}
