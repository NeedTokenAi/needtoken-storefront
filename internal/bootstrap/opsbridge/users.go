package opsbridge

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dujiao-next/internal/cache"
	"github.com/dujiao-next/internal/constants"
	fulfillmentapp "github.com/dujiao-next/internal/modules/fulfillment/application"
	externalidentitycontract "github.com/dujiao-next/internal/modules/identity/externalidentity/contract"
	usercontract "github.com/dujiao-next/internal/modules/identity/user/contract"
	userdomain "github.com/dujiao-next/internal/modules/identity/user/domain"
	userauthapp "github.com/dujiao-next/internal/modules/identity/userauth/application"
	opscontract "github.com/dujiao-next/internal/modules/opsbridge/contract"
	orderapp "github.com/dujiao-next/internal/modules/order/application"
	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	paymentapp "github.com/dujiao-next/internal/modules/payment/application"
	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
	procurementapp "github.com/dujiao-next/internal/modules/procurement/application"
	procurementcontract "github.com/dujiao-next/internal/modules/procurement/contract"
	settingsapp "github.com/dujiao-next/internal/modules/settings/application"
	walletapp "github.com/dujiao-next/internal/modules/wallet/application"
	walletcontract "github.com/dujiao-next/internal/modules/wallet/contract"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
)

type userStoreAdapter struct {
	users    usercontract.Store
	wallets  *walletapp.Service
	oauth    externalidentitycontract.Store
	userAuth *userauthapp.Service
	settings *settingsapp.Service
}

func (a *userStoreAdapter) siteCurrency() string {
	if a.settings != nil {
		if cur, err := a.settings.GetSiteCurrency(constants.SiteCurrencyDefault); err == nil && strings.TrimSpace(cur) != "" {
			return strings.ToUpper(strings.TrimSpace(cur))
		}
	}
	return constants.SiteCurrencyDefault
}

