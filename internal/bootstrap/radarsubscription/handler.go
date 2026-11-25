package radarsubscriptionbootstrap

import (
	"github.com/dujiao-next/internal/app/container"
	radarsubscriptionhttp "github.com/dujiao-next/internal/modules/radarsubscription/transport/http"
)

func NewHandler(c *container.Container) *radarsubscriptionhttp.Handler {
	return radarsubscriptionhttp.NewHandler(c.RadarSubscriptionService)
}
