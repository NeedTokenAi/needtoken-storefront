package contract

import (
	"context"
	"errors"

	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
)

var (
	ErrAuthorizationUnavailable = errors.New("subscription authorization unavailable")
	ErrAuthorizationInvalid     = errors.New("subscription authorization invalid")
	ErrAuthorizationConflict    = errors.New("subscription authorization conflict")
	ErrAuthorizationRateLimited = errors.New("subscription authorization rate limited")
	ErrAuthorizationNotFound    = errors.New("subscription authorization not found")
)

type SubscriptionAuthorizationStore interface {
	Get(context.Context, string) (*orderdomain.SubscriptionAuthorization, error)
	Create(context.Context, *orderdomain.SubscriptionAuthorization, bool) (*orderdomain.SubscriptionAuthorization, bool, error)
	Update(context.Context, string, func(*orderdomain.SubscriptionAuthorization) error) (*orderdomain.SubscriptionAuthorization, error)
}
