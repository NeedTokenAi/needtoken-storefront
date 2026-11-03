package application

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	mappingcontract "github.com/dujiao-next/internal/modules/catalog/mapping/contract"
	mappingdomain "github.com/dujiao-next/internal/modules/catalog/mapping/domain"
	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"

	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/dujiao-next/internal/upstream"

	"github.com/shopspring/decimal"
)

func TestConvertUpstreamWholesalePricesRemapsUpstreamSKUScope(t *testing.T) {
	tiers := convertUpstreamWholesalePrices(productdomain.WholesalePriceTiers{
		{SKUID: 201, MinQuantity: 5, UnitPrice: money.FromDecimal(decimal.NewFromInt(80))},
	}, decimal.NewFromInt(1), decimal.Zero, "none", buildUpstreamWholesaleSKUIndex(
		[]productdomain.ProductSKU{{ID: 11, SKUCode: "SKU-A"}},
		[]upstream.UpstreamSKU{{ID: 201, SKUCode: "SKU-A"}},
		nil,
	))

	if len(tiers) != 1 {
		t.Fatalf("expected 1 tier, got %d", len(tiers))
	}
	if tiers[0].SKUID != 11 || tiers[0].SKUCode != "SKU-A" {
		t.Fatalf("expected upstream SKU scope to be remapped, got %+v", tiers[0])
	}
}

func TestConvertUpstreamWholesalePricesDropsUnmappedUpstreamSKUID(t *testing.T) {
	tiers := convertUpstreamWholesalePrices(productdomain.WholesalePriceTiers{
		{SKUID: 201, MinQuantity: 5, UnitPrice: money.FromDecimal(decimal.NewFromInt(80))},
	}, decimal.NewFromInt(1), decimal.Zero, "none")

	if len(tiers) != 0 {
		t.Fatalf("expected unmapped upstream sku_id tier to be dropped, got %+v", tiers)
	}
}

// pagedProductsAdapter 只实现 ListProducts：前 total 条商品按页返回，ignorePage 模拟上游忽略 page 参数。
type pagedProductsAdapter struct {
	upstream.Adapter
	total      int
	ignorePage bool
	calls      int
}

func (a *pagedProductsAdapter) ListProducts(_ context.Context, opts upstream.ListProductsOpts) (*upstream.ProductListResult, error) {
	a.calls++
	page := opts.Page
	if a.ignorePage {
		page = 1
	}
	var items []upstream.UpstreamProduct
	for id := (page-1)*opts.PageSize + 1; id <= a.total && len(items) < opts.PageSize; id++ {
		items = append(items, upstream.UpstreamProduct{ID: uint(id), CategoryID: uint(id % 3)})
	}
	if a.ignorePage {
		return &upstream.ProductListResult{Total: 1 << 30, Items: items}, nil
	}
	return &upstream.ProductListResult{Total: a.total, Items: items}, nil
}

