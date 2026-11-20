package accountlinkwiring

import (
	"context"
	"errors"
	"testing"

	"github.com/dujiao-next/internal/app/container"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/modules/order/application/accountlink"
	siteconnectionapp "github.com/dujiao-next/internal/modules/siteconnection/application"
	siteconnectioncontract "github.com/dujiao-next/internal/modules/siteconnection/contract"
	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
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
