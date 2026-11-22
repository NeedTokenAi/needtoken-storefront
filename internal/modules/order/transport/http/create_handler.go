package orderhttp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"unicode/utf8"

	paymentpresenter "github.com/dujiao-next/internal/modules/payment/transport/presenter"

	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"

	orderdomain "github.com/dujiao-next/internal/modules/order/domain"

	"github.com/dujiao-next/internal/i18n"
	captcha "github.com/dujiao-next/internal/modules/captcha/contract"
	captchahttp "github.com/dujiao-next/internal/modules/captcha/transport/http"
	orderpresenter "github.com/dujiao-next/internal/modules/order/transport/presenter"
	"github.com/dujiao-next/internal/platform/http/ginutil"
	"github.com/dujiao-next/internal/platform/http/response"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"

	"github.com/gin-gonic/gin"
)

// OrderCreateService 订单创建端口。
type OrderCreateService interface {
	CreateOrder(input CreateOrderInput) (*orderdomain.Order, error)
	CreateGuestOrder(input CreateGuestOrderInput) (*orderdomain.Order, error)
}

// GuestCreateCaptcha 游客下单验证码端口。
type GuestCreateCaptcha interface {
	VerifyGuestCreateOrder(payload captchahttp.CaptchaPayloadRequest, clientIP string) error
}

// CreatePaymentInput 创建支付输入。
type CreatePaymentInput struct {
	OrderID       uint
	ChannelID     uint
	UseBalance    bool
	ClientIP      string
	Context       context.Context
	RequestScheme string
}

// CreatePaymentResult 创建支付结果。
type CreatePaymentResult struct {
	Payment          *paymentdomain.Payment
	OrderPaid        bool
	WalletPaidAmount money.Amount
	OnlinePayAmount  money.Amount
}

// OrderPaymentCreator 订单支付创建端口。
type OrderPaymentCreator interface {
	CreatePayment(input CreatePaymentInput) (*CreatePaymentResult, error)
}

// CreateOrderAndPayRequest 创建订单并发起支付请求
type CreateOrderAndPayRequest struct {
	InvoiceRequest      *orderdomain.InvoiceRequest `json:"invoice_request"`
	Items               []OrderItemRequest          `json:"items" binding:"required"`
	CouponCode          string                      `json:"coupon_code"`
	AffiliateCode       string                      `json:"affiliate_code"`
	AffiliateVisitorKey string                      `json:"affiliate_visitor_key"`
	ManualFormData      map[string]jsonmap.JSON     `json:"manual_form_data"`
	ChannelID           uint                        `json:"channel_id"`
	UseBalance          bool                        `json:"use_balance"`
}

// CreateGuestOrderAndPayRequest 游客创建订单并发起支付请求
type CreateGuestOrderAndPayRequest struct {
	InvoiceRequest      *orderdomain.InvoiceRequest       `json:"invoice_request"`
	Email               string                            `json:"email" binding:"required"`
	OrderPassword       string                            `json:"order_password" binding:"required"`
	Items               []OrderItemRequest                `json:"items" binding:"required"`
	CouponCode          string                            `json:"coupon_code"`
	AffiliateCode       string                            `json:"affiliate_code"`
	AffiliateVisitorKey string                            `json:"affiliate_visitor_key"`
	ManualFormData      map[string]jsonmap.JSON           `json:"manual_form_data"`
	CaptchaPayload      captchahttp.CaptchaPayloadRequest `json:"captcha_payload"`
	ChannelID           uint                              `json:"channel_id"`
}

// CreateHandler 处理前台订单创建与 create-and-pay HTTP。
type CreateHandler struct {
	orders   OrderCreateService
	payments PaymentChannelPolicy
	captcha  GuestCreateCaptcha
	pay      OrderPaymentCreator
}

func NewCreateHandler(orders OrderCreateService, payments PaymentChannelPolicy, captcha GuestCreateCaptcha, pay OrderPaymentCreator) *CreateHandler {
	if orders == nil {
		panic("order create handler: orders is nil")
	}
	return &CreateHandler{orders: orders, payments: payments, captcha: captcha, pay: pay}
}

// CreateOrder 创建订单
func (h *CreateHandler) CreateOrder(c *gin.Context) {
	uid, ok := ginutil.GetUserID(c)
	if !ok {
		return
	}

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutil.RespondBindError(c, err)
		return
	}

	userAgent, fingerprint := extractClientDeviceMetadata(c)
	order, err := h.orders.CreateOrder(CreateOrderInput{
		UserID:              uid,
		Tenant:              tenantFromRequest(c),
		Items:               mapOrderItems(req.Items),
		CouponCode:          req.CouponCode,
		AffiliateCode:       req.AffiliateCode,
		AffiliateVisitorKey: req.AffiliateVisitorKey,
		ClientIP:            c.ClientIP(),
		UserAgent:           userAgent,
		ClientFingerprint:   fingerprint,
		ManualFormData:      req.ManualFormData,
		InvoiceRequest:      req.InvoiceRequest,
	})
	if err != nil {
		respondUserOrderCreateError(c, err)
		return
	}

	orderDetail := orderpresenter.NewOrderDetail(order)
	enrichOrderWithAllowedChannels(h.payments, order, &orderDetail)
	response.Success(c, orderDetail)
}

