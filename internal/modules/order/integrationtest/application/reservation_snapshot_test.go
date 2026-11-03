package application_test

import (
	"testing"

	"github.com/dujiao-next/internal/constants"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	"github.com/dujiao-next/internal/shared/jsonmap"
)

func TestCheckoutFreezesRequiredReservationAndMappingIdentityIgnoringBuyerFields(t *testing.T) {
	f := newCouponLimitRaceFixture(t, "reservation-freeze", 0, 0)
	contract := jsonmap.JSON{"version": 1, "sku": "ChatGPT-Plus-1mo", "region": "US", "currency": "USD", "required_minor": "2500", "ttl_seconds": 900}
	if err := f.db.Model(&f.sku).Updates(map[string]interface{}{"requires_reservation": true, "reservation_policy_known": true, "reservation_contract_json": contract, "reservation_connection_id": uint64(9007199254740993), "reservation_upstream_sku_id": uint64(9007199254740995)}).Error; err != nil {
		t.Fatal(err)
	}
	schema := jsonmap.JSON{"fields": []interface{}{map[string]interface{}{"key": "requires_reservation", "type": "text", "label": map[string]interface{}{"en-US": "Customer note"}, "required": false}}}
	if err := f.db.Model(&f.product).Update("manual_form_schema_json", schema).Error; err != nil {
		t.Fatal(err)
	}
	input := f.input(f.users[0].ID)
	input.CouponCode = ""
	input.ManualFormData = map[string]jsonmap.JSON{orderdomain.ItemKey(f.product.ID, f.sku.ID): {"requires_reservation": "false"}}
	if _, err := f.service(f.couponRepo, f.usageRepo).CreateOrder(input); err != nil {
		t.Fatal(err)
	}
	if err := f.db.Model(&f.sku).Updates(map[string]interface{}{"requires_reservation": false, "reservation_policy_known": false, "reservation_contract_json": jsonmap.JSON{"required_minor": "1"}, "reservation_connection_id": 2, "reservation_upstream_sku_id": 3}).Error; err != nil {
		t.Fatal(err)
	}
	var item orderdomain.OrderItem
	if err := f.db.Where("sku_id = ?", f.sku.ID).First(&item).Error; err != nil {
		t.Fatal(err)
	}
	snapshot := item.SKUSnapshotJSON
	if snapshot["requires_reservation"] != true || snapshot["reservation_policy_known"] != true || snapshot["reservation_connection_id"] != "9007199254740993" || snapshot["reservation_upstream_sku_id"] != "9007199254740995" {
		t.Fatal("server reservation requirement/identity changed", snapshot)
	}
	frozen, ok := snapshot["reservation_contract"].(map[string]interface{})
	if !ok || frozen["required_minor"] != "2500" || frozen["sku"] != "ChatGPT-Plus-1mo" {
		t.Fatal("terms were not frozen", snapshot)
	}
	if item.ManualFormSubmissionJSON["requires_reservation"] != "false" {
		t.Fatal("test did not carry the untrusted buyer field")
	}
}

func TestLegacyUnknownPolicySnapshotStaysUnknownAfterCatalogRefresh(t *testing.T) {
	f := newCouponLimitRaceFixture(t, "reservation-unknown-freeze", 0, 0)
	if err := f.db.Model(&f.product).Update("fulfillment_type", constants.FulfillmentTypeUpstream).Error; err != nil {
		t.Fatal(err)
	}
	input := f.input(f.users[0].ID)
	input.CouponCode = ""
	if _, err := f.service(f.couponRepo, f.usageRepo).CreateOrder(input); err != nil {
		t.Fatal(err)
	}
	if err := f.db.Model(&f.sku).Update("reservation_policy_known", true).Error; err != nil {
		t.Fatal(err)
	}
	var item orderdomain.OrderItem
	if err := f.db.Where("sku_id = ?", f.sku.ID).First(&item).Error; err != nil {
		t.Fatal(err)
	}
	if item.FulfillmentType != constants.FulfillmentTypeUpstream || item.SKUSnapshotJSON["reservation_policy_known"] != false {
		t.Fatal("catalog refresh guessed a legacy order's policy", item.SKUSnapshotJSON)
	}
}
