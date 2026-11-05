package orderhttp_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	orderhttp "github.com/dujiao-next/internal/modules/order/transport/http"
	"github.com/gin-gonic/gin"
)

type invoiceCheckoutPort struct{ received *orderdomain.InvoiceRequest }

func (s *invoiceCheckoutPort) CreateOrder(input orderhttp.CreateOrderInput) (*orderdomain.Order, error) {
	s.received = input.InvoiceRequest
	return &orderdomain.Order{OrderNo: "INVOICE", InvoiceRequest: input.InvoiceRequest}, nil
}
func (s *invoiceCheckoutPort) CreateGuestOrder(input orderhttp.CreateGuestOrderInput) (*orderdomain.Order, error) {
	s.received = input.InvoiceRequest
	return &orderdomain.Order{OrderNo: "INVOICE", InvoiceRequest: input.InvoiceRequest}, nil
}
func (s *invoiceCheckoutPort) PreviewOrder(input orderhttp.CreateOrderInput) (*orderhttp.OrderPreview, error) {
	s.received = input.InvoiceRequest
	return &orderhttp.OrderPreview{InvoiceRequest: input.InvoiceRequest}, nil
}
func (s *invoiceCheckoutPort) PreviewGuestOrder(input orderhttp.CreateGuestOrderInput) (*orderhttp.OrderPreview, error) {
	s.received = input.InvoiceRequest
	return &orderhttp.OrderPreview{InvoiceRequest: input.InvoiceRequest}, nil
}

func TestInvoiceForwardedByEveryCheckoutHTTPPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, path := range []string{"member-preview", "guest-preview", "member-create", "guest-create", "member-create-pay", "guest-create-pay"} {
		t.Run(path, func(t *testing.T) {
			port := &invoiceCheckoutPort{}
			create, preview := orderhttp.NewCreateHandler(port, nil, nil, nil), orderhttp.NewPreviewHandler(port)
			handlers := map[string]gin.HandlerFunc{"member-preview": preview.PreviewOrder, "guest-preview": preview.PreviewGuestOrder, "member-create": create.CreateOrder, "guest-create": create.CreateGuestOrder, "member-create-pay": create.CreateOrderAndPay, "guest-create-pay": create.CreateGuestOrderAndPay}
			router := gin.New()
			router.Use(func(c *gin.Context) { c.Set("user_id", uint(7)); c.Next() })
			router.POST("/checkout", handlers[path])
			body := `{"items":[{"product_id":1,"sku_id":2,"quantity":1}],"email":"buyer@example.test","order_password":"order-secret","invoice_request":{"type":"ordinary","title_type":"personal","title":"测试买家","email":"invoice@example.test"}}`
			request := httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			var result struct {
				StatusCode int             `json:"status_code"`
				Data       json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || result.StatusCode != 0 || port.received == nil || port.received.Email != "invoice@example.test" || !strings.Contains(string(result.Data), "invoice_request") {
				t.Fatalf("invoice lost: %s", response.Body.String())
			}
			port.received = nil
			request = httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(strings.Replace(body, `"type":"ordinary"`, `"unknown":"lost","type":"ordinary"`, 1)))
			request.Header.Set("Content-Type", "application/json")
			response = httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if port.received != nil {
				t.Fatal("unknown invoice fields must not reach the use case")
			}
		})
	}
}

func TestInvoiceVisibleOnlyInExplicitAuthorizedDetailPresenters(t *testing.T) {
	invoice := &orderdomain.InvoiceRequest{Type: "ordinary", TitleType: "company", Title: "Private Co", TaxID: "91310000ABCDEFGH12", Email: "finance@example.test"}
	order := orderdomain.Order{OrderNo: "INVOICE", InvoiceRequest: invoice}
	raw, _ := json.Marshal(order)
	if strings.Contains(string(raw), "invoice_request") || strings.Contains(string(raw), invoice.TaxID) {
		t.Fatal("raw domain projection leaked invoice data")
	}
	raw, _ = json.Marshal(orderhttp.AdminOrderDetail{Order: order, InvoiceRequest: invoice})
	if !strings.Contains(string(raw), "invoice_request") {
		t.Fatal("authorized finance detail lost invoice request")
	}
}
