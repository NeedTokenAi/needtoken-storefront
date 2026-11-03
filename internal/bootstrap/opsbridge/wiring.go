package opsbridge

import (
	"github.com/dujiao-next/internal/app/container"
	opsapp "github.com/dujiao-next/internal/modules/opsbridge/application"
	opstransport "github.com/dujiao-next/internal/modules/opsbridge/transport/http"
)

func NewHandler(c *container.Container) *opstransport.Handler {
	return opstransport.New(opsapp.New(c.OrderStore, c.PaymentStore, c.ProcurementOrderRepo))
}
