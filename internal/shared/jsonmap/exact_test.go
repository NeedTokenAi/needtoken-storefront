package jsonmap

import (
	"crypto/sha256"
	"encoding/json"
	"testing"
)

func TestExactJSONPreservesSignedLargeIntegerThroughSQLRoundtrip(t *testing.T) {
	raw := []byte(`{"kind":"plan","plan":{"quota":9007199254740993},"issue_api_key":true}`)
	var contract ExactJSON
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatal(err)
	}
	canonical, _ := json.Marshal(contract)
	value, err := contract.Value()
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []interface{}{value, string(value.([]byte))} {
		var stored ExactJSON
		if err := stored.Scan(source); err != nil {
			t.Fatal(err)
		}
		again, _ := json.Marshal(stored)
		if sha256.Sum256(canonical) != sha256.Sum256(again) || string(again) != `{"issue_api_key":true,"kind":"plan","plan":{"quota":9007199254740993}}` {
			t.Fatalf("signed contract changed through SQL: %s", again)
		}
	}
}

func TestExactJSONRejectsInvalidOrMultipleDocuments(t *testing.T) {
	for _, raw := range []string{`{} {}`, `{} trailing`, `{"quota":01}`, `{"quota":NaN}`, `[]`, `true`} {
		var value ExactJSON
		if err := value.UnmarshalJSON([]byte(raw)); err == nil {
			t.Fatalf("accepted invalid entitlement JSON: %s", raw)
		}
	}
	var value ExactJSON
	if err := value.Scan(100); err == nil {
		t.Fatal("unsupported SQL type silently accepted")
	}
}
