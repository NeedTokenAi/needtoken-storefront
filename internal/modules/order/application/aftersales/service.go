package aftersales

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrUnavailable = errors.New("after-sales service unavailable")
	ErrInvalid     = errors.New("invalid after-sales request")
	ErrNotFound    = errors.New("order or after-sales ticket not found")
	ErrConflict    = errors.New("after-sales ticket is closed")
)

const maxSummaryRunes = 2000

var allowedCategories = map[string]struct{}{
	"fulfillment_failed": {},
	"amount_mismatch":    {},
	"customer_inquiry":   {},
	"refund_dispute":     {},
	"other":              {},
}

type Subject struct {
	OrderNo       string
	UserID        uint
	GuestEmail    string
	GuestPassword string
	ResellerID    *uint
}

type OrderSnapshot struct {
	OrderID       uint
	OrderNo       string
	UserID        uint
	CustomerEmail string
	ConnectionID  uint
}

type Supplement struct {
	Author    string `json:"author"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

type Ticket struct {
	ID             string       `json:"id"`
	OrderNo        string       `json:"order_no"`
	Category       string       `json:"category"`
	Priority       string       `json:"priority"`
	Status         string       `json:"status"`
	Summary        string       `json:"summary"`
	ResolutionNote string       `json:"resolution_note,omitempty"`
	Supplements    []Supplement `json:"supplements,omitempty"`
	CreatedAt      string       `json:"created_at"`
	UpdatedAt      string       `json:"updated_at"`
	ResolvedAt     string       `json:"resolved_at,omitempty"`
}

type ListResult struct {
	OrderNo string   `json:"order_no"`
	Items   []Ticket `json:"items"`
}

type CreateInput struct {
	Category       string
	Summary        string
	IdempotencyKey string
}

type Directory interface {
	Resolve(ctx context.Context, sub Subject) (OrderSnapshot, error)
}

type Bridge interface {
	List(ctx context.Context, connectionID uint, orderNo string) ([]Ticket, error)
	Create(ctx context.Context, connectionID uint, orderNo, customerEmail, category, summary, idempotencyKey string) (Ticket, error)
	AppendSupplement(ctx context.Context, connectionID uint, orderNo, ticketID, message string) (Ticket, error)
}

type Service struct {
	directory Directory
	bridge    Bridge
}

func New(directory Directory, bridge Bridge) *Service {
	return &Service{directory: directory, bridge: bridge}
}

func (s *Service) List(ctx context.Context, sub Subject) (ListResult, error) {
	order, err := s.resolveOrder(ctx, sub)
	if err != nil {
		return ListResult{}, err
	}
	items, err := s.bridge.List(ctx, order.ConnectionID, order.OrderNo)
	if err != nil {
		return ListResult{}, err
	}
	if items == nil {
		items = []Ticket{}
	}
	return ListResult{OrderNo: order.OrderNo, Items: items}, nil
}

func (s *Service) Create(ctx context.Context, sub Subject, in CreateInput) (Ticket, error) {
	category := strings.ToLower(strings.TrimSpace(in.Category))
	if category == "" {
		category = "customer_inquiry"
	}
	if _, ok := allowedCategories[category]; !ok {
		return Ticket{}, ErrInvalid
	}
	summary := strings.TrimSpace(in.Summary)
	if summary == "" || len([]rune(summary)) > maxSummaryRunes {
		return Ticket{}, ErrInvalid
	}
	order, err := s.resolveOrder(ctx, sub)
	if err != nil {
		return Ticket{}, err
	}
	return s.bridge.Create(
		ctx,
		order.ConnectionID,
		order.OrderNo,
		strings.TrimSpace(order.CustomerEmail),
		category,
		summary,
		strings.TrimSpace(in.IdempotencyKey),
	)
}

func (s *Service) AppendSupplement(ctx context.Context, sub Subject, ticketID, message string) (Ticket, error) {
	ticketID = strings.TrimSpace(ticketID)
	message = strings.TrimSpace(message)
	if ticketID == "" || message == "" || len([]rune(message)) > maxSummaryRunes {
		return Ticket{}, ErrInvalid
	}
	order, err := s.resolveOrder(ctx, sub)
	if err != nil {
		return Ticket{}, err
	}
	return s.bridge.AppendSupplement(ctx, order.ConnectionID, order.OrderNo, ticketID, message)
}

func (s *Service) resolveOrder(ctx context.Context, sub Subject) (OrderSnapshot, error) {
	if s == nil || s.directory == nil || s.bridge == nil {
		return OrderSnapshot{}, ErrUnavailable
	}
	sub.OrderNo = strings.TrimSpace(sub.OrderNo)
	if sub.OrderNo == "" {
		return OrderSnapshot{}, ErrInvalid
	}
	if sub.UserID == 0 && (strings.TrimSpace(sub.GuestEmail) == "" || strings.TrimSpace(sub.GuestPassword) == "") {
		return OrderSnapshot{}, ErrNotFound
	}
	order, err := s.directory.Resolve(ctx, sub)
	if err != nil {
		return OrderSnapshot{}, err
	}
	if order.OrderID == 0 || strings.TrimSpace(order.OrderNo) == "" || order.ConnectionID == 0 {
		return OrderSnapshot{}, ErrUnavailable
	}
	return order, nil
}
