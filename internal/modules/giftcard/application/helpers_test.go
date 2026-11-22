package application

import (
	"errors"
	"testing"
	"time"

	giftcardcontract "github.com/dujiao-next/internal/modules/giftcard/contract"
	giftcarddomain "github.com/dujiao-next/internal/modules/giftcard/domain"
	walletdomain "github.com/dujiao-next/internal/modules/wallet/domain"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/shopspring/decimal"
)

func TestNormalizeIDsDeduplicatesAndDropsZero(t *testing.T) {
	got := normalizeIDs([]uint{0, 2, 2, 3, 0, 3})
	if len(got) != 2 || got[0] != 2 || got[1] != 3 {
		t.Fatalf("unexpected ids: %#v", got)
	}
}

func TestNormalizeExpireAtUTC(t *testing.T) {
	raw := time.Date(2026, 7, 21, 10, 0, 0, 0, time.FixedZone("CST", 8*3600))
	got := normalizeExpireAt(&raw)
	if got == nil || got.Location() != time.UTC {
		t.Fatalf("expected UTC expire at, got %#v", got)
	}
}

type stubGiftCardRepo struct {
	giftcardcontract.Repository
}

type stubCurrency struct{ c string }

func (s stubCurrency) SiteCurrency() string { return s.c }

type stubRedeemTx struct {
	localCard   *giftcarddomain.GiftCard
	creditErr   error
	creditCalls []giftcardcontract.WalletCreditInput
}

func (s *stubRedeemTx) GetByCodeForUpdate(code string) (*giftcarddomain.GiftCard, error) {
	return s.localCard, nil
}
func (s *stubRedeemTx) UpdateCard(card *giftcarddomain.GiftCard) error { return nil }
func (s *stubRedeemTx) CreditWallet(input giftcardcontract.WalletCreditInput) (*walletdomain.Account, *walletdomain.Transaction, error) {
	s.creditCalls = append(s.creditCalls, input)
	if s.creditErr != nil {
		return nil, nil, s.creditErr
	}
	return &walletdomain.Account{UserID: input.UserID, Balance: input.Amount}, &walletdomain.Transaction{ID: 88, UserID: input.UserID, Amount: input.Amount}, nil
}

type stubRedeemRunner struct {
	tx *stubRedeemTx
}

func (r *stubRedeemRunner) WithinRedeemTransaction(fn func(tx giftcardcontract.RedeemTransaction) error) error {
	return fn(r.tx)
}

type stubUniversalRedeemer struct {
	claimOut      *giftcardcontract.UniversalGiftCardClaim
	claimErr      error
	confirmed     bool
	released      bool
	confirmTxnRef string
	releaseReason string
}

func (u *stubUniversalRedeemer) Claim(code string, userID uint, siteCurrency string) (*giftcardcontract.UniversalGiftCardClaim, error) {
	if u.claimErr != nil {
		return nil, u.claimErr
	}
	return u.claimOut, nil
}
func (u *stubUniversalRedeemer) Confirm(code, claimID, txnRef, creditDisplay string) error {
	u.confirmed = true
	u.confirmTxnRef = txnRef
	return nil
}
func (u *stubUniversalRedeemer) Release(code, claimID, reason string) error {
	u.released = true
	u.releaseReason = reason
	return nil
}

func TestRedeemUniversalGiftCardFallbackAndRollbackRelease(t *testing.T) {
	tx := &stubRedeemTx{}
	univ := &stubUniversalRedeemer{
		claimOut: &giftcardcontract.UniversalGiftCardClaim{
			ClaimID:      "sf_claim_1",
			Code:         "NTGC-ABCD-1234",
			Name:         "NeedToken 通用卡 $10",
			BatchNo:      "UNI-01",
			CreditAmount: money.FromDecimal(decimal.RequireFromString("72.50")),
			Currency:     "CNY",
			FaceUSDCents: 1000,
		},
	}
	svc := NewService(Options{
		Repo:      stubGiftCardRepo{},
		Currency:  stubCurrency{c: "CNY"},
		Redeemer:  &stubRedeemRunner{tx: tx},
		Universal: univ,
	})

	card, acc, txn, err := svc.RedeemGiftCard(RedeemInput{UserID: 42, Code: " ntgc-abcd-1234 "})
	if err != nil {
		t.Fatalf("expected universal fallback redeem to succeed, got %v", err)
	}
	if card.Code != "NTGC-ABCD-1234" || card.Currency != "CNY" || card.Amount.StringFixed(2) != "72.50" || acc == nil || txn == nil {
		t.Fatalf("unexpected universal redeem output: card=%+v acc=%+v txn=%+v", card, acc, txn)
	}
	if !univ.confirmed || univ.released || univ.confirmTxnRef != "storefront:wallet_txn:88" {
		t.Fatalf("expected Confirm called once and Release not called: %+v", univ)
	}
	if len(tx.creditCalls) != 1 || tx.creditCalls[0].Reference != "universal_gift_card:NTGC-ABCD-1234" {
		t.Fatalf("unexpected wallet credit reference: %+v", tx.creditCalls)
	}

	// Failure path: if local wallet credit fails, Release MUST be invoked so card is never stranded.
	txFail := &stubRedeemTx{creditErr: errors.New("wallet db write failed")}
	univFail := &stubUniversalRedeemer{claimOut: univ.claimOut}
	svcFail := NewService(Options{
		Repo:      stubGiftCardRepo{},
		Currency:  stubCurrency{c: "CNY"},
		Redeemer:  &stubRedeemRunner{tx: txFail},
		Universal: univFail,
	})
	if _, _, _, err := svcFail.RedeemGiftCard(RedeemInput{UserID: 42, Code: "NTGC-ABCD-1234"}); err == nil {
		t.Fatalf("expected wallet failure to propagate")
	}
	if univFail.confirmed || !univFail.released || univFail.releaseReason != "storefront_wallet_credit_failed" {
		t.Fatalf("expected Release invoked on wallet credit error: %+v", univFail)
	}
}
