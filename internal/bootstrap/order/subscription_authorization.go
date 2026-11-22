package orderwiring

import (
	"context"
	"encoding/json"
	"os"
	"strconv"

	"github.com/dujiao-next/internal/app/container"
	"github.com/dujiao-next/internal/constants"
	subscriptionauth "github.com/dujiao-next/internal/modules/order/application/subscriptionauthorization"
	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	authorizationhttp "github.com/dujiao-next/internal/modules/order/transport/http/subscriptionauthorization"
	"github.com/dujiao-next/internal/shared/mailbrand"
	"github.com/dujiao-next/internal/upstream"
)

func NewSubscriptionAuthorizationHandler(c *container.Container) *authorizationhttp.Handler {
	key := os.Getenv("STOREFRONT_ORDER_AUTH_KEY")
	if key == "" {
		key = c.Config.App.SecretKey
	}
	service := subscriptionauth.New(subscriptionauth.Config{Enabled: os.Getenv("STOREFRONT_ORDER_AUTH_ENABLED") == "1", Key: []byte(key)}, c.OrderAuthStore, authorizationDirectory{c}, authorizationBridge{c}, authorizationMailer{c})
	return authorizationhttp.New(service)
}

type authorizationDirectory struct{ c *container.Container }

func (a authorizationDirectory) Resolve(ctx context.Context, subject subscriptionauth.Subject) (subscriptionauth.OrderSnapshot, error) {
	var order *orderdomain.Order
	var err error
	scope := ordercontract.TenantScope{ResellerID: subject.ResellerID}
	if subject.UserID > 0 {
		order, err = a.c.OrderStore.GetAnyByOrderNoAndUserScoped(subject.OrderNo, subject.UserID, scope)
	} else {
		order, err = a.c.OrderStore.GetAnyByOrderNoAndGuestScoped(subject.OrderNo, subject.GuestEmail, subject.GuestPassword, scope)
	}
	if err != nil {
		return subscriptionauth.OrderSnapshot{}, ordercontract.ErrAuthorizationUnavailable
	}
	if order == nil {
		return subscriptionauth.OrderSnapshot{}, ordercontract.ErrAuthorizationNotFound
	}
	return a.snapshot(ctx, order)
}
func (a authorizationDirectory) Current(ctx context.Context, id uint) (subscriptionauth.OrderSnapshot, error) {
	order, err := a.c.OrderStore.GetByID(id)
	if err != nil || order == nil {
		return subscriptionauth.OrderSnapshot{}, ordercontract.ErrAuthorizationNotFound
	}
	return a.snapshot(ctx, order)
}
func (a authorizationDirectory) snapshot(_ context.Context, order *orderdomain.Order) (subscriptionauth.OrderSnapshot, error) {
	// Existing payment rows do not freeze an authoritative gateway environment.
	// Today's channel configuration cannot upgrade a historical payment to live.
	local := subscriptionauth.OrderSnapshot{ID: order.ID, OrderNo: order.OrderNo, TenantRef: "main", GuestEmail: order.GuestEmail, Leaf: len(order.Children) == 0 && len(order.Items) == 1, PaymentEnvironment: "unknown"}
	if order.ResellerID != nil {
		local.TenantRef = "reseller:" + strconv.FormatUint(uint64(*order.ResellerID), 10)
	}
	if order.UserID > 0 {
		local.BuyerRef = "needtoken-storefront:user:" + strconv.FormatUint(uint64(order.UserID), 10)
	} else {
		local.BuyerRef = "needtoken-storefront:guest:" + strconv.FormatUint(uint64(order.ID), 10)
	}
	local.Paid = order.PaidAt != nil && order.RefundedAmount.IsZero() && (order.Status == constants.OrderStatusPaid || order.Status == constants.OrderStatusFulfilling)
	if local.Paid && order.WalletPaidAmount.IsZero() && a.c.PaymentStore != nil {
		payOrderID := order.ID
		if order.ParentID != nil && *order.ParentID > 0 {
			payOrderID = *order.ParentID
		}
		if payments, err := a.c.PaymentStore.ListByOrderID(payOrderID); err == nil {
			matched := 0
			resolvedEnv := "unknown"
			for i := range payments {
				p := &payments[i]
				if p.Status != constants.PaymentStatusSuccess || p.SupersededAt != nil || p.ExceptionCode != "" || p.ProviderType == constants.PaymentProviderWallet {
					continue
				}
				matched++
				resolvedEnv = p.FrozenPaymentEnvironment()
			}
			if matched == 1 && (resolvedEnv == "live" || resolvedEnv == "sandbox") {
				local.PaymentEnvironment = resolvedEnv
			}
		}
	}
	if !local.Leaf {
		return local, nil
	}
	item := order.Items[0]
	if item.Quantity != 1 {
		return local, nil
	}
	local.Contract, _ = json.Marshal(item.SKUSnapshotJSON["entitlement_contract"])
	local.SnapshotHash, _ = item.SKUSnapshotJSON["entitlement_contract_hash"].(string)
	if a.c.ProcurementOrderRepo == nil {
		return local, nil
	}
	procurement, err := a.c.ProcurementOrderRepo.GetByLocalOrderID(order.ID)
	if err != nil {
		return local, ordercontract.ErrAuthorizationUnavailable
	}
	if procurement != nil {
		local.ConnectionID = procurement.ConnectionID
		local.UpstreamOrderID = procurement.UpstreamOrderID
		local.UpstreamOrderNo = procurement.UpstreamOrderNo
	}
	return local, nil
}

