package procurement_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	orderdomain "github.com/dujiao-next/internal/modules/order/domain"

	mappingdomain "github.com/dujiao-next/internal/modules/catalog/mapping/domain"

	"github.com/dujiao-next/internal/constants"
	procurementcontract "github.com/dujiao-next/internal/modules/procurement/contract"
	siteconnectionapp "github.com/dujiao-next/internal/modules/siteconnection/application"
	"github.com/dujiao-next/internal/shared/jsonmap"
)

func TestAPIPlanProcurementUsesFrozenRouteAfterMappingMovedOrDeleted(t *testing.T) {
	for _, state := range []string{"moved", "deleted"} {
		t.Run(state, func(t *testing.T) {
			db := setupProcurementTestDB(t)
			order := createProcTestOrder(t, db, "API-PLAN-"+state, constants.OrderStatusPaid, constants.FulfillmentTypeUpstream)
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				decoder := json.NewDecoder(r.Body)
				decoder.UseNumber()
				var request map[string]interface{}
				if err := decoder.Decode(&request); err != nil {
					t.Error(err)
				}
				contract, _ := request["entitlement_contract"].(map[string]interface{})
				plan, _ := contract["plan"].(map[string]interface{})
				if request["sku_id"] != json.Number("901") || plan["quota"] != json.Number("9007199254740993") || request["entitlement_contract_hash"] != strings.Repeat("a", 64) {
					t.Errorf("frozen route or exact quota changed: %#v", request)
				}
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "order_id": 999, "order_no": "UP-PLAN-999", "status": "accepted", "amount": "50.00", "currency": "CNY"})
			}))
			defer server.Close()
			connections := newTestSiteConnectionService(db, "test-key", t.TempDir())
			connection, err := connections.Create(siteconnectionapp.CreateInput{Name: "frozen-plan", BaseURL: server.URL, ApiKey: "key", ApiSecret: "secret", Protocol: constants.ConnectionProtocolDujiaoNext})
			if err != nil {
				t.Fatal(err)
			}
			snapshot := jsonmap.JSON{"api_plan": true, "requires_reservation": false, "reservation_policy_known": true, "plan_checkout_connection_id": strconv.FormatUint(uint64(connection.ID), 10), "plan_checkout_upstream_sku_id": "901", "plan_checkout_entitlement_raw": `{"kind":"plan","issue_api_key":true,"plan":{"quota":9007199254740993}}`, "entitlement_contract_hash": strings.Repeat("a", 64)}
			if err := db.Model(&orderdomain.OrderItem{}).Where("order_id = ?", order.ID).Update("sku_snapshot_json", snapshot).Error; err != nil {
				t.Fatal(err)
			}
			if state == "moved" {
				mapping := &mappingdomain.Mapping{ConnectionID: connection.ID + 100, LocalProductID: 1, UpstreamProductID: 999, IsActive: true}
				if err := db.Create(mapping).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.Create(&mappingdomain.SKUMapping{ProductMappingID: mapping.ID, LocalSKUID: 1, UpstreamSKUID: 999, UpstreamIsActive: true}).Error; err != nil {
					t.Fatal(err)
				}
			}
			service := newTestProcurementService(db, connections)
			if err := service.CreateForOrder(order.ID); err != nil {
				t.Fatal(err)
			}
			var procurement ProcurementOrder
			if err := db.Where("local_order_id = ?", order.ID).First(&procurement).Error; err != nil {
				t.Fatal(err)
			}
			if procurement.ConnectionID != connection.ID {
				t.Fatalf("paid plan followed live connection: %d", procurement.ConnectionID)
			}
			if err := service.SubmitToUpstream(procurement.ID); err != nil || calls != 1 {
				t.Fatalf("frozen plan did not procure: calls=%d err=%v", calls, err)
			}
		})
	}
}

