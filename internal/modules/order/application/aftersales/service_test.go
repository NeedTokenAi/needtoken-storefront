package aftersales

import (
	"context"
	"errors"
	"testing"
)

type stubDirectory struct {
	snap OrderSnapshot
	err  error
	got  Subject
}

func (s *stubDirectory) Resolve(_ context.Context, sub Subject) (OrderSnapshot, error) {
	s.got = sub
	return s.snap, s.err
}

type stubBridge struct {
	items        []Ticket
	ticket       Ticket
	err          error
	createdCalls int
	lastOrderNo  string
	lastEmail    string
	lastCategory string
	lastSummary  string
	lastIdem     string
	lastTicketID string
	lastMessage  string
}

func (b *stubBridge) List(_ context.Context, _ uint, orderNo string) ([]Ticket, error) {
	b.lastOrderNo = orderNo
	return b.items, b.err
}

func (b *stubBridge) Create(_ context.Context, _ uint, orderNo, customerEmail, category, summary, idem string) (Ticket, error) {
	b.createdCalls++
	b.lastOrderNo = orderNo
	b.lastEmail = customerEmail
	b.lastCategory = category
	b.lastSummary = summary
	b.lastIdem = idem
	return b.ticket, b.err
}

func (b *stubBridge) AppendSupplement(_ context.Context, _ uint, orderNo, ticketID, message string) (Ticket, error) {
	b.lastOrderNo = orderNo
	b.lastTicketID = ticketID
	b.lastMessage = message
	return b.ticket, b.err
}

func TestServiceValidatesOwnershipAndInputs(t *testing.T) {
	dir := &stubDirectory{
		snap: OrderSnapshot{OrderID: 7, OrderNo: "NT-100", UserID: 42, CustomerEmail: "alice@example.com", ConnectionID: 3},
	}
	br := &stubBridge{ticket: Ticket{ID: "otk_1", OrderNo: "NT-100", Status: "open"}}
	svc := New(dir, br)

	// Unauthenticated subject (no UserID and incomplete guest creds) must return ErrNotFound without hitting directory/bridge.
	if _, err := svc.List(context.Background(), Subject{OrderNo: "NT-100", GuestEmail: "alice@example.com"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing guest password, got %v", err)
	}
	if dir.got.OrderNo != "" {
		t.Fatalf("directory should not be called without valid credentials")
	}

	// Invalid category rejected before resolving order.
	if _, err := svc.Create(context.Background(), Subject{OrderNo: "NT-100", UserID: 42}, CreateInput{
		Category: "force_refund",
		Summary:  "give money back",
	}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid for disallowed category, got %v", err)
	}

	// Valid member create derives email and order_no from resolved snapshot.
	created, err := svc.Create(context.Background(), Subject{OrderNo: " NT-100 ", UserID: 42}, CreateInput{
		Category:       "fulfillment_failed",
		Summary:        "  entitlement missing after payment  ",
		IdempotencyKey: "k-1",
	})
	if err != nil {
		t.Fatalf("unexpected create err: %v", err)
	}
	if created.ID != "otk_1" || br.lastOrderNo != "NT-100" || br.lastEmail != "alice@example.com" || br.lastIdem != "k-1" {
		t.Fatalf("unexpected bridge call: order=%q email=%q idem=%q", br.lastOrderNo, br.lastEmail, br.lastIdem)
	}

	// Blank supplement message rejected.
	if _, err := svc.AppendSupplement(context.Background(), Subject{OrderNo: "NT-100", UserID: 42}, "otk_1", "   "); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid for blank message, got %v", err)
	}
}
