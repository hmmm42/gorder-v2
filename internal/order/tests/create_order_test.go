package tests

import (
	"context"
	"fmt"
	"log"
	"testing"

	sw "github.com/hmmm42/gorder-v2/common/client/order"
	_ "github.com/hmmm42/gorder-v2/common/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var (
	ctx    = context.Background()
	server = fmt.Sprintf("http://%s/api", viper.GetString("order.http-addr"))
)

func TestMain(m *testing.M) {
	before()
	m.Run()
}

func before() {
	log.Printf("server=%s", server)
}

func TestCreateOrder_success(t *testing.T) {
	response := getResponse(t, "123", sw.PostCustomerCustomerIdOrdersJSONRequestBody{
		CustomerId: "123",
		Items: []sw.ItemWithQuantity{
			{
				Id:       "prod_RsZJd3tytJeDWz",
				Quantity: 1,
			},
			{
				Id:       "prod_RsFRFxXTR91vZl",
				Quantity: 10,
			},
		},
	})

	t.Logf("body=%s", string(response.Body))
	assert.Equal(t, 200, response.StatusCode())
	assert.Equal(t, 0, response.JSON200.Errno)
}

func TestCreateOrder_invalidParams(t *testing.T) {
	response := getResponse(t, "123", sw.PostCustomerCustomerIdOrdersJSONRequestBody{
		CustomerId: "123",
		Items:      nil,
	})

	t.Logf("body=%s", string(response.Body))
	assert.Equal(t, 200, response.StatusCode())
	assert.Equal(t, 2, response.JSON200.Errno)
}

func getResponse(t *testing.T, customerID string, body sw.PostCustomerCustomerIdOrdersJSONRequestBody) *sw.PostCustomerCustomerIdOrdersResponse {
	t.Helper()
	client, err := sw.NewClientWithResponses(server)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	response, err := client.PostCustomerCustomerIdOrdersWithResponse(ctx, customerID, body)
	if err != nil {
		t.Fatalf("failed to get response: %v", err)
	}
	return response
}
