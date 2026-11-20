package assetclaim

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
)

var (
	ErrUnavailable = errors.New("asset claim unavailable")
	ErrNotFound    = errors.New("asset claim order not found")
	ErrInvalid     = errors.New("asset claim invalid")
	ErrForbidden   = errors.New("asset claim forbidden")
	ErrExhausted   = errors.New("asset claim reveal limit reached")
	ErrConflict    = errors.New("asset claim conflict")
)

type Subject struct {
	OrderNo       string
	ResellerID    *uint
	UserID        uint
	GuestEmail    string
	GuestPassword string
}

type OrderSnapshot struct {
	OrderID         uint
	OrderNo         string
	UserID          uint
	Leaf            bool
	Delivered       bool
	ConnectionID    uint
	UpstreamOrderID uint
}

type Verification struct {
	Source          string     `json:"source"`
	ObservedAt      time.Time  `json:"observed_at"`
	RemoteAccountID string     `json:"remote_account_id,omitempty"`
	PlanType        string     `json:"plan_type,omitempty"`
	SubscriptionEnd *time.Time `json:"subscription_end,omitempty"`
	EvidenceRef     string     `json:"evidence_ref,omitempty"`
	EvidenceSHA256  string     `json:"evidence_sha256"`
}

type Status struct {
	Eligible         bool          `json:"eligible"`
	AssetID          string        `json:"asset_id,omitempty"`
	SKUCode          string        `json:"sku_code,omitempty"`
	Platform         string        `json:"platform,omitempty"`
	SaleMode         string        `json:"sale_mode,omitempty"`
	MaskedIdentifier string        `json:"masked_identifier,omitempty"`
	DeliveredAt      *time.Time    `json:"delivered_at,omitempty"`
	ExpiresAt        *time.Time    `json:"expires_at,omitempty"`
	RentalExpiresAt  *time.Time    `json:"rental_expires_at,omitempty"`
	Verification     *Verification `json:"verification,omitempty"`
	MaxReveals       int           `json:"max_reveals"`
	RevealCount      int           `json:"reveal_count"`
	RemainingReveals int           `json:"remaining_reveals"`
	FirstRevealedAt  *time.Time    `json:"first_revealed_at,omitempty"`
	LastRevealedAt   *time.Time    `json:"last_revealed_at,omitempty"`
}

type RevealResult struct {
	Status
	AccountIdentifier string            `json:"account_identifier"`
	SecretPayload     string            `json:"secret_payload"`
	DeliveryNotes     string            `json:"delivery_notes,omitempty"`
	Extra             map[string]string `json:"extra,omitempty"`
}

type Directory interface {
	Resolve(ctx context.Context, subject Subject) (OrderSnapshot, error)
}

type Bridge interface {
	Status(ctx context.Context, connectionID, upstreamOrderID uint, buyerRef string) (Status, error)
	Reveal(ctx context.Context, connectionID, upstreamOrderID uint, buyerRef, actor string) (RevealResult, error)
}

type Service struct {
	directory Directory
	bridge    Bridge
}

func New(directory Directory, bridge Bridge) *Service {
	return &Service{directory: directory, bridge: bridge}
}

func subjectRefs(snap OrderSnapshot, subject Subject) (buyerRef, actor string) {
	if snap.UserID > 0 {
		return "needtoken-storefront:user:" + strconv.FormatUint(uint64(snap.UserID), 10), "member_" + strconv.FormatUint(uint64(snap.UserID), 10)
	}
	if subject.UserID > 0 {
		return "needtoken-storefront:user:" + strconv.FormatUint(uint64(subject.UserID), 10), "member_" + strconv.FormatUint(uint64(subject.UserID), 10)
	}
	return "", "guest_order_" + strconv.FormatUint(uint64(snap.OrderID), 10)
}

func (s *Service) Status(ctx context.Context, subject Subject) (Status, error) {
	if strings.TrimSpace(subject.OrderNo) == "" {
		return Status{}, ErrInvalid
	}
	if s.directory == nil || s.bridge == nil {
		return Status{Eligible: false}, nil
	}
	snap, err := s.directory.Resolve(ctx, subject)
	if err != nil {
		return Status{}, err
	}
	if !snap.Leaf || !snap.Delivered || snap.ConnectionID == 0 || snap.UpstreamOrderID == 0 {
		return Status{Eligible: false}, nil
	}
	buyerRef, _ := subjectRefs(snap, subject)
	return s.bridge.Status(ctx, snap.ConnectionID, snap.UpstreamOrderID, buyerRef)
}

func (s *Service) Reveal(ctx context.Context, subject Subject) (RevealResult, error) {
	if strings.TrimSpace(subject.OrderNo) == "" {
		return RevealResult{}, ErrInvalid
	}
	if s.directory == nil || s.bridge == nil {
		return RevealResult{}, ErrUnavailable
	}
	snap, err := s.directory.Resolve(ctx, subject)
	if err != nil {
		return RevealResult{}, err
	}
	if !snap.Leaf || !snap.Delivered || snap.ConnectionID == 0 || snap.UpstreamOrderID == 0 {
		return RevealResult{}, ErrConflict
	}
	buyerRef, actor := subjectRefs(snap, subject)
	return s.bridge.Reveal(ctx, snap.ConnectionID, snap.UpstreamOrderID, buyerRef, actor)
}