func TestListAllUpstreamProductsWalksEveryPage(t *testing.T) {
	adapter := &pagedProductsAdapter{total: 120}
	products, err := listAllUpstreamProducts(context.Background(), adapter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 120 || adapter.calls != 3 {
		t.Fatalf("expected 120 products in 3 calls, got %d in %d", len(products), adapter.calls)
	}
}

func TestListAllUpstreamProductsStopsAtPageLimit(t *testing.T) {
	adapter := &pagedProductsAdapter{total: 50, ignorePage: true}
	if _, err := listAllUpstreamProducts(context.Background(), adapter); err == nil {
		t.Fatal("expected error when upstream pagination never terminates")
	}
	if adapter.calls != maxUpstreamProductPages {
		t.Fatalf("expected %d calls, got %d", maxUpstreamProductPages, adapter.calls)
	}
}

func reservationContract() jsonmap.JSON {
	return jsonmap.JSON{"version": 1, "sku": "ChatGPT-Plus-1mo", "region": "US", "currency": "USD", "required_minor": "2500", "ttl_seconds": 900}
}

func TestReservationAssignmentUsesMappingIdentityAndHidesServerFields(t *testing.T) {
	product := &upstream.UpstreamProduct{RequiresReservation: true, ReservationContract: reservationContract()}
	local := productdomain.ProductSKU{IsActive: true}
	if err := assignReservationMetadata(&local, product, upstream.UpstreamSKU{ID: 901}, 77); err != nil {
		t.Fatal(err)
	}
	if !local.RequiresReservation || !local.ReservationPolicyKnown || local.ReservationConnectionID != 77 || local.ReservationUpstreamSKUID != 901 || local.ReservationContractJSON["required_minor"] != "2500" {
		t.Fatal(local)
	}
	raw, _ := json.Marshal(local)
	for _, key := range []string{"requires_reservation", "reservation_policy_known", "reservation_contract", "reservation_connection_id", "reservation_upstream_sku_id"} {
		if strings.Contains(string(raw), key) {
			t.Fatal("public SKU exposed server field", key)
		}
	}
	product.ReservationContract["required_minor"] = "1"
	if local.ReservationContractJSON["required_minor"] != "2500" {
		t.Fatal("local terms changed with fetched map")
	}
	if err := assignReservationMetadata(&local, &upstream.UpstreamProduct{}, upstream.UpstreamSKU{ID: 901}, 77); err == nil || local.IsActive || !local.RequiresReservation || local.ReservationPolicyKnown {
		t.Fatal("vanished terms downgraded scarcity", local, err)
	}
}

func TestReservationAssignmentRequiresActualUpstreamSKUAndConnection(t *testing.T) {
	product := &upstream.UpstreamProduct{RequiresReservation: true, ReservationContract: reservationContract()}
	for _, pair := range [][2]uint{{0, 901}, {77, 0}} {
		local := productdomain.ProductSKU{IsActive: true}
		if err := assignReservationMetadata(&local, product, upstream.UpstreamSKU{ID: pair[1]}, pair[0]); err == nil || !local.RequiresReservation || local.IsActive {
			t.Fatal(local, err)
		}
	}
	product.ReservationContract = nil
	local := productdomain.ProductSKU{IsActive: true}
	if err := assignReservationMetadata(&local, product, upstream.UpstreamSKU{ID: 901}, 77); err == nil || !local.RequiresReservation || local.IsActive {
		t.Fatal(local, err)
	}
}

type reservationMappingRepo struct {
	mappingcontract.MappingRepository
	row *mappingdomain.Mapping
}

func (r *reservationMappingRepo) GetByID(uint) (*mappingdomain.Mapping, error) { return r.row, nil }
func (r *reservationMappingRepo) GetByConnectionAndUpstreamID(uint, uint) (*mappingdomain.Mapping, error) {
	return nil, nil
}
func (r *reservationMappingRepo) Update(m *mappingdomain.Mapping) error { r.row = m; return nil }

type reservationSKUMappingRepo struct {
	mappingcontract.SKUMappingRepository
	rows []mappingdomain.SKUMapping
}

func (r *reservationSKUMappingRepo) ListByProductMapping(uint) ([]mappingdomain.SKUMapping, error) {
	return r.rows, nil
}
func (r *reservationSKUMappingRepo) Update(mapping *mappingdomain.SKUMapping) error {
	for i := range r.rows {
		if r.rows[i].LocalSKUID == mapping.LocalSKUID {
			r.rows[i] = *mapping
		}
	}
	return nil
}

type reservationProductRepo struct {
	mappingcontract.ProductRepository
	row *productdomain.Product
}

func (r *reservationProductRepo) GetByID(string) (*productdomain.Product, error) { return r.row, nil }
func (r *reservationProductRepo) Update(p *productdomain.Product) error          { r.row = p; return nil }

type reservationSKURepo struct {
	mappingcontract.SKURepository
	rows map[uint]*productdomain.ProductSKU
}

func (r *reservationSKURepo) GetByID(id uint) (*productdomain.ProductSKU, error) {
	return r.rows[id], nil
}
func (r *reservationSKURepo) ListByProduct(productID uint, onlyActive bool) ([]productdomain.ProductSKU, error) {
	out := []productdomain.ProductSKU{}
	for _, sku := range r.rows {
		if sku.ProductID == productID && (!onlyActive || sku.IsActive) {
			out = append(out, *sku)
		}
	}
	return out, nil
}
func (r *reservationSKURepo) Update(p *productdomain.ProductSKU) error { r.rows[p.ID] = p; return nil }

type reservationCatalogAdapter struct {
	upstream.Adapter
	product *upstream.UpstreamProduct
}

func (a *reservationCatalogAdapter) GetProduct(context.Context, uint) (*upstream.UpstreamProduct, error) {
	return a.product, nil
}

type reservationConnections struct{ adapter upstream.Adapter }

func (c reservationConnections) GetByID(uint) (*siteconnectiondomain.Connection, error) {
	return &siteconnectiondomain.Connection{ID: 77}, nil
}
func (c reservationConnections) GetAdapter(*siteconnectiondomain.Connection) (upstream.Adapter, error) {
	return c.adapter, nil
}

func TestMalformedRequiredCatalogCannotImportOrLeaveExistingSKUActive(t *testing.T) {
	product := &upstream.UpstreamProduct{ID: 22, IsActive: true, RequiresReservation: true, SKUs: []upstream.UpstreamSKU{{ID: 901, IsActive: true}}}
	mappings := &reservationMappingRepo{row: &mappingdomain.Mapping{ID: 1, ConnectionID: 77, LocalProductID: 11, UpstreamProductID: 22}}
	products := &reservationProductRepo{row: &productdomain.Product{ID: 11, IsActive: true}}
	skus := &reservationSKURepo{rows: map[uint]*productdomain.ProductSKU{111: {ID: 111, ProductID: 11, IsActive: true}}}
	svc := &Service{mappings: mappings, products: products, skus: skus, skuMappings: &reservationSKUMappingRepo{rows: []mappingdomain.SKUMapping{{LocalSKUID: 111, UpstreamSKUID: 901}}}, connections: reservationConnections{adapter: &reservationCatalogAdapter{product: product}}}
	if _, err := svc.ImportUpstreamProduct(77, 22, 0, ""); err == nil {
		t.Fatal("malformed catalog reached import writes")
	}
	if err := svc.SyncProduct(1); err == nil {
		t.Fatal("malformed sync accepted")
	}
	if products.row.IsActive || skus.rows[111].IsActive || !skus.rows[111].RequiresReservation || skus.rows[111].ReservationPolicyKnown || skus.rows[111].ReservationContractJSON != nil || skus.rows[111].ReservationConnectionID != 77 || skus.rows[111].ReservationUpstreamSKUID != 901 {
		t.Fatal("old unrestricted SKU remained sellable", products.row, skus.rows[111])
	}
}

type reservationImportCapture struct {
	product *productdomain.Product
	skus    []productdomain.ProductSKU
	mapping *mappingdomain.Mapping
	links   []mappingdomain.SKUMapping
}
type reservationImportProducts struct{ capture *reservationImportCapture }

func (r reservationImportProducts) Create(p *productdomain.Product) error {
	p.ID = 11
	r.capture.product = p
	return nil
}
func (r reservationImportProducts) QuickUpdate(string, map[string]interface{}) error { return nil }

type reservationImportSKUs struct{ capture *reservationImportCapture }

func (r reservationImportSKUs) Create(p *productdomain.ProductSKU) error {
	p.ID = uint(111 + len(r.capture.skus))
	r.capture.skus = append(r.capture.skus, *p)
	return nil
}

type reservationImportMappings struct{ capture *reservationImportCapture }

func (r reservationImportMappings) Create(m *mappingdomain.Mapping) error {
	m.ID = 1
	r.capture.mapping = m
	return nil
}

type reservationImportSKUMappings struct{ capture *reservationImportCapture }

func (r reservationImportSKUMappings) Create(m *mappingdomain.SKUMapping) error {
	r.capture.links = append(r.capture.links, *m)
	return nil
}

type reservationImportTransaction struct{ capture *reservationImportCapture }

func (r reservationImportTransaction) WithinTransaction(fn func(mappingcontract.ImportRepositories) error) error {
	return fn(mappingcontract.ImportRepositories{Products: reservationImportProducts{r.capture}, SKUs: reservationImportSKUs{r.capture}, Mappings: reservationImportMappings{r.capture}, SKUMappings: reservationImportSKUMappings{r.capture}})
}

func TestActualImportPersistsTrustedReservationMetadataWithSKUMap(t *testing.T) {
	product := &upstream.UpstreamProduct{ID: 22, IsActive: true, RequiresReservation: true, ReservationContract: reservationContract(), PriceAmount: "100", SKUs: []upstream.UpstreamSKU{{ID: 901, SKUCode: "US-Plus", PriceAmount: "100", IsActive: true}}}
	capture := &reservationImportCapture{}
	svc := &Service{mappings: &reservationMappingRepo{}, connections: reservationConnections{adapter: &reservationCatalogAdapter{product: product}}, transactions: reservationImportTransaction{capture}}
	if _, err := svc.ImportUpstreamProduct(77, 22, 0, "local-scarce"); err != nil {
		t.Fatal(err)
	}
	if len(capture.skus) != 1 || len(capture.links) != 1 || capture.product.IsActive {
		t.Fatal("import did not remain closed by default", capture)
	}
	sku := capture.skus[0]
	if !sku.RequiresReservation || !sku.ReservationPolicyKnown || sku.ReservationConnectionID != capture.mapping.ConnectionID || sku.ReservationUpstreamSKUID != capture.links[0].UpstreamSKUID || sku.ReservationContractJSON["required_minor"] != "2500" {
		t.Fatal("import lost reservation identity/terms", sku, capture.links)
	}
}

func TestReservationPolicyKnownRequiresFreshSuccessfulCatalogProcessing(t *testing.T) {
	local := productdomain.ProductSKU{IsActive: true}
	if local.ReservationPolicyKnown {
		t.Fatal("legacy zero value claimed a known policy")
	}
	if err := assignReservationMetadata(&local, &upstream.UpstreamProduct{}, upstream.UpstreamSKU{ID: 901}, 77); err != nil || local.ReservationPolicyKnown {
		t.Fatal("omitted legacy policy became known after refresh", local, err)
	}
	if err := assignReservationMetadata(&local, &upstream.UpstreamProduct{ReservationPolicyDeclared: true}, upstream.UpstreamSKU{ID: 901}, 77); err != nil || !local.ReservationPolicyKnown || local.RequiresReservation {
		t.Fatal("successful ordinary catalog pull did not establish policy", local, err)
	}
	if err := assignReservationMetadata(&local, &upstream.UpstreamProduct{RequiresReservation: true}, upstream.UpstreamSKU{ID: 901}, 77); err == nil || local.ReservationPolicyKnown || local.IsActive {
		t.Fatal("failed refresh claimed known policy", local, err)
	}
}

func TestOrdinaryDefaultSKUImportAndRefreshEstablishKnownNonScarcePolicy(t *testing.T) {
	product := &upstream.UpstreamProduct{ID: 22, IsActive: true, PriceAmount: "100", ReservationPolicyDeclared: true}
	capture := &reservationImportCapture{}
	svc := &Service{mappings: &reservationMappingRepo{}, connections: reservationConnections{adapter: &reservationCatalogAdapter{product: product}}, transactions: reservationImportTransaction{capture}}
	if _, err := svc.ImportUpstreamProduct(77, 22, 0, "local-ordinary"); err != nil {
		t.Fatal(err)
	}
	if len(capture.skus) != 1 || !capture.skus[0].ReservationPolicyKnown || capture.skus[0].RequiresReservation || capture.skus[0].SKUCode != productdomain.DefaultSKUCode {
		t.Fatal("ordinary import policy unknown", capture.skus)
	}
	mappings := &reservationMappingRepo{row: &mappingdomain.Mapping{ID: 1, ConnectionID: 77, LocalProductID: 11, UpstreamProductID: 22}}
	products := &reservationProductRepo{row: &productdomain.Product{ID: 11, IsActive: true}}
	skus := &reservationSKURepo{rows: map[uint]*productdomain.ProductSKU{111: {ID: 111, ProductID: 11, SKUCode: productdomain.DefaultSKUCode, IsActive: true}}}
	svc = &Service{mappings: mappings, products: products, skus: skus, skuMappings: &reservationSKUMappingRepo{}, connections: reservationConnections{adapter: &reservationCatalogAdapter{product: product}}}
	if err := svc.SyncProduct(1); err != nil {
		t.Fatal(err)
	}
	if !skus.rows[111].ReservationPolicyKnown || skus.rows[111].RequiresReservation || !skus.rows[111].IsActive {
		t.Fatal("ordinary default SKU did not acquire a fresh policy", skus.rows[111])
	}
}

func TestBatchSyncRefreshesPolicyBeforePriceParsingAndNeverInfersMissingDeclaration(t *testing.T) {
	for _, declared := range []bool{false, true} {
		mappings := &reservationMappingRepo{row: &mappingdomain.Mapping{ID: 1, ConnectionID: 77, LocalProductID: 11, UpstreamProductID: 22}}
		products := &reservationProductRepo{row: &productdomain.Product{ID: 11, IsActive: true}}
		skus := &reservationSKURepo{rows: map[uint]*productdomain.ProductSKU{111: {ID: 111, ProductID: 11, SKUCode: "US-Plus", IsActive: true, ReservationPolicyKnown: true}}}
		links := &reservationSKUMappingRepo{rows: []mappingdomain.SKUMapping{{LocalSKUID: 111, UpstreamSKUID: 901}}}
		svc := &Service{mappings: mappings, products: products, skus: skus, skuMappings: links}
		product := &upstream.UpstreamProduct{ID: 22, IsActive: true, ReservationPolicyDeclared: declared, SKUs: []upstream.UpstreamSKU{{ID: 901, SKUCode: "US-Plus", PriceAmount: "invalid-price", IsActive: true}}}
		now := time.Now()
		svc.syncProductFromData(mappings.row, &siteconnectiondomain.Connection{ID: 77}, product, &now)
		if skus.rows[111].ReservationPolicyKnown != declared || skus.rows[111].RequiresReservation {
			t.Fatal("batch sync inferred a policy or skipped its update after price error", declared, skus.rows[111])
		}
	}
}
