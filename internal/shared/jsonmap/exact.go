package jsonmap

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
)

// ExactJSON preserves JSON number literals for signed immutable entitlements.
// Ordinary merchandising JSON keeps its existing JSON type and behavior.
type ExactJSON map[string]interface{}

func (value *ExactJSON) UnmarshalJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var decoded map[string]interface{}
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("entitlement JSON contains trailing data")
	}
	*value = decoded
	return nil
}

func (value ExactJSON) Value() (driver.Value, error) {
	if value == nil {
		return nil, nil
	}
	return json.Marshal(value)
}

func (value *ExactJSON) Scan(source interface{}) error {
	switch data := source.(type) {
	case nil:
		*value = make(ExactJSON)
		return nil
	case []byte:
		return value.UnmarshalJSON(data)
	case string:
		return value.UnmarshalJSON([]byte(data))
	default:
		return fmt.Errorf("unsupported entitlement JSON database type %T", source)
	}
}
