package application

import (
	"strings"

	opscontract "github.com/dujiao-next/internal/modules/opsbridge/contract"
	settingscontract "github.com/dujiao-next/internal/modules/settings/contract"
)

func (s *Service) WithGrowth(growth opscontract.GrowthStore) *Service {
	s.growth = growth
	return s
}

func (s *Service) GrowthSummary() (*opscontract.GrowthSummaryResult, error) {
	if s.growth == nil {
		return nil, opscontract.ErrNotConfigured
	}
	return s.growth.GrowthSummary()
}

func (s *Service) UpdateAffiliateSettings(input opscontract.AffiliateSettingDTO) (*opscontract.AffiliateSettingDTO, error) {
	if s.growth == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if input.CommissionRate < 0 || input.CommissionRate > 100 || input.ConfirmDays < 0 || input.ConfirmDays > 3650 || input.MinWithdrawAmount < 0 {
		return nil, opscontract.ErrInvalidInput
	}
	return s.growth.UpdateAffiliateSettings(input)
}

func (s *Service) ListAffiliateWithdraws(page, pageSize int, status string) (*opscontract.AffiliateWithdrawsResult, error) {
	if s.growth == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	return s.growth.ListAffiliateWithdraws(page, pageSize, strings.TrimSpace(status))
}

func (s *Service) ReviewAffiliateWithdraw(id uint, action, rejectReason string) (*opscontract.AffiliateWithdrawItem, error) {
	if s.growth == nil {
		return nil, opscontract.ErrNotConfigured
	}
	act := strings.ToLower(strings.TrimSpace(action))
	if id == 0 || (act != "pay" && act != "reject") {
		return nil, opscontract.ErrInvalidInput
	}
	return s.growth.ReviewAffiliateWithdraw(id, act, strings.TrimSpace(rejectReason))
}

func (s *Service) UpdateRechargeBonus(input settingscontract.WalletRechargeBonusConfig) (*settingscontract.WalletRechargeBonusConfig, error) {
	if s.growth == nil {
		return nil, opscontract.ErrNotConfigured
	}
	return s.growth.UpdateRechargeBonus(settingscontract.NormalizeWalletRechargeBonusConfig(input))
}

func (s *Service) ListResellerProfiles(page, pageSize int, status, keyword string) (*opscontract.ResellerProfilesResult, error) {
	if s.growth == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	return s.growth.ListResellerProfiles(page, pageSize, strings.TrimSpace(status), strings.TrimSpace(keyword))
}

func (s *Service) ApproveResellerProfile(id uint, input opscontract.ApproveResellerInput) (*opscontract.ResellerProfileItem, error) {
	if s.growth == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if id == 0 || input.DefaultMarkupPercent < 0 || input.MaxMarkupPercent < 0 || input.MaxMarkupPercent < input.DefaultMarkupPercent {
		return nil, opscontract.ErrInvalidInput
	}
	return s.growth.ApproveResellerProfile(id, input)
}

func (s *Service) RejectResellerProfile(id uint, input opscontract.RejectResellerInput) (*opscontract.ResellerProfileItem, error) {
	if s.growth == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if id == 0 {
		return nil, opscontract.ErrInvalidInput
	}
	input.Reason = strings.TrimSpace(input.Reason)
	return s.growth.RejectResellerProfile(id, input)
}

func (s *Service) UpdateResellerProfileStatus(id uint, input opscontract.UpdateResellerStatusInput) (*opscontract.ResellerProfileItem, error) {
	if s.growth == nil {
		return nil, opscontract.ErrNotConfigured
	}
	st := strings.ToLower(strings.TrimSpace(input.Status))
	if id == 0 || (st != "active" && st != "disabled") {
		return nil, opscontract.ErrInvalidInput
	}
	input.Status = st
	input.Reason = strings.TrimSpace(input.Reason)
	return s.growth.UpdateResellerProfileStatus(id, input)
}

func (s *Service) ListResellerWithdraws(page, pageSize int, status string) (*opscontract.ResellerWithdrawsResult, error) {
	if s.growth == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	return s.growth.ListResellerWithdraws(page, pageSize, strings.TrimSpace(status))
}

func (s *Service) ReviewResellerWithdraw(id uint, action, rejectReason string) (*opscontract.ResellerWithdrawItem, error) {
	if s.growth == nil {
		return nil, opscontract.ErrNotConfigured
	}
	act := strings.ToLower(strings.TrimSpace(action))
	if id == 0 || (act != "pay" && act != "reject") {
		return nil, opscontract.ErrInvalidInput
	}
	return s.growth.ReviewResellerWithdraw(id, act, strings.TrimSpace(rejectReason))
}
