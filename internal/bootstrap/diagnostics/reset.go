package diagnosticsbootstrap

import (
	"time"

	diagnosticsapplication "github.com/dujiao-next/internal/modules/diagnostics/application"
	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
	"github.com/dujiao-next/internal/modules/diagnostics/infrastructure/codexresets"
	"github.com/dujiao-next/internal/modules/diagnostics/infrastructure/resetsources"
	diagnosticstransport "github.com/dujiao-next/internal/modules/diagnostics/transport/http"
)

func NewResetHandler() *diagnosticstransport.ResetHandler {
	tsalon := resetsources.NewTSalonClient(nil)
	feeds := diagnosticsapplication.NewMultiChannelResetFeed(map[string]diagnosticscontract.ResetFeedReader{
		"codex":  diagnosticsapplication.NewResetFeedWithOptions(codexresets.NewClient(nil), "https://codex-resets.com/", 5*time.Minute),
		"claude": diagnosticsapplication.NewResetFeedWithOptions(tsalon.ChannelSource("claude"), resetsources.TSalonSourceURL, 5*time.Minute),
		"gemini": diagnosticsapplication.NewResetFeedWithOptions(resetsources.NewAntigravityClient(nil), resetsources.AntigravitySourceURL, 10*time.Minute),
		"grok":   diagnosticsapplication.NewResetFeedWithOptions(tsalon.ChannelSource("grok"), resetsources.TSalonSourceURL, 5*time.Minute),
	})
	return diagnosticstransport.NewResetHandler(feeds)
}
