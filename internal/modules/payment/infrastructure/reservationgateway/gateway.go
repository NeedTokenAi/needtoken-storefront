package reservationgateway

import (
	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
	"github.com/dujiao-next/internal/upstream"
)

type Provider interface {
	GetByID(uint) (*siteconnectiondomain.Connection, error)
	GetAdapter(*siteconnectiondomain.Connection) (upstream.Adapter, error)
}

type Gateway struct{ provider Provider }

func New(p Provider) *Gateway { return &Gateway{provider: p} }

func (g *Gateway) OpenReservation(id uint) (paymentcontract.ReservationConnection, error) {
	if g == nil || g.provider == nil {
		return nil, paymentcontract.ErrReservationUnavailable
	}
	conn, err := g.provider.GetByID(id)
	if err != nil || conn == nil {
		return nil, paymentcontract.ErrReservationUnavailable
	}
	if conn.Status != "active" {
		return nil, paymentcontract.ErrReservationUnavailable
	}
	a, err := g.provider.GetAdapter(conn)
	if err != nil {
		return nil, paymentcontract.ErrReservationUnavailable
	}
	capability, ok := a.(paymentcontract.ReservationConnection)
	if !ok {
		return nil, paymentcontract.ErrReservationUnavailable
	}
	return capability, nil
}

var _ paymentcontract.ReservationProvider = (*Gateway)(nil)
