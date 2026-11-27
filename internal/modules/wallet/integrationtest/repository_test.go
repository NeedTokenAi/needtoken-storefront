package integrationtest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	userdomain "github.com/dujiao-next/internal/modules/identity/user/domain"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"

	"github.com/dujiao-next/internal/constants"
	walletapp "github.com/dujiao-next/internal/modules/wallet/application"
	walletcontract "github.com/dujiao-next/internal/modules/wallet/contract"
	walletdomain "github.com/dujiao-next/internal/modules/wallet/domain"
	walletgormstore "github.com/dujiao-next/internal/modules/wallet/infrastructure/gormstore"
	wallethttp "github.com/dujiao-next/internal/modules/wallet/transport/http"
	"github.com/dujiao-next/internal/shared/money"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func TestAdminAdjustmentPersistsOperatorIdentity(t *testing.T) {
	repo, db := setupWalletRepositoryTest(t)
	service := walletapp.NewService(walletapp.Options{Repository: repo, Transactions: repo})
	const (
		userID  = uint(7001)
		adminID = uint(42)
	)

	_, transaction, err := service.AdminAdjustBalance(walletcontract.AdjustBalanceInput{
		UserID:          userID,
		OperatorAdminID: adminID,
		Delta:           money.FromDecimal(decimal.NewFromInt(25)),
		Currency:        "CNY",
		Remark:          "manual reconciliation",
	})
	if err != nil {
		t.Fatalf("AdminAdjustBalance failed: %v", err)
	}
	if transaction.OperatorAdminID == nil || *transaction.OperatorAdminID != adminID {
		t.Fatalf("returned operator_admin_id = %v, want %d", transaction.OperatorAdminID, adminID)
	}

	var stored walletdomain.Transaction
	if err := db.First(&stored, transaction.ID).Error; err != nil {
		t.Fatalf("reload wallet transaction: %v", err)
	}
	if stored.OperatorAdminID == nil || *stored.OperatorAdminID != adminID {
		t.Fatalf("stored operator_admin_id = %v, want %d", stored.OperatorAdminID, adminID)
	}
}

func setupWalletRepositoryTest(t *testing.T) (*walletgormstore.Store, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:wallet_repo_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&userdomain.User{},
		&walletdomain.Account{},
		&walletdomain.Transaction{},
		&walletdomain.RechargeOrder{},
		&walletdomain.UnifiedOp{},
		&paymentdomain.Payment{},
	); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
	return walletgormstore.New(db), db
}

