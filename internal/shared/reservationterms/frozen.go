package reservationterms

import (
	"errors"
	"regexp"
	"strconv"
	"time"

	"github.com/dujiao-next/internal/shared/jsonmap"
)

type HoldProof struct{ OrderRef, HoldID, Version, State, ExpiresAt string }
type Frozen struct {
	Required                    bool
	Terms                       Terms
	ConnectionID, UpstreamSKUID uint
	Hold                        *HoldProof
}

var frozenReference = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{2,127}$`)

// ParseFrozen reads only a persisted server SKU snapshot, never form input.
// Supplier identity and optional hold receipt are separate from asset amounts.
func ParseFrozen(snapshot jsonmap.JSON) (Frozen, error) {
	bad := errors.New("invalid frozen checkout reservation")
	f := Frozen{}
	if raw, exists := snapshot["requires_reservation"]; exists {
		v, ok := raw.(bool)
		if !ok {
			return f, bad
		}
		f.Required = v
	}
	if raw, exists := snapshot["reservation_contract"]; exists && raw != nil {
		f.Required = true
	}
	if !f.Required {
		return f, nil
	}
	terms, err := Parse(snapshot["reservation_contract"])
	if err != nil {
		return f, err
	}
	f.Terms = terms
	readID := func(raw any) (uint, error) {
		s, ok := raw.(string)
		if !ok || s == "" || s[0] == '0' {
			return 0, bad
		}
		v, e := strconv.ParseUint(s, 10, strconv.IntSize)
		if e != nil || v == 0 || strconv.FormatUint(v, 10) != s {
			return 0, bad
		}
		return uint(v), nil
	}
	f.ConnectionID, err = readID(snapshot["reservation_connection_id"])
	if err != nil {
		return f, err
	}
	f.UpstreamSKUID, err = readID(snapshot["reservation_upstream_sku_id"])
	if err != nil {
		return f, err
	}
	if raw := snapshot["reservation_hold"]; raw != nil {
		var p map[string]any
		switch v := raw.(type) {
		case map[string]any:
			p = v
		case jsonmap.JSON:
			p = map[string]any(v)
		default:
			return f, bad
		}
		text := func(key string) string { v, _ := p[key].(string); return v }
		h := &HoldProof{OrderRef: text("order_ref"), HoldID: text("hold_id"), Version: text("version"), State: text("state"), ExpiresAt: text("expires_at")}
		// A pending/unknown receipt is useful for operations but cannot be
		// forwarded as proof authorizing upstream procurement.
		if h.HoldID != "" {
			if !frozenReference.MatchString(h.OrderRef) || !frozenReference.MatchString(h.HoldID) {
				return f, bad
			}
			v, e := strconv.ParseUint(h.Version, 10, 64)
			if e != nil || v == 0 || strconv.FormatUint(v, 10) != h.Version {
				return f, bad
			}
		}
		if _, e := time.Parse(time.RFC3339Nano, h.ExpiresAt); e != nil {
			return f, bad
		}
		f.Hold = h
	}
	return f, nil
}