// CreateGuestOrder 游客创建订单
func (h *CreateHandler) CreateGuestOrder(c *gin.Context) {
	var req CreateGuestOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutil.RespondBindError(c, err)
		return
	}
	if !h.verifyGuestCreateCaptcha(c, req.CaptchaPayload) {
		return
	}

	userAgent, fingerprint := extractClientDeviceMetadata(c)
	order, err := h.orders.CreateGuestOrder(CreateGuestOrderInput{
		Email:               req.Email,
		OrderPassword:       req.OrderPassword,
		Locale:              i18n.ResolveLocale(c),
		Tenant:              tenantFromRequest(c),
		Items:               mapOrderItems(req.Items),
		CouponCode:          req.CouponCode,
		AffiliateCode:       req.AffiliateCode,
		AffiliateVisitorKey: req.AffiliateVisitorKey,
		ClientIP:            c.ClientIP(),
		UserAgent:           userAgent,
		ClientFingerprint:   fingerprint,
		ManualFormData:      req.ManualFormData,
		InvoiceRequest:      req.InvoiceRequest,
	})
	if err != nil {
		respondGuestOrderCreateError(c, err)
		return
	}
	orderDetail := orderpresenter.NewOrderDetail(order)
	enrichOrderWithAllowedChannels(h.payments, order, &orderDetail)
	response.Success(c, orderDetail)
}

// CreateOrderAndPay 创建订单并发起支付（合并接口）
func (h *CreateHandler) CreateOrderAndPay(c *gin.Context) {
	uid, ok := ginutil.GetUserID(c)
	if !ok {
		return
	}

	var req CreateOrderAndPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutil.RespondBindError(c, err)
		return
	}

	userAgent, fingerprint := extractClientDeviceMetadata(c)
	order, err := h.orders.CreateOrder(CreateOrderInput{
		UserID:              uid,
		Tenant:              tenantFromRequest(c),
		Items:               mapOrderItems(req.Items),
		CouponCode:          req.CouponCode,
		AffiliateCode:       req.AffiliateCode,
		AffiliateVisitorKey: req.AffiliateVisitorKey,
		ClientIP:            c.ClientIP(),
		UserAgent:           userAgent,
		ClientFingerprint:   fingerprint,
		ManualFormData:      req.ManualFormData,
		InvoiceRequest:      req.InvoiceRequest,
	})
	if err != nil {
		respondUserOrderCreateError(c, err)
		return
	}

	orderResp := orderpresenter.NewOrderDetail(order)
	enrichOrderWithAllowedChannels(h.payments, order, &orderResp)

	if req.ChannelID == 0 && !req.UseBalance {
		response.Success(c, gin.H{
			"order":    orderResp,
			"order_no": order.OrderNo,
		})
		return
	}

	h.respondCreateAndPay(c, order, orderResp, req.ChannelID, req.UseBalance)
}

// CreateGuestOrderAndPay 游客创建订单并发起支付（合并接口）
func (h *CreateHandler) CreateGuestOrderAndPay(c *gin.Context) {
	var req CreateGuestOrderAndPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutil.RespondBindError(c, err)
		return
	}
	if !h.verifyGuestCreateCaptcha(c, req.CaptchaPayload) {
		return
	}

	userAgent, fingerprint := extractClientDeviceMetadata(c)
	order, err := h.orders.CreateGuestOrder(CreateGuestOrderInput{
		Email:               req.Email,
		OrderPassword:       req.OrderPassword,
		Locale:              i18n.ResolveLocale(c),
		Tenant:              tenantFromRequest(c),
		Items:               mapOrderItems(req.Items),
		CouponCode:          req.CouponCode,
		AffiliateCode:       req.AffiliateCode,
		AffiliateVisitorKey: req.AffiliateVisitorKey,
		ClientIP:            c.ClientIP(),
		UserAgent:           userAgent,
		ClientFingerprint:   fingerprint,
		ManualFormData:      req.ManualFormData,
		InvoiceRequest:      req.InvoiceRequest,
	})
	if err != nil {
		respondGuestOrderCreateError(c, err)
		return
	}
	orderResp := orderpresenter.NewOrderDetail(order)
	enrichOrderWithAllowedChannels(h.payments, order, &orderResp)

	if req.ChannelID == 0 {
		response.Success(c, gin.H{
			"order":    orderResp,
			"order_no": order.OrderNo,
		})
		return
	}

	h.respondCreateAndPay(c, order, orderResp, req.ChannelID, false)
}

