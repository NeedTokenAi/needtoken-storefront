package publicconfigwiring

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	paymentapp "github.com/dujiao-next/internal/modules/payment/application"
	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/jsonslice"
)

type displayChannelStore struct {
	paymentcontract.ChannelStore
	channels []paymentdomain.PaymentChannel
	err      error
	filter   paymentcontract.ChannelListFilter
}

func (s *displayChannelStore) List(filter paymentcontract.ChannelListFilter) ([]paymentdomain.PaymentChannel, int64, error) {
	s.filter = filter
	return s.channels, int64(len(s.channels)), s.err
}

func TestDisplayPaymentChannelsExposeOnlyActiveBrandMetadata(t *testing.T) {
	deletedAt := time.Now()
	store := &displayChannelStore{channels: []paymentdomain.PaymentChannel{
		{ID: 1, Name: "Alipay", ChannelType: constants.PaymentChannelTypeAlipay, IsActive: true},
		{
			ID: 2, Name: "WeChat", ChannelType: constants.PaymentChannelTypeWechat, IsActive: true,
			PaymentRoles: jsonslice.Strings{constants.PaymentRoleMember}, MemberLevels: jsonslice.Uints{3},
			ConfigJSON: jsonmap.JSON{"merchant_secret": "must-not-appear"},
		},
		{ID: 3, Name: "Crypto top-up", ChannelType: "tron-usdt", IsActive: true, PaymentTypes: jsonslice.Strings{constants.PaymentTypeWallet}},
		{ID: 4, Name: "Disabled", ChannelType: constants.PaymentChannelTypeAlipay},
		{ID: 5, Name: "Deleted", ChannelType: constants.PaymentChannelTypeWechat, IsActive: true, DeletedAt: &deletedAt},
		{ID: 6, Name: "Balance", ChannelType: constants.PaymentChannelTypeBalance, IsActive: true},
	}}
	adapter := publicConfigPaymentAdapter{payments: paymentapp.NewPaymentService(paymentapp.PaymentServiceOptions{ChannelStore: store})}
	got, err := adapter.GetDisplayPaymentChannels()
	if err != nil {
		t.Fatal(err)
	}
	want := []map[string]interface{}{
		{"id": uint(1), "name": "Alipay", "channel_type": "alipay"},
		{"id": uint(2), "name": "WeChat", "channel_type": "wechat"},
		{"id": uint(3), "name": "Crypto top-up", "channel_type": "tron-usdt"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("display metadata = %#v, want %#v", got, want)
	}
	if !store.filter.ActiveOnly || store.filter.Page != 1 || store.filter.PageSize != 200 {
		t.Fatalf("display query must only request active channels: %#v", store.filter)
	}

	// Showing a configured brand does not make its channel eligible for guests.
	store.channels = store.channels[:3]
	orderChannels, err := adapter.GetOrderPaymentChannels()
	if err != nil {
		t.Fatal(err)
	}
	if len(orderChannels) != 1 || orderChannels[0]["id"] != uint(1) {
		t.Fatalf("guest order eligibility changed: %#v", orderChannels)
	}
}

func TestDisplayPaymentChannelsKeepEmptyAndFailedConfigurationDistinct(t *testing.T) {
	store := &displayChannelStore{}
	adapter := publicConfigPaymentAdapter{payments: paymentapp.NewPaymentService(paymentapp.PaymentServiceOptions{ChannelStore: store})}
	got, err := adapter.GetDisplayPaymentChannels()
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty configured channels = %#v, %v", got, err)
	}
	store.err = errors.New("database unavailable")
	if _, err := adapter.GetDisplayPaymentChannels(); !errors.Is(err, store.err) {
		t.Fatalf("configuration error = %v", err)
	}
}
