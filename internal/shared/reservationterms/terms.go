// Package reservationterms validates the server-sourced scarce-capacity
// contract shared by catalog import, order snapshots and the checkout gate.
package reservationterms

import (
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"strconv"

	"github.com/dujiao-next/internal/shared/jsonmap"
)

var ErrInvalid = errors.New("invalid authoritative reservation terms")

var (
	skuCode         = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{2,95}$`)
	regionCode      = regexp.MustCompile(`^[A-Z]{2}$`)
	currencyCode    = regexp.MustCompile(`^[A-Z]{3}$`)
	positiveInteger = regexp.MustCompile(`^[1-9][0-9]*$`)
)

type Terms struct {
	Version       int64
	SKU           string
	Region        string
	Currency      string
	RequiredMinor int64
	TTLSeconds    int64
}

// Parse accepts an already decoded server JSON object. Amounts are strings so
// neither JSON float decoding nor a browser can round the committed minor units.
// Extra fields and permissive string/number coercions are deliberately rejected.
func Parse(value any) (Terms, error) {
	var raw map[string]interface{}
	switch v := value.(type) {
	case jsonmap.JSON:
		raw = v
	case map[string]interface{}:
		raw = v
	default:
		return Terms{}, ErrInvalid
	}
	if len(raw) != 6 {
		return Terms{}, ErrInvalid
	}
	for _, key := range []string{"version", "sku", "region", "currency", "required_minor", "ttl_seconds"} {
		if _, ok := raw[key]; !ok {
			return Terms{}, ErrInvalid
		}
	}
	version, ok := smallInteger(raw["version"])
	if !ok || version != 1 {
		return Terms{}, ErrInvalid
	}
	ttl, ok := smallInteger(raw["ttl_seconds"])
	if !ok || ttl < 1 || ttl > 86400 {
		return Terms{}, ErrInvalid
	}
	sku, ok := raw["sku"].(string)
	if !ok || !skuCode.MatchString(sku) {
		return Terms{}, ErrInvalid
	}
	region, ok := raw["region"].(string)
	if !ok || !regionCode.MatchString(region) {
		return Terms{}, ErrInvalid
	}
	currency, ok := raw["currency"].(string)
	if !ok || !currencyCode.MatchString(currency) {
		return Terms{}, ErrInvalid
	}
	amount, ok := raw["required_minor"].(string)
	if !ok || !positiveInteger.MatchString(amount) {
		return Terms{}, ErrInvalid
	}
	minor, err := strconv.ParseInt(amount, 10, 64)
	if err != nil || minor <= 0 {
		return Terms{}, ErrInvalid
	}
	return Terms{Version: version, SKU: sku, Region: region, Currency: currency, RequiredMinor: minor, TTLSeconds: ttl}, nil
}

func smallInteger(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case int32:
		return int64(n), true
	case uint:
		if uint64(n) > math.MaxInt64 {
			return 0, false
		}
		return int64(n), true
	case uint64:
		if n > math.MaxInt64 {
			return 0, false
		}
		return int64(n), true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	case float64:
		if math.IsNaN(n) || math.IsInf(n, 0) || n < 0 || n > 86400 || math.Trunc(n) != n {
			return 0, false
		}
		return int64(n), true
	default:
		return 0, false
	}
}

// Snapshot returns a new flat object; changing a catalog value after checkout
// cannot mutate an order's promised capacity dimensions or amount.
func (t Terms) Snapshot() jsonmap.JSON {
	return jsonmap.JSON{"version": t.Version, "sku": t.SKU, "region": t.Region, "currency": t.Currency, "required_minor": strconv.FormatInt(t.RequiredMinor, 10), "ttl_seconds": t.TTLSeconds}
}
