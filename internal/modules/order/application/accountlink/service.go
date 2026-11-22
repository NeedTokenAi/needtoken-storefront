package accountlink

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
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

type AutoLinkInput struct {
	MerchantCustomerID string
	Email              string
	EmailVerified      bool
	DisplayName        string
}

type AutoLinkResult struct {
	Linked         bool   `json:"linked"`
	Status         string `json:"status"`
	NativeUserID   int    `json:"native_user_id,omitempty"`
	AwakenedOrders int    `json:"awakened_orders,omitempty"`
}

type APISubscriptionSummary struct {
	ID                  int64  `json:"id"`
	PlanID              int    `json:"plan_id"`
	PlanTitle           string `json:"plan_title,omitempty"`
	Status              string `json:"status"`
	AmountTotal         int64  `json:"amount_total"`
	AmountUsed          int64  `json:"amount_used"`
	StartTime           int64  `json:"start_time"`
	EndTime             int64  `json:"end_time"`
	AllowWalletOverflow bool   `json:"allow_wallet_overflow"`
}

type APISummaryResult struct {
	Linked              bool                     `json:"linked"`
	NativeUserID        int                      `json:"native_user_id,omitempty"`
	Quota               int64                    `json:"quota,omitempty"`
	UsedQuota           int64                    `json:"used_quota,omitempty"`
	RequestCount        int64                    `json:"request_count,omitempty"`
	Group               string                   `json:"group,omitempty"`
	ActiveSubscriptions []APISubscriptionSummary `json:"active_subscriptions,omitempty"`
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
	AutoLink(ctx context.Context, input AutoLinkInput) (AutoLinkResult, error)
	APISummary(ctx context.Context, merchantCustomerID string) (APISummaryResult, error)
}

type ensureFlight struct {
	done chan struct{}
	res  AutoLinkResult
	err  error
}

type Service struct {
	bridge         Bridge
	identity       IdentityReader
	ensureMu       sync.Mutex
	ensureInflight map[uint]*ensureFlight
}

func New(bridge Bridge) *Service {
	return &Service{
		bridge:         bridge,
		ensureInflight: make(map[uint]*ensureFlight),
	}
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

// Ensure idempotently links a verified storefront member to a freshly provisioned
// console account. Concurrent calls for the same user are coalesced (singleflight)
// to prevent re-entrant race conditions, and pre-existing console accounts never
// leak remote identities (native_user_id is stripped when existing_requires_link).
func (s *Service) Ensure(ctx context.Context, userID uint) (AutoLinkResult, error) {
	ref, err := merchantCustomerID(userID)
	if err != nil {
		return AutoLinkResult{}, err
	}
	if s.bridge == nil || s.identity == nil {
		return AutoLinkResult{Linked: false, Status: "disabled"}, nil
	}

	s.ensureMu.Lock()
	if s.ensureInflight == nil {
		s.ensureInflight = make(map[uint]*ensureFlight)
	}
	if flight, running := s.ensureInflight[userID]; running {
		s.ensureMu.Unlock()
		select {
		case <-flight.done:
			return flight.res, flight.err
		case <-ctx.Done():
			return AutoLinkResult{}, ctx.Err()
		}
	}
	flight := &ensureFlight{done: make(chan struct{})}
	s.ensureInflight[userID] = flight
	s.ensureMu.Unlock()

	res, callErr := s.runEnsure(ctx, userID, ref)

	s.ensureMu.Lock()
	flight.res = res
	flight.err = callErr
	delete(s.ensureInflight, userID)
	close(flight.done)
	s.ensureMu.Unlock()

	return res, callErr
}

func (s *Service) runEnsure(ctx context.Context, userID uint, ref string) (AutoLinkResult, error) {
	member, err := s.identity.LookupMember(ctx, userID)
	if err != nil {
		return AutoLinkResult{}, ErrUnavailable
	}
	if !member.Active {
		return AutoLinkResult{}, ErrForbidden
	}

	email := strings.TrimSpace(member.Email)
	emailVerified := member.EmailVerified && email != ""
	if !emailVerified {
		// Never forward unverified or placeholder emails for auto-provisioning.
		email = ""
	}

	res, err := s.bridge.AutoLink(ctx, AutoLinkInput{
		MerchantCustomerID: ref,
		Email:              email,
		EmailVerified:      emailVerified,
		DisplayName:        strings.TrimSpace(member.DisplayName),
	})
	if err != nil {
		return AutoLinkResult{}, err
	}
	res.Status = strings.TrimSpace(res.Status)
	if !res.Linked || res.Status == "existing_requires_link" {
		res.Linked = false
		res.NativeUserID = 0
		res.AwakenedOrders = 0
	}
	return res, nil
}

// APISummary returns read-only API quota and active subscription summaries for
// a verified linked storefront member.
func (s *Service) APISummary(ctx context.Context, userID uint) (APISummaryResult, error) {
	ref, err := merchantCustomerID(userID)
	if err != nil {
		return APISummaryResult{}, err
	}
	if s.bridge == nil {
		return APISummaryResult{Linked: false}, nil
	}
	res, err := s.bridge.APISummary(ctx, ref)
	if err != nil {
		return APISummaryResult{}, err
	}
	if !res.Linked {
		return APISummaryResult{Linked: false}, nil
	}
	return res, nil
}
