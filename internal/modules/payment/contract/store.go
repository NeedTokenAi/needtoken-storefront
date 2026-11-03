package contract

import (
	"time"

	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"
)

// Store 是支付记录持久化与事务边界。
type Store interface {
	Create(payment *paymentdomain.Payment) error
	ListCheckoutReservations(rootOrderID uint) ([]paymentdomain.CheckoutReservation, error)
	CreateCheckoutReservation(record paymentdomain.CheckoutReservation) (*paymentdomain.CheckoutReservation, error)
	SaveCheckoutReservation(record paymentdomain.CheckoutReservation) (*paymentdomain.CheckoutReservation, error)
	RecordPaidFulfillment(orderID uint, now time.Time) error
	BackfillPaidFulfillment(now time.Time, limit int) error
	ListPendingPaidFulfillment(now time.Time, limit int) ([]paymentdomain.PaidFulfillmentEvent, error)
	ClaimPaidFulfillment(orderID uint, leaseID string, now time.Time) (bool, error)
	FinishPaidFulfillment(orderID uint, leaseID string, completed bool, now time.Time) error
	Update(payment *paymentdomain.Payment) error
	GetByID(id uint) (*paymentdomain.Payment, error)
	GetByIDs(ids []uint) ([]paymentdomain.Payment, error)
	GetByGatewayOrderNo(gatewayOrderNo string) (*paymentdomain.Payment, error)
	GetLatestByProviderRef(providerRef string) (*paymentdomain.Payment, error)
	ListByOrderID(orderID uint) ([]paymentdomain.Payment, error)
	GetLatestPendingByOrder(orderID uint, now time.Time) (*paymentdomain.Payment, error)
	GetLatestPendingByOrderChannel(orderID, channelID uint, now time.Time) (*paymentdomain.Payment, error)
	SupersedePendingByOrderID(orderID, replacementPaymentID uint, supersededAt time.Time) (int64, error)
	ExpirePendingByOrderIDs(orderIDs []uint, expiredAt time.Time) (int64, error)
	ListAdmin(filter ListFilter) ([]paymentdomain.Payment, int64, error)
	GetByIDForUpdate(id uint) (*paymentdomain.Payment, error)
	WithinTransaction(fn func(Transaction) error) error
}

// ChannelStore 是支付渠道配置持久化端口。
type ChannelStore interface {
	Create(channel *paymentdomain.PaymentChannel) error
	Update(channel *paymentdomain.PaymentChannel) error
	Delete(id uint) error
	GetByID(id uint) (*paymentdomain.PaymentChannel, error)
	ListByIDs(ids []uint) ([]paymentdomain.PaymentChannel, error)
	List(filter ChannelListFilter) ([]paymentdomain.PaymentChannel, int64, error)
}

// Transaction 复用订单工作单元的全部领域端口，并追加支付聚合。
// 应用层只接触端口，不感知 GORM 事务句柄。
type Transaction interface {
	ordercontract.Transaction
	Payments() Store
	PaymentChannels() ChannelStore
}
