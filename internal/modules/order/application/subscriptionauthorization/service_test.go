package subscriptionauthorization

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
)

type memoryStore struct {
	mu   sync.Mutex
	rows map[string]orderdomain.SubscriptionAuthorization
}

func (s *memoryStore) Get(_ context.Context, id string) (*orderdomain.SubscriptionAuthorization, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.rows[id]
	if !ok {
		return nil, ordercontract.ErrAuthorizationNotFound
	}
	return &r, nil
}
func (s *memoryStore) Create(_ context.Context, r *orderdomain.SubscriptionAuthorization, _ bool) (*orderdomain.SubscriptionAuthorization, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.rows[r.ID]; ok {
		if old.ContextHash != r.ContextHash || old.OrderID != r.OrderID {
			return nil, false, ordercontract.ErrAuthorizationConflict
		}
		return &old, false, nil
	}
	s.rows[r.ID] = *r
	copy := *r
	return &copy, true, nil
}
func (s *memoryStore) Update(_ context.Context, id string, fn func(*orderdomain.SubscriptionAuthorization) error) (*orderdomain.SubscriptionAuthorization, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.rows[id]
	if !ok {
		return nil, ordercontract.ErrAuthorizationNotFound
	}
	if err := fn(&r); err != nil {
		return nil, err
	}
	s.rows[id] = r
	return &r, nil
}

type directory struct {
	snapshot        OrderSnapshot
	member          uint
	email, password string
}

func (d *directory) Resolve(_ context.Context, s Subject) (OrderSnapshot, error) {
	if s.OrderNo != d.snapshot.OrderNo || s.UserID != d.member || (s.UserID == 0 && (s.GuestEmail != d.email || s.GuestPassword != d.password)) {
		return OrderSnapshot{}, ordercontract.ErrAuthorizationNotFound
	}
	return d.snapshot, nil
}
func (d *directory) Current(_ context.Context, id uint) (OrderSnapshot, error) {
	if id != d.snapshot.ID {
		return OrderSnapshot{}, ordercontract.ErrAuthorizationNotFound
	}
	return d.snapshot, nil
}

type bridge struct {
	mu             sync.Mutex
	remote         Context
	assertions     []Assertion
	validSignature bool
	fail           bool
}

func (b *bridge) Context(context.Context, uint, uint) (Context, error) {
	if b.fail {
		return Context{}, ordercontract.ErrAuthorizationUnavailable
	}
	return b.remote, nil
}
func (b *bridge) Open(_ context.Context, _ uint, a Assertion) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.assertions = append(b.assertions, a)
	if b.fail {
		return ordercontract.ErrAuthorizationUnavailable
	}
	return nil
}
func (b *bridge) CredentialURL(context.Context, uint) (string, error) {
	return "https://integration.example.test/api/authorization/v1/credential", nil
}
func (b *bridge) Verify(context.Context, uint, Signature) bool { return b.validSignature }

type mailer struct {
	code, email string
	count       int
}

func (m *mailer) Send(_ context.Context, email, code, _ string) error {
	m.code = code
	m.email = email
	m.count++
	return nil
}

func fixture(t *testing.T, guest bool) (*Service, *directory, *bridge, *mailer, *memoryStore, *time.Time, Subject) {
	t.Helper()
	now := time.Date(2026, 11, 15, 8, 0, 0, 0, time.UTC)
	terms := Terms{Version: 1, Platform: "openai", Route: "web_card", RouteVersion: 1, Plan: "plus-monthly", Period: "month", RenewalPolicy: "single_cycle", BudgetMinor: "2500", Currency: "USD"}
	raw, _ := json.Marshal(map[string]any{"kind": "recharge", "subscription_contract": terms})
	d := &directory{snapshot: OrderSnapshot{ID: 1, OrderNo: "ORDER-1", TenantRef: "main", BuyerRef: "needtoken-storefront:user:1", GuestEmail: "original@example.test", Paid: true, Leaf: true, ConnectionID: 3, UpstreamOrderID: 9, UpstreamOrderNo: "UP-9", SnapshotHash: strings.Repeat("a", 64), PaymentEnvironment: "unknown", Contract: raw}, member: 1}
	subject := Subject{OrderNo: "ORDER-1", UserID: 1}
	if guest {
		d.member = 0
		d.email = "original@example.test"
		d.password = "private-order-password"
		d.snapshot.BuyerRef = "needtoken-storefront:guest:1"
		subject = Subject{OrderNo: "ORDER-1", GuestEmail: d.email, GuestPassword: d.password}
	}
	b := &bridge{remote: Context{OrderID: 9, OrderNo: "UP-9", SnapshotHash: strings.Repeat("b", 64), Platform: "openai", Route: "web_card", Plan: "plus-monthly", Period: "month"}, validSignature: true}
	m := &mailer{}
	store := &memoryStore{rows: map[string]orderdomain.SubscriptionAuthorization{}}
	s := New(Config{Enabled: true, Key: []byte(strings.Repeat("k", 32)), Clock: func() time.Time { return now }}, store, d, b, m)
	return s, d, b, m, store, &now, subject
}

