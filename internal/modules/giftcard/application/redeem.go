package application

import (
	"errors"
	"fmt"
	"strings"
	"time"

	walletdomain "github.com/dujiao-next/internal/modules/wallet/domain"

	"github.com/dujiao-next/internal/constants"
	giftcardcontract "github.com/dujiao-next/internal/modules/giftcard/contract"
	giftcarddomain "github.com/dujiao-next/internal/modules/giftcard/domain"

	"github.com/shopspring/decimal"
)

// RedeemGiftCard 在单一事务中完成礼品卡锁定、钱包入账和兑换状态更新。
func (s *Service) RedeemGiftCard(input RedeemInput) (*giftcarddomain.GiftCard, *walletdomain.Account, *walletdomain.Transaction, error) {
	if s == nil || s.redeemer == nil {
		return nil, nil, nil, giftcardcontract.ErrFetchFailed
	}
	code := strings.TrimSpace(strings.ToUpper(input.Code))
	if input.UserID == 0 || code == "" {
		return nil, nil, nil, giftcardcontract.ErrInvalid
	}

	var (
		resultCard *giftcarddomain.GiftCard
		resultAcc  *walletdomain.Account
		resultTxn  *walletdomain.Transaction
	)
	err := s.redeemer.WithinRedeemTransaction(func(tx giftcardcontract.RedeemTransaction) error {
		card, err := tx.GetByCodeForUpdate(code)
		if err != nil {
			return giftcardcontract.ErrFetchFailed
		}
		if card == nil {
			return giftcardcontract.ErrNotFound
		}
		switch card.Status {
		case giftcarddomain.GiftCardStatusRedeemed:
			return giftcardcontract.ErrRedeemed
		case giftcarddomain.GiftCardStatusDisabled:
			return giftcardcontract.ErrDisabled
		case giftcarddomain.GiftCardStatusActive:
		default:
			return giftcardcontract.ErrInvalid
		}
		if isGiftCardExpired(card.ExpiresAt, time.Now()) {
			return giftcardcontract.ErrExpired
		}
		if card.Amount.Decimal.Round(2).LessThanOrEqual(decimal.Zero) {
			return giftcardcontract.ErrInvalid
		}

		now := time.Now()
		account, txn, err := tx.CreditWallet(giftcardcontract.WalletCreditInput{
			UserID:    input.UserID,
			Amount:    card.Amount,
			Currency:  card.Currency,
			TxnType:   constants.WalletTxnTypeGiftCard,
			Reference: fmt.Sprintf("gift_card:%d", card.ID),
			Remark:    fmt.Sprintf("礼品卡兑换：%s", card.Code),
		})
		if err != nil {
			return err
		}

		card.Status = giftcarddomain.GiftCardStatusRedeemed
		card.RedeemedUserID = &input.UserID
		card.RedeemedAt = &now
		if txn != nil && txn.ID > 0 {
			card.WalletTxnID = &txn.ID
		}
		card.UpdatedAt = now
		if err := tx.UpdateCard(card); err != nil {
			return giftcardcontract.ErrUpdateFailed
		}
		resultCard = card
		resultAcc = account
		resultTxn = txn
		return nil
	})
	if err != nil {
		if errors.Is(err, giftcardcontract.ErrNotFound) && s.universal != nil {
			return s.redeemUniversalGiftCard(input.UserID, code)
		}
		return nil, nil, nil, err
	}
	return resultCard, resultAcc, resultTxn, nil
}

// redeemUniversalGiftCard 在本地事务外通过权威源（Glue）原子预锁双端通用礼品卡，
// 再在独立本地短事务内完成钱包入账，最后确认核销或在入账失败时释放预锁。
func (s *Service) redeemUniversalGiftCard(userID uint, code string) (*giftcarddomain.GiftCard, *walletdomain.Account, *walletdomain.Transaction, error) {
	siteCurrency := ""
	if s.currency != nil {
		siteCurrency = strings.TrimSpace(strings.ToUpper(s.currency.SiteCurrency()))
	}
	if siteCurrency == "" {
		siteCurrency = constants.SiteCurrencyDefault
	}

	claim, err := s.universal.Claim(code, userID, siteCurrency)
	if err != nil {
		return nil, nil, nil, err
	}
	if claim == nil || claim.CreditAmount.Decimal.Round(2).LessThanOrEqual(decimal.Zero) {
		if claim != nil && claim.ClaimID != "" {
			_ = s.universal.Release(code, claim.ClaimID, "invalid_credit_amount")
		}
		return nil, nil, nil, giftcardcontract.ErrInvalid
	}

	creditCurrency := strings.TrimSpace(strings.ToUpper(claim.Currency))
	if creditCurrency == "" {
		creditCurrency = siteCurrency
	}
	remark := fmt.Sprintf("通用礼品卡兑换：%s", claim.Code)
	if claim.FaceUSDCents > 0 {
		remark = fmt.Sprintf("通用礼品卡兑换：%s（$%.2f USD）", claim.Code, float64(claim.FaceUSDCents)/100.0)
	}

	var (
		resultAcc *walletdomain.Account
		resultTxn *walletdomain.Transaction
	)
	txErr := s.redeemer.WithinRedeemTransaction(func(tx giftcardcontract.RedeemTransaction) error {
		account, txn, creditErr := tx.CreditWallet(giftcardcontract.WalletCreditInput{
			UserID:    userID,
			Amount:    claim.CreditAmount,
			Currency:  creditCurrency,
			TxnType:   constants.WalletTxnTypeGiftCard,
			Reference: "universal_gift_card:" + claim.Code,
			Remark:    remark,
		})
		if creditErr != nil {
			return creditErr
		}
		resultAcc = account
		resultTxn = txn
		return nil
	})
	if txErr != nil {
		_ = s.universal.Release(claim.Code, claim.ClaimID, "storefront_wallet_credit_failed")
		return nil, nil, nil, txErr
	}

	txnRef := "storefront:wallet"
	if resultTxn != nil && resultTxn.ID > 0 {
		txnRef = fmt.Sprintf("storefront:wallet_txn:%d", resultTxn.ID)
	}
	creditDisplay := fmt.Sprintf("%s %s", claim.CreditAmount.StringFixed(2), creditCurrency)
	_ = s.universal.Confirm(claim.Code, claim.ClaimID, txnRef, creditDisplay)

	now := time.Now()
	var txnIDPtr *uint
	if resultTxn != nil && resultTxn.ID > 0 {
		id := resultTxn.ID
		txnIDPtr = &id
	}
	card := &giftcarddomain.GiftCard{
		Name:           claim.Name,
		Code:           claim.Code,
		Amount:         claim.CreditAmount,
		Currency:       creditCurrency,
		Status:         giftcarddomain.GiftCardStatusRedeemed,
		ExpiresAt:      claim.ExpiresAt,
		RedeemedUserID: &userID,
		RedeemedAt:     &now,
		WalletTxnID:    txnIDPtr,
	}
	return card, resultAcc, resultTxn, nil
}

func isGiftCardExpired(expiresAt *time.Time, now time.Time) bool {
	if expiresAt == nil || expiresAt.IsZero() {
		return false
	}
	return expiresAt.Before(now)
}
