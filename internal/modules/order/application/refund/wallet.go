package refund

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	orderapp "github.com/dujiao-next/internal/modules/order/application"
	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	settingsapp "github.com/dujiao-next/internal/modules/settings/application"
	walletcontract "github.com/dujiao-next/internal/modules/wallet/contract"
	walletdomain "github.com/dujiao-next/internal/modules/wallet/domain"
)

// AdminRefundToWalletInput describes an order refund whose settlement target
// is the registered user's wallet.
type AdminRefundToWalletInput struct {
	OperationID string
	OrderID     uint
	Amount      money.Amount
	Remark      string
}

func buildWalletRefundDigestAndRef(operationID string, orderID uint, amount decimal.Decimal, remark string) (string, string) {
	raw, _ := json.Marshal(struct {
		Type           string
		OrderID        uint
		Amount, Remark string
	}{constants.OrderRefundTypeWallet, orderID, amount.StringFixed(2), remark})
	digestSum := sha256.Sum256(raw)
	opSum := sha256.Sum256([]byte(operationID))
	reference := fmt.Sprintf("order:%d:wrefund:%s", orderID, hex.EncodeToString(opSum[:12]))
	return hex.EncodeToString(digestSum[:]), reference
}

func (s *Service) resolveExistingWalletRefund(
	existing *orderdomain.OrderRefundRecord,
	orderID uint,
	digest string,
	reference string,
) (*orderdomain.Order, *walletdomain.Transaction, *orderdomain.OrderRefundRecord, error) {
	if existing.Type != constants.OrderRefundTypeWallet || existing.OrderID != orderID || existing.OperationDigest != digest {
		return nil, nil, nil, ErrRefundOperationConflict
	}
	order, err := s.orderStore.GetByID(orderID)
	if err != nil {
		return nil, nil, nil, ErrOrderFetchFailed
	}
	var txn *walletdomain.Transaction
	_ = s.orderStore.WithinTransaction(func(tx ordercontract.Transaction) error {
		loaded, loadErr := tx.Wallets().Wallets().GetTransactionByReference(reference)
		if loadErr != nil {
			return loadErr
		}
		txn = loaded
		return nil
	})
	return order, txn, existing, nil
}