func TestMemberOpenIsOrderScopedInspectionOnlyAndReplayable(t *testing.T) {
	s, _, b, _, st, now, subject := fixture(t, false)
	ctx := context.Background()
	input := SessionRequest{OperationID: "auth_member_1", ConsentPurchase: true, ConsentCancelRenewal: true}
	v, err := s.Open(ctx, subject, input)
	if err != nil {
		t.Fatal(err)
	}
	if v.ExecutionScope != "inspection_only" || len(v.SessionToken) != 43 || len(b.assertions) != 1 || b.assertions[0].PaymentEnvironment != "unknown" {
		t.Fatalf("unexpected session %#v", v)
	}
	*now = now.Add(time.Minute)
	replay, err := s.Open(ctx, subject, input)
	if err != nil || replay.SessionToken != v.SessionToken || len(b.assertions) != 1 {
		t.Fatalf("replay changed session: %#v %v", replay, err)
	}
	stored, _ := st.Get(ctx, input.OperationID)
	raw, _ := json.Marshal(stored)
	if strings.Contains(string(raw), v.SessionToken) || stored.SessionTokenHash != digestToken(v.SessionToken) {
		t.Fatal("raw session token leaked or hash mismatch")
	}
	other := subject
	other.UserID = 2
	if _, err = s.Open(ctx, other, input); err != ordercontract.ErrAuthorizationNotFound {
		t.Fatalf("other owner accepted: %v", err)
	}
	input.ConsentCancelRenewal = false
	if _, err = s.Open(ctx, subject, input); err != ordercontract.ErrAuthorizationInvalid {
		t.Fatalf("missing cancellation consent accepted: %v", err)
	}
}

func TestGuestRequiresPrivateLookupAndSingleUseOrderMailboxProof(t *testing.T) {
	s, _, _, mail, st, _, subject := fixture(t, true)
	ctx := context.Background()
	id := "auth_guest_1"
	wrong := subject
	wrong.GuestEmail = "attacker@example.test"
	if _, err := s.Challenge(ctx, wrong, id, "en-US"); err != ordercontract.ErrAuthorizationNotFound {
		t.Fatalf("wrong guest accepted: %v", err)
	}
	if _, err := s.Challenge(ctx, subject, id, "en-US"); err != nil {
		t.Fatal(err)
	}
	if mail.email != "original@example.test" || len(mail.code) != 6 {
		t.Fatal("challenge did not use persisted order email")
	}
	if _, err := s.Challenge(ctx, subject, id, "en-US"); err != nil || mail.count != 1 {
		t.Fatal("replay resent challenge")
	}
	input := SessionRequest{OperationID: id, Code: mail.code, ConsentPurchase: true, ConsentCancelRenewal: true}
	v, err := s.Open(ctx, subject, input)
	if err != nil {
		t.Fatal(err)
	}
	row, _ := st.Get(ctx, id)
	if row.CodeHash != "" || row.VerifiedAt == nil || row.Attempts != 1 {
		t.Fatal("proof not consumed atomically")
	}
	replay, err := s.Open(ctx, subject, input)
	if err != nil || replay.SessionToken != v.SessionToken {
		t.Fatal("guest replay created another session")
	}
}

