package upstream

import (
	"bytes"
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	subscriptionauth "github.com/dujiao-next/internal/modules/order/application/subscriptionauthorization"
)

var ErrSubscriptionAuthorizationUnavailable = errors.New("subscription authorization unavailable")

func (a *DujiaoNextAdapter) SubscriptionAuthorizationContext(ctx context.Context, id uint) (subscriptionauth.Context, error) {
	var result struct {
		OK bool `json:"ok"`
		subscriptionauth.Context
	}
	err := a.authorizationRequest(ctx, http.MethodGet, "/api/v1/upstream/orders/"+strconv.FormatUint(uint64(id), 10)+"/authorization-context", nil, &result)
	if err != nil || !result.OK {
		return subscriptionauth.Context{}, ErrSubscriptionAuthorizationUnavailable
	}
	return result.Context, nil
}

func (a *DujiaoNextAdapter) OpenSubscriptionAuthorization(ctx context.Context, assertion subscriptionauth.Assertion) error {
	var result struct {
		OK          bool   `json:"ok"`
		Accepted    bool   `json:"accepted"`
		OperationID string `json:"operation_id"`
	}
	if err := a.authorizationRequest(ctx, http.MethodPost, "/api/v1/upstream/order-authorizations", assertion, &result); err != nil || !result.OK || !result.Accepted || result.OperationID != assertion.OperationID {
		return ErrSubscriptionAuthorizationUnavailable
	}
	return nil
}

func (a *DujiaoNextAdapter) SubscriptionCredentialURL() (string, error) {
	u, err := url.Parse(a.baseURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return "", ErrSubscriptionAuthorizationUnavailable
	}
	u.Path = "/api/authorization/v1/credential"
	return u.String(), nil
}

func (a *DujiaoNextAdapter) VerifySubscriptionAuthorization(signature subscriptionauth.Signature) bool {
	return len(a.apiSecret) > 0 && hmac.Equal([]byte(a.apiKey), []byte(signature.APIKey)) && IsTimestampValid(signature.Timestamp) && Verify(a.apiSecret, http.MethodGet, signature.Path, signature.Value, signature.Timestamp, nil)
}

// This sensitive bridge has bounded, origin-locked I/O and never logs a
// response body. Generic upstream errors can contain untrusted server content.
func (a *DujiaoNextAdapter) authorizationRequest(ctx context.Context, method, path string, input, output any) error {
	if _, err := a.SubscriptionCredentialURL(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	var raw []byte
	var err error
	if input != nil {
		raw, err = json.Marshal(input)
		if err != nil {
			return ErrSubscriptionAuthorizationUnavailable
		}
	}
	request, err := http.NewRequestWithContext(ctx, method, a.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return ErrSubscriptionAuthorizationUnavailable
	}
	now := time.Now().Unix()
	request.Header.Set(HeaderApiKey, a.apiKey)
	request.Header.Set(HeaderTimestamp, strconv.FormatInt(now, 10))
	request.Header.Set(HeaderSignature, Sign(a.apiSecret, method, path, now, raw))
	request.Header.Set("Accept", "application/json")
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	client := *a.client
	client.Jar = nil
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(request)
	if err != nil {
		return ErrSubscriptionAuthorizationUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ErrSubscriptionAuthorizationUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 16385))
	if err != nil || len(body) > 16384 {
		return ErrSubscriptionAuthorizationUnavailable
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if decoder.Decode(output) != nil || decoder.Decode(new(any)) != io.EOF {
		return ErrSubscriptionAuthorizationUnavailable
	}
	return nil
}
