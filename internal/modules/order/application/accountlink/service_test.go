package accountlink

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubBridge struct {
	gotRef    string
	gotNext   string
	gotCID    string
	gotAppr   string
	statusRes Status
	startRes  Challenge
	doneRes   CompleteResult
	err       error
}

func (s *stubBridge) Status(_ context.Context, ref string) (Status, error) {
	s.gotRef = ref
	return s.statusRes, s.err
}
func (s *stubBridge) Start(_ context.Context, ref, next string) (Challenge, error) {
	s.gotRef = ref
	s.gotNext = next
	return s.startRes, s.err
}
func (s *stubBridge) Complete(_ context.Context, ref, cid, appr string) (CompleteResult, error) {
	s.gotRef = ref
	s.gotCID = cid
	s.gotAppr = appr
	return s.doneRes, s.err
}

func TestServiceEnforcesSubjectAndSafeNextPath(t *testing.T) {
	bridge := &stubBridge{
		statusRes: Status{Enabled: true, Linked: false, Status: "unlinked", PendingOrders: 1},
		startRes:  Challenge{ChallengeID: "lnk_1", AuthorizeURL: "https://api.needtoken.ai/account-link?challenge=tok", ExpiresAt: time.Now().Add(time.Minute)},
		doneRes:   CompleteResult{Linked: true, Status: "self_verified", NativeUserID: 9, NextPath: "/orders/NT-1", AwakenedOrders: 1},
	}
	svc := New(bridge)

	st, err := svc.Status(context.Background(), 7)
	if err != nil || bridge.gotRef != "needtoken-storefront:user:7" || st.PendingOrders != 1 {
		t.Fatalf("status=%+v ref=%q err=%v", st, bridge.gotRef, err)
	}
	if _, err := svc.Start(context.Background(), 7, "https://evil.example/phish"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("open redirect next_path must be rejected: %v", err)
	}
	if _, err := svc.Start(context.Background(), 7, "/orders/../../etc"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("traversal next_path must be rejected: %v", err)
	}
	ch, err := svc.Start(context.Background(), 7, "")
	if err != nil || bridge.gotNext != "/plans" || ch.ChallengeID != "lnk_1" {
		t.Fatalf("default start failed: ch=%+v next=%q err=%v", ch, bridge.gotNext, err)
	}

	bridge.startRes.AuthorizeURL = "javascript:alert(1)"
	if _, err := svc.Start(context.Background(), 7, "/plans"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unsafe authorize_url must fail closed: %v", err)
	}

	done, err := svc.Complete(context.Background(), 7, "lnk_1", "v1.abc.def")
	if err != nil || !done.Linked || done.NextPath != "/orders/NT-1" || bridge.gotRef != "needtoken-storefront:user:7" {
		t.Fatalf("complete=%+v err=%v", done, err)
	}

	bridge.doneRes.NextPath = "https://evil.example"
	done, err = svc.Complete(context.Background(), 7, "lnk_1", "v1.abc.def")
	if err != nil || done.NextPath != "/orders" {
		t.Fatalf("unsafe remote next_path must be sanitized: %+v err=%v", done, err)
	}

	if _, err := svc.Complete(context.Background(), 7, "", "tok"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty challenge_id must fail: %v", err)
	}
	if _, err := svc.Status(context.Background(), 0); !errors.Is(err, ErrInvalid) {
		t.Fatalf("zero user id must fail: %v", err)
	}
}

func TestServiceNilBridgeDegradesGracefully(t *testing.T) {
	svc := New(nil)
	st, err := svc.Status(context.Background(), 12)
	if err != nil || st.Enabled || st.Status != "disabled" {
		t.Fatalf("unexpected status=%+v err=%v", st, err)
	}
	if _, err := svc.Start(context.Background(), 12, "/plans"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("want ErrUnavailable, got %v", err)
	}
}