func TestSubmitScarceOrderRequiresPaidProofAndUsesFrozenSupplierSKU(t *testing.T) {
	db := setupProcurementTestDB(t)
	order := createProcTestOrder(t, db, "SCARCE-PAID-PROOF", constants.OrderStatusPaid, constants.FulfillmentTypeUpstream)
	calls := 0
	expiry := time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond).Format(time.RFC3339Nano)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if request["sku_id"] != float64(990) || request["reservation_order_ref"] != "sf_order_proof_01" || request["reservation_hold_id"] != "hold_proof_01" || request["reservation_hold_version"] != "2" || request["reservation_expires_at"] != expiry {
			t.Errorf("frozen hold association lost: %+v", request)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "order_id": 999, "order_no": "UP-PROOF-999", "status": "accepted", "amount": "50.00", "currency": "CNY"})
	}))
	defer server.Close()
	connections := newTestSiteConnectionService(db, "test-key", t.TempDir())
	conn, err := connections.Create(siteconnectionapp.CreateInput{Name: "frozen-upstream", BaseURL: server.URL, ApiKey: "key", ApiSecret: "secret", Protocol: constants.ConnectionProtocolDujiaoNext})
	if err != nil {
		t.Fatal(err)
	}
	hold := jsonmap.JSON{"order_ref": "sf_order_proof_01", "hold_id": "hold_proof_01", "version": "1", "state": "reserved", "expires_at": expiry}
	snapshot := jsonmap.JSON{"reservation_policy_known": true, "requires_reservation": true, "reservation_connection_id": strconv.FormatUint(uint64(conn.ID), 10), "reservation_upstream_sku_id": "990", "reservation_contract": jsonmap.JSON{"version": 1, "sku": "ChatGPT-Plus-US", "region": "US", "currency": "USD", "required_minor": "2000", "ttl_seconds": 900}, "reservation_hold": hold}
	if err := db.Model(&orderdomain.OrderItem{}).Where("order_id = ?", order.ID).Updates(map[string]any{"sku_snapshot_json": snapshot, "manual_form_submission_json": jsonmap.JSON{"reservation_hold_id": "attacker"}}).Error; err != nil {
		t.Fatal(err)
	}
	service := newTestProcurementService(db, connections)
	// No current mappings exist: scarce routing must retain the supplier/SKU
	// frozen at checkout rather than consult today's mutable mapping tables.
	if err := service.CreateForOrder(order.ID); err != nil {
		t.Fatal(err)
	}
	var proc ProcurementOrder
	if err := db.Where("local_order_id = ?", order.ID).First(&proc).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.SubmitToUpstream(proc.ID); err == nil || calls != 0 {
		t.Fatal("unpaid hold was procured", err, calls)
	}
	hold["state"], hold["version"] = "paid", "2"
	if err := db.Model(&orderdomain.OrderItem{}).Where("order_id = ?", order.ID).Update("sku_snapshot_json", snapshot).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.SubmitToUpstream(proc.ID); err != nil || calls != 1 {
		t.Fatal("paid frozen hold not forwarded", err, calls)
	}
}

// ── SubmitToUpstream tests ──