func TestGuestAttemptsPersistAndExpire(t *testing.T) {
	s, _, _, mail, st, now, subject := fixture(t, true)
	ctx := context.Background()
	id := "auth_guest_attempts"
	if _, err := s.Challenge(ctx, subject, id, "en-US"); err != nil {
		t.Fatal(err)
	}
	wrong := "111111"
	if mail.code == wrong {
		wrong = "222222"
	}
	input := SessionRequest{OperationID: id, Code: wrong, ConsentPurchase: true, ConsentCancelRenewal: true}
	for range 5 {
		if _, err := s.Open(ctx, subject, input); err != ordercontract.ErrAuthorizationInvalid {
			t.Fatalf("bad proof accepted: %v", err)
		}
	}
	row, _ := st.Get(ctx, id)
	if row.Attempts != 5 {
		t.Fatalf("failed attempts rolled back: %d", row.Attempts)
	}
	input.Code = mail.code
	if _, err := s.Open(ctx, subject, input); err != ordercontract.ErrAuthorizationInvalid {
		t.Fatal("exhausted proof accepted")
	}
	*now = now.Add(31 * time.Minute)
	if _, err := s.Open(ctx, subject, input); err != ordercontract.ErrAuthorizationConflict {
		t.Fatal("expired session accepted")
	}
}

func TestConsentChecksCurrentOrderContractAndSignature(t *testing.T) {
	s, d, b, _, _, _, subject := fixture(t, false)
	ctx := context.Background()
	input := SessionRequest{OperationID: "auth_consent_1", ConsentPurchase: true, ConsentCancelRenewal: true}
	if _, err := s.Open(ctx, subject, input); err != nil {
		t.Fatal(err)
	}
	v, err := s.Consent(ctx, input.OperationID, Signature{})
	if err != nil || !v.Purchase || !v.CancelRenewal {
		t.Fatal("valid consent unavailable")
	}
	b.validSignature = false
	if _, err = s.Consent(ctx, input.OperationID, Signature{}); err != ordercontract.ErrAuthorizationNotFound {
		t.Fatal("invalid signature accepted")
	}
	b.validSignature = true
	d.snapshot.Paid = false
	if _, err = s.Consent(ctx, input.OperationID, Signature{}); err != ordercontract.ErrAuthorizationConflict {
		t.Fatal("cancelled order consent accepted")
	}
	d.snapshot.Paid = true
	d.snapshot.Contract = []byte(`{"kind":"recharge","subscription_contract":{"plan":"changed"}}`)
	if _, err = s.Consent(ctx, input.OperationID, Signature{}); err != ordercontract.ErrAuthorizationConflict {
		t.Fatal("changed contract accepted")
	}
}

func TestDisabledOrMismatchedUpstreamNeverOpensSession(t *testing.T) {
	s, d, b, _, _, _, subject := fixture(t, false)
	ctx := context.Background()
	s.cfg.Enabled = false
	v, err := s.Eligibility(ctx, subject)
	if err != nil || v.Eligible {
		t.Fatal("disabled service eligible")
	}
	s.cfg.Enabled = true
	b.remote.Route = "apple_iap"
	v, err = s.Eligibility(ctx, subject)
	if err != nil || v.Eligible {
		t.Fatal("mismatched upstream route eligible")
	}
	b.remote.Route = "web_card"
	d.snapshot.Paid = false
	v, err = s.Eligibility(ctx, subject)
	if err != nil || v.Eligible {
		t.Fatal("unpaid order eligible")
	}
	if len(b.assertions) > 0 {
		t.Fatal("eligibility caused external authorization mutation")
	}
}

func TestConcurrentOpensUseOneImmutableAuthorization(t *testing.T) {
	s, _, b, _, st, _, subject := fixture(t, false)
	ctx := context.Background()
	input := SessionRequest{OperationID: "auth_concurrent_1", ConsentPurchase: true, ConsentCancelRenewal: true}
	var wg sync.WaitGroup
	tokens := make(chan string, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := s.Open(ctx, subject, input)
			if err != nil {
				t.Errorf("open failed: %v", err)
				return
			}
			tokens <- v.SessionToken
		}()
	}
	wg.Wait()
	close(tokens)
	var token string
	for value := range tokens {
		if token != "" && value != token {
			t.Fatal("concurrent token changed")
		}
		token = value
	}
	if len(st.rows) != 1 {
		t.Fatal("multiple durable authorizations")
	}
	for _, assertion := range b.assertions {
		if assertion.OperationID != input.OperationID || assertion.SessionTokenHash != digestToken(token) {
			t.Fatal("different upstream authorization snapshot")
		}
	}
}
