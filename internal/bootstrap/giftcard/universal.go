package giftcardwiring

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/dujiao-next/internal/app/container"
	"github.com/dujiao-next/internal/constants"
	giftcardcontract "github.com/dujiao-next/internal/modules/giftcard/contract"
	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/dujiao-next/internal/upstream"
	"github.com/shopspring/decimal"
)

type universalGlueRedeemer struct {
	c                      *container.Container
	configuredConnectionID uint
}

// NewUniversalRedeemer 构造指向 NeedToken Glue 权威源的双端通用礼品卡核销适配器。
func NewUniversalRedeemer(c *container.Container) giftcardcontract.UniversalGiftCardRedeemer {
	var connID uint
	if raw := strings.TrimSpace(os.Getenv("STOREFRONT_ACCOUNT_LINK_CONNECTION_ID")); raw != "" {
		if parsed, err := strconv.ParseUint(raw, 10, strconv.IntSize); err == nil && parsed > 0 {
			connID = uint(parsed)
		}
	}
	return &universalGlueRedeemer{c: c, configuredConnectionID: connID}
}

func (r *universalGlueRedeemer) resolveAdapter() (*upstream.DujiaoNextAdapter, *siteconnectiondomain.Connection, error) {
	if r == nil || r.c == nil || r.c.SiteConnectionService == nil {
		return nil, nil, giftcardcontract.ErrNotFound
	}
	var conn *siteconnectiondomain.Connection
	if r.configuredConnectionID > 0 {
		found, err := r.c.SiteConnectionService.GetByID(r.configuredConnectionID)
		if err != nil || found == nil || found.Status != constants.ConnectionStatusActive || found.Protocol != constants.ConnectionProtocolDujiaoNext {
			return nil, nil, giftcardcontract.ErrNotFound
		}
		conn = found
	} else {
		if r.c.SiteConnectionRepo == nil {
			return nil, nil, giftcardcontract.ErrNotFound
		}
		active, err := r.c.SiteConnectionRepo.ListActive()
		if err != nil {
			return nil, nil, giftcardcontract.ErrFetchFailed
		}
		var matches []siteconnectiondomain.Connection
		for _, item := range active {
			if item.Protocol == constants.ConnectionProtocolDujiaoNext {
				matches = append(matches, item)
			}
		}
		if len(matches) != 1 {
			return nil, nil, giftcardcontract.ErrNotFound
		}
		conn = &matches[0]
	}
	adapter, err := r.c.SiteConnectionService.GetAdapter(conn)
	if err != nil {
		return nil, nil, giftcardcontract.ErrFetchFailed
	}
	dujiaoAdapter, ok := adapter.(*upstream.DujiaoNextAdapter)
	if !ok {
		return nil, nil, giftcardcontract.ErrNotFound
	}
	return dujiaoAdapter, conn, nil
}

func newUniversalClaimID() string {
	var buf [12]byte
	_, _ = rand.Read(buf[:])
	return "sf_" + hex.EncodeToString(buf[:])
}

func (r *universalGlueRedeemer) Claim(code string, userID uint, siteCurrency string) (*giftcardcontract.UniversalGiftCardClaim, error) {
	adapter, conn, err := r.resolveAdapter()
	if err != nil {
		return nil, err
	}
	claimID := newUniversalClaimID()
	res, err := adapter.ClaimUniversalGiftCard(context.Background(), upstream.UniversalGiftCardClaimInput{
		Code:    code,
		Target:  "storefront",
		UserRef: fmt.Sprintf("storefront:user:%d", userID),
		ClaimID: claimID,
	})
	if err != nil {
		return nil, mapUniversalError(err)
	}

	usdAmount := decimal.New(res.FaceUSDCents, -2)
	targetCurrency := strings.TrimSpace(strings.ToUpper(siteCurrency))
	if targetCurrency == "" {
		targetCurrency = constants.SiteCurrencyDefault
	}
	creditDec := usdAmount
	if targetCurrency != "USD" && conn != nil && conn.ExchangeRate.GreaterThan(decimal.Zero) {
		creditDec = usdAmount.Mul(conn.ExchangeRate).Round(2)
	} else {
		creditDec = usdAmount.Round(2)
	}

	name := strings.TrimSpace(res.Name)
	if name == "" {
		name = "NeedToken 通用礼品卡"
	}
	return &giftcardcontract.UniversalGiftCardClaim{
		ClaimID:      res.ClaimID,
		Code:         res.Code,
		Name:         name,
		BatchNo:      res.BatchNo,
		CreditAmount: money.FromDecimal(creditDec),
		Currency:     targetCurrency,
		FaceUSDCents: res.FaceUSDCents,
		ExpiresAt:    res.ExpiresAt,
	}, nil
}

func (r *universalGlueRedeemer) Confirm(code, claimID, txnRef, creditDisplay string) error {
	adapter, _, err := r.resolveAdapter()
	if err != nil {
		return err
	}
	return adapter.ConfirmUniversalGiftCard(context.Background(), upstream.UniversalGiftCardConfirmInput{
		Code:          code,
		ClaimID:       claimID,
		TxnRef:        txnRef,
		CreditDisplay: creditDisplay,
	})
}

func (r *universalGlueRedeemer) Release(code, claimID, reason string) error {
	adapter, _, err := r.resolveAdapter()
	if err != nil {
		return err
	}
	return adapter.ReleaseUniversalGiftCard(context.Background(), upstream.UniversalGiftCardReleaseInput{
		Code:    code,
		ClaimID: claimID,
		Reason:  reason,
	})
}

func mapUniversalError(err error) error {
	switch {
	case errors.Is(err, upstream.ErrUniversalGiftCardNotFound):
		return giftcardcontract.ErrNotFound
	case errors.Is(err, upstream.ErrUniversalGiftCardRedeemed):
		return giftcardcontract.ErrRedeemed
	case errors.Is(err, upstream.ErrUniversalGiftCardDisabled):
		return giftcardcontract.ErrDisabled
	case errors.Is(err, upstream.ErrUniversalGiftCardExpired):
		return giftcardcontract.ErrExpired
	case errors.Is(err, upstream.ErrUniversalGiftCardInvalid):
		return giftcardcontract.ErrInvalid
	default:
		return giftcardcontract.ErrFetchFailed
	}
}