func TestSubmitToUpstream_Success(t *testing.T) {
	db := setupProcurementTestDB(t)

	order := createProcTestOrder(t, db, "PROC-SUBMIT-001", constants.OrderStatusPaid, constants.FulfillmentTypeUpstream)
	// An immutable checkout snapshot, separate from self-entered form claims.
	frozen := jsonmap.JSON{"version": 1, "token_quota_per_unit": 10}
	if err := db.Model(&orderdomain.OrderItem{}).Where("order_id = ?", order.ID).Updates(map[string]interface{}{
		"sku_snapshot_json":           jsonmap.JSON{"entitlement_contract": frozen, "entitlement_contract_hash": "frozen-hash"},
		"manual_form_submission_json": jsonmap.JSON{"customer_reference": "attacker", "entitlement_contract_hash": "attacker"},
	}).Error; err != nil {
		t.Fatal(err)
	}
	// 创建 product mapping 和 sku mapping
	pm := &mappingdomain.Mapping{
		ConnectionID:      1,
		LocalProductID:    1,
		UpstreamProductID: 101,
		IsActive:          true,
	}
	db.Create(pm)
	sm := &mappingdomain.SKUMapping{
		ProductMappingID: pm.ID,
		LocalSKUID:       1,
		UpstreamSKUID:    201,
		UpstreamIsActive: true,
	}
	db.Create(sm)

	// mock upstream server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		customer, ok := request["customer_reference"].(map[string]interface{})
		if !ok || customer["merchant_customer_id"] != "needtoken-storefront:user:1" || customer["verification"] != "merchant_session" {
			t.Errorf("untrusted customer reference: %+v", request)
		}
		if request["entitlement_contract_hash"] != "frozen-hash" {
			t.Errorf("frozen contract lost/replaced: %+v", request)
		}
		contract, ok := request["entitlement_contract"].(map[string]interface{})
		if !ok || contract["token_quota_per_unit"] != float64(10) {
			t.Errorf("invalid frozen quota: %+v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":       true,
			"order_id": 999,
			"order_no": "UP-999",
			"status":   "accepted",
			"amount":   "50.00",
			"currency": "CNY",
		})
	}))
	defer server.Close()

	connSvc := newTestSiteConnectionService(db, "test-key", t.TempDir())
	conn, err := connSvc.Create(siteconnectionapp.CreateInput{
		Name:      "test-upstream",
		BaseURL:   server.URL,
		ApiKey:    "key",
		ApiSecret: "secret",
		Protocol:  constants.ConnectionProtocolDujiaoNext,
	})
	if err != nil {
		t.Fatalf("create connection: %v", err)
	}

	proc := createTestProcurementOrder(t, db, conn.ID, order.ID, order.OrderNo, "pending")

	svc := newTestProcurementService(db, connSvc)

	if err := svc.SubmitToUpstream(proc.ID); err != nil {
		t.Fatalf("SubmitToUpstream: %v", err)
	}

	// 验证采购单状态 = accepted
	var updatedProc ProcurementOrder
	db.First(&updatedProc, proc.ID)
	if updatedProc.Status != "accepted" {
		t.Errorf("expected procurement status 'accepted', got %q", updatedProc.Status)
	}
	if updatedProc.UpstreamOrderID != 999 {
		t.Errorf("expected upstream_order_id=999, got %d", updatedProc.UpstreamOrderID)
	}

	// 验证本地订单状态 = fulfilling
	var updatedOrder orderdomain.Order
	db.First(&updatedOrder, order.ID)
	if updatedOrder.Status != constants.OrderStatusFulfilling {
		t.Errorf("expected order status %q, got %q", constants.OrderStatusFulfilling, updatedOrder.Status)
	}
}

func TestSubmitToUpstream_NonRetryableError_Rejects(t *testing.T) {
	db := setupProcurementTestDB(t)

	order := createProcTestOrder(t, db, "PROC-NONRETRY-001", constants.OrderStatusFulfilling, constants.FulfillmentTypeUpstream)
	pm := &mappingdomain.Mapping{ConnectionID: 1, LocalProductID: 1, UpstreamProductID: 101, IsActive: true}
	db.Create(pm)
	sm := &mappingdomain.SKUMapping{ProductMappingID: pm.ID, LocalSKUID: 1, UpstreamSKUID: 201, UpstreamIsActive: true}
	db.Create(sm)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":            false,
			"error_code":    "product_out_of_stock",
			"error_message": "product out of stock",
		})
	}))
	defer server.Close()

	connSvc := newTestSiteConnectionService(db, "test-key", t.TempDir())
	conn, _ := connSvc.Create(siteconnectionapp.CreateInput{
		Name: "test-upstream", BaseURL: server.URL,
		ApiKey: "key", ApiSecret: "secret", Protocol: constants.ConnectionProtocolDujiaoNext,
	})

	proc := createTestProcurementOrder(t, db, conn.ID, order.ID, order.OrderNo, "pending")
	svc := newTestProcurementService(db, connSvc)

	// 不可重试错误应返回 error
	_ = svc.SubmitToUpstream(proc.ID)

	// 验证采购单状态 = rejected
	var updatedProc ProcurementOrder
	db.First(&updatedProc, proc.ID)
	if updatedProc.Status != "rejected" {
		t.Errorf("expected procurement status 'rejected', got %q", updatedProc.Status)
	}

	// 验证本地订单状态回退到 paid
	var updatedOrder orderdomain.Order
	db.First(&updatedOrder, order.ID)
	if updatedOrder.Status != constants.OrderStatusPaid {
		t.Errorf("expected order status %q after rejection, got %q", constants.OrderStatusPaid, updatedOrder.Status)
	}
}

