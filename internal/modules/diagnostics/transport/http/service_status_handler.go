package diagnosticshttp

import (
	"errors"
	"net/http"

	"github.com/dujiao-next/internal/i18n"
	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
	"github.com/dujiao-next/internal/platform/http/response"

	"github.com/gin-gonic/gin"
)

type ServiceStatusHandler struct {
	statuses diagnosticscontract.ServiceStatusReader
}

func NewServiceStatusHandler(statuses diagnosticscontract.ServiceStatusReader) *ServiceStatusHandler {
	return &ServiceStatusHandler{statuses: statuses}
}

func RegisterServiceStatusRoutes(group gin.IRoutes, handler *ServiceStatusHandler) {
	group.GET("/service-status/:provider", handler.GetServiceStatus)
}

func (h *ServiceStatusHandler) GetServiceStatus(c *gin.Context) {
	provider := diagnosticscontract.ServiceStatusProvider(c.Param("provider"))
	err := diagnosticscontract.ErrServiceStatusUnavailable
	if !diagnosticscontract.ValidServiceStatusProvider(provider) {
		err = diagnosticscontract.ErrServiceStatusProvider
	} else if h.statuses != nil {
		var snapshot diagnosticscontract.ServiceStatusSnapshot
		snapshot, err = h.statuses.Latest(c.Request.Context(), provider)
		if err == nil {
			response.Success(c, snapshot)
			return
		}
	}
	code, key, errorCode := http.StatusServiceUnavailable, "error.service_status_unavailable", "unavailable"
	if errors.Is(err, diagnosticscontract.ErrServiceStatusProvider) {
		code, key, errorCode = response.CodeBadRequest, "error.service_status_provider_invalid", "invalid_provider"
	}
	c.JSON(http.StatusOK, response.Response{StatusCode: code, Msg: i18n.T(i18n.ResolveLocale(c), key), Data: gin.H{"error_code": errorCode}})
}
