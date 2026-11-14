package diagnosticsbootstrap

import (
	diagnosticsapplication "github.com/dujiao-next/internal/modules/diagnostics/application"
	"github.com/dujiao-next/internal/modules/diagnostics/infrastructure/chatgpt"
	"github.com/dujiao-next/internal/modules/diagnostics/infrastructure/statuspage"
	diagnosticstransport "github.com/dujiao-next/internal/modules/diagnostics/transport/http"
)

func NewHandler() *diagnosticstransport.Handler {
	return diagnosticstransport.NewHandler(chatgpt.NewClient(nil))
}

func NewServiceStatusHandler() *diagnosticstransport.ServiceStatusHandler {
	return diagnosticstransport.NewServiceStatusHandler(diagnosticsapplication.NewServiceStatus(statuspage.NewClient(nil)))
}
