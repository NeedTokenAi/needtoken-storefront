package contract

import "errors"

var (
	ErrMappingNotFound           = errors.New("product mapping not found")
	ErrMappingAlreadyExists      = errors.New("product mapping already exists for this upstream product")
	ErrUpstreamProductNotFound   = errors.New("upstream product not found")
	ErrMappingInactive           = errors.New("product mapping is inactive")
	ErrMediaRecorderRequired     = errors.New("product mapping media recorder is required")
	ErrUpstreamStockInsufficient = errors.New("upstream stock insufficient")
	ErrBatchImportInProgress     = errors.New("batch import already running for this connection")
)

var (
	ErrPlanAccountLinkRequired    = errors.New("plan API account link required")
	ErrPlanAccountUnavailable     = errors.New("plan API account unavailable")
	ErrPlanUnavailable            = errors.New("API plan unavailable")
	ErrPlanChanged                = errors.New("API plan changed")
	ErrPlanPurchaseLimitReached   = errors.New("API plan purchase limit reached")
	ErrPlanTokenLimitReached      = errors.New("API account token limit reached")
	ErrPlanEligibilityUnavailable = errors.New("API plan eligibility unavailable")
)
