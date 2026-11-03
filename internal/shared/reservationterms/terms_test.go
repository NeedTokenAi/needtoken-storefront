package reservationterms

import (
	"encoding/json"
	"math"
	"strconv"
	"testing"

	"github.com/dujiao-next/internal/shared/jsonmap"
)

func contract() jsonmap.JSON {
	return jsonmap.JSON{"version": 1, "sku": "ChatGPT-Plus-1mo", "region": "US", "currency": "USD", "required_minor": "2500", "ttl_seconds": 900}
}

func TestParseExactMinorUnitsAndCanonicalSnapshot(t *testing.T) {
	raw := contract()
	raw["required_minor"] = strconv.FormatInt(math.MaxInt64, 10)
	b, _ := json.Marshal(raw)
	var decoded jsonmap.JSON
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	terms, err := Parse(decoded)
	if err != nil || terms.RequiredMinor != math.MaxInt64 || terms.TTLSeconds != 900 {
		t.Fatal(terms, err)
	}
	snapshot := terms.Snapshot()
	if snapshot["required_minor"] != strconv.FormatInt(math.MaxInt64, 10) {
		t.Fatal(snapshot)
	}
	snapshot["required_minor"] = "1"
	if terms.RequiredMinor != math.MaxInt64 {
		t.Fatal("snapshot aliases terms")
	}
}

func TestParseRejectsMissingMalformedAndCoercedTerms(t *testing.T) {
	for _, change := range []func(jsonmap.JSON){
		func(r jsonmap.JSON) { delete(r, "required_minor") }, func(r jsonmap.JSON) { r["unknown"] = "extra" }, func(r jsonmap.JSON) { r["version"] = "1" }, func(r jsonmap.JSON) { r["version"] = 2 }, func(r jsonmap.JSON) { r["sku"] = "token from buyer" }, func(r jsonmap.JSON) { r["region"] = "us" }, func(r jsonmap.JSON) { r["currency"] = "usd" }, func(r jsonmap.JSON) { r["required_minor"] = 2500 }, func(r jsonmap.JSON) { r["required_minor"] = "02500" }, func(r jsonmap.JSON) { r["required_minor"] = "0" }, func(r jsonmap.JSON) { r["required_minor"] = "-1" }, func(r jsonmap.JSON) { r["required_minor"] = "9223372036854775808" }, func(r jsonmap.JSON) { r["ttl_seconds"] = "900" }, func(r jsonmap.JSON) { r["ttl_seconds"] = 0 }, func(r jsonmap.JSON) { r["ttl_seconds"] = 86401 }, func(r jsonmap.JSON) { r["ttl_seconds"] = 1.5 },
	} {
		raw := contract()
		change(raw)
		if _, err := Parse(raw); err == nil {
			t.Fatalf("accepted malformed contract: %#v", raw)
		}
	}
	for _, raw := range []any{nil, "buyer-supplied-json", []any{contract()}} {
		if _, err := Parse(raw); err == nil {
			t.Fatal("accepted nonobject", raw)
		}
	}
}
