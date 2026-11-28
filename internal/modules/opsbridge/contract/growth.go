package contract

import (
	"time"

	settingscontract "github.com/dujiao-next/internal/modules/settings/contract"
)

type AffiliateSettingDTO struct {
	Enabled           bool     `json:"enabled"`
	CommissionRate    float64  `json:"commission_rate"`
	ConfirmDays       int      `json:"confirm_days"`
	MinWithdrawAmount float64  `json:"min_withdraw_amount"`
	WithdrawChannels  []string `json:"withdraw_channels"`
}

type AffiliateStatsSummary struct {
	ProfileCount                  int64            `json:"profile_count"`
	PendingWithdrawCount          int64            `json:"pending_withdraw_count"`
	PendingCommissionByCurrency   map[string]int64 `json:"pending_commission_by_currency"`
	AvailableCommissionByCurrency map[string]int64 `json:"available_commission_by_currency"`
	WithdrawnCommissionByCurrency map[string]int64 `json:"withdrawn_commission_by_currency"`
	EffectiveCostByCurrency       map[string]int64 `json:"effective_cost_by_currency"`
}

type AffiliateWithdrawItem struct {
	ID                 uint       `json:"id"`
	AffiliateProfileID uint       `json:"affiliate_profile_id"`
	UserID             uint       `json:"user_id,omitempty"`
	UserEmail          string     `json:"user_email,omitempty"`
	AffiliateCode      string     `json:"affiliate_code,omitempty"`
	Amount             string     `json:"amount"`
	Currency           string     `json:"currency"`
	Channel            string     `json:"channel"`
	Account            string     `json:"account"`
	Status             string     `json:"status"`
	RejectReason       string     `json:"reject_reason,omitempty"`
	ProcessedAt        *time.Time `json:"processed_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

type AffiliateWithdrawsResult struct {
	Source      string                  `json:"source"`
	CollectedAt time.Time               `json:"collected_at"`
	Items       []AffiliateWithdrawItem `json:"items"`
	Total       int64                   `json:"total"`
	Page        int                     `json:"page"`
	PageSize    int                     `json:"page_size"`
}

type RechargeBonusStatsSummary struct {
	BonusCount           int64            `json:"bonus_count"`
	TotalBonusByCurrency map[string]int64 `json:"total_bonus_by_currency"`
	CostByCurrency       map[string]int64 `json:"cost_by_currency"`
}

type ResellerConfigSummary struct {
	Enabled               bool   `json:"enabled"`
	SelfApplyEnabled      bool   `json:"self_apply_enabled"`
	SubdomainBase         string `json:"subdomain_base,omitempty"`
	SettlementConfirmDays int    `json:"settlement_confirm_days"`
}

type ResellerStatsSummary struct {
	TotalProfiles        int64            `json:"total_profiles"`
	PendingReviewCount   int64            `json:"pending_review_count"`
	ActiveCount          int64            `json:"active_count"`
	PendingWithdrawCount int64            `json:"pending_withdraw_count"`
	ProfitCostByCurrency map[string]int64 `json:"profit_cost_by_currency"`
}

type ResellerProfileItem struct {
	ID                   uint       `json:"id"`
	UserID               uint       `json:"user_id"`
	UserEmail            string     `json:"user_email,omitempty"`
	UserDisplayName      string     `json:"user_display_name,omitempty"`
	Status               string     `json:"status"`
	SettlementStatus     string     `json:"settlement_status"`
	ApplyReason          string     `json:"apply_reason,omitempty"`
	RejectReason         string     `json:"reject_reason,omitempty"`
	DefaultMarkupPercent string     `json:"default_markup_percent"`
	MaxMarkupPercent     string     `json:"max_markup_percent"`
	PrimaryDomain        string     `json:"primary_domain,omitempty"`
	ReviewedAt           *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
}

type ResellerProfilesResult struct {
	Source      string                `json:"source"`
	CollectedAt time.Time             `json:"collected_at"`
	Items       []ResellerProfileItem `json:"items"`
	Total       int64                 `json:"total"`
	Page        int                   `json:"page"`
	PageSize    int                   `json:"page_size"`
}

type ResellerWithdrawItem struct {
	ID           uint       `json:"id"`
	ResellerID   uint       `json:"reseller_id"`
	UserID       uint       `json:"user_id,omitempty"`
	UserEmail    string     `json:"user_email,omitempty"`
	Amount       string     `json:"amount"`
	Currency     string     `json:"currency"`
	Channel      string     `json:"channel"`
	Account      string     `json:"account"`
	Status       string     `json:"status"`
	RejectReason string     `json:"reject_reason,omitempty"`
	ProcessedAt  *time.Time `json:"processed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type ResellerWithdrawsResult struct {
	Source      string                 `json:"source"`
	CollectedAt time.Time              `json:"collected_at"`
	Items       []ResellerWithdrawItem `json:"items"`
	Total       int64                  `json:"total"`
	Page        int                    `json:"page"`
	PageSize    int                    `json:"page_size"`
}

