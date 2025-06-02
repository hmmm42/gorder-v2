package integration

import (
	"context"

	"github.com/stripe/stripe-go/v81"
)

type StripeProductFetcher interface {
	GetProductByID(ctx context.Context, productID string) (*stripe.Product, error)
}
