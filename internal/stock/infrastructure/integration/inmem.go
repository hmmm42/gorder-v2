// internal/stock/infrastructure/integration/mock_stripe_api.go
package integration

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go/v81"
)

// MockStripeAPI is a mock implementation of the StripeProductFetcher interface.
type MockStripeAPI struct {
	// You can add fields here if you want to make the mock configurable,
	// e.g., a map of productID to mock stripe.Product objects.
	// For simplicity, we'll return some generic data.
	MockProducts map[string]*stripe.Product
}

// NewMockStripeAPI creates a new instance of MockStripeAPI.
func NewMockStripeAPI() *MockStripeAPI {
	// Initialize with some default mock products
	// Ensure these product IDs match what your stress tests might use,
	// or make this map configurable.
	mockProducts := make(map[string]*stripe.Product)

	// Example product 1 (matches one from your init.sql)
	mockProducts["prod_RsZJd3tytJeDWz"] = &stripe.Product{
		ID:   "prod_RsZJd3tytJeDWz",
		Name: "Mocked Coke",
		DefaultPrice: &stripe.Price{
			ID: "price_mock_coke", // Mocked price ID
		},
		// Add other fields if your code consuming this product object uses them
	}

	// Example product 2 (matches one from your init.sql)
	mockProducts["prod_RsFRFxXTR91vZl"] = &stripe.Product{
		ID:   "prod_RsFRFxXTR91vZl",
		Name: "Mocked Fries",
		DefaultPrice: &stripe.Price{
			ID: "price_mock_fries", // Mocked price ID
		},
	}

	// Add a generic fallback for any other product ID encountered during tests
	// This prevents panics if a ProductID is not in the map.
	// Alternatively, you can make it return an error for unknown IDs.

	return &MockStripeAPI{
		MockProducts: mockProducts,
	}
}

// GetProductByID mocks the retrieval of a product by its ID.
// It implements the StripeProductFetcher interface.
func (m *MockStripeAPI) GetProductByID(_ context.Context, productID string) (*stripe.Product, error) {
	// For stress testing, we want this to be fast and always succeed.
	// You can return a generic product or look up from a predefined map.

	if product, ok := m.MockProducts[productID]; ok {
		// fmt.Printf("MockStripeAPI: Returning mocked product for ID %s\n", productID)
		return product, nil
	}

	// Fallback for any other product ID if not found in the predefined map
	// This helps if your tests use arbitrary product IDs not explicitly mocked above.
	// fmt.Printf("MockStripeAPI: Product ID %s not explicitly mocked, returning generic mock product\n", productID)
	return &stripe.Product{
		ID:   productID,
		Name: fmt.Sprintf("Mocked Product %s", productID),
		DefaultPrice: &stripe.Price{
			ID: fmt.Sprintf("price_mock_%s", productID),
		},
		// Populate other necessary fields of stripe.Product that your application might use
		// For example, if you use product.Description, product.Active, etc.
	}, nil
}

// Ensure MockStripeAPI satisfies the StripeProductFetcher interface
var _ StripeProductFetcher = (*MockStripeAPI)(nil)
var _ StripeProductFetcher = (*StripeAPI)(nil) // Also good to assert for the real one