func TestSubmitToUpstream_RetryableError_Retries(t *testing.T) {
	db := setupProcurementTestDB(t)

	order := createProcTestOrder(t, db, "PROC-RETRY-001", constants.OrderStatusFulfilling, constants.FulfillmentTypeUpstream)
	pm := &mappingdomain.Mapping{ConnectionID: 1, LocalProductID: 1, UpstreamProductID: 101, IsActive: true}
	db.Create(pm)
	sm := &mappingdomain.SKUMapping{ProductMappingID: pm.ID, LocalSKUID: 1, UpstreamSKUID: 201, UpstreamIsActive: true}
	db.Create(sm)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":            false,
			"error_code":    "server_error",
			"error_message": "temporary failure",
		})
	}))
	defer server.Close()

	connSvc := newTestSiteConnectionService(db, "test-key", t.TempDir())
	conn, _ := connSvc.Create(siteconnectionapp.CreateInput{
		Name: "test-upstream", BaseURL: server.URL,
		ApiKey: "key", ApiSecret: "secret", Protocol: constants.ConnectionProtocolDujiaoNext,
		RetryMax: 3,
	})

	proc := createTestProcurementOrder(t, db, conn.ID, order.ID, order.OrderNo, "pending")
	svc := newTestProcurementService(db, connSvc)

	// 可重试错误不应返回 error（已入队重试）
	if err := svc.SubmitToUpstream(proc.ID); err != nil {
		t.Fatalf("expected no error for retryable failure, got: %v", err)
	}

	// 验证采购单状态 = failed（而非 rejected）
	var updatedProc ProcurementOrder
	db.First(&updatedProc, proc.ID)
	if updatedProc.Status != "failed" {
		t.Errorf("expected procurement status 'failed', got %q", updatedProc.Status)
	}
	if updatedProc.RetryCount != 1 {
		t.Errorf("expected retry_count=1, got %d", updatedProc.RetryCount)
	}
}

func TestHandleSubmitFailure_MaxRetriesExhausted(t *testing.T) {
	db := setupProcurementTestDB(t)

	order := createProcTestOrder(t, db, "PROC-MAXRETRY-001", constants.OrderStatusFulfilling, constants.FulfillmentTypeUpstream)
	productMapping := &mappingdomain.Mapping{ConnectionID: 1, LocalProductID: 1, UpstreamProductID: 101, IsActive: true}
	db.Create(productMapping)
	db.Create(&mappingdomain.SKUMapping{
		ProductMappingID: productMapping.ID,
		LocalSKUID:       1,
		UpstreamSKUID:    201,
		UpstreamIsActive: true,
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":            false,
			"error_code":    "server_error",
			"error_message": "timeout after retries",
		})
	}))
	defer server.Close()

	connSvc := newTestSiteConnectionService(db, "test-key", t.TempDir())
	conn, err := connSvc.Create(siteconnectionapp.CreateInput{
		Name: "test-upstream", BaseURL: server.URL,
		ApiKey: "key", ApiSecret: "secret", Protocol: constants.ConnectionProtocolDujiaoNext,
		RetryMax: 2, RetryIntervals: "[30,60]",
	})
	if err != nil {
		t.Fatalf("create connection: %v", err)
	}

	proc := createTestProcurementOrder(t, db, conn.ID, order.ID, order.OrderNo, "failed")
	// 设置 retry_count 已达上限
	db.Model(proc).Update("retry_count", 2)

	svc := newTestProcurementService(db, connSvc)

	// 通过公开提交入口验证：可重试错误在次数耗尽后仍必须转为 rejected。
	_ = svc.SubmitToUpstream(proc.ID)

	// 验证采购单状态 = rejected
	var updatedProc ProcurementOrder
	db.First(&updatedProc, proc.ID)
	if updatedProc.Status != "rejected" {
		t.Errorf("expected procurement status 'rejected', got %q", updatedProc.Status)
	}

	// 验证本地订单回退到 paid
	var updatedOrder orderdomain.Order
	db.First(&updatedOrder, order.ID)
	if updatedOrder.Status != constants.OrderStatusPaid {
		t.Errorf("expected order status %q, got %q", constants.OrderStatusPaid, updatedOrder.Status)
	}
}

