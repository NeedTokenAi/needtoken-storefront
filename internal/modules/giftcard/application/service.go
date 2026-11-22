package application

import giftcardcontract "github.com/dujiao-next/internal/modules/giftcard/contract"

// Service 礼品卡管理用例（不含兑换写路径）。
type Service struct {
	repo      giftcardcontract.Repository
	users     giftcardcontract.UserDirectory
	currency  giftcardcontract.CurrencyProvider
	redeemer  giftcardcontract.RedeemTransactionRunner
	universal giftcardcontract.UniversalGiftCardRedeemer
}

// Options 组装管理用例依赖。
type Options struct {
	Repo      giftcardcontract.Repository
	Users     giftcardcontract.UserDirectory
	Currency  giftcardcontract.CurrencyProvider
	Redeemer  giftcardcontract.RedeemTransactionRunner
	Universal giftcardcontract.UniversalGiftCardRedeemer
}

func NewService(opts Options) *Service {
	if opts.Repo == nil {
		panic("giftcard service: repo is nil")
	}
	return &Service{
		repo:      opts.Repo,
		users:     opts.Users,
		currency:  opts.Currency,
		redeemer:  opts.Redeemer,
		universal: opts.Universal,
	}
}

// SetUniversalRedeemer 挂载可选的双端通用礼品卡二阶段核销端口（由 bootstrap 装配）。
func (s *Service) SetUniversalRedeemer(u giftcardcontract.UniversalGiftCardRedeemer) {
	if s == nil {
		return
	}
	s.universal = u
}