type authorizationBridge struct{ c *container.Container }

func (a authorizationBridge) adapter(id uint) (*upstream.DujiaoNextAdapter, error) {
	if a.c.SiteConnectionService == nil {
		return nil, ordercontract.ErrAuthorizationUnavailable
	}
	connection, err := a.c.SiteConnectionService.GetByID(id)
	if err != nil || connection == nil || connection.Status != constants.ConnectionStatusActive {
		return nil, ordercontract.ErrAuthorizationUnavailable
	}
	adapter, err := a.c.SiteConnectionService.GetAdapter(connection)
	if err != nil {
		return nil, ordercontract.ErrAuthorizationUnavailable
	}
	value, ok := adapter.(*upstream.DujiaoNextAdapter)
	if !ok {
		return nil, ordercontract.ErrAuthorizationUnavailable
	}
	return value, nil
}
func (a authorizationBridge) Context(ctx context.Context, connectionID, orderID uint) (subscriptionauth.Context, error) {
	v, e := a.adapter(connectionID)
	if e != nil {
		return subscriptionauth.Context{}, e
	}
	return v.SubscriptionAuthorizationContext(ctx, orderID)
}
func (a authorizationBridge) Open(ctx context.Context, connectionID uint, assertion subscriptionauth.Assertion) error {
	v, e := a.adapter(connectionID)
	if e != nil {
		return e
	}
	return v.OpenSubscriptionAuthorization(ctx, assertion)
}
func (a authorizationBridge) CredentialURL(_ context.Context, connectionID uint) (string, error) {
	v, e := a.adapter(connectionID)
	if e != nil {
		return "", e
	}
	return v.SubscriptionCredentialURL()
}
func (a authorizationBridge) Verify(_ context.Context, connectionID uint, signature subscriptionauth.Signature) bool {
	v, e := a.adapter(connectionID)
	return e == nil && v.VerifySubscriptionAuthorization(signature)
}

type authorizationMailer struct{ c *container.Container }

func (a authorizationMailer) Send(ctx context.Context, email, code, locale string) error {
	if a.c.EmailSender == nil {
		return ordercontract.ErrAuthorizationUnavailable
	}
	brand := mailbrand.Brand{}
	if a.c.EmailBrandResolver != nil {
		var err error
		brand, err = a.c.EmailBrandResolver.ResolveEmailBrand(ctx, mailbrand.Scope{})
		if err != nil {
			return ordercontract.ErrAuthorizationUnavailable
		}
	}
	return a.c.EmailSender.SendVerifyCode(email, code, "order_authorization", locale, brand)
}
