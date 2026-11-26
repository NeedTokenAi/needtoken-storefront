package opsbridge

import (
	"fmt"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	categorydomain "github.com/dujiao-next/internal/modules/catalog/category/domain"
	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	productgormstore "github.com/dujiao-next/internal/modules/catalog/product/store/gormstore"
	fulfillmentapp "github.com/dujiao-next/internal/modules/fulfillment/application"
	fulfillmentdomain "github.com/dujiao-next/internal/modules/fulfillment/domain"
	fulfillmentstore "github.com/dujiao-next/internal/modules/fulfillment/infrastructure/gormstore"
	externalidentitydomain "github.com/dujiao-next/internal/modules/identity/externalidentity/domain"
	externalidentitystore "github.com/dujiao-next/internal/modules/identity/externalidentity/infrastructure/gormstore"
	userdomain "github.com/dujiao-next/internal/modules/identity/user/domain"
	userstore "github.com/dujiao-next/internal/modules/identity/user/infrastructure/gormstore"
	opsapp "github.com/dujiao-next/internal/modules/opsbridge/application"
	opscontract "github.com/dujiao-next/internal/modules/opsbridge/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	orderstore "github.com/dujiao-next/internal/modules/order/infrastructure/gormstore"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"
	paymentstore "github.com/dujiao-next/internal/modules/payment/infrastructure/gormstore"
	procurementapp "github.com/dujiao-next/internal/modules/procurement/application"
	procurementdomain "github.com/dujiao-next/internal/modules/procurement/domain"
	procurementstore "github.com/dujiao-next/internal/modules/procurement/infrastructure/gormstore"
	radarsubscriptiondomain "github.com/dujiao-next/internal/modules/radarsubscription/domain"
	radarsubscriptiongormstore "github.com/dujiao-next/internal/modules/radarsubscription/infrastructure/gormstore"
	walletapp "github.com/dujiao-next/internal/modules/wallet/application"
	walletdomain "github.com/dujiao-next/internal/modules/wallet/domain"
	walletgormstore "github.com/dujiao-next/internal/modules/wallet/infrastructure/gormstore"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func TestOpsBridgeUsersLifecycleAndWalletAdjust(t *testing.T) {
	dsn := fmt.Sprintf("file:opsbridge_users_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&userdomain.User{},
		&externalidentitydomain.Identity{},
		&walletdomain.Account{},
		&walletdomain.Transaction{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	uStore := userstore.New(db)
	oStore := externalidentitystore.New(db)
	wStore := walletgormstore.New(db)
	wSvc := walletapp.NewService(walletapp.Options{
		Repository:   wStore,
		Transactions: wStore,
	})

	svc := opsapp.New(nil, nil, nil).WithUsers(&userStoreAdapter{
		users:   uStore,
		wallets: wSvc,
		oauth:   oStore,
	})

	// 1. Create user
	created, err := svc.CreateUser(opscontract.CreateUserInput{
		Email:       "Unified@Example.com",
		Password:    "StrongPass123!",
		DisplayName: "Unified Alice",
	})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if created.Email != "unified@example.com" || created.Status != constants.UserStatusActive || !created.EmailVerified {
		t.Fatalf("unexpected created user: %+v", created)
	}

	// Duplicate email rejected
	if _, err := svc.CreateUser(opscontract.CreateUserInput{
		Email:    "unified@example.com",
		Password: "StrongPass123!",
	}); err == nil {
		t.Fatalf("expected duplicate email error")
	}

	// 2. Wallet credit & debit
	adj, err := svc.AdjustUserWallet(created.ID, opscontract.AdjustUserWalletInput{
		Operation: "credit",
		Amount:    "50.00",
		Remark:    "ops test credit",
	})
	if err != nil || adj.Balance != "50.00" {
		t.Fatalf("credit failed: adj=%+v err=%v", adj, err)
	}
	adj2, err := svc.AdjustUserWallet(created.ID, opscontract.AdjustUserWalletInput{
		Operation: "debit",
		Amount:    "12.50",
		Remark:    "ops test debit",
	})
	if err != nil || adj2.Balance != "37.50" {
		t.Fatalf("debit failed: adj=%+v err=%v", adj2, err)
	}

	// 3. List users includes wallet balance
	listed, err := svc.Users(1, 20, "unified", "")
	if err != nil || listed.Total != 1 || len(listed.Users) != 1 || listed.Users[0].WalletBalance != "37.50" {
		t.Fatalf("unexpected list result: %+v err=%v", listed, err)
	}

	// 4. Update email, password, status bumps TokenVersion
	newEmail := "renamed@example.com"
	newPass := "NextSecret456!"
	disabled := constants.UserStatusDisabled
	updated, err := svc.UpdateUser(created.ID, opscontract.UpdateUserInput{
		Email:    &newEmail,
		Password: &newPass,
		Status:   &disabled,
	})
	if err != nil || updated.Email != newEmail || updated.Status != disabled {
		t.Fatalf("UpdateUser failed: %+v err=%v", updated, err)
	}
	stored, err := uStore.GetByID(created.ID)
	if err != nil || stored == nil || stored.TokenVersion == 0 {
		t.Fatalf("expected TokenVersion bumped after security update, got %+v err=%v", stored, err)
	}

	// 5. Clear TOTP
	if err := svc.ClearUserTOTP(created.ID); err != nil {
		t.Fatalf("ClearUserTOTP failed: %v", err)
	}

	// 6. SoftDelete archives email & hides user from queries
	if err := svc.DeleteUser(created.ID); err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}
	afterDel, err := uStore.GetByID(created.ID)
	if err != nil || afterDel != nil {
		t.Fatalf("expected user hidden after soft delete, got %+v err=%v", afterDel, err)
	}
	// Original email can be recreated after soft delete
	if _, err := svc.CreateUser(opscontract.CreateUserInput{
		Email:    newEmail,
		Password: "FreshUser789!",
	}); err != nil {
		t.Fatalf("expected recreated user with released email to succeed: %v", err)
	}
}

func TestOpsBridgeOrdersFilteredAndActions(t *testing.T) {
	dsn := fmt.Sprintf("file:opsbridge_orders_d_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&userdomain.User{},
		&categorydomain.Category{},
		&productdomain.Product{},
		&orderdomain.Order{},
		&orderdomain.OrderItem{},
		&orderdomain.OrderRefundRecord{},
		&paymentdomain.Payment{},
		&paymentdomain.CheckoutReservation{},
		&fulfillmentdomain.Fulfillment{},
		&procurementdomain.Order{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	member := userdomain.User{
		Email:       "buyer@example.com",
		DisplayName: "Buyer One",
		Status:      constants.UserStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := db.Create(&member).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	cat := categorydomain.Category{
		Slug:     "api-plans",
		NameJSON: jsonmap.JSON{"zh-CN": "API 套餐"},
		IsActive: true,
	}
	if err := db.Create(&cat).Error; err != nil {
		t.Fatalf("seed category: %v", err)
	}
	prod := productdomain.Product{
		CategoryID:      cat.ID,
		Slug:            "pro-plan",
		TitleJSON:       jsonmap.JSON{"zh-CN": "Pro 月卡"},
		FulfillmentType: constants.FulfillmentTypeManual,
		PriceAmount:     money.FromDecimal(decimal.NewFromInt(99)),
		IsActive:        true,
	}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatalf("seed product: %v", err)
	}

	paidAt := now
	memberOrder := orderdomain.Order{
		OrderNo:           "DJ-MEM-001",
		UserID:            member.ID,
		Status:            constants.OrderStatusPaid,
		Currency:          "CNY",
		OriginalAmount:    money.FromDecimal(decimal.NewFromInt(99)),
		TotalAmount:       money.FromDecimal(decimal.NewFromInt(99)),
		OnlinePaidAmount:  money.FromDecimal(decimal.NewFromInt(99)),
		ClientIP:          "203.0.113.5",
		UserAgent:         "Mozilla/5.0 Chrome/130.0",
		ClientFingerprint: "fp_member_0001_abcdef012345",
		PaidAt:            &paidAt,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := db.Create(&memberOrder).Error; err != nil {
		t.Fatalf("seed member order: %v", err)
	}
	if err := db.Create(&orderdomain.OrderItem{
		OrderID:         memberOrder.ID,
		ProductID:       prod.ID,
		TitleJSON:       jsonmap.JSON{"zh-CN": "Pro 月卡"},
		SKUSnapshotJSON: jsonmap.JSON{"sku_code": "PRO-30D"},
		Quantity:        1,
		UnitPrice:       money.FromDecimal(decimal.NewFromInt(99)),
		TotalPrice:      money.FromDecimal(decimal.NewFromInt(99)),
		FulfillmentType: constants.FulfillmentTypeManual,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatalf("seed member item: %v", err)
	}
	if err := db.Create(&paymentdomain.Payment{
		OrderID:         memberOrder.ID,
		ChannelID:       1,
		ProviderType:    "stripe",
		ChannelType:     "card",
		InteractionMode: "redirect",
		Amount:          money.FromDecimal(decimal.NewFromInt(80)), // 实付 80 vs 下单 99 -> amount_mismatch
		Currency:        "CNY",
		Status:          constants.PaymentStatusSuccess,
		ExceptionCode:   "underpaid_review",
		ProviderRef:     "pi_12345",
		GatewayOrderNo:  "gw_999",
		ProviderPayload: jsonmap.JSON{
			"event_id":  "evt_safe_01",
			"signature": "TOP_SECRET_SIG",
			"api_token": "TOP_SECRET_TOK",
		},
		PaidAt:     &paidAt,
		CallbackAt: &paidAt,
		CreatedAt:  now,
		UpdatedAt:  now,
	}).Error; err != nil {
		t.Fatalf("seed payment: %v", err)
	}
	procOrder := procurementdomain.Order{
		ConnectionID:    1,
		LocalOrderID:    memberOrder.ID,
		LocalOrderNo:    memberOrder.OrderNo,
		Status:          constants.ProcurementStatusFailed,
		ErrorMessage:    "upstream timeout",
		Currency:        "CNY",
		LocalSellAmount: money.FromDecimal(decimal.NewFromInt(99)),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(&procOrder).Error; err != nil {
		t.Fatalf("seed procurement: %v", err)
	}

	guestOrder := orderdomain.Order{
		OrderNo:           "DJ-GUEST-002",
		UserID:            0,
		GuestEmail:        "guest@example.com",
		Status:            constants.OrderStatusPendingPayment,
		Currency:          "CNY",
		TotalAmount:       money.FromDecimal(decimal.NewFromInt(20)),
		OnlinePaidAmount:  money.FromDecimal(decimal.NewFromInt(20)),
		ClientIP:          "198.51.100.9",
		UserAgent:         "Safari/18.0",
		ClientFingerprint: "fp_guest_0002_abcdef012345",
		CreatedAt:         now.Add(-48 * time.Hour),
		UpdatedAt:         now.Add(-48 * time.Hour),
	}
	if err := db.Create(&guestOrder).Error; err != nil {
		t.Fatalf("seed guest order: %v", err)
	}

	oStore := orderstore.New(db, "sqlite")
	pStore := paymentstore.New(db, "sqlite")
	procStore := procurementstore.New(db)
	prodStore := productgormstore.NewProductStore(db)
	uStore := userstore.New(db)
	fStore := fulfillmentstore.New(db)
	fSvc := fulfillmentapp.New(fulfillmentapp.Options{
		OrderStore:       oStore,
		FulfillmentStore: fStore,
	})
	procSvc := procurementapp.NewService(procurementapp.Options{
		Repository: procStore,
	})

	svc := opsapp.NewWithCatalog(oStore, pStore, procStore, nil, prodStore, nil, nil).
		WithOrderEnricher(&orderEnricherAdapter{users: uStore}).
		WithOrderActions(&orderActionStoreAdapter{
			orders:       oStore,
			payments:     pStore,
			fulfillments: fSvc,
			procurements: procSvc,
		})

	// 1. Filter by category + buyer_type + exception_only + payment_channel
	res, err := svc.OrdersFiltered(opscontract.OrdersFilter{
		Page:           1,
		PageSize:       20,
		CategoryID:     cat.ID,
		BuyerType:      "member",
		PaymentChannel: "stripe",
		ExceptionOnly:  true,
	})
	if err != nil || res.Total != 1 || len(res.Orders) != 1 {
		t.Fatalf("expected 1 filtered exception member order, got res=%+v err=%v", res, err)
	}
	got := res.Orders[0]
	if got.BuyerEmail != "buyer@example.com" || got.UserAgent != "Mozilla/5.0 Chrome/130.0" || got.ClientFingerprint != "fp_member_0001_abcdef012345" {
		t.Fatalf("buyer/device metadata missing: %+v", got)
	}
	if !got.HasException || len(got.ExceptionReasons) < 3 {
		t.Fatalf("expected amount_mismatch + payment_exception + procurement_failed, got %+v", got.ExceptionReasons)
	}
	if len(got.Payments) != 1 || got.Payments[0].CallbackSummary["event_id"] != "evt_safe_01" {
		t.Fatalf("callback summary missing safe key: %+v", got.Payments)
	}
	if _, leaked := got.Payments[0].CallbackSummary["signature"]; leaked {
		t.Fatalf("sensitive signature key leaked in callback summary")
	}
	if _, leaked := got.Payments[0].CallbackSummary["api_token"]; leaked {
		t.Fatalf("sensitive api_token key leaked in callback summary")
	}

	// 2. Keyword filter matches fingerprint for guest order
	guestRes, err := svc.OrdersFiltered(opscontract.OrdersFilter{
		Keyword: "fp_guest_0002",
	})
	if err != nil || guestRes.Total != 1 || guestRes.Orders[0].OrderNo != "DJ-GUEST-002" || guestRes.Orders[0].BuyerType != "guest" || guestRes.Orders[0].BuyerEmail != "guest@example.com" {
		t.Fatalf("unexpected guest filter result: %+v err=%v", guestRes, err)
	}

	// 3. Manual fulfill paid order
	fulRes, err := svc.ManualFulfillOrder("DJ-MEM-001", opscontract.ManualFulfillOrderInput{
		Payload: "CDK-MANUAL-DELIVERED-001",
	})
	if err != nil || fulRes.FulfillmentID == 0 || fulRes.Status != constants.FulfillmentStatusDelivered {
		t.Fatalf("ManualFulfillOrder failed: %+v err=%v", fulRes, err)
	}

	// 4. Retry failed procurement
	retryRes, err := svc.RetryProcurement(procOrder.ID)
	if err != nil || retryRes.Status != constants.ProcurementStatusPending {
		t.Fatalf("RetryProcurement failed: %+v err=%v", retryRes, err)
	}
}

func TestOpsBridgeRadarSubscribersAggregationAndExclusion(t *testing.T) {
	dsn := fmt.Sprintf("file:opsbridge_radar_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&userdomain.User{},
		&radarsubscriptiondomain.Subscription{},
		&radarsubscriptiondomain.EventDelivery{},
		&radarsubscriptiondomain.MemberExclusion{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	uStore := userstore.New(db)
	rStore := radarsubscriptiongormstore.New(db)

	alice := &userdomain.User{Email: "alice@example.com", DisplayName: "Alice", Status: constants.UserStatusActive}
	bob := &userdomain.User{Email: "bob@example.com", DisplayName: "Bob", Status: constants.UserStatusActive}
	if err := uStore.Create(alice); err != nil {
		t.Fatalf("create alice: %v", err)
	}
	if err := uStore.Create(bob); err != nil {
		t.Fatalf("create bob: %v", err)
	}

	if err := rStore.UpsertUserChannels(alice.ID, map[string]bool{
		radarsubscriptiondomain.ChannelCodex:  true,
		radarsubscriptiondomain.ChannelClaude: true,
		radarsubscriptiondomain.ChannelGemini: true,
	}); err != nil {
		t.Fatalf("upsert alice subs: %v", err)
	}
	if err := rStore.UpsertUserChannels(bob.ID, map[string]bool{
		radarsubscriptiondomain.ChannelGrok: true,
	}); err != nil {
		t.Fatalf("upsert bob subs: %v", err)
	}

	now := time.Now().UTC()
	sentAt := now.Add(-10 * time.Minute)
	if err := rStore.UpsertDelivery(&radarsubscriptiondomain.EventDelivery{
		Channel:  radarsubscriptiondomain.ChannelCodex,
		RecordID: "rec-1",
		UserID:   alice.ID,
		Email:    alice.Email,
		Status:   radarsubscriptiondomain.DeliveryStatusSent,
		Attempts: 1,
		SentAt:   &sentAt,
	}); err != nil {
		t.Fatalf("upsert alice delivery: %v", err)
	}
	if err := rStore.UpsertDelivery(&radarsubscriptiondomain.EventDelivery{
		Channel:   radarsubscriptiondomain.ChannelGrok,
		RecordID:  "rec-2",
		UserID:    bob.ID,
		Email:     bob.Email,
		Status:    radarsubscriptiondomain.DeliveryStatusFailed,
		Attempts:  1,
		LastError: "smtp 550 mailbox unavailable",
	}); err != nil {
		t.Fatalf("upsert bob delivery: %v", err)
	}

	svc := opsapp.New(nil, nil, nil).WithRadarSubscriptions(&radarSubscriptionAdapter{
		radar: rStore,
		users: uStore,
	})

	// 1. Exclude Bob
	exRes, err := svc.SetRadarSubscriberExclusion(bob.ID, opscontract.UpdateRadarExclusionInput{
		Excluded: true,
		Reason:   "mailbox bounced",
		Operator: "ops-owner",
	})
	if err != nil {
		t.Fatalf("exclude bob: %v", err)
	}
	if !exRes.Excluded || exRes.Reason != "mailbox bounced" || exRes.UpdatedBy != "ops-owner" {
		t.Fatalf("unexpected exclusion result: %+v", exRes)
	}

	// 2. Query summary & all items
	res, err := svc.RadarSubscribers(opscontract.RadarSubscribersFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("RadarSubscribers: %v", err)
	}
	if res.Summary.SubscribedMembers != 2 || res.Summary.TotalSent != 1 || res.Summary.TotalFailed != 1 ||
		res.Summary.FailedMembers != 1 || res.Summary.ExcludedMembers != 1 ||
		res.Summary.ChannelCounts["codex"] != 1 || res.Summary.ChannelCounts["grok"] != 1 {
		t.Fatalf("unexpected summary: %+v", res.Summary)
	}
	if len(res.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(res.Items))
	}
	// Bob has failure so sorts ahead of Alice among active subscribers.
	if res.Items[0].UserID != bob.ID || !res.Items[0].HasFailure || !res.Items[0].Excluded || res.Items[0].LastError != "smtp 550 mailbox unavailable" {
		t.Fatalf("unexpected first item (bob): %+v", res.Items[0])
	}
	if res.Items[1].UserID != alice.ID || res.Items[1].SubscribedCount != 3 || res.Items[1].SentCount != 1 || res.Items[1].HasFailure {
		t.Fatalf("unexpected second item (alice): %+v", res.Items[1])
	}

	// 3. Filter failed & excluded
	failedOnly, err := svc.RadarSubscribers(opscontract.RadarSubscribersFilter{
		Page:          1,
		PageSize:      20,
		DeliveryState: "failed",
		Excluded:      "excluded",
	})
	if err != nil {
		t.Fatalf("filtered query failed: %v", err)
	}
	if failedOnly.Total != 1 || failedOnly.Items[0].UserID != bob.ID {
		t.Fatalf("unexpected filtered items: %+v", failedOnly)
	}

	// 4. Fetch Bob deliveries
	delivRes, err := svc.RadarSubscriberDeliveries(bob.ID, 20)
	if err != nil {
		t.Fatalf("RadarSubscriberDeliveries: %v", err)
	}
	if delivRes.Total != 1 || delivRes.Items[0].LastError != "smtp 550 mailbox unavailable" {
		t.Fatalf("unexpected deliveries: %+v", delivRes)
	}

	// 5. Re-include Bob
	incRes, err := svc.SetRadarSubscriberExclusion(bob.ID, opscontract.UpdateRadarExclusionInput{
		Excluded: false,
		Reason:   "ignored",
		Operator: "ops-owner",
	})
	if err != nil || incRes.Excluded || incRes.Reason != "" {
		t.Fatalf("unexpected re-include result: %+v err=%v", incRes, err)
	}
}