func (h *CreateHandler) verifyGuestCreateCaptcha(c *gin.Context, payload captchahttp.CaptchaPayloadRequest) bool {
	if h.captcha == nil {
		return true
	}
	if captchaErr := h.captcha.VerifyGuestCreateOrder(payload, c.ClientIP()); captchaErr != nil {
		switch {
		case errors.Is(captchaErr, captcha.ErrRequired):
			ginutil.RespondError(c, response.CodeBadRequest, "error.captcha_required", nil)
		case errors.Is(captchaErr, captcha.ErrInvalid):
			ginutil.RespondError(c, response.CodeBadRequest, "error.captcha_invalid", nil)
		case errors.Is(captchaErr, captcha.ErrConfigInvalid):
			ginutil.RespondError(c, response.CodeInternal, "error.captcha_config_invalid", captchaErr)
		default:
			ginutil.RespondError(c, response.CodeInternal, "error.captcha_verify_failed", captchaErr)
		}
		return false
	}
	return true
}

func (h *CreateHandler) respondCreateAndPay(c *gin.Context, order *orderdomain.Order, orderResp orderpresenter.OrderDetail, channelID uint, useBalance bool) {
	if h.pay == nil {
		response.Success(c, gin.H{
			"order":         orderResp,
			"order_no":      order.OrderNo,
			"payment_error": "payment creator unavailable",
		})
		return
	}
	result, err := h.pay.CreatePayment(CreatePaymentInput{
		OrderID:       order.ID,
		ChannelID:     channelID,
		UseBalance:    useBalance,
		ClientIP:      c.ClientIP(),
		Context:       c.Request.Context(),
		RequestScheme: ginutil.RequestScheme(c),
	})
	if err != nil {
		response.Success(c, gin.H{
			"order":         orderResp,
			"order_no":      order.OrderNo,
			"payment_error": err.Error(),
		})
		return
	}

	resp := gin.H{
		"order":              orderResp,
		"order_no":           order.OrderNo,
		"order_paid":         result.OrderPaid,
		"wallet_paid_amount": result.WalletPaidAmount,
		"online_pay_amount":  result.OnlinePayAmount,
	}
	if result.Payment != nil {
		resp["payment_id"] = result.Payment.ID
		resp["provider_type"] = result.Payment.ProviderType
		resp["channel_type"] = result.Payment.ChannelType
		resp["interaction_mode"] = result.Payment.InteractionMode
		resp["pay_url"] = result.Payment.PayURL
		resp["qr_code"] = result.Payment.QRCode
		if info := paymentpresenter.ExtractCryptoWalletInfo(result.Payment.ProviderType, result.Payment.InteractionMode, result.Payment.ProviderPayload); info.HasAny() {
			if info.Address != "" {
				resp["wallet_address"] = info.Address
			}
			if info.ChainAmount != "" {
				resp["chain_amount"] = info.ChainAmount
			}
			if info.Chain != "" {
				resp["chain"] = info.Chain
			}
			if info.TokenID != "" {
				resp["token_id"] = info.TokenID
			}
		}
		resp["expires_at"] = result.Payment.ExpiredAt
	}
	response.Success(c, resp)
}

func extractClientDeviceMetadata(c *gin.Context) (string, string) {
	if c == nil || c.Request == nil {
		return "", ""
	}
	ua := truncateUTF8Bytes(strings.TrimSpace(c.Request.UserAgent()), 512)
	for _, header := range []string{"X-Client-Fingerprint", "X-Device-Fingerprint"} {
		if raw := sanitizeFingerprint(c.GetHeader(header)); raw != "" {
			return ua, raw
		}
	}
	acceptLang := strings.TrimSpace(c.GetHeader("Accept-Language"))
	secUA := strings.TrimSpace(c.GetHeader("Sec-CH-UA"))
	secPlatform := strings.TrimSpace(c.GetHeader("Sec-CH-UA-Platform"))
	secMobile := strings.TrimSpace(c.GetHeader("Sec-CH-UA-Mobile"))
	if ua == "" && acceptLang == "" && secUA == "" && secPlatform == "" && secMobile == "" {
		return "", ""
	}
	sum := sha256.Sum256([]byte(strings.Join([]string{ua, acceptLang, secUA, secPlatform, secMobile}, "|")))
	return ua, "fp_" + hex.EncodeToString(sum[:16])
}

func sanitizeFingerprint(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) < 8 {
		return ""
	}
	var b strings.Builder
	for i := 0; i < len(trimmed) && b.Len() < 128; i++ {
		ch := trimmed[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.' || ch == ':' {
			b.WriteByte(ch)
		}
	}
	if b.Len() < 8 {
		return ""
	}
	return b.String()
}

func truncateUTF8Bytes(s string, maxBytes int) string {
	if maxBytes <= 0 || len(s) <= maxBytes {
		return s
	}
	for len(s) > maxBytes {
		_, size := utf8.DecodeLastRuneInString(s)
		if size <= 0 {
			break
		}
		s = s[:len(s)-size]
	}
	return s
}