func TestCancelManual_AcceptedKeepsStatusWhenUpstreamRefuses(t *testing.T) {
	db := setupProcurementTestDB(t)
	order := createProcTestOrder(t, db, "PROC-CANCEL-001", constants.OrderStatusFulfilling, constants.FulfillmentTypeUpstream)

	// 上游已支付的订单不允许取消
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error_code": "cancel_not_allowed"})
	}))
	defer server.Close()

	connSvc := newTestSiteConnectionService(db, "test-key", t.TempDir())
	conn, err := connSvc.Create(siteconnectionapp.CreateInput{
		Name: "test-upstream", BaseURL: server.URL, ApiKey: "key", ApiSecret: "secret",
		Protocol: constants.ConnectionProtocolDujiaoNext,
	})
	if err != nil {
		t.Fatalf("create connection: %v", err)
	}
	proc := createTestProcurementOrder(t, db, conn.ID, order.ID, order.OrderNo, constants.ProcurementStatusAccepted)
	db.Model(proc).Update("upstream_order_id", 999)

	svc := newTestProcurementService(db, connSvc)
	if err := svc.CancelManual(proc.ID); !errors.Is(err, procurementcontract.ErrUpstreamCancelRefused) {
		t.Fatalf("expected ErrUpstreamCancelRefused, got %v", err)
	}
	var reloaded ProcurementOrder
	db.First(&reloaded, proc.ID)
	if reloaded.Status != constants.ProcurementStatusAccepted {
		t.Fatalf("expected procurement to stay accepted, got %q", reloaded.Status)
	}
}

// 上游以非 200 + 错误码拒单（如余额不足）时按错误码判断，不可重试的直接拒绝，不反复打上游。
func TestSubmitToUpstream_HTTPErrorWithNonRetryableCode_Rejects(t *testing.T) {
	db := setupProcurementTestDB(t)
	order := createProcTestOrder(t, db, "PROC-HTTP402-001", constants.OrderStatusFulfilling, constants.FulfillmentTypeUpstream)
	pm := &mappingdomain.Mapping{ConnectionID: 1, LocalProductID: 1, UpstreamProductID: 101, IsActive: true}
	db.Create(pm)
	db.Create(&mappingdomain.SKUMapping{ProductMappingID: pm.ID, LocalSKUID: 1, UpstreamSKUID: 201, UpstreamIsActive: true})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPaymentRequired)
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error_code": "insufficient_balance", "error_message": "balance not enough"})
	}))
	defer server.Close()

	connSvc := newTestSiteConnectionService(db, "test-key", t.TempDir())
	conn, _ := connSvc.Create(siteconnectionapp.CreateInput{
		Name: "test-upstream", BaseURL: server.URL,
		ApiKey: "key", ApiSecret: "secret", Protocol: constants.ConnectionProtocolDujiaoNext,
	})
	proc := createTestProcurementOrder(t, db, conn.ID, order.ID, order.OrderNo, "pending")
	_ = newTestProcurementService(db, connSvc).SubmitToUpstream(proc.ID)

	var updated ProcurementOrder
	db.First(&updated, proc.ID)
	if updated.Status != constants.ProcurementStatusRejected || updated.RetryCount != 0 {
		t.Fatalf("expected rejected without retries, got status=%q retry_count=%d", updated.Status, updated.RetryCount)
	}
}

