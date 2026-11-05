package invoice

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
)

func TestNormalizeOptionalInvoiceRequest(t *testing.T) {
	if got, err := Normalize(nil); got != nil || err != nil {
		t.Fatalf("omitted request = %+v, %v", got, err)
	}
	input := &orderdomain.InvoiceRequest{Type: " special_vat ", TitleType: "company", Title: " 示例科技有限公司 ", TaxID: " 91310000abcdefgh12 ", Email: " finance@example.test "}
	got, err := Normalize(input)
	if err != nil || got.Title != "示例科技有限公司" || got.TaxID != "91310000ABCDEFGH12" || got.Email != "finance@example.test" || got.BankAccount != "" {
		t.Fatalf("minimal special VAT request = %+v, %v", got, err)
	}
	if input.TaxID != " 91310000abcdefgh12 " {
		t.Fatal("normalization must not mutate caller-owned draft")
	}
	input.Type, input.BankAccount, input.CompanyPhone = "ordinary", "1234 5678-9012", "+86 (21) 12345678"
	got, err = Normalize(input)
	if err != nil || got.BankAccount != "123456789012" {
		t.Fatalf("optional details = %+v, %v", got, err)
	}
	input.TitleType = "personal"
	got, err = Normalize(input)
	if err != nil || got.TaxID != "" || got.BankAccount != "" || got.CompanyPhone != "" {
		t.Fatalf("personal request retained stale company data: %+v, %v", got, err)
	}
}

func TestNormalizeRejectsInvalidInvoiceRequests(t *testing.T) {
	for name, mutate := range map[string]func(*orderdomain.InvoiceRequest){
		"type":                 func(r *orderdomain.InvoiceRequest) { r.Type = "issued" },
		"title type":           func(r *orderdomain.InvoiceRequest) { r.TitleType = "other" },
		"personal special VAT": func(r *orderdomain.InvoiceRequest) { r.TitleType = "personal" },
		"missing title":        func(r *orderdomain.InvoiceRequest) { r.Title = " " },
		"long title":           func(r *orderdomain.InvoiceRequest) { r.Title = strings.Repeat("字", 121) },
		"title control":        func(r *orderdomain.InvoiceRequest) { r.Title = "公司\n抬头" },
		"tax ID":               func(r *orderdomain.InvoiceRequest) { r.TaxID = "short" },
		"email":                func(r *orderdomain.InvoiceRequest) { r.Email = "Name <finance@example.test>" },
		"long address":         func(r *orderdomain.InvoiceRequest) { r.CompanyAddress = strings.Repeat("字", 301) },
		"phone":                func(r *orderdomain.InvoiceRequest) { r.CompanyPhone = "++++++" },
		"bank account":         func(r *orderdomain.InvoiceRequest) { r.BankAccount = "account number" },
	} {
		t.Run(name, func(t *testing.T) {
			r := &orderdomain.InvoiceRequest{Type: "special_vat", TitleType: "company", Title: "示例科技有限公司", TaxID: "91310000ABCDEFGH12", Email: "finance@example.test"}
			mutate(r)
			if _, err := Normalize(r); !errors.Is(err, ErrInvalid) {
				t.Fatalf("invalid invoice accepted: %v", err)
			}
		})
	}
}

func TestInvoiceJSONRejectsUnknownFieldsAndWrongTypes(t *testing.T) {
	for _, raw := range []string{`{"type":"ordinary","unknown":"lost"}`, `{"type":12}`, `[]`, `"ordinary"`} {
		var request orderdomain.InvoiceRequest
		if err := json.Unmarshal([]byte(raw), &request); err == nil {
			t.Fatalf("malformed invoice accepted: %s", raw)
		}
	}
}
