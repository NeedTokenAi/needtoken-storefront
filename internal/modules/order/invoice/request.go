package invoice

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
)

var ErrInvalid = errors.New("invalid invoice request")
var taxIDPattern = regexp.MustCompile(`^[A-Z0-9]{15,20}$`)
var bankAccountPattern = regexp.MustCompile(`^[0-9]{8,40}$`)
var phonePattern = regexp.MustCompile(`^[0-9+() -]{6,32}$`)

// Normalize validates the optional request. Actual issuance remains a finance
// operation; no tax rate, invoice status, or invoice number is inferred here.
func Normalize(input *orderdomain.InvoiceRequest) (*orderdomain.InvoiceRequest, error) {
	if input == nil {
		return nil, nil
	}
	r := *input
	r.Type, r.TitleType = strings.TrimSpace(r.Type), strings.TrimSpace(r.TitleType)
	r.Title, r.Email = strings.TrimSpace(r.Title), strings.TrimSpace(r.Email)
	r.TaxID = strings.ToUpper(strings.TrimSpace(r.TaxID))
	r.CompanyAddress, r.CompanyPhone = strings.TrimSpace(r.CompanyAddress), strings.TrimSpace(r.CompanyPhone)
	r.BankName, r.BankAccount = strings.TrimSpace(r.BankName), strings.TrimSpace(r.BankAccount)
	if (r.Type != "ordinary" && r.Type != "special_vat") || (r.TitleType != "personal" && r.TitleType != "company") || (r.Type == "special_vat" && r.TitleType != "company") {
		return nil, ErrInvalid
	}
	if !validText(r.Title, 120, true) || !validText(r.Email, 254, true) {
		return nil, ErrInvalid
	}
	address, err := mail.ParseAddress(r.Email)
	if err != nil || address.Address != r.Email {
		return nil, ErrInvalid
	}
	if r.TitleType == "personal" {
		// Avoid retaining company/bank details from an earlier form selection.
		r.TaxID, r.CompanyAddress, r.CompanyPhone, r.BankName, r.BankAccount = "", "", "", "", ""
		return &r, nil
	}
	if !taxIDPattern.MatchString(r.TaxID) || !validText(r.CompanyAddress, 300, false) || !validText(r.BankName, 120, false) {
		return nil, ErrInvalid
	}
	if r.CompanyPhone != "" {
		if !phonePattern.MatchString(r.CompanyPhone) {
			return nil, ErrInvalid
		}
		digits := 0
		for _, digit := range r.CompanyPhone {
			if digit >= '0' && digit <= '9' {
				digits++
			}
		}
		if digits < 6 {
			return nil, ErrInvalid
		}
	}
	if r.BankAccount != "" {
		r.BankAccount = strings.ReplaceAll(strings.ReplaceAll(r.BankAccount, " ", ""), "-", "")
		if !bankAccountPattern.MatchString(r.BankAccount) {
			return nil, ErrInvalid
		}
	}
	return &r, nil
}

func validText(value string, maximum int, required bool) bool {
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > maximum || (required && value == "") {
		return false
	}
	for _, char := range value {
		if unicode.IsControl(char) {
			return false
		}
	}
	return true
}