type GrowthSummaryResult struct {
	Source       string    `json:"source"`
	CollectedAt  time.Time `json:"collected_at"`
	SiteCurrency string    `json:"site_currency"`
	Affiliate    struct {
		Settings AffiliateSettingDTO   `json:"settings"`
		Stats    AffiliateStatsSummary `json:"stats"`
	} `json:"affiliate"`
	RechargeBonus struct {
		Settings settingscontract.WalletRechargeBonusConfig `json:"settings"`
		Stats    RechargeBonusStatsSummary                  `json:"stats"`
	} `json:"recharge_bonus"`
	Reseller struct {
		Config ResellerConfigSummary `json:"config"`
		Stats  ResellerStatsSummary  `json:"stats"`
	} `json:"reseller"`
}

type ReviewWithdrawInput struct {
	RejectReason string `json:"reject_reason,omitempty"`
}

type ApproveResellerInput struct {
	DefaultMarkupPercent float64 `json:"default_markup_percent"`
	MaxMarkupPercent     float64 `json:"max_markup_percent"`
}

type RejectResellerInput struct {
	Reason string `json:"reason,omitempty"`
}

type UpdateResellerStatusInput struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type GrowthStore interface {
	GrowthSummary() (*GrowthSummaryResult, error)
	UpdateAffiliateSettings(input AffiliateSettingDTO) (*AffiliateSettingDTO, error)
	ListAffiliateWithdraws(page, pageSize int, status string) (*AffiliateWithdrawsResult, error)
	ReviewAffiliateWithdraw(id uint, action, rejectReason string) (*AffiliateWithdrawItem, error)
	UpdateRechargeBonus(input settingscontract.WalletRechargeBonusConfig) (*settingscontract.WalletRechargeBonusConfig, error)
	ListResellerProfiles(page, pageSize int, status, keyword string) (*ResellerProfilesResult, error)
	ApproveResellerProfile(id uint, input ApproveResellerInput) (*ResellerProfileItem, error)
	RejectResellerProfile(id uint, input RejectResellerInput) (*ResellerProfileItem, error)
	UpdateResellerProfileStatus(id uint, input UpdateResellerStatusInput) (*ResellerProfileItem, error)
	ListResellerWithdraws(page, pageSize int, status string) (*ResellerWithdrawsResult, error)
	ReviewResellerWithdraw(id uint, action, rejectReason string) (*ResellerWithdrawItem, error)
}

type GrowthHandler interface {
	GrowthSummary() (*GrowthSummaryResult, error)
	UpdateAffiliateSettings(input AffiliateSettingDTO) (*AffiliateSettingDTO, error)
	ListAffiliateWithdraws(page, pageSize int, status string) (*AffiliateWithdrawsResult, error)
	ReviewAffiliateWithdraw(id uint, action, rejectReason string) (*AffiliateWithdrawItem, error)
	UpdateRechargeBonus(input settingscontract.WalletRechargeBonusConfig) (*settingscontract.WalletRechargeBonusConfig, error)
	ListResellerProfiles(page, pageSize int, status, keyword string) (*ResellerProfilesResult, error)
	ApproveResellerProfile(id uint, input ApproveResellerInput) (*ResellerProfileItem, error)
	RejectResellerProfile(id uint, input RejectResellerInput) (*ResellerProfileItem, error)
	UpdateResellerProfileStatus(id uint, input UpdateResellerStatusInput) (*ResellerProfileItem, error)
	ListResellerWithdraws(page, pageSize int, status string) (*ResellerWithdrawsResult, error)
	ReviewResellerWithdraw(id uint, action, rejectReason string) (*ResellerWithdrawItem, error)
}
