package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dujiao-next/internal/app/container"
	"github.com/dujiao-next/internal/bootstrap/opsbridge"
	channelclientapp "github.com/dujiao-next/internal/modules/channelclient/application"
	channelclientdomain "github.com/dujiao-next/internal/modules/channelclient/domain"
	giftcardcontract "github.com/dujiao-next/internal/modules/giftcard/contract"
	giftcarddomain "github.com/dujiao-next/internal/modules/giftcard/domain"
	opsapp "github.com/dujiao-next/internal/modules/opsbridge/application"
	opscontract "github.com/dujiao-next/internal/modules/opsbridge/contract"
	opstransport "github.com/dujiao-next/internal/modules/opsbridge/transport/http"
	"github.com/dujiao-next/internal/upstream"
	"github.com/gin-gonic/gin"
)

func TestDedicatedOpsCredentialScope(t *testing.T) {
	for _, test := range []struct {
		kind, path string
		allowed    bool
	}{
		{"needtoken_ops", "/api/v1/channel/needtoken-ops/overview", true},
		{"needtoken_ops", "/api/v1/channel/needtoken-ops/orders", true},
		{"needtoken_ops", "/api/v1/channel/needtoken-ops/gift-cards", true},
		{"needtoken_ops", "/api/v1/channel/needtoken-ops/gift-cards/generate", true},
		{"telegram", "/api/v1/channel/needtoken-ops/orders", false},
		{"telegram", "/api/v1/channel/needtoken-ops/gift-cards/generate", false},
		{"needtoken_ops", "/api/v1/channel/wallet/recharge", false},
		{"telegram", "/api/v1/channel/wallet/recharge", true},
		{"", "/api/v1/channel/needtoken-ops/orders", false},
	} {
		if got := channelTypeAllowsPath(test.kind, test.path); got != test.allowed {
			t.Errorf("%s %s: %v", test.kind, test.path, got)
		}
	}
}

type memChannelClientStore struct {
	mu    sync.Mutex
	items map[uint]*channelclientdomain.Client
	next  uint
}

func (s *memChannelClientStore) Create(client *channelclientdomain.Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.items == nil {
		s.items = map[uint]*channelclientdomain.Client{}
	}
	s.next++
	client.ID = s.next
	cp := *client
	s.items[client.ID] = &cp
	return nil
}
func (s *memChannelClientStore) FindByID(id uint) (*channelclientdomain.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.items[id]; ok {
		cp := *c
		return &cp, nil
	}
	return nil, nil
}
func (s *memChannelClientStore) FindByChannelKey(key string) (*channelclientdomain.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.items {
		if c.ChannelKey == key {
			cp := *c
			return &cp, nil
		}
	}
	return nil, nil
}
func (s *memChannelClientStore) FindActiveByChannelType(channelType string) (*channelclientdomain.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.items {
		if c.ChannelType == channelType && c.Status == 1 {
			cp := *c
			return &cp, nil
		}
	}
	return nil, nil
}
func (s *memChannelClientStore) FindAll() ([]channelclientdomain.Client, error)  { return nil, nil }
func (s *memChannelClientStore) Update(client *channelclientdomain.Client) error { return nil }
func (s *memChannelClientStore) Delete(uint, time.Time) error                    { return nil }
func (s *memChannelClientStore) UpdateLastUsed(id uint, usedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.items[id]; ok {
		c.LastUsedAt = &usedAt
	}
	return nil
}

type memGiftCardRepo struct {
	giftcardcontract.Repository
	cards  []giftcarddomain.GiftCard
	nextID uint
}

func (m *memGiftCardRepo) WithinTransaction(fn func(repo giftcardcontract.Repository) error) error {
	return fn(m)
}
func (m *memGiftCardRepo) CreateBatch(batch *giftcarddomain.GiftCardBatch, cards []giftcarddomain.GiftCard) error {
	m.nextID++
	batch.ID = m.nextID
	bCopy := *batch
	for i := range cards {
		m.nextID++
		cards[i].ID = m.nextID
		cards[i].BatchID = &batch.ID
		cCopy := cards[i]
		cCopy.Batch = &bCopy
		m.cards = append(m.cards, cCopy)
	}
	return nil
}
func (m *memGiftCardRepo) List(filter giftcardcontract.ListFilter) ([]giftcarddomain.GiftCard, int64, error) {
	return m.cards, int64(len(m.cards)), nil
}

