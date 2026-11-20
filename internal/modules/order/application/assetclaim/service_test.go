package assetclaim

import (
	"context"
	"errors"
	"testing"
)

type stubDir struct {
	snap OrderSnapshot
	err  error
}

func (s stubDir) Resolve(context.Context, Subject) (OrderSnapshot, error) { return s.snap, s.err }

type stubBridge struct {
	gotBuyer string
	gotActor string
	st       Status
	rev      RevealResult
	err      error
}

func (b *stubBridge) Status(_ context.Context, _, _ uint, buyerRef string) (Status, error) {
	b.gotBuyer = buyerRef
	return b.st, b.err
}
func (b *stubBridge) Reveal(_ context.Context, _, _ uint, buyerRef, actor string) (RevealResult, error) {
	b.gotBuyer = buyerRef
	b.gotActor = actor
	return b.rev, b.err
}

func TestServiceHandlesMemberGuestAndNonDeliveredOrders(t *testing.T) {
	bridge := &stubBridge{
		st:  Status{Eligible: true, AssetID: "av_1", RemainingReveals: 1},
		rev: RevealResult{Status: Status{Eligible: true, AssetID: "av_1"}, AccountIdentifier: "a@b.com", SecretPayload: "pw"},
	}

	// Undelivered order returns Eligible=false and blocks reveal.
	svc := New(stubDir{snap: OrderSnapshot{OrderID: 1, Leaf: true, Delivered: false, ConnectionID: 2, UpstreamOrderID: 3}}, bridge)
	st, err := svc.Status(context.Background(), Subject{OrderNo: "ORD-1", UserID: 10})
	if err != nil || st.Eligible {
		t.Fatalf("undelivered order should not be eligible: %+v err=%v", st, err)
	}
	if _, err := svc.Reveal(context.Background(), Subject{OrderNo: "ORD-1", UserID: 10}); !errors.Is(err, ErrConflict) {
		t.Fatalf("undelivered reveal must fail conflict, got %v", err)
	}

	// Member delivered order binds member buyer_ref.
	svc = New(stubDir{snap: OrderSnapshot{OrderID: 1, UserID: 10, Leaf: true, Delivered: true, ConnectionID: 2, UpstreamOrderID: 3}}, bridge)
	rev, err := svc.Reveal(context.Background(), Subject{OrderNo: "ORD-1", UserID: 10})
	if err != nil || bridge.gotBuyer != "needtoken-storefront:user:10" || bridge.gotActor != "member_10" || rev.SecretPayload != "pw" {
		t.Fatalf("member reveal mismatch: rev=%+v buyer=%q actor=%q err=%v", rev, bridge.gotBuyer, bridge.gotActor, err)
	}

	// Guest delivered order passes empty buyer_ref and guest actor.
	svc = New(stubDir{snap: OrderSnapshot{OrderID: 9, UserID: 0, Leaf: true, Delivered: true, ConnectionID: 2, UpstreamOrderID: 4}}, bridge)
	if _, err := svc.Reveal(context.Background(), Subject{OrderNo: "ORD-9", GuestEmail: "g@b.com", GuestPassword: "pwd"}); err != nil || bridge.gotBuyer != "" || bridge.gotActor != "guest_order_9" {
		t.Fatalf("guest reveal mismatch: buyer=%q actor=%q err=%v", bridge.gotBuyer, bridge.gotActor, err)
	}
}