func (a *userStoreAdapter) mapUser(u userdomain.User, balance money.Amount, currency string, providers []string) opscontract.StorefrontUser {
	return opscontract.StorefrontUser{
		ID:             u.ID,
		Email:          u.Email,
		DisplayName:    u.DisplayName,
		Locale:         u.Locale,
		Status:         u.Status,
		MemberLevelID:  u.MemberLevelID,
		WalletBalance:  balance.Decimal.Round(2).StringFixed(2),
		WalletCurrency: currency,
		TotalRecharged: u.TotalRecharged.Decimal.Round(2).StringFixed(2),
		TotalSpent:     u.TotalSpent.Decimal.Round(2).StringFixed(2),
		AdminNote:      u.AdminNote,
		TOTPEnabled:    u.TOTPEnabledAt != nil,
		EmailVerified:  u.EmailVerifiedAt != nil,
		OAuthProviders: providers,
		LastLoginAt:    u.LastLoginAt,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}

func (a *userStoreAdapter) loadOAuthProviders(userID uint) []string {
	if a.oauth == nil || userID == 0 {
		return nil
	}
	ids, err := a.oauth.ListByUserID(userID)
	if err != nil || len(ids) == 0 {
		return nil
	}
	out := make([]string, 0, len(ids))
	for _, item := range ids {
		if p := strings.TrimSpace(item.Provider); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (a *userStoreAdapter) ListUsers(page, pageSize int, keyword, status string) (*opscontract.UsersResult, error) {
	if a.users == nil {
		return nil, opscontract.ErrNotConfigured
	}
	users, total, err := a.users.List(usercontract.ListFilter{
		Page:     page,
		PageSize: pageSize,
		Keyword:  keyword,
		Status:   status,
	})
	if err != nil {
		return nil, err
	}
	balances := map[uint]money.Amount{}
	if a.wallets != nil && len(users) > 0 {
		ids := make([]uint, 0, len(users))
		for _, u := range users {
			ids = append(ids, u.ID)
		}
		if loaded, bErr := a.wallets.GetBalancesByUserIDs(ids); bErr == nil && loaded != nil {
			balances = loaded
		}
	}
	currency := a.siteCurrency()
	items := make([]opscontract.StorefrontUser, 0, len(users))
	for _, u := range users {
		items = append(items, a.mapUser(u, balances[u.ID], currency, a.loadOAuthProviders(u.ID)))
	}
	return &opscontract.UsersResult{
		Source:      "needtoken-storefront",
		CollectedAt: time.Now().UTC(),
		Freshness:   "fresh",
		Users:       items,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
	}, nil
}

func (a *userStoreAdapter) CreateUser(input opscontract.CreateUserInput) (*opscontract.StorefrontUser, error) {
	if a.users == nil {
		return nil, opscontract.ErrNotConfigured
	}
	normalized, err := userauthapp.NormalizeEmail(input.Email)
	if err != nil {
		return nil, opscontract.ErrInvalidInput
	}
	existing, err := a.users.GetByEmail(normalized)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, &opscontract.ConflictError{Code: "email_exists"}
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	displayName := userdomain.SanitizeDisplayName(input.DisplayName)
	if displayName == "" {
		if local, _, ok := strings.Cut(normalized, "@"); ok {
			displayName = userdomain.SanitizeDisplayName(local)
		}
	}
	locale := strings.TrimSpace(input.Locale)
	if locale == "" {
		locale = constants.LocaleZhCN
	}
	now := time.Now().UTC()
	var verifiedAt *time.Time
	if input.EmailVerified == nil || *input.EmailVerified {
		verifiedAt = &now
	}
	user := &userdomain.User{
		Email:           normalized,
		PasswordHash:    string(hashed),
		DisplayName:     displayName,
		Locale:          locale,
		Status:          input.Status,
		MemberLevelID:   input.MemberLevelID,
		AdminNote:       strings.TrimSpace(input.AdminNote),
		EmailVerifiedAt: verifiedAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := a.users.Create(user); err != nil {
		return nil, err
	}
	mapped := a.mapUser(*user, money.Amount{}, a.siteCurrency(), nil)
	return &mapped, nil
}

func (a *userStoreAdapter) UpdateUser(id uint, input opscontract.UpdateUserInput) (*opscontract.StorefrontUser, error) {
	if a.users == nil {
		return nil, opscontract.ErrNotConfigured
	}
	user, err := a.users.GetByID(id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, opscontract.ErrNotFound
	}
	updated := false
	revokeToken := false

	if input.Email != nil {
		normalized, err := userauthapp.NormalizeEmail(*input.Email)
		if err != nil {
			return nil, opscontract.ErrInvalidInput
		}
		if normalized != user.Email {
			existing, err := a.users.GetByEmail(normalized)
			if err != nil {
				return nil, err
			}
			if existing != nil && existing.ID != user.ID {
				return nil, &opscontract.ConflictError{Code: "email_exists"}
			}
			user.Email = normalized
			updated = true
			revokeToken = true
		}
	}
	if input.DisplayName != nil {
		trimmed := strings.TrimSpace(*input.DisplayName)
		if trimmed != "" {
			if !userdomain.ValidDisplayName(trimmed) {
				return nil, opscontract.ErrInvalidInput
			}
			if trimmed != user.DisplayName {
				user.DisplayName = trimmed
				updated = true
			}
		}
	}
	if input.Password != nil && *input.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(*input.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.PasswordHash = string(hashed)
		user.PasswordSetupRequired = false
		updated = true
		revokeToken = true
	}
	if input.Locale != nil {
		if trimmed := strings.TrimSpace(*input.Locale); trimmed != "" && trimmed != user.Locale {
			user.Locale = trimmed
			updated = true
		}
	}
	if input.Status != nil && *input.Status != user.Status {
		user.Status = *input.Status
		updated = true
		if *input.Status == constants.UserStatusDisabled {
			revokeToken = true
		}
	}
	if input.MemberLevelID != nil && *input.MemberLevelID != user.MemberLevelID {
		user.MemberLevelID = *input.MemberLevelID
		updated = true
	}
	if input.AdminNote != nil && *input.AdminNote != user.AdminNote {
		user.AdminNote = *input.AdminNote
		updated = true
	}
	if input.EmailVerified != nil {
		if *input.EmailVerified && user.EmailVerifiedAt == nil {
			now := time.Now().UTC()
			user.EmailVerifiedAt = &now
			updated = true
		} else if !*input.EmailVerified && user.EmailVerifiedAt != nil {
			user.EmailVerifiedAt = nil
			updated = true
		}
	}

	if updated {
		now := time.Now().UTC()
		user.UpdatedAt = now
		if revokeToken {
			user.TokenVersion++
			user.TokenInvalidBefore = &now
		}
		if err := a.users.Update(user); err != nil {
			return nil, err
		}
		_ = cache.SetUserAuthState(context.Background(), cache.BuildUserAuthState(user))
	}
	balance := money.Amount{}
	if a.wallets != nil {
		if acc, wErr := a.wallets.GetAccount(user.ID); wErr == nil && acc != nil {
			balance = acc.Balance
		}
	}
	mapped := a.mapUser(*user, balance, a.siteCurrency(), a.loadOAuthProviders(user.ID))
	return &mapped, nil
}

func (a *userStoreAdapter) DeleteUser(id uint) error {
	if a.users == nil {
		return opscontract.ErrNotConfigured
	}
	user, err := a.users.GetByID(id)
	if err != nil {
		return err
	}
	if user == nil {
		return opscontract.ErrNotFound
	}
	now := time.Now().UTC()
	archivedEmail := fmt.Sprintf("deleted_%d_%d@deleted.needtoken.invalid", user.ID, now.Unix())
	if err := a.users.SoftDelete(user.ID, archivedEmail, now); err != nil {
		return err
	}
	_ = cache.DelUserAuthState(context.Background(), user.ID)
	return nil
}

func (a *userStoreAdapter) AdjustUserWallet(id uint, input opscontract.AdjustUserWalletInput) (*opscontract.WalletAdjustmentResult, error) {
	if a.users == nil || a.wallets == nil {
		return nil, opscontract.ErrNotConfigured
	}
	user, err := a.users.GetByID(id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, opscontract.ErrNotFound
	}
	amtDec, err := decimal.NewFromString(input.Amount)
	if err != nil || amtDec.LessThanOrEqual(decimal.Zero) {
		return nil, opscontract.ErrInvalidInput
	}
	delta := amtDec.Round(2)
	if input.Operation == "debit" {
		delta = delta.Neg()
	}
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if currency == "" {
		currency = a.siteCurrency()
	}
	remark := strings.TrimSpace(input.Remark)
	if remark == "" {
		remark = "NeedToken Ops 管理员调额"
	}
	acc, txn, err := a.wallets.AdminAdjustBalance(walletcontract.AdjustBalanceInput{
		UserID:          user.ID,
		OperatorAdminID: 1,
		Delta:           money.FromDecimal(delta),
		Currency:        currency,
		Remark:          remark,
	})
	if err != nil {
		return nil, &opscontract.ConflictError{Code: err.Error()}
	}
	return &opscontract.WalletAdjustmentResult{
		Source:        "needtoken-storefront",
		CollectedAt:   time.Now().UTC(),
		UserID:        user.ID,
		Balance:       acc.Balance.Decimal.Round(2).StringFixed(2),
		Currency:      currency,
		TransactionID: txn.ID,
		Direction:     txn.Direction,
		Amount:        txn.Amount.Decimal.Round(2).StringFixed(2),
	}, nil
}

func (a *userStoreAdapter) ClearUserTOTP(id uint) error {
	if a.users == nil {
		return opscontract.ErrNotConfigured
	}
	user, err := a.users.GetByID(id)
	if err != nil {
		return err
	}
	if user == nil {
		return opscontract.ErrNotFound
	}
	if err := a.users.ClearTOTP(user.ID); err != nil {
		return err
	}
	if refreshed, rErr := a.users.GetByID(user.ID); rErr == nil && refreshed != nil {
		_ = cache.SetUserAuthState(context.Background(), cache.BuildUserAuthState(refreshed))
	}
	return nil
}

func (a *userStoreAdapter) UnbindUserOAuth(id uint, provider string) error {
	if a.userAuth == nil {
		return opscontract.ErrNotConfigured
	}
	var err error
	switch provider {
	case "telegram":
		err = a.userAuth.UnbindTelegram(id)
	case "google":
		err = a.userAuth.UnbindGoogle(id)
	default:
		return opscontract.ErrInvalidInput
	}
	if err != nil {
		return &opscontract.ConflictError{Code: err.Error()}
	}
	return nil
}

type orderEnricherAdapter struct {
	users usercontract.Store
}

func (a *orderEnricherAdapter) LookupBuyers(userIDs []uint) (map[uint]opscontract.OrderBuyerInfo, error) {
	if a.users == nil || len(userIDs) == 0 {
		return map[uint]opscontract.OrderBuyerInfo{}, nil
	}
	rows, err := a.users.ListByIDs(userIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[uint]opscontract.OrderBuyerInfo, len(rows))
	for _, u := range rows {
		out[u.ID] = opscontract.OrderBuyerInfo{
			Email:       strings.TrimSpace(u.Email),
			DisplayName: strings.TrimSpace(u.DisplayName),
		}
	}
	return out, nil
}

type orderActionStoreAdapter struct {
	orders       ordercontract.Store
	orderService *orderapp.OrderService
	payments     paymentcontract.Store
	paymentSvc   *paymentapp.PaymentService
	fulfillments *fulfillmentapp.Service
	procurements *procurementapp.Service
}

func (a *orderActionStoreAdapter) findOrderByNo(orderNo string) (*orderdomain.Order, error) {
	rows, _, err := a.orders.ListAdmin(ordercontract.ListFilter{
		OrderNo:  strings.TrimSpace(orderNo),
		Page:     1,
		PageSize: 1,
	})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	order := rows[0]
	return &order, nil
}

func (a *orderActionStoreAdapter) ManualConfirmPayment(orderNo string, input opscontract.ManualConfirmPaymentInput) (*opscontract.ManualConfirmPaymentResult, error) {
	if a.orders == nil || a.payments == nil || a.paymentSvc == nil {
		return nil, opscontract.ErrNotConfigured
	}
	order, err := a.findOrderByNo(orderNo)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, opscontract.ErrNotFound
	}
	paymentID := input.PaymentID
	if paymentID == 0 {
		list, listErr := a.payments.ListByOrderID(order.ID)
		if listErr != nil {
			return nil, listErr
		}
		for i := len(list) - 1; i >= 0; i-- {
			if list[i].ExceptionCode == constants.PaymentExceptionUnderpaidSucceeded {
				continue
			}
			paymentID = list[i].ID
			break
		}
		if paymentID == 0 {
			return nil, opscontract.ErrNotFound
		}
	}
	remark := strings.TrimSpace(input.Remark)
	if remark == "" {
		remark = "NeedToken Ops 管理员人工确认支付"
	}
	confirmed, err := a.paymentSvc.ManualConfirmPayment(paymentapp.ManualConfirmPaymentInput{
		OrderID:     order.ID,
		PaymentID:   paymentID,
		AdminID:     1,
		ProviderRef: strings.TrimSpace(input.ProviderRef),
		Remark:      remark,
	})
	if err != nil {
		switch {
		case errors.Is(err, orderapp.ErrOrderNotFound), errors.Is(err, paymentapp.ErrPaymentNotFound):
			return nil, opscontract.ErrNotFound
		case errors.Is(err, paymentapp.ErrPaymentInvalid):
			return nil, opscontract.ErrInvalidInput
		default:
			return nil, &opscontract.ConflictError{Code: err.Error()}
		}
	}
	orderStatus := order.Status
	if refreshed, rErr := a.orders.GetByID(order.ID); rErr == nil && refreshed != nil {
		orderStatus = refreshed.Status
	}
	return &opscontract.ManualConfirmPaymentResult{
		Source:        "storefront",
		CollectedAt:   time.Now().UTC(),
		OrderNo:       order.OrderNo,
		PaymentID:     confirmed.ID,
		PaymentStatus: confirmed.Status,
		OrderStatus:   orderStatus,
	}, nil
}

func (a *orderActionStoreAdapter) ManualFulfillOrder(orderNo string, input opscontract.ManualFulfillOrderInput) (*opscontract.ManualFulfillOrderResult, error) {
	if a.orders == nil || a.fulfillments == nil {
		return nil, opscontract.ErrNotConfigured
	}
	order, err := a.findOrderByNo(orderNo)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, opscontract.ErrNotFound
	}
	targetID := input.TargetOrderID
	if targetID == 0 {
		if order.ParentID == nil && len(order.Children) > 0 {
			for _, child := range order.Children {
				if child.Status == constants.OrderStatusPaid || child.Status == constants.OrderStatusFulfilling {
					targetID = child.ID
					break
				}
			}
			if targetID == 0 {
				targetID = order.Children[0].ID
			}
		} else {
			targetID = order.ID
		}
	} else if targetID != order.ID && !orderHasChildID(order, targetID) {
		return nil, opscontract.ErrInvalidInput
	}
	created, err := a.fulfillments.CreateManual(fulfillmentapp.CreateManualInput{
		OrderID: targetID,
		AdminID: 1,
		Payload: strings.TrimSpace(input.Payload),
	})
	if err != nil {
		switch {
		case errors.Is(err, fulfillmentapp.ErrOrderNotFound):
			return nil, opscontract.ErrNotFound
		case errors.Is(err, fulfillmentapp.ErrFulfillmentInvalid):
			return nil, opscontract.ErrInvalidInput
		case errors.Is(err, fulfillmentapp.ErrOrderStatusInvalid), errors.Is(err, fulfillmentapp.ErrFulfillmentExists):
			return nil, &opscontract.ConflictError{Code: err.Error()}
		default:
			return nil, err
		}
	}
	return &opscontract.ManualFulfillOrderResult{
		Source:        "storefront",
		CollectedAt:   time.Now().UTC(),
		OrderNo:       order.OrderNo,
		TargetOrderID: targetID,
		FulfillmentID: created.ID,
		Status:        created.Status,
	}, nil
}

func orderHasChildID(order *orderdomain.Order, childID uint) bool {
	if order == nil {
		return false
	}
	for _, child := range order.Children {
		if child.ID == childID {
			return true
		}
	}
	return false
}

func (a *orderActionStoreAdapter) UpdateOrderStatus(orderNo string, input opscontract.UpdateOrderStatusInput) (*opscontract.UpdateOrderStatusResult, error) {
	if a.orders == nil || a.orderService == nil {
		return nil, opscontract.ErrNotConfigured
	}
	order, err := a.findOrderByNo(orderNo)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, opscontract.ErrNotFound
	}
	updated, err := a.orderService.UpdateOrderStatus(order.ID, input.Status)
	if err != nil {
		switch {
		case errors.Is(err, orderapp.ErrOrderNotFound):
			return nil, opscontract.ErrNotFound
		case errors.Is(err, orderapp.ErrOrderStatusInvalid):
			return nil, &opscontract.ConflictError{Code: err.Error()}
		default:
			return nil, err
		}
	}
	return &opscontract.UpdateOrderStatusResult{
		Source:      "storefront",
		CollectedAt: time.Now().UTC(),
		OrderNo:     updated.OrderNo,
		Status:      updated.Status,
	}, nil
}

func (a *orderActionStoreAdapter) RetryProcurement(id uint) (*opscontract.RetryProcurementResult, error) {
	if a.procurements == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if err := a.procurements.RetryManual(id); err != nil {
		switch {
		case errors.Is(err, procurementcontract.ErrNotFound):
			return nil, opscontract.ErrNotFound
		case errors.Is(err, procurementcontract.ErrStatusInvalid):
			return nil, &opscontract.ConflictError{Code: err.Error()}
		default:
			return nil, err
		}
	}
	return &opscontract.RetryProcurementResult{
		Source:        "storefront",
		CollectedAt:   time.Now().UTC(),
		ProcurementID: id,
		Status:        constants.ProcurementStatusPending,
	}, nil
}