// AdminRefundToWallet executes the order refund workflow and delegates only
// the account credit to Wallet. Order state, refund records, affiliate
// reversals, and reseller accounting remain owned by the order context.
func (s *Service) AdminRefundToWallet(
	input AdminRefundToWalletInput,
) (*orderdomain.Order, *walletdomain.Transaction, *orderdomain.OrderRefundRecord, error) {
	if input.OrderID == 0 {
		return nil, nil, nil, ErrOrderNotFound
	}
	amount := input.Amount.Decimal.Round(2)
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, nil, nil, walletcontract.ErrInvalidAmount
	}
	if s == nil || s.wallets == nil || s.orderStore == nil {
		return nil, nil, nil, walletcontract.ErrAccountNotFound
	}

	remark := strings.TrimSpace(input.Remark)
	walletRemark := remark
	if walletRemark == "" {
		walletRemark = "管理员退款到余额"
	}
	operationID := strings.TrimSpace(input.OperationID)
	if operationID == "" {
		operationID = uuid.NewString()
	}
	if len(operationID) > 128 {
		return nil, nil, nil, ErrRefundOperationInvalid
	}
	opDigest, reference := buildWalletRefundDigestAndRef(operationID, input.OrderID, amount, remark)

	if existing, err := s.orderStore.GetRefundRecordByOperationID(operationID); err != nil {
		return nil, nil, nil, err
	} else if existing != nil {
		return s.resolveExistingWalletRefund(existing, input.OrderID, opDigest, reference)
	}

	var (
		transactionResult  *walletdomain.Transaction
		refundRecordResult *orderdomain.OrderRefundRecord
		unifiedCreditOp    *walletdomain.UnifiedOp
	)

	config := settingsapp.DefaultOrderRefundConfig()
	if s.settingService != nil {
		loaded, err := s.settingService.GetOrderRefundConfig()
		if err != nil {
			return nil, nil, nil, err
		}
		config = loaded
	}

	err := s.orderStore.WithinTransaction(func(tx ordercontract.Transaction) error {
		orderRepository := tx.Orders()
		locked, err := orderRepository.GetByIDForUpdate(input.OrderID)
		if err != nil {
			return err
		}
		if locked == nil {
			return ErrOrderNotFound
		}
		if existing, lookupErr := orderRepository.GetRefundRecordByOperationID(operationID); lookupErr != nil {
			return lookupErr
		} else if existing != nil {
			if existing.Type != constants.OrderRefundTypeWallet || existing.OrderID != input.OrderID || existing.OperationDigest != opDigest {
				return ErrRefundOperationConflict
			}
			refundRecordResult = existing
			if existingTxn, txnErr := tx.Wallets().Wallets().GetTransactionByReference(reference); txnErr == nil {
				transactionResult = existingTxn
			}
			return nil
		}
		order := *locked
		if order.UserID == 0 {
			return walletcontract.ErrNotSupportedForGuest
		}
		if order.PaidAt == nil {
			return ErrOrderStatusInvalid
		}
		now := time.Now()
		if settingsapp.IsOrderRefundWindowExpired(order.CreatedAt, order.PaidAt, config.MaxRefundDays, now) {
			return ErrOrderRefundExpired
		}
		if order.TotalAmount.Decimal.LessThanOrEqual(decimal.Zero) {
			return ErrOrderStatusInvalid
		}
		refundedBefore := order.RefundedAmount.Decimal.Round(2)
		refundable := order.TotalAmount.Decimal.Sub(refundedBefore).Round(2)
		if amount.GreaterThan(refundable) {
			return walletcontract.ErrRefundExceeded
		}
		if err := reserveParentRefund(orderRepository, &order, amount, now); err != nil {
			return err
		}

		_, transaction, err := s.wallets.CreditInTransaction(
			tx.Wallets(),
			walletcontract.CreditInput{
				UserID:    order.UserID,
				Amount:    money.FromDecimal(amount),
				Currency:  order.Currency,
				Type:      constants.WalletTxnTypeAdminRefund,
				Reference: reference,
				Remark:    walletRemark,
				OrderID:   &order.ID,
			},
		)
		if err != nil {
			return err
		}

		newRefunded := refundedBefore.Add(amount).Round(2)
		updates := map[string]interface{}{
			"refunded_amount": money.FromDecimal(newRefunded),
			"updated_at":      now,
		}
		markRefunded := newRefunded.GreaterThanOrEqual(order.TotalAmount.Decimal.Round(2))
		if markRefunded {
			updates["status"] = constants.OrderStatusRefunded
		} else {
			updates["status"] = constants.OrderStatusPartiallyRefunded
		}
		if err := orderRepository.UpdateFields(order.ID, updates); err != nil {
			return ErrOrderUpdateFailed
		}
		if order.ParentID == nil {
			targetStatus := constants.OrderStatusPartiallyRefunded
			if markRefunded {
				targetStatus = constants.OrderStatusRefunded
			}
			if err := applyParentRefundChildStatusUpdates(orderRepository, order.ID, targetStatus, now); err != nil {
				return ErrOrderUpdateFailed
			}
		} else if _, err := orderapp.SyncParentStatus(orderRepository, *order.ParentID, now); err != nil {
			return ErrOrderUpdateFailed
		}
		if s.affiliateRefund != nil {
			if err := s.affiliateRefund.HandleOrderRefunded(
				tx.Affiliates(),
				&order,
				amount,
				refundedBefore,
				"order_refunded_to_wallet",
			); err != nil {
				return err
			}
		}

		currency := strings.ToUpper(strings.TrimSpace(order.Currency))
		if currency == "" {
			currency = "CNY"
		}
		opIDCopy := operationID
		record := &orderdomain.OrderRefundRecord{
			OperationID:              &opIDCopy,
			OperationDigest:          opDigest,
			UserID:                   order.UserID,
			GuestEmail:               order.GuestEmail,
			OrderID:                  order.ID,
			Type:                     constants.OrderRefundTypeWallet,
			Amount:                   money.FromDecimal(amount),
			PaymentFeeRefunded:       false,
			PaymentFeeRefundedAmount: money.FromDecimal(decimal.Zero),
			Currency:                 currency,
			Remark:                   remark,
			CreatedAt:                now,
			UpdatedAt:                now,
		}
		if err := orderRepository.CreateRefundRecord(record); err != nil {
			return ErrRefundRecordCreateFailed
		}
		if s.resellerAccounting != nil {
			if err := s.resellerAccounting.HandleRefundDeduct(tx.ResellerAccounting(), &order, record, refundedBefore); err != nil {
				return err
			}
		}
		if err := orderRepository.ApplyOrderRefund(order.UserID, amount); err != nil {
			return err
		}
		if s.wallets != nil && s.wallets.HasUnifiedBalance() {
			oid := order.ID
			op, opErr := s.wallets.EnqueueUnifiedCreditInTx(
				tx.Wallets(),
				order.UserID,
				&oid,
				money.FromDecimal(amount),
				currency,
				fmt.Sprintf("order:%d:refund:%d", order.ID, record.ID),
				walletRemark,
			)
			if opErr != nil {
				return opErr
			}
			unifiedCreditOp = op
		}
		transactionResult = transaction
		refundRecordResult = record
		return nil
	})
	if err != nil {
		if existing, lookupErr := s.orderStore.GetRefundRecordByOperationID(operationID); lookupErr == nil && existing != nil {
			return s.resolveExistingWalletRefund(existing, input.OrderID, opDigest, reference)
		}
		return nil, nil, nil, err
	}
	if unifiedCreditOp != nil && s.wallets != nil {
		s.wallets.FlushUnifiedCreditOp(unifiedCreditOp)
	}

	order, err := s.orderStore.GetByID(input.OrderID)
	if err != nil {
		return nil, nil, nil, ErrOrderFetchFailed
	}
	if order == nil {
		return nil, nil, nil, ErrOrderNotFound
	}
	return order, transactionResult, refundRecordResult, nil
}
