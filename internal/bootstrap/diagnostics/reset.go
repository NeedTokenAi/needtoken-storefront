package diagnosticsbootstrap

import (
	"time"

	diagnosticsapplication "github.com/dujiao-next/internal/modules/diagnostics/application"
	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
	"github.com/dujiao-next/internal/modules/diagnostics/infrastructure/codexresets"
	"github.com/dujiao-next/internal/modules/diagnostics/infrastructure/resetsources"
	diagnosticstransport "github.com/dujiao-next/internal/modules/diagnostics/transport/http"
)

func NewResetFeed() diagnosticscontract.ChannelResetFeedReader {
	tsalon := resetsources.NewTSalonClient(nil)
	return diagnosticsapplication.NewMultiChannelResetFeed(map[string]diagnosticscontract.ResetFeedReader{
		"codex":  diagnosticsapplication.NewResetFeedWithOptions(codexresets.NewClient(nil), "https://codex-resets.com/", 5*time.Minute),
		"claude": diagnosticsapplication.NewResetFeedWithOptions(tsalon.ChannelSource("claude"), resetsources.TSalonSourceURL, 5*time.Minute),
		"gemini": diagnosticsapplication.NewResetFeedWithOptions(resetsources.NewAntigravityClient(nil), resetsources.AntigravitySourceURL, 10*time.Minute),
		"grok":   diagnosticsapplication.NewResetFeedWithOptions(tsalon.ChannelSource("grok"), resetsources.TSalonSourceURL, 5*time.Minute),
	})
}

func NewResetHandler() *diagnosticstransport.ResetHandler {
	return diagnosticstransport.NewResetHandler(NewResetFeed())
}

func NewResetHandlerWithFeed(feed diagnosticscontract.ResetFeedReader) *diagnosticstransport.ResetHandler {
	if feed == nil {
		return NewResetHandler()
	}
	return diagnosticstransport.NewResetHandler(feed)
}