func TestWalletRepositoryListRechargeOrdersAdmin(t *testing.T) {
	repo, db := setupWalletRepositoryTest(t)
	now := time.Now().UTC().Truncate(time.Second)

	user1 := userdomain.User{
		Email:        "alpha_wallet_repo@example.com",
		DisplayName:  "Alpha",
		PasswordHash: "hash",
		Status:       constants.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	user2 := userdomain.User{
		Email:        "beta_wallet_repo@example.com",
		DisplayName:  "Beta",
		PasswordHash: "hash",
		Status:       constants.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := db.Create(&user1).Error; err != nil {
		t.Fatalf("create user1 failed: %v", err)
	}
	if err := db.Create(&user2).Error; err != nil {
		t.Fatalf("create user2 failed: %v", err)
	}

	paidAt1 := now.Add(-2 * time.Hour)
	paidAt3 := now.Add(-30 * time.Minute)
	orders := []walletdomain.RechargeOrder{
		{
			RechargeNo:      "DJR-A001",
			UserID:          user1.ID,
			PaymentID:       1001,
			ChannelID:       11,
			ProviderType:    constants.PaymentProviderOfficial,
			ChannelType:     constants.PaymentChannelTypeAlipay,
			InteractionMode: constants.PaymentInteractionRedirect,
			Amount:          money.FromDecimal(decimal.RequireFromString("50.00")),
			PayableAmount:   money.FromDecimal(decimal.RequireFromString("50.00")),
			FeeRate:         money.FromDecimal(decimal.Zero),
			FeeAmount:       money.FromDecimal(decimal.Zero),
			Currency:        "CNY",
			Status:          constants.WalletRechargeStatusSuccess,
			PaidAt:          &paidAt1,
			CreatedAt:       now.Add(-3 * time.Hour),
			UpdatedAt:       now.Add(-2 * time.Hour),
		},
		{
			RechargeNo:      "DJR-A002",
			UserID:          user1.ID,
			PaymentID:       1002,
			ChannelID:       11,
			ProviderType:    constants.PaymentProviderOfficial,
			ChannelType:     constants.PaymentChannelTypeWechat,
			InteractionMode: constants.PaymentInteractionQR,
			Amount:          money.FromDecimal(decimal.RequireFromString("80.00")),
			PayableAmount:   money.FromDecimal(decimal.RequireFromString("80.00")),
			FeeRate:         money.FromDecimal(decimal.Zero),
			FeeAmount:       money.FromDecimal(decimal.Zero),
			Currency:        "CNY",
			Status:          constants.WalletRechargeStatusPending,
			CreatedAt:       now.Add(-20 * time.Minute),
			UpdatedAt:       now.Add(-20 * time.Minute),
		},
		{
			RechargeNo:      "DJR-B001",
			UserID:          user2.ID,
			PaymentID:       2001,
			ChannelID:       12,
			ProviderType:    constants.PaymentProviderOfficial,
			ChannelType:     constants.PaymentChannelTypeAlipay,
			InteractionMode: constants.PaymentInteractionRedirect,
			Amount:          money.FromDecimal(decimal.RequireFromString("120.00")),
			PayableAmount:   money.FromDecimal(decimal.RequireFromString("120.00")),
			FeeRate:         money.FromDecimal(decimal.Zero),
			FeeAmount:       money.FromDecimal(decimal.Zero),
			Currency:        "CNY",
			Status:          constants.WalletRechargeStatusSuccess,
			PaidAt:          &paidAt3,
			CreatedAt:       now.Add(-40 * time.Minute),
			UpdatedAt:       now.Add(-30 * time.Minute),
		},
	}
	if err := db.Create(&orders).Error; err != nil {
		t.Fatalf("create recharge orders failed: %v", err)
	}

	t.Run("filter by user keyword", func(t *testing.T) {
		rows, total, err := repo.ListRechargeOrdersAdmin(walletcontract.RechargeListFilter{
			Page:        1,
			PageSize:    20,
			UserKeyword: "alpha_wallet_repo",
		})
		if err != nil {
			t.Fatalf("list by user keyword failed: %v", err)
		}
		if total != 2 {
			t.Fatalf("total want 2 got %d", total)
		}
		if len(rows) != 2 {
			t.Fatalf("rows len want 2 got %d", len(rows))
		}
		for _, row := range rows {
			if row.UserID != user1.ID {
				t.Fatalf("expected only user1 rows, got user_id=%d", row.UserID)
			}
		}
	})

	t.Run("filter by status and paid range", func(t *testing.T) {
		from := now.Add(-3 * time.Hour)
		to := now.Add(-90 * time.Minute)
		rows, total, err := repo.ListRechargeOrdersAdmin(walletcontract.RechargeListFilter{
			Page:      1,
			PageSize:  20,
			Status:    constants.WalletRechargeStatusSuccess,
			PaidFrom:  &from,
			PaidTo:    &to,
			ChannelID: 11,
		})
		if err != nil {
			t.Fatalf("list by status/paid range failed: %v", err)
		}
		if total != 1 {
			t.Fatalf("total want 1 got %d", total)
		}
		if len(rows) != 1 {
			t.Fatalf("rows len want 1 got %d", len(rows))
		}
		if rows[0].RechargeNo != "DJR-A001" {
			t.Fatalf("unexpected recharge_no=%s", rows[0].RechargeNo)
		}
	})
}

func TestWalletStoreExcludesSoftDeletedRecords(t *testing.T) {
	repo, db := setupWalletRepositoryTest(t)
	now := time.Now()
	account := walletdomain.Account{
		UserID:    991,
		Balance:   money.FromDecimal(decimal.NewFromInt(10)),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("create account: %v", err)
	}
	transaction := walletdomain.Transaction{
		UserID:        account.UserID,
		Type:          constants.WalletTxnTypeRecharge,
		Direction:     constants.WalletTxnDirectionIn,
		Amount:        money.FromDecimal(decimal.NewFromInt(10)),
		BalanceBefore: money.FromDecimal(decimal.Zero),
		BalanceAfter:  money.FromDecimal(decimal.NewFromInt(10)),
		Currency:      "CNY",
		Reference:     "soft-delete-proof",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := db.Create(&transaction).Error; err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	recharge := walletdomain.RechargeOrder{
		RechargeNo:      "SOFT-DELETE-RECHARGE",
		UserID:          account.UserID,
		PaymentID:       992,
		ChannelID:       1,
		ProviderType:    constants.PaymentProviderOfficial,
		ChannelType:     constants.PaymentChannelTypeAlipay,
		InteractionMode: constants.PaymentInteractionRedirect,
		Amount:          money.FromDecimal(decimal.NewFromInt(10)),
		PayableAmount:   money.FromDecimal(decimal.NewFromInt(10)),
		FeeRate:         money.FromDecimal(decimal.Zero),
		FeeAmount:       money.FromDecimal(decimal.Zero),
		Currency:        "CNY",
		Status:          constants.WalletRechargeStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(&recharge).Error; err != nil {
		t.Fatalf("create recharge: %v", err)
	}
	deletedAt := now.Add(time.Second)
	if err := db.Model(&walletdomain.Account{}).Where("id = ?", account.ID).Update("deleted_at", deletedAt).Error; err != nil {
		t.Fatalf("soft delete account: %v", err)
	}
	if err := db.Model(&walletdomain.Transaction{}).Where("id = ?", transaction.ID).Update("deleted_at", deletedAt).Error; err != nil {
		t.Fatalf("soft delete transaction: %v", err)
	}
	if err := db.Model(&walletdomain.RechargeOrder{}).Where("id = ?", recharge.ID).Update("deleted_at", deletedAt).Error; err != nil {
		t.Fatalf("soft delete recharge: %v", err)
	}

	foundAccount, err := repo.GetAccountByUserID(account.UserID)
	if err != nil {
		t.Fatalf("get deleted account: %v", err)
	}
	if foundAccount != nil {
		t.Fatalf("soft-deleted account leaked: %+v", foundAccount)
	}
	foundTransaction, err := repo.GetTransactionByReference(transaction.Reference)
	if err != nil {
		t.Fatalf("get deleted transaction: %v", err)
	}
	if foundTransaction != nil {
		t.Fatalf("soft-deleted transaction leaked: %+v", foundTransaction)
	}
	foundRecharge, err := repo.GetRechargeOrderByPaymentID(recharge.PaymentID)
	if err != nil {
		t.Fatalf("get deleted recharge: %v", err)
	}
	if foundRecharge != nil {
		t.Fatalf("soft-deleted recharge leaked: %+v", foundRecharge)
	}
}

type rechargeSyncUnifiedStub struct {
	creditErr error
	balance   money.Amount
}

func (s *rechargeSyncUnifiedStub) Enabled() bool        { return true }
func (s *rechargeSyncUnifiedStub) SiteCurrency() string { return "CNY" }
func (s *rechargeSyncUnifiedStub) FetchBalance(_ context.Context, _ uint, _ string) (money.Amount, bool, error) {
	return s.balance, true, nil
}
func (s *rechargeSyncUnifiedStub) Credit(_ context.Context, _ uint, amount money.Amount, _, _, _ string) (money.Amount, error) {
	if s.creditErr != nil {
		return money.Amount{}, s.creditErr
	}
	s.balance = money.FromDecimal(s.balance.Decimal.Add(amount.Decimal))
	return s.balance, nil
}
func (s *rechargeSyncUnifiedStub) Debit(_ context.Context, _ uint, _ money.Amount, _ string, _ bool, _, _ string) (money.Amount, money.Amount, bool, error) {
	return money.Amount{}, money.Amount{}, false, nil
}
func (s *rechargeSyncUnifiedStub) Redeem(_ context.Context, _ uint, _, _ string) (*walletcontract.UnifiedBalanceRedeemResult, error) {
	return nil, nil
}

type rechargeSyncPaymentReader struct {
	payment *paymentdomain.Payment
}

func (p *rechargeSyncPaymentReader) GetAvailableWalletRechargeChannels(_ money.Amount, _ *userdomain.User) ([]map[string]interface{}, error) {
	return nil, nil
}
func (p *rechargeSyncPaymentReader) CreateWalletRechargePayment(_ wallethttp.CreateRechargePaymentInput) (*wallethttp.CreateRechargePaymentResult, error) {
	return nil, nil
}
func (p *rechargeSyncPaymentReader) GetPayment(_ uint) (*paymentdomain.Payment, error) {
	return p.payment, nil
}
func (p *rechargeSyncPaymentReader) CapturePayment(_ wallethttp.CapturePaymentInput) (*paymentdomain.Payment, error) {
	return p.payment, nil
}

type rechargeSyncUserReader struct{}

func (u *rechargeSyncUserReader) GetByID(id uint) (*userdomain.User, error) {
	return &userdomain.User{ID: id}, nil
}

type rechargeSyncWalletWrapper struct {
	*walletapp.Service
}

func (w *rechargeSyncWalletWrapper) ListTransactions(userID uint, page, pageSize int) ([]walletdomain.Transaction, int64, error) {
	return w.Service.ListTransactions(walletcontract.TransactionListFilter{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
	})
}

func TestUserGetRechargeExposesUnifiedBalanceSyncState(t *testing.T) {
	repo, db := setupWalletRepositoryTest(t)
	now := time.Now().UTC()
	user := userdomain.User{
		Email: "sync_recharge@example.com", DisplayName: "Sync", PasswordHash: "h",
		Status: constants.UserStatusActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	payment := paymentdomain.Payment{
		ID: 551, ChannelID: 1, ProviderType: constants.PaymentProviderOfficial,
		ChannelType: constants.PaymentChannelTypeAlipay, InteractionMode: constants.PaymentInteractionQR,
		Amount: money.FromDecimal(decimal.NewFromInt(20)), Currency: "CNY", Status: constants.PaymentStatusSuccess,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&payment).Error; err != nil {
		t.Fatal(err)
	}
	recharge := walletdomain.RechargeOrder{
		ID: 881, RechargeNo: "RCH-SYNC-881", UserID: user.ID, PaymentID: payment.ID,
		ChannelID: 1, ProviderType: constants.PaymentProviderOfficial, ChannelType: constants.PaymentChannelTypeAlipay,
		InteractionMode: constants.PaymentInteractionQR,
		Amount:          money.FromDecimal(decimal.NewFromInt(20)), PayableAmount: money.FromDecimal(decimal.NewFromInt(20)),
		FeeRate: money.FromDecimal(decimal.Zero), FeeAmount: money.FromDecimal(decimal.Zero),
		Currency: "CNY", Status: constants.WalletRechargeStatusSuccess, PaidAt: &now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&recharge).Error; err != nil {
		t.Fatal(err)
	}

	bridge := &rechargeSyncUnifiedStub{creditErr: fmt.Errorf("gateway 503")}
	svc := walletapp.NewService(walletapp.Options{Repository: repo, Transactions: repo, Unified: bridge})

	opID := fmt.Sprintf("recharge:%d:success", recharge.ID)
	var op *walletdomain.UnifiedOp
	if err := repo.WithinTransaction(func(tx walletcontract.Transaction) error {
		var enqErr error
		op, enqErr = svc.EnqueueUnifiedCreditInTx(tx, user.ID, &recharge.ID, recharge.Amount, "CNY", opID, "recharge")
		return enqErr
	}); err != nil {
		t.Fatal(err)
	}
	svc.FlushUnifiedCreditOp(op)

	// R11 check: unresolved unified ops list includes the errored credit_pending op.
	unresolved, err := svc.ListUnresolvedUnifiedOps(20)
	if err != nil || len(unresolved) != 1 || unresolved[0].OpID != opID {
		t.Fatalf("expected 1 unresolved unified op %q, got %+v err=%v", opID, unresolved, err)
	}

	gin.SetMode(gin.TestMode)
	handler := wallethttp.NewUserHandler(&rechargeSyncWalletWrapper{Service: svc}, &rechargeSyncPaymentReader{payment: &payment}, &rechargeSyncUserReader{})
	callDetail := func() map[string]any {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Set("user_id", user.ID)
		ctx.Params = gin.Params{{Key: "recharge_no", Value: recharge.RechargeNo}}
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/user/wallet/recharges/"+recharge.RechargeNo, nil)
		handler.GetRecharge(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected HTTP %d: %s", rec.Code, rec.Body.String())
		}
		var out map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out["data"].(map[string]any)
	}

	data1 := callDetail()
	bs1, ok := data1["balance_sync"].(map[string]any)
	if !ok || bs1["status"] != "sync_error" || bs1["last_error"] != "gateway 503" {
		t.Fatalf("expected balance_sync sync_error with last_error, got %+v", data1["balance_sync"])
	}

	// Recover after gateway resumes and clear backoff to simulate immediate retry.
	bridge.creditErr = nil
	if err := db.Model(&walletdomain.UnifiedOp{}).Where("op_id = ?", opID).Update("next_retry_at", nil).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.RecoverUnifiedOps(); err != nil {
		t.Fatal(err)
	}

	data2 := callDetail()
	bs2, ok := data2["balance_sync"].(map[string]any)
	if !ok || bs2["status"] != "credited" {
		t.Fatalf("expected balance_sync credited after recovery, got %+v", data2["balance_sync"])
	}
}
