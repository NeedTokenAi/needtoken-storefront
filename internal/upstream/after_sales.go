package upstream

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrAfterSalesUnavailable = errors.New("after-sales service unavailable")
	ErrAfterSalesInvalid     = errors.New("after-sales request invalid")
	ErrAfterSalesNotFound    = errors.New("after-sales ticket not found")
	ErrAfterSalesConflict    = errors.New("after-sales ticket closed or conflicted")
)

type OrderTicketSupplement struct {
	Author    string `json:"author"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

type OrderAfterSalesTicket struct {
	ID             string                  `json:"id"`
	OrderNo        string                  `json:"order_no"`
	Category       string                  `json:"category"`
	Priority       string                  `json:"priority"`
	Status         string                  `json:"status"`
	Summary        string                  `json:"summary"`
	ResolutionNote string                  `json:"resolution_note,omitempty"`
	Supplements    []OrderTicketSupplement `json:"supplements,omitempty"`
	CreatedAt      string                  `json:"created_at"`
	UpdatedAt      string                  `json:"updated_at"`
	ResolvedAt     string                  `json:"resolved_at,omitempty"`
}

func (a *DujiaoNextAdapter) ListOrderAfterSalesTickets(ctx context.Context, orderNo string) ([]OrderAfterSalesTicket, error) {
	var result struct {
		OK    bool                    `json:"ok"`
		Items []OrderAfterSalesTicket `json:"items"`
	}
	path := "/api/v1/upstream/after-sales/orders/" + url.PathEscape(strings.TrimSpace(orderNo))
	if err := a.afterSalesRequest(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	if !result.OK {
		return nil, ErrAfterSalesUnavailable
	}
	if result.Items == nil {
		return []OrderAfterSalesTicket{}, nil
	}
	return result.Items, nil
}

func (a *DujiaoNextAdapter) CreateOrderAfterSalesTicket(ctx context.Context, orderNo, customerEmail, category, summary, idempotencyKey string) (OrderAfterSalesTicket, error) {
	var result struct {
		OK     bool                  `json:"ok"`
		Ticket OrderAfterSalesTicket `json:"ticket"`
	}
	path := "/api/v1/upstream/after-sales/orders/" + url.PathEscape(strings.TrimSpace(orderNo))
	payload := struct {
		CustomerEmail  string `json:"customer_email,omitempty"`
		Category       string `json:"category"`
		Summary        string `json:"summary"`
		IdempotencyKey string `json:"idempotency_key,omitempty"`
	}{
		CustomerEmail:  strings.TrimSpace(customerEmail),
		Category:       strings.TrimSpace(category),
		Summary:        strings.TrimSpace(summary),
		IdempotencyKey: strings.TrimSpace(idempotencyKey),
	}
	if err := a.afterSalesRequest(ctx, http.MethodPost, path, payload, &result); err != nil {
		return OrderAfterSalesTicket{}, err
	}
	if !result.OK || strings.TrimSpace(result.Ticket.ID) == "" {
		return OrderAfterSalesTicket{}, ErrAfterSalesUnavailable
	}
	return result.Ticket, nil
}

func (a *DujiaoNextAdapter) AppendOrderAfterSalesSupplement(ctx context.Context, orderNo, ticketID, message string) (OrderAfterSalesTicket, error) {
	var result struct {
		OK     bool                  `json:"ok"`
		Ticket OrderAfterSalesTicket `json:"ticket"`
	}
	path := "/api/v1/upstream/after-sales/orders/" +
		url.PathEscape(strings.TrimSpace(orderNo)) + "/" +
		url.PathEscape(strings.TrimSpace(ticketID)) + "/supplement"
	payload := struct {
		Message string `json:"message"`
	}{
		Message: strings.TrimSpace(message),
	}
	if err := a.afterSalesRequest(ctx, http.MethodPost, path, payload, &result); err != nil {
		return OrderAfterSalesTicket{}, err
	}
	if !result.OK || strings.TrimSpace(result.Ticket.ID) == "" {
		return OrderAfterSalesTicket{}, ErrAfterSalesUnavailable
	}
	return result.Ticket, nil
}

func (a *DujiaoNextAdapter) afterSalesRequest(ctx context.Context, method, path string, input, output any) error {
	base, err := url.Parse(a.baseURL)
	if err != nil || (base.Scheme != "https" && base.Scheme != "http") || base.Host == "" || base.User != nil || base.Path != "" || base.RawQuery != "" || base.Fragment != "" {
		return ErrAfterSalesUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()

	var raw []byte
	if input != nil {
		raw, err = json.Marshal(input)
		if err != nil {
			return ErrAfterSalesInvalid
		}
	}
	signPath := path
	if idx := strings.IndexByte(path, '?'); idx >= 0 {
		signPath = path[:idx]
	}

	req, err := http.NewRequestWithContext(ctx, method, a.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return ErrAfterSalesUnavailable
	}
	now := time.Now().Unix()
	req.Header.Set(HeaderApiKey, a.apiKey)
	req.Header.Set(HeaderTimestamp, strconv.FormatInt(now, 10))
	req.Header.Set(HeaderSignature, Sign(a.apiSecret, method, signPath, now, raw))
	req.Header.Set("Accept", "application/json")
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := *a.client
	client.Jar = nil
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	resp, err := client.Do(req)
	if err != nil {
		return ErrAfterSalesUnavailable
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 131073))
	if err != nil || len(body) > 131072 {
		return ErrAfterSalesUnavailable
	}
	if resp.StatusCode != http.StatusOK {
		return mapAfterSalesError(resp.StatusCode, body)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if decoder.Decode(output) != nil || decoder.Decode(new(any)) != io.EOF {
		return ErrAfterSalesUnavailable
	}
	return nil
}

func mapAfterSalesError(statusCode int, body []byte) error {
	var payload struct {
		ErrorCode string `json:"error_code"`
	}
	_ = json.Unmarshal(body, &payload)
	switch payload.ErrorCode {
	case "invalid_request", "after_sales_invalid":
		return ErrAfterSalesInvalid
	case "after_sales_not_found", "order_not_found":
		return ErrAfterSalesNotFound
	case "after_sales_closed":
		return ErrAfterSalesConflict
	}
	switch statusCode {
	case http.StatusBadRequest:
		return ErrAfterSalesInvalid
	case http.StatusNotFound:
		return ErrAfterSalesNotFound
	case http.StatusConflict:
		return ErrAfterSalesConflict
	default:
		return ErrAfterSalesUnavailable
	}
}