// 本地订单已退款时（如巡检补投历史积压单），不能再去上游下单扣款。
func TestSubmitToUpstream_LocalOrderRefunded_RejectsWithoutCallingUpstream(t *testing.T) {
	db := setupProcurementTestDB(t)
	order := createProcTestOrder(t, db, "PROC-REFUNDED-001", constants.OrderStatusRefunded, constants.FulfillmentTypeUpstream)
	pm := &mappingdomain.Mapping{ConnectionID: 1, LocalProductID: 1, UpstreamProductID: 101, IsActive: true}
	db.Create(pm)
	db.Create(&mappingdomain.SKUMapping{ProductMappingID: pm.ID, LocalSKUID: 1, UpstreamSKUID: 201, UpstreamIsActive: true})

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "order_id": 999, "order_no": "UP-999"})
	}))
	defer server.Close()

	connSvc := newTestSiteConnectionService(db, "test-key", t.TempDir())
	conn, _ := connSvc.Create(siteconnectionapp.CreateInput{
		Name: "test-upstream", BaseURL: server.URL,
		ApiKey: "key", ApiSecret: "secret", Protocol: constants.ConnectionProtocolDujiaoNext,
	})
	proc := createTestProcurementOrder(t, db, conn.ID, order.ID, order.OrderNo, "failed")
	if err := newTestProcurementService(db, connSvc).SubmitToUpstream(proc.ID); err != nil {
		t.Fatalf("SubmitToUpstream: %v", err)
	}

	var updated ProcurementOrder
	db.First(&updated, proc.ID)
	if called || updated.Status != constants.ProcurementStatusRejected {
		t.Fatalf("expected rejected without upstream call, got called=%v status=%q", called, updated.Status)
	}
}

// 下单响应回来前采购单已被回调推进或被管理员取消：不回退状态，但必须补记上游订单号供对账与退款同步。
func TestSubmitToUpstream_StatusChangedDuringRequest_KeepsStatusAndRecordsUpstreamOrder(t *testing.T) {
	for _, concurrent := range []string{constants.ProcurementStatusFulfilled, constants.ProcurementStatusCanceled} {
		t.Run(concurrent, func(t *testing.T) {
			db := setupProcurementTestDB(t)
			order := createProcTestOrder(t, db, "PROC-RACE-"+concurrent, constants.OrderStatusPaid, constants.FulfillmentTypeUpstream)
			pm := &mappingdomain.Mapping{ConnectionID: 1, LocalProductID: 1, UpstreamProductID: 101, IsActive: true}
			db.Create(pm)
			db.Create(&mappingdomain.SKUMapping{ProductMappingID: pm.ID, LocalSKUID: 1, UpstreamSKUID: 201, UpstreamIsActive: true})

			var procID uint
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				db.Model(&ProcurementOrder{}).Where("id = ?", procID).Update("status", concurrent)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{"ok": true, "order_id": 999, "order_no": "UP-999", "amount": "50.00", "currency": "CNY"})
			}))
			defer server.Close()

			connSvc := newTestSiteConnectionService(db, "test-key", t.TempDir())
			conn, _ := connSvc.Create(siteconnectionapp.CreateInput{
				Name: "test-upstream", BaseURL: server.URL,
				ApiKey: "key", ApiSecret: "secret", Protocol: constants.ConnectionProtocolDujiaoNext,
			})
			procID = createTestProcurementOrder(t, db, conn.ID, order.ID, order.OrderNo, "pending").ID
			if err := newTestProcurementService(db, connSvc).SubmitToUpstream(procID); err != nil {
				t.Fatalf("SubmitToUpstream: %v", err)
			}

			var updated ProcurementOrder
			db.First(&updated, procID)
			if updated.Status != concurrent || updated.UpstreamOrderID != 999 || updated.UpstreamOrderNo != "UP-999" {
				t.Fatalf("expected status=%q with upstream order recorded, got status=%q upstream_order_id=%d no=%q",
					concurrent, updated.Status, updated.UpstreamOrderID, updated.UpstreamOrderNo)
			}
		})
	}
}
