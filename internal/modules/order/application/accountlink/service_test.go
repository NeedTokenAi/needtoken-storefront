package accountlink

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type stubBridge struct {
	gotRef        string
	gotNext       string
	gotCID        string
	gotAppr       string
	gotSSO        ConsoleSSOInput
	gotAuto       AutoLinkInput
	autoCalls     atomic.Int32
	autoDelayGate chan struct{}
	statusRes     Status
	startRes      Challenge
	doneRes       CompleteResult
	ssoRes        ConsoleSSOResult
	autoRes       AutoLinkResult
	summaryRes    APISummaryResult
	err           error
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
func (s *stubBridge) ConsoleSSO(_ context.Context, input ConsoleSSOInput) (ConsoleSSOResult, error) {
	s.gotSSO = input
	return s.ssoRes, s.err
}
func (s *stubBridge) AutoLink(_ context.Context, input AutoLinkInput) (AutoLinkResult, error) {
	s.autoCalls.Add(1)
	s.gotAuto = input
	if s.autoDelayGate != nil {
		<-s.autoDelayGate
	}
	return s.autoRes, s.err
}
func (s *stubBridge) APISummary(_ context.Context, ref string) (APISummaryResult, error) {
	s.gotRef = ref
	return s.summaryRes, s.err
}

type stubIdentity struct {
	member MemberIdentity
	err    error
}

func (s *stubIdentity) LookupMember(_ context.Context, _ uint) (MemberIdentity, error) {
	return s.member, s.err
}

func TestServiceEnforcesSubjectAndSafeNextPath(t *testing.T) {
	bridge := &stubBridge{
		statusRes: Status{Enabled: true, Linked: false, Status: "unlinked", PendingOrders: 1},
		startRes:  Challenge{ChallengeID: "lnk_1", AuthorizeURL: "https://api.needtoken.ai/account-link?challenge=tok", ExpiresAt: time.Now().Add(time.Minute)},
		doneRes:   CompleteResult{Linked: true, Status: "self_verified", NativeUserID: 9, NextPath: "/orders/NT-1", AwakenedOrders: 1},
		ssoRes:    ConsoleSSOResult{Mode: "sso", RedirectURL: "https://api.needtoken.ai/oauth?redirect=%2Fdashboard#needtoken_ticket=t1", Linked: true, NativeUserID: 9},
	}
	identity := &stubIdentity{member: MemberIdentity{UserID: 7, Email: "member@example.com", EmailVerified: true, DisplayName: "Member", Active: true}}
	svc := New(bridge).WithIdentityReader(identity)

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

	// ConsoleSSO enforces target_path allowlist, server-side identity, and safe redirect_url.
	sso, err := svc.ConsoleSSO(context.Background(), 7, "", "")
	if err != nil || !sso.Linked || bridge.gotSSO.TargetPath != "/dashboard" || bridge.gotSSO.NextPath != "/me" || bridge.gotSSO.Email != "member@example.com" || !bridge.gotSSO.EmailVerified {
		t.Fatalf("default consoleSSO failed: res=%+v got=%+v err=%v", sso, bridge.gotSSO, err)
	}
	if _, err := svc.ConsoleSSO(context.Background(), 7, "/admin", "/me"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("disallowed target_path must fail: %v", err)
	}
	if _, err := svc.ConsoleSSO(context.Background(), 7, "/keys", "https://evil.example"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("disallowed next_path must fail: %v", err)
	}
	bridge.ssoRes.RedirectURL = "javascript:alert(1)"
	if _, err := svc.ConsoleSSO(context.Background(), 7, "/keys", "/me"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unsafe redirect_url must fail closed: %v", err)
	}
	bridge.ssoRes.RedirectURL = "https://api.needtoken.ai/oauth"
	identity.member.Active = false
	if _, err := svc.ConsoleSSO(context.Background(), 7, "/keys", "/me"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("disabled member must be forbidden: %v", err)
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
	autoRes, err := svc.Ensure(context.Background(), 12)
	if err != nil || autoRes.Linked || autoRes.Status != "disabled" {
		t.Fatalf("nil bridge Ensure should degrade gracefully: %+v err=%v", autoRes, err)
	}
	sumRes, err := svc.APISummary(context.Background(), 12)
	if err != nil || sumRes.Linked {
		t.Fatalf("nil bridge APISummary should degrade gracefully: %+v err=%v", sumRes, err)
	}
}

func TestServiceEnsureSingleflightSecurityAndAPISummary(t *testing.T) {
	gate := make(chan struct{})
	bridge := &stubBridge{
		autoDelayGate: gate,
		autoRes:       AutoLinkResult{Linked: true, Status: "auto_provisioned", NativeUserID: 77, AwakenedOrders: 1},
		summaryRes: APISummaryResult{
			Linked:       true,
			NativeUserID: 77,
			Quota:        2500000,
			UsedQuota:    500000,
			RequestCount: 15,
			Group:        "default",
		},
	}
	identity := &stubIdentity{member: MemberIdentity{UserID: 9, Email: "verified@example.com", EmailVerified: true, DisplayName: "Verified", Active: true}}
	svc := New(bridge).WithIdentityReader(identity)

	// Concurrent Ensure calls for the same user coalesce into a single upstream call.
	var wg sync.WaitGroup
	results := make([]AutoLinkResult, 4)
	errs := make([]error, 4)
	for i := 0; i < 4; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = svc.Ensure(context.Background(), 9)
		}()
	}
	time.Sleep(20 * time.Millisecond)
	close(gate)
	wg.Wait()

	if bridge.autoCalls.Load() != 1 {
		t.Fatalf("concurrent Ensure calls must coalesce into 1 upstream request, got %d", bridge.autoCalls.Load())
	}
	for i := range results {
		if errs[i] != nil || !results[i].Linked || results[i].NativeUserID != 77 {
			t.Fatalf("worker %d unexpected Ensure result: %+v err=%v", i, results[i], errs[i])
		}
	}

	// existing_requires_link must strip NativeUserID and never claim linked.
	bridge.autoDelayGate = nil
	bridge.autoRes = AutoLinkResult{Linked: true, Status: "existing_requires_link", NativeUserID: 999, AwakenedOrders: 3}
	res, err := svc.Ensure(context.Background(), 9)
	if err != nil || res.Linked || res.Status != "existing_requires_link" || res.NativeUserID != 0 || res.AwakenedOrders != 0 {
		t.Fatalf("existing_requires_link must strip native identity: %+v err=%v", res, err)
	}

	// Unverified member email must not be forwarded to upstream AutoLink.
	identity.member.EmailVerified = false
	bridge.autoRes = AutoLinkResult{Linked: false, Status: "email_unverified"}
	res, err = svc.Ensure(context.Background(), 9)
	if err != nil || res.Linked || bridge.gotAuto.Email != "" || bridge.gotAuto.EmailVerified || bridge.gotAuto.AllowWithoutEmail {
		t.Fatalf("unverified member must not forward email: res=%+v gotAuto=%+v err=%v", res, bridge.gotAuto, err)
	}

	// Explicit AllowWithoutEmail forwards empty email with AllowWithoutEmail=true.
	bridge.autoRes = AutoLinkResult{Linked: true, Status: "auto_provisioned", NativeUserID: 88}
	res, err = svc.EnsureWithOptions(context.Background(), 9, EnsureOptions{AllowWithoutEmail: true})
	if err != nil || !res.Linked || res.NativeUserID != 88 || bridge.gotAuto.Email != "" || bridge.gotAuto.EmailVerified || !bridge.gotAuto.AllowWithoutEmail {
		t.Fatalf("explicit allow_without_email EnsureWithOptions failed: res=%+v gotAuto=%+v err=%v", res, bridge.gotAuto, err)
	}

	// Disabled member is forbidden.
	identity.member.Active = false
	if _, err := svc.Ensure(context.Background(), 9); !errors.Is(err, ErrForbidden) {
		t.Fatalf("disabled member Ensure must return ErrForbidden, got %v", err)
	}

	// APISummary passes merchant ref and strips data when unlinked.
	sum, err := svc.APISummary(context.Background(), 9)
	if err != nil || !sum.Linked || sum.NativeUserID != 77 || sum.Quota != 2500000 || bridge.gotRef != "needtoken-storefront:user:9" {
		t.Fatalf("APISummary failed: %+v ref=%q err=%v", sum, bridge.gotRef, err)
	}
	bridge.summaryRes = APISummaryResult{Linked: false, NativeUserID: 77, Quota: 999}
	sum, err = svc.APISummary(context.Background(), 9)
	if err != nil || sum.Linked || sum.NativeUserID != 0 || sum.Quota != 0 {
		t.Fatalf("unlinked APISummary must not expose remote fields: %+v err=%v", sum, err)
	}
}
