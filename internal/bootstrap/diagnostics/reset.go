package diagnosticsbootstrap

import (
	diagnosticsapplication "github.com/dujiao-next/internal/modules/diagnostics/application"
	"github.com/dujiao-next/internal/modules/diagnostics/infrastructure/codexresets"
	diagnosticstransport "github.com/dujiao-next/internal/modules/diagnostics/transport/http"
)

func NewResetHandler() *diagnosticstransport.ResetHandler {
	return diagnosticstransport.NewResetHandler(diagnosticsapplication.NewResetFeed(codexresets.NewClient(nil)))
}
