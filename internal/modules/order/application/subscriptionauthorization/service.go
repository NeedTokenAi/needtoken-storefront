// Package subscriptionauthorization owns paid-order identity proofs and buyer
// consent. Opening an inspection session never permits a purchase by itself.
package subscriptionauthorization

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"regexp"
	"strings"
	"time"

	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
)

type Subject struct {
	OrderNo                   string
	UserID                    uint
	GuestEmail, GuestPassword string
	ResellerID                *uint
}

type OrderSnapshot struct {
	ID                                                    uint
	OrderNo, OrderStatus, TenantRef, BuyerRef, GuestEmail string
	Paid                                                  bool
	Leaf                                                  bool
	ConnectionID, UpstreamOrderID                         uint
	UpstreamOrderNo, SnapshotHash, PaymentEnvironment     string
	Contract                                              json.RawMessage
}

type Directory interface {
	Resolve(context.Context, Subject) (OrderSnapshot, error)
	Current(context.Context, uint) (OrderSnapshot, error)
}
type Mailer interface {
	Send(context.Context, string, string, string) error
}

type FulfillmentProgress struct {
	Stage                  string `json:"stage"`
	AuthorizationSubmitted bool   `json:"authorization_submitted"`
	NextAction             string `json:"next_action"`
	UpdatedAt              string `json:"updated_at,omitempty"`
	Terminal               bool   `json:"terminal"`
}

type Context struct {
	OrderID             uint                 `json:"order_id"`
	OrderNo             string               `json:"order_no"`
	SnapshotHash        string               `json:"snapshot_hash"`
	Platform            string               `json:"authorization_platform"`
	Route               string               `json:"route"`
	Plan                string               `json:"plan"`
	Period              string               `json:"period"`
	FulfillmentProgress *FulfillmentProgress `json:"fulfillment_progress,omitempty"`
}

// Assertion deliberately matches the existing Glue inspection-only contract.
// Consent is stored separately and is read over its own authenticated bridge.
type Assertion struct {
	OperationID        string    `json:"operation_id"`
	Source             string    `json:"source"`
	TenantRef          string    `json:"tenant_ref"`
	OrderRef           string    `json:"order_ref"`
	SnapshotHash       string    `json:"snapshot_hash"`
	GlueSnapshotHash   string    `json:"glue_snapshot_hash"`
	BuyerRef           string    `json:"buyer_ref"`
	ProofRef           string    `json:"proof_ref"`
	ExpiresAt          time.Time `json:"expires_at"`
	PaymentEnvironment string    `json:"payment_environment"`
	ExecutionScope     string    `json:"execution_scope"`
	Platform           string    `json:"platform"`
	SessionTokenHash   string    `json:"session_token_hash"`
}

type Signature struct {
	APIKey, Value, Path string
	Timestamp           int64
}
type Bridge interface {
	Context(context.Context, uint, uint) (Context, error)
	Open(context.Context, uint, Assertion) error
	CredentialURL(context.Context, uint) (string, error)
	Verify(context.Context, uint, Signature) bool
}

type Config struct {
	Enabled bool
	Key     []byte
	Clock   func() time.Time
}
type Service struct {
	cfg       Config
	store     ordercontract.SubscriptionAuthorizationStore
	directory Directory
	bridge    Bridge
	mail      Mailer
}

func New(cfg Config, store ordercontract.SubscriptionAuthorizationStore, directory Directory, bridge Bridge, mail Mailer) *Service {
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	cfg.Key = append([]byte(nil), cfg.Key...)
	return &Service{cfg: cfg, store: store, directory: directory, bridge: bridge, mail: mail}
}