func TestOpsBridgeGiftCardsHMACAuthAndGeneration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_ = opsbridge.NewHandler // verify bootstrap symbol compiles

	clientStore := &memChannelClientStore{}
	clientSvc := channelclientapp.NewService(clientStore, "0123456789abcdef0123456789abcdef")
	opsCred, err := clientSvc.CreateChannelClient("NeedToken Ops", "needtoken_ops", "", "", "")
	if err != nil {
		t.Fatalf("create ops channel client: %v", err)
	}
	botCred, err := clientSvc.CreateChannelClient("Telegram Bot", "telegram", "", "", "")
	if err != nil {
		t.Fatalf("create telegram channel client: %v", err)
	}

	gcRepo := &memGiftCardRepo{}
	opsSvc := opsapp.New(nil, nil, nil).WithGiftCards(gcRepo)
	opsHandler := opstransport.New(opsSvc)

	c := &container.Container{ChannelClientService: clientSvc}
	router := gin.New()
	ch := router.Group("/api/v1/channel")
	ch.Use(ChannelAPIAuthMiddleware(c))
	ch.GET("/needtoken-ops/gift-cards", opsHandler.GiftCards)
	ch.POST("/needtoken-ops/gift-cards/generate", opsHandler.GenerateGiftCards)

	// 1. Unauthenticated call is rejected (401).
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/channel/needtoken-ops/gift-cards", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without channel headers, got %d", w.Code)
	}

	// 2. Non-ops (telegram) channel credential signed validly is rejected with 403 scope error.
	bodyBytes := []byte(`{"name":"VIP Card","quantity":2,"amount":"88.00"}`)
	nowTs := time.Now().Unix()
	path := "/api/v1/channel/needtoken-ops/gift-cards/generate"
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(channelHeaderKey, botCred.ChannelKey)
	req.Header.Set(channelHeaderTimestamp, strconv.FormatInt(nowTs, 10))
	req.Header.Set(channelHeaderSignature, upstream.Sign(botCred.ChannelSecret, http.MethodPost, path, nowTs, bodyBytes))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-ops channel credential, got %d: %s", w.Code, w.Body.String())
	}

	// 3. Dedicated needtoken_ops credential generates batch & returns plaintext cards + batch_no.
	req = httptest.NewRequest(http.MethodPost, path, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(channelHeaderKey, opsCred.ChannelKey)
	req.Header.Set(channelHeaderTimestamp, strconv.FormatInt(nowTs, 10))
	req.Header.Set(channelHeaderSignature, upstream.Sign(opsCred.ChannelSecret, http.MethodPost, path, nowTs, bodyBytes))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for ops gift card generate, got %d: %s", w.Code, w.Body.String())
	}
	var genEnvelope struct {
		StatusCode int                                 `json:"status_code"`
		Data       opscontract.GenerateGiftCardsResult `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &genEnvelope); err != nil {
		t.Fatalf("decode generate response: %v", err)
	}
	if genEnvelope.StatusCode != 0 || !strings.HasPrefix(genEnvelope.Data.BatchNo, "GCB") || len(genEnvelope.Data.Cards) != 2 {
		t.Fatalf("unexpected generate payload: %+v", genEnvelope)
	}

	// 4. GET /needtoken-ops/gift-cards returns generated cards.
	listPath := "/api/v1/channel/needtoken-ops/gift-cards"
	req = httptest.NewRequest(http.MethodGet, listPath+"?page=1&page_size=10", nil)
	req.Header.Set(channelHeaderKey, opsCred.ChannelKey)
	req.Header.Set(channelHeaderTimestamp, strconv.FormatInt(nowTs, 10))
	req.Header.Set(channelHeaderSignature, upstream.Sign(opsCred.ChannelSecret, http.MethodGet, listPath, nowTs, nil))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for ops gift card list, got %d: %s", w.Code, w.Body.String())
	}
	var listEnvelope struct {
		StatusCode int                         `json:"status_code"`
		Data       opscontract.GiftCardsResult `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listEnvelope); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if listEnvelope.Data.Total != 2 || len(listEnvelope.Data.Cards) != 2 {
		t.Fatalf("unexpected list payload: %+v", listEnvelope)
	}
}
