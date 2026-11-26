package accountlinkwiring

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dujiao-next/internal/app/container"
	"github.com/dujiao-next/internal/constants"
	usercontract "github.com/dujiao-next/internal/modules/identity/user/contract"
	userdomain "github.com/dujiao-next/internal/modules/identity/user/domain"
	"github.com/dujiao-next/internal/modules/order/application/accountlink"
	settingsapp "github.com/dujiao-next/internal/modules/settings/application"
	settingscontract "github.com/dujiao-next/internal/modules/settings/contract"
	siteconnectionapp "github.com/dujiao-next/internal/modules/siteconnection/application"
	siteconnectioncontract "github.com/dujiao-next/internal/modules/siteconnection/contract"
	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/dujiao-next/internal/telegramidentity"
	"github.com/shopspring/decimal"
)

type stubConnRepo struct {
	items []siteconnectiondomain.Connection
}

func (s *stubConnRepo) GetByID(id uint) (*siteconnectiondomain.Connection, error) {
	for i := range s.items {
		if s.items[i].ID == id {
			c := s.items[i]
			return &c, nil
		}
	}
	return nil, nil
}
func (s *stubConnRepo) GetByApiKey(string) (*siteconnectiondomain.Connection, error) { return nil, nil }
func (s *stubConnRepo) Create(*siteconnectiondomain.Connection) error                { return nil }
func (s *stubConnRepo) Update(*siteconnectiondomain.Connection) error                { return nil }
func (s *stubConnRepo) Delete(uint) error                                            { return nil }
func (s *stubConnRepo) List(siteconnectioncontract.ListFilter) ([]siteconnectiondomain.Connection, int64, error) {
	return s.items, int64(len(s.items)), nil
}
func (s *stubConnRepo) ListActive() ([]siteconnectiondomain.Connection, error) {
	var out []siteconnectiondomain.Connection
	for _, item := range s.items {
		if item.Status == constants.ConnectionStatusActive {
			out = append(out, item)
		}
	}
	return out, nil
}

func TestUpstreamBridgeDisabledWhenNoConnectionsAndFailsClosedWhenAmbiguous(t *testing.T) {
	repo := &stubConnRepo{}
	svc := siteconnectionapp.NewService(repo, "test-secret-key-at-least-32-bytes!", "")
	c := &container.Container{
		SiteConnectionRepo:    repo,
		SiteConnectionService: svc,
	}

	b := upstreamBridge{c: c}
	st, err := b.Status(context.Background(), "needtoken-storefront:user:1")
	if err != nil || st.Enabled || st.Status != "disabled" {
		t.Fatalf("empty connections should disable gracefully: st=%+v err=%v", st, err)
	}

	repo.items = []siteconnectiondomain.Connection{
		{ID: 1, Status: constants.ConnectionStatusActive, Protocol: constants.ConnectionProtocolDujiaoNext},
		{ID: 2, Status: constants.ConnectionStatusActive, Protocol: constants.ConnectionProtocolDujiaoNext},
	}
	if _, err := b.Status(context.Background(), "needtoken-storefront:user:1"); !errors.Is(err, accountlink.ErrUnavailable) {
		t.Fatalf("ambiguous active connections must fail closed, got %v", err)
	}
}

type stubUserStore struct {
	usercontract.Store
	users map[uint]*userdomain.User
}

func (s *stubUserStore) GetByID(id uint) (*userdomain.User, error) {
	return s.users[id], nil
}

func TestLookupMemberRejectsPlaceholderAndUnverifiedEmails(t *testing.T) {
	now := time.Now().UTC()
	reader := memberIdentityReader{
		c: &container.Container{
			UserStore: &stubUserStore{
				users: map[uint]*userdomain.User{
					1: {ID: 1, Email: telegramidentity.BuildPlaceholderEmail("12345"), Status: constants.UserStatusActive, EmailVerifiedAt: &now, DisplayName: "TG User"},
					2: {ID: 2, Email: "unverified@example.com", Status: constants.UserStatusActive, EmailVerifiedAt: nil, DisplayName: "Unverified"},
					3: {ID: 3, Email: "real@example.com", Status: constants.UserStatusActive, EmailVerifiedAt: &now, DisplayName: "Real User"},
				},
			},
		},
	}

	tgMember, err := reader.LookupMember(context.Background(), 1)
	if err != nil || tgMember.EmailVerified {
		t.Fatalf("telegram placeholder email must never be treated as verified: member=%+v err=%v", tgMember, err)
	}
	unverified, err := reader.LookupMember(context.Background(), 2)
	if err != nil || unverified.EmailVerified {
		t.Fatalf("unverified email must have EmailVerified=false: member=%+v err=%v", unverified, err)
	}
	verified, err := reader.LookupMember(context.Background(), 3)
	if err != nil || !verified.EmailVerified || verified.Email != "real@example.com" || !verified.Active {
		t.Fatalf("verified real email should be accepted: member=%+v err=%v", verified, err)
	}
}

type stubSettingStore struct {
	settingscontract.Store
	values map[string]jsonmap.JSON
}

func (s *stubSettingStore) GetByKey(key string) (jsonmap.JSON, bool, error) {
	v, ok := s.values[key]
	return v, ok, nil
}

func TestUnifiedBalanceDirectionalRoundingAndTruncation(t *testing.T) {
	adapter := &UnifiedBalanceAdapter{
		c: &container.Container{
			SettingService: settingsapp.NewService(&stubSettingStore{
				values: map[string]jsonmap.JSON{
					constants.SettingKeySiteConfig: {
						"needtoken_pricing": map[string]interface{}{
							"usd_cny_rate": "6.67",
						},
					},
				},
			}),
		},
	}

	// $100.00 - $0.00072 API call = $99.99928 = 49,999,640 quota; * 6.67 = 666.9951976 -> truncated to 666.99 CNY
	cnyAfter := adapter.quotaToCurrencyAmount(49_999_640, "CNY")
	if got := cnyAfter.Decimal.StringFixed(2); got != "666.99" {
		t.Fatalf("expected truncated 666.99 CNY after $0.00072 call, got %s", got)
	}

	// Credit floors fractional quota, Debit ceils fractional quota (100.00 CNY / 6.67 * 500,000 = 7,496,251.874...)
	amt := money.FromDecimal(decimal.RequireFromString("100.00"))
	creditQ, err := adapter.currencyAmountToCreditQuota(amt, "CNY")
	if err != nil || creditQ != 7_496_251 {
		t.Fatalf("expected credit floor 7496251, got %d err=%v", creditQ, err)
	}
	debitQ, err := adapter.currencyAmountToDebitQuota(amt, "CNY")
	if err != nil || debitQ != 7_496_252 {
		t.Fatalf("expected debit ceil 7496252, got %d err=%v", debitQ, err)
	}
	if debitQ < creditQ {
		t.Fatalf("debit quota %d must never be less than credit quota %d", debitQ, creditQ)
	}
}