type Terms struct {
	Version       int    `json:"version"`
	Platform      string `json:"platform"`
	Route         string `json:"route"`
	RouteVersion  int    `json:"route_version"`
	Plan          string `json:"plan"`
	Period        string `json:"period"`
	RenewalPolicy string `json:"renewal_policy"`
	BudgetMinor   string `json:"budget_minor"`
	Currency      string `json:"currency"`
}
type Eligibility struct {
	Eligible            bool                 `json:"eligible"`
	Reason              string               `json:"reason"`
	Platform            string               `json:"platform,omitempty"`
	Route               string               `json:"route,omitempty"`
	Plan                string               `json:"plan,omitempty"`
	Period              string               `json:"period,omitempty"`
	GuestProofRequired  bool                 `json:"guest_proof_required"`
	FulfillmentProgress *FulfillmentProgress `json:"fulfillment_progress,omitempty"`
}
type SessionRequest struct {
	OperationID          string `json:"operation_id"`
	Code                 string `json:"code,omitempty"`
	ConsentPurchase      bool   `json:"consent_purchase"`
	ConsentCancelRenewal bool   `json:"consent_cancel_renewal"`
}
type Session struct {
	AuthorizationID string    `json:"authorization_id"`
	SessionToken    string    `json:"session_token"`
	CredentialURL   string    `json:"credential_url"`
	ExpiresAt       time.Time `json:"expires_at"`
	ExecutionScope  string    `json:"execution_scope"`
	ConsentRef      string    `json:"consent_ref"`
}
type Challenge struct {
	OperationID string    `json:"operation_id"`
	ExpiresAt   time.Time `json:"expires_at"`
}
type Consent struct {
	AuthorizationID    string    `json:"authorization_id"`
	Source             string    `json:"source"`
	TenantRef          string    `json:"tenant_ref"`
	OrderRef           string    `json:"order_ref"`
	BuyerRef           string    `json:"buyer_ref"`
	SnapshotHash       string    `json:"snapshot_hash"`
	GlueSnapshotHash   string    `json:"glue_snapshot_hash"`
	Platform           string    `json:"platform"`
	Route              string    `json:"route"`
	Plan               string    `json:"plan"`
	Period             string    `json:"period"`
	Purchase           bool      `json:"consent_purchase"`
	CancelRenewal      bool      `json:"consent_cancel_renewal"`
	PaymentEnvironment string    `json:"payment_environment"`
	ApprovedAt         time.Time `json:"approved_at"`
	ExpiresAt          time.Time `json:"expires_at"`
}

