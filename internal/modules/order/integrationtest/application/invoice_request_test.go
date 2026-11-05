package application_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	. "github.com/dujiao-next/internal/modules/order/application"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	orderinvoice "github.com/dujiao-next/internal/modules/order/invoice"
	orderpresenter "github.com/dujiao-next/internal/modules/order/transport/presenter"
	resellercontract "github.com/dujiao-next/internal/modules/reseller/contract"
)

func TestInvoiceRequestPersistsAcrossMemberGuestAndSplitOrders(t *testing.T) {
	for _, guest := range []bool{false, true} {
		t.Run(map[bool]string{false: "member", true: "guest"}[guest], func(t *testing.T) {
			f := newOrderResellerSnapshotFixture(t)
			tenant := resellercontract.MainTenantContext("main.example.test")
			invoice := &orderdomain.InvoiceRequest{Type: "special_vat", TitleType: "company", Title: " 示例科技有限公司 ", TaxID: "91310000abcdefgh12", Email: " finance@example.test "}
			items := []CreateOrderItem{{ProductID: f.product.ID, SKUID: f.sku.ID, Quantity: 1}}
			secondSKU := f.sku
			secondSKU.ID, secondSKU.SKUCode = 0, "invoice-second-spec"
			if err := f.db.Create(&secondSKU).Error; err != nil {
				t.Fatal(err)
			}
			items = append(items, CreateOrderItem{ProductID: f.product.ID, SKUID: secondSKU.ID, Quantity: 1})
			var preview *OrderPreview
			var created *orderdomain.Order
			var err error
			if guest {
				input := CreateGuestOrderInput{Email: "buyer@example.test", OrderPassword: "invoice-order-secret", Tenant: tenant, Items: items, InvoiceRequest: invoice}
				preview, err = f.svc.PreviewGuestOrder(input)
				if err == nil {
					created, err = f.svc.CreateGuestOrder(input)
				}
			} else {
				input := CreateOrderInput{UserID: f.buyer.ID, Tenant: tenant, Items: items, InvoiceRequest: invoice}
				preview, err = f.svc.PreviewOrder(input)
				if err == nil {
					created, err = f.svc.CreateOrder(input)
				}
			}
			if err != nil || preview.InvoiceRequest == nil || preview.InvoiceRequest.Title != "示例科技有限公司" {
				t.Fatalf("invoice checkout = %+v, %v", preview, err)
			}
			var loaded orderdomain.Order
			if err := f.db.Preload("Children").First(&loaded, created.ID).Error; err != nil {
				t.Fatal(err)
			}
			if loaded.InvoiceRequest == nil || loaded.InvoiceRequest.TaxID != "91310000ABCDEFGH12" || len(loaded.Children) != 2 {
				t.Fatalf("invoice request lost on split order: %+v", loaded)
			}
			for _, child := range loaded.Children {
				if child.InvoiceRequest == nil || child.InvoiceRequest.Email != "finance@example.test" {
					t.Fatalf("cart child did not inherit root invoice request: %+v", child)
				}
			}
			var authorized *orderdomain.Order
			if guest {
				authorized, err = f.svc.GetOrderByGuestOrderNoForTenant(tenant, created.OrderNo, "buyer@example.test", "invoice-order-secret")
				if _, wrongErr := f.svc.GetOrderByGuestOrderNoForTenant(tenant, created.OrderNo, "buyer@example.test", "wrong-password"); !errors.Is(wrongErr, ErrGuestOrderNotFound) {
					t.Fatalf("wrong guest password = %v", wrongErr)
				}
			} else {
				authorized, err = f.svc.GetOrderByUserOrderNoForTenant(tenant, created.OrderNo, f.buyer.ID)
				if _, wrongErr := f.svc.GetOrderByUserOrderNoForTenant(tenant, created.OrderNo, f.owner.ID); !errors.Is(wrongErr, ErrOrderNotFound) {
					t.Fatalf("other member read = %v", wrongErr)
				}
			}
			if err != nil || orderpresenter.NewOrderDetail(authorized).InvoiceRequest == nil {
				t.Fatalf("owner invoice read = %+v, %v", authorized, err)
			}
			raw, _ := json.Marshal(authorized)
			if strings.Contains(string(raw), "invoice_request") || strings.Contains(string(raw), "91310000ABCDEFGH12") {
				t.Fatal("raw domain order leaked invoice data to unrelated adapters")
			}
			raw, _ = json.Marshal(orderpresenter.NewOrderSummary(authorized))
			if strings.Contains(string(raw), "invoice_request") {
				t.Fatal("order lists must omit invoice PII")
			}
		})
	}
}

func TestInvalidInvoiceStopsBeforeOrderPersistence(t *testing.T) {
	f := newOrderResellerSnapshotFixture(t)
	input := CreateOrderInput{UserID: f.buyer.ID, Items: []CreateOrderItem{{ProductID: f.product.ID, SKUID: f.sku.ID, Quantity: 1}}, InvoiceRequest: &orderdomain.InvoiceRequest{Type: "unknown"}}
	if _, err := f.svc.PreviewOrder(input); !errors.Is(err, orderinvoice.ErrInvalid) {
		t.Fatalf("invalid preview = %v", err)
	}
	if _, err := f.svc.CreateOrder(input); !errors.Is(err, orderinvoice.ErrInvalid) {
		t.Fatalf("invalid create = %v", err)
	}
	var count int64
	if err := f.db.Model(&orderdomain.Order{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("invalid request persisted orders: %d, %v", count, err)
	}
}
