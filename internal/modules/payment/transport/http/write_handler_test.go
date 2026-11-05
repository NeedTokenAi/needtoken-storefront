package paymenthttp

import (
	"encoding/json"
	"errors"
	"fmt"
	mappingcontract "github.com/dujiao-next/internal/modules/catalog/mapping/contract"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"

	"github.com/dujiao-next/internal/platform/http/response"

	"github.com/gin-gonic/gin"
)

type latestPaymentErrorLookup struct{ err error }

func (l latestPaymentErrorLookup) GetLatestPendingByOrder(uint, time.Time) (*paymentdomain.Payment, error) {
	return nil, l.err
}

func TestLatestPaymentReturnsLocalizedPlanGuardErrors(t *testing.T) {
	for _, test := range []struct {
		err  error
		want string
	}{
		{mappingcontract.ErrPlanChanged, "该 API 套餐已更新，请返回套餐页重新选择规格并下单。"},
		{mappingcontract.ErrPlanAccountUnavailable, "已关联的 API 账号当前不可用，请联系客服核查后重试。"},
	} {
		c, recorder := newResponseTestContext()
		handler := &LatestHandler{payments: latestPaymentErrorLookup{err: test.err}}
		handler.respondLatestPayment(c, &orderdomain.Order{ID: 1, Status: constants.OrderStatusPendingPayment})
		assertErrorResponse(t, recorder, response.CodeBadRequest, test.want)
	}
}

func TestRespondPaymentCreateError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		err  error
		code int
		msg  string
	}{
		{name: "verified link required", err: fmt.Errorf("checkout: %w", mappingcontract.ErrPlanAccountLinkRequired), code: response.CodeBadRequest, msg: "请先联系客服，验证并关联您已有的 API 账号后再购买；原有余额与用量会保留。"},
		{name: "eligibility unavailable", err: mappingcontract.ErrPlanEligibilityUnavailable, code: response.CodeBadRequest, msg: "暂时无法核实 API 套餐开通条件，尚未创建付款。请稍后重试或联系客服。"},
		{name: "plan changed", err: mappingcontract.ErrPlanChanged, code: response.CodeBadRequest, msg: "该 API 套餐已更新，请返回套餐页重新选择规格并下单。"},
		{
			name: "gateway response invalid",
			err:  ErrPaymentGatewayResponseInvalid,
			code: response.CodeBadRequest,
			msg:  "支付网关响应异常",
		},
		{
			name: "recharge channel not allowed",
			err:  ErrPaymentChannelNotAllowedForRecharge,
			code: response.CodeBadRequest,
			msg:  "钱包充值不支持此支付渠道",
		},
		{
			name: "unknown error",
			err:  errors.New("boom"),
			code: response.CodeInternal,
			msg:  "创建支付失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, recorder := newResponseTestContext()

			respondPaymentCreateError(c, tt.err)

			assertErrorResponse(t, recorder, tt.code, tt.msg)
		})
	}
}

func TestRespondPaymentCaptureError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		err  error
		code int
		msg  string
	}{
		{
			name: "amount mismatch",
			err:  ErrPaymentAmountMismatch,
			code: response.CodeBadRequest,
			msg:  "支付金额不匹配",
		},
		{
			name: "unknown error",
			err:  errors.New("boom"),
			code: response.CodeInternal,
			msg:  "支付回调处理失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, recorder := newResponseTestContext()

			respondPaymentCaptureError(c, tt.err)

			assertErrorResponse(t, recorder, tt.code, tt.msg)
		})
	}
}

func newResponseTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	return c, recorder
}

func assertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, wantCode int, wantMsg string) {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var body response.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.StatusCode != wantCode {
		t.Fatalf("status_code = %d, want %d; body=%s", body.StatusCode, wantCode, recorder.Body.String())
	}
	if body.Msg != wantMsg {
		t.Fatalf("msg = %q, want %q; body=%s", body.Msg, wantMsg, recorder.Body.String())
	}
}