var opaque = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{7,100}$`)
var hash64 = regexp.MustCompile(`^[0-9a-f]{64}$`)
var positiveMinor = regexp.MustCompile(`^[1-9][0-9]{0,11}$`)
var currencyCode = regexp.MustCompile(`^[A-Z]{3}$`)

func isAllowedSubscriptionPlan(plan string) bool {
	switch plan {
	case "plus-monthly", "pro-monthly", "pro-5x-monthly", "pro-20x-monthly":
		return true
	default:
		return false
	}
}

func (s *Service) load(ctx context.Context, subject Subject) (OrderSnapshot, Context, Terms, error) {
	var terms Terms
	if s.directory == nil {
		return OrderSnapshot{}, Context{}, terms, ordercontract.ErrAuthorizationUnavailable
	}
	local, err := s.directory.Resolve(ctx, subject)
	if err != nil {
		return local, Context{}, terms, err
	}
	if !s.cfg.Enabled || len(s.cfg.Key) < 32 || s.store == nil || s.bridge == nil {
		return local, Context{}, terms, ordercontract.ErrAuthorizationUnavailable
	}
	if !local.Paid || !local.Leaf || local.ConnectionID == 0 || local.UpstreamOrderID == 0 || !hash64.MatchString(local.SnapshotHash) {
		return local, Context{}, terms, ordercontract.ErrAuthorizationConflict
	}
	var contract struct {
		Kind         string `json:"kind"`
		Subscription Terms  `json:"subscription_contract"`
	}
	if json.Unmarshal(local.Contract, &contract) != nil {
		return local, Context{}, terms, ordercontract.ErrAuthorizationConflict
	}
	terms = contract.Subscription
	if contract.Kind != "recharge" || terms.Version != 1 || terms.Platform != "openai" || terms.Route != "web_card" || terms.RouteVersion != 1 || !isAllowedSubscriptionPlan(terms.Plan) || terms.Period != "month" || terms.RenewalPolicy != "single_cycle" || !positiveMinor.MatchString(terms.BudgetMinor) || !currencyCode.MatchString(terms.Currency) {
		return local, Context{}, terms, ordercontract.ErrAuthorizationConflict
	}
	remote, err := s.bridge.Context(ctx, local.ConnectionID, local.UpstreamOrderID)
	if err != nil {
		return local, remote, terms, ordercontract.ErrAuthorizationUnavailable
	}
	if remote.OrderID != local.UpstreamOrderID || remote.OrderNo != local.UpstreamOrderNo || !hash64.MatchString(remote.SnapshotHash) || remote.Platform != terms.Platform || remote.Route != terms.Route || remote.Plan != terms.Plan || remote.Period != terms.Period {
		return local, remote, terms, ordercontract.ErrAuthorizationConflict
	}
	return local, remote, terms, nil
}

func isResubmittableStage(progress *FulfillmentProgress) bool {
	if progress == nil {
		return true
	}
	switch progress.Stage {
	case "", "awaiting_authorization", "credential_expired":
		return !progress.AuthorizationSubmitted
	default:
		return false
	}
}

func fallbackProgressFromOrderStatus(status string) *FulfillmentProgress {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "delivered", "completed":
		return &FulfillmentProgress{Stage: "completed", AuthorizationSubmitted: true, NextAction: "none", Terminal: true}
	case "canceled":
		return &FulfillmentProgress{Stage: "canceled", AuthorizationSubmitted: false, NextAction: "none", Terminal: true}
	case "refunded", "partially_refunded":
		return &FulfillmentProgress{Stage: "refunded", AuthorizationSubmitted: false, NextAction: "none", Terminal: true}
	default:
		return nil
	}
}

func (s *Service) Eligibility(ctx context.Context, subject Subject) (Eligibility, error) {
	local, remote, terms, err := s.load(ctx, subject)
	if err == ordercontract.ErrAuthorizationNotFound {
		return Eligibility{}, err
	}
	v := Eligibility{Reason: "unavailable", GuestProofRequired: subject.UserID == 0}
	if err != nil {
		if s.directory != nil {
			if snap, rErr := s.directory.Resolve(ctx, subject); rErr == nil && snap.Leaf && snap.OrderNo != "" {
				var progress *FulfillmentProgress
				if s.cfg.Enabled && s.bridge != nil && snap.ConnectionID > 0 && snap.UpstreamOrderID > 0 {
					if rem, cErr := s.bridge.Context(ctx, snap.ConnectionID, snap.UpstreamOrderID); cErr == nil && rem.FulfillmentProgress != nil {
						progress = rem.FulfillmentProgress
					}
				}
				if progress == nil {
					progress = fallbackProgressFromOrderStatus(snap.OrderStatus)
				}
				if progress != nil {
					v.FulfillmentProgress = progress
					v.Reason = progress.Stage
				}
			}
		}
		return v, nil
	}
	if local.OrderNo == "" || remote.OrderNo == "" {
		return v, nil
	}
	progress := remote.FulfillmentProgress
	eligible := isResubmittableStage(progress)
	reason := "ready"
	if !eligible && progress != nil && progress.Stage != "" {
		reason = progress.Stage
	}
	return Eligibility{
		Eligible:            eligible,
		Reason:              reason,
		Platform:            terms.Platform,
		Route:               terms.Route,
		Plan:                terms.Plan,
		Period:              terms.Period,
		GuestProofRequired:  subject.UserID == 0,
		FulfillmentProgress: progress,
	}, nil
}

func contextDigest(local OrderSnapshot, remote Context, terms Terms) string {
	raw, _ := json.Marshal(struct {
		Local  OrderSnapshot
		Remote Context
		Terms  Terms
	}{local, remote, terms})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func (s *Service) codeHash(id, code string) string { return s.mac("email-proof", id, code) }
func (s *Service) mac(domain, id, value string) string {
	m := hmac.New(sha256.New, s.cfg.Key)
	fmt.Fprintf(m, "needtoken:subscription-authorization:%s\x00%s\x00%s", domain, id, value)
	return hex.EncodeToString(m.Sum(nil))
}
func (s *Service) token(row *orderdomain.SubscriptionAuthorization) string {
	raw, _ := hex.DecodeString(s.mac("session", row.ID, row.ContextHash+":"+row.ExpiresAt.UTC().Format(time.RFC3339Nano)))
	return base64.RawURLEncoding.EncodeToString(raw)
}
func digestToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func (s *Service) record(id string, local OrderSnapshot, remote Context, terms Terms) *orderdomain.SubscriptionAuthorization {
	now := s.cfg.Clock().UTC()
	return &orderdomain.SubscriptionAuthorization{ID: id, OrderID: local.ID, OrderNo: local.OrderNo, TenantRef: local.TenantRef, BuyerRef: local.BuyerRef, ConnectionID: local.ConnectionID, UpstreamOrderID: local.UpstreamOrderID, SnapshotHash: local.SnapshotHash, GlueSnapshotHash: remote.SnapshotHash, ContextHash: contextDigest(local, remote, terms), Platform: terms.Platform, Route: terms.Route, Plan: terms.Plan, Period: terms.Period, PaymentEnvironment: local.PaymentEnvironment, Status: "awaiting_email_proof", CreatedAt: now, ExpiresAt: now.Add(30 * time.Minute)}
}

func (s *Service) Challenge(ctx context.Context, subject Subject, id, locale string) (Challenge, error) {
	if subject.UserID != 0 || !opaque.MatchString(id) {
		return Challenge{}, ordercontract.ErrAuthorizationInvalid
	}
	local, remote, terms, err := s.load(ctx, subject)
	if err != nil {
		return Challenge{}, err
	}
	if local.GuestEmail == "" || s.mail == nil {
		return Challenge{}, ordercontract.ErrAuthorizationUnavailable
	}
	number, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return Challenge{}, ordercontract.ErrAuthorizationUnavailable
	}
	code := fmt.Sprintf("%06d", number)
	row := s.record(id, local, remote, terms)
	row.CodeHash = s.codeHash(id, code)
	row.CodeExpiresAt = row.CreatedAt.Add(10 * time.Minute)
	row, created, err := s.store.Create(ctx, row, true)
	if err != nil {
		return Challenge{}, err
	}
	if !created {
		if row.Status == "send_failed" || !s.cfg.Clock().Before(row.CodeExpiresAt) {
			return Challenge{}, ordercontract.ErrAuthorizationInvalid
		}
		return Challenge{OperationID: row.ID, ExpiresAt: row.CodeExpiresAt}, nil
	}
	if err = s.mail.Send(ctx, local.GuestEmail, code, locale); err != nil {
		_, _ = s.store.Update(ctx, row.ID, func(r *orderdomain.SubscriptionAuthorization) error {
			r.Status = "send_failed"
			r.CodeHash = ""
			return nil
		})
		return Challenge{}, ordercontract.ErrAuthorizationUnavailable
	}
	return Challenge{OperationID: row.ID, ExpiresAt: row.CodeExpiresAt}, nil
}

func (s *Service) Open(ctx context.Context, subject Subject, input SessionRequest) (Session, error) {
	if !opaque.MatchString(input.OperationID) || !input.ConsentPurchase || !input.ConsentCancelRenewal {
		return Session{}, ordercontract.ErrAuthorizationInvalid
	}
	local, remote, terms, err := s.load(ctx, subject)
	if err != nil {
		return Session{}, err
	}
	wanted := s.record(input.OperationID, local, remote, terms)
	var row *orderdomain.SubscriptionAuthorization
	if subject.UserID != 0 {
		wanted.VerifiedAt = &wanted.CreatedAt
		row, _, err = s.store.Create(ctx, wanted, false)
	} else {
		row, err = s.store.Get(ctx, input.OperationID)
	}
	if err != nil {
		return Session{}, err
	}
	if row.ContextHash != wanted.ContextHash || row.BuyerRef != wanted.BuyerRef || row.TenantRef != wanted.TenantRef || !s.cfg.Clock().Before(row.ExpiresAt) {
		return Session{}, ordercontract.ErrAuthorizationConflict
	}
	var proofErr error
	row, err = s.store.Update(ctx, row.ID, func(r *orderdomain.SubscriptionAuthorization) error {
		if !s.cfg.Clock().Before(r.ExpiresAt) || r.Status == "send_failed" {
			return ordercontract.ErrAuthorizationConflict
		}
		if r.VerifiedAt == nil {
			if !s.cfg.Clock().Before(r.CodeExpiresAt) || r.Attempts >= 5 {
				return ordercontract.ErrAuthorizationInvalid
			}
			r.Attempts++
			if len(input.Code) != 6 || subtle.ConstantTimeCompare([]byte(r.CodeHash), []byte(s.codeHash(r.ID, input.Code))) != 1 {
				proofErr = ordercontract.ErrAuthorizationInvalid
				return nil
			}
			now := s.cfg.Clock().UTC()
			r.VerifiedAt = &now
			r.CodeHash = ""
		}
		r.ConsentPurchase = true
		r.ConsentCancel = true
		r.SessionTokenHash = digestToken(s.token(r))
		if r.Status != "opened" {
			r.Status = "opening"
		}
		return nil
	})
	if err != nil {
		return Session{}, err
	}
	if proofErr != nil {
		return Session{}, proofErr
	}
	credentialURL, err := s.bridge.CredentialURL(ctx, row.ConnectionID)
	if err != nil || !validCredentialURL(credentialURL) {
		return Session{}, ordercontract.ErrAuthorizationUnavailable
	}
	if row.Status != "opened" {
		a := Assertion{OperationID: row.ID, Source: "needtoken-storefront", TenantRef: row.TenantRef, OrderRef: row.OrderNo, SnapshotHash: row.SnapshotHash, GlueSnapshotHash: row.GlueSnapshotHash, BuyerRef: row.BuyerRef, ProofRef: "proof_" + row.ID, ExpiresAt: row.ExpiresAt, PaymentEnvironment: row.PaymentEnvironment, ExecutionScope: "inspection_only", Platform: row.Platform, SessionTokenHash: row.SessionTokenHash}
		if err = s.bridge.Open(ctx, row.ConnectionID, a); err != nil {
			return Session{}, ordercontract.ErrAuthorizationUnavailable
		}
		row, err = s.store.Update(ctx, row.ID, func(r *orderdomain.SubscriptionAuthorization) error { r.Status = "opened"; return nil })
		if err != nil {
			return Session{}, err
		}
	}
	return Session{AuthorizationID: row.ID, SessionToken: s.token(row), CredentialURL: credentialURL, ExpiresAt: row.ExpiresAt, ExecutionScope: "inspection_only", ConsentRef: row.ID}, nil
}

func validCredentialURL(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && u.Scheme == "https" && u.Host != "" && u.User == nil && u.Path == "/api/authorization/v1/credential" && u.RawQuery == "" && u.Fragment == ""
}

// Consent returns only a currently valid, opened consent after verifying the
// signature against the procurement connection frozen into this record.
func (s *Service) Consent(ctx context.Context, id string, signature Signature) (Consent, error) {
	if !s.cfg.Enabled || s.store == nil || s.bridge == nil || !opaque.MatchString(id) {
		return Consent{}, ordercontract.ErrAuthorizationUnavailable
	}
	row, err := s.store.Get(ctx, id)
	if err != nil || !s.bridge.Verify(ctx, row.ConnectionID, signature) {
		return Consent{}, ordercontract.ErrAuthorizationNotFound
	}
	if row.Status != "opened" || row.VerifiedAt == nil || !row.ConsentPurchase || !row.ConsentCancel || !s.cfg.Clock().Before(row.ExpiresAt) {
		return Consent{}, ordercontract.ErrAuthorizationConflict
	}
	current, err := s.directory.Current(ctx, row.OrderID)
	if err != nil || !current.Paid || !current.Leaf || current.OrderNo != row.OrderNo || current.BuyerRef != row.BuyerRef || current.TenantRef != row.TenantRef || current.SnapshotHash != row.SnapshotHash || current.ConnectionID != row.ConnectionID || current.UpstreamOrderID != row.UpstreamOrderID || current.PaymentEnvironment != row.PaymentEnvironment {
		return Consent{}, ordercontract.ErrAuthorizationConflict
	}
	remote, err := s.bridge.Context(ctx, row.ConnectionID, row.UpstreamOrderID)
	var contract struct {
		Subscription Terms `json:"subscription_contract"`
	}
	if err != nil || json.Unmarshal(current.Contract, &contract) != nil || contextDigest(current, remote, contract.Subscription) != row.ContextHash {
		return Consent{}, ordercontract.ErrAuthorizationConflict
	}
	return Consent{AuthorizationID: row.ID, Source: "needtoken-storefront", TenantRef: row.TenantRef, OrderRef: row.OrderNo, BuyerRef: row.BuyerRef, SnapshotHash: row.SnapshotHash, GlueSnapshotHash: row.GlueSnapshotHash, Platform: row.Platform, Route: row.Route, Plan: row.Plan, Period: row.Period, Purchase: row.ConsentPurchase, CancelRenewal: row.ConsentCancel, PaymentEnvironment: row.PaymentEnvironment, ApprovedAt: *row.VerifiedAt, ExpiresAt: row.ExpiresAt}, nil
}
