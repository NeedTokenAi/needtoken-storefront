package opsbridge

import (
	"fmt"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	externalidentitydomain "github.com/dujiao-next/internal/modules/identity/externalidentity/domain"
	externalidentitystore "github.com/dujiao-next/internal/modules/identity/externalidentity/infrastructure/gormstore"
	userdomain "github.com/dujiao-next/internal/modules/identity/user/domain"
	userstore "github.com/dujiao-next/internal/modules/identity/user/infrastructure/gormstore"
	opsapp "github.com/dujiao-next/internal/modules/opsbridge/application"
	opscontract "github.com/dujiao-next/internal/modules/opsbridge/contract"
	walletapp "github.com/dujiao-next/internal/modules/wallet/application"
	walletdomain "github.com/dujiao-next/internal/modules/wallet/domain"
	walletgormstore "github.com/dujiao-next/internal/modules/wallet/infrastructure/gormstore"
	"github.com/glebarez/sqlite"
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
