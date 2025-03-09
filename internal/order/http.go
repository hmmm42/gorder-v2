package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/hmmm42/gorder-v2/common"
	client "github.com/hmmm42/gorder-v2/common/client/order"
	"github.com/hmmm42/gorder-v2/order/app"
	"github.com/hmmm42/gorder-v2/order/app/command"
	"github.com/hmmm42/gorder-v2/order/app/dto"
	"github.com/hmmm42/gorder-v2/order/app/query"
	"github.com/hmmm42/gorder-v2/order/convertor"
)

type HTTPServer struct {
	app app.Application
	common.BaseResponse
}

func (H HTTPServer) PostCustomerCustomerIdOrders(c *gin.Context, customerID string) {
	var (
		req  client.CreateOrderRequest
		resp dto.CreateOrderResponse
		err  error
	)
	defer func() {
		H.Response(c, err, &resp)
	}()

	if err = c.ShouldBindJSON(&req); err != nil {
		return
	}
	if err = H.validate(req); err != nil {
		return
	}
	r, err := H.app.Commands.CreateOrder.Handle(c.Request.Context(), command.CreateOrder{
		CustomerID: req.CustomerId,
		Items:      convertor.NewItemWithQuantityConvertor().ClientsToEntities(req.Items),
	})
	if err != nil {
		return
	}

	resp = dto.CreateOrderResponse{
		OrderID:     r.OrderID,
		CustomerID:  req.CustomerId,
		RedirectURL: fmt.Sprintf("http://localhost:8282/success?customerID=%s&orderID=%s", req.CustomerId, r.OrderID),
	}
}

func (H HTTPServer) GetCustomerCustomerIdOrdersOrderId(c *gin.Context, customerID string, orderID string) {
	var (
		err  error
		resp interface{}
	)
	defer func() {
		H.Response(c, err, resp)
	}()
	o, err := H.app.Queries.GetCustomerOrder.Handle(c.Request.Context(), query.GetCustomerOrder{
		CustomerID: customerID,
		OrderID:    orderID,
	})
	if err != nil {
		return
	}
	resp = convertor.NewOrderConvertor().EntityToClient(o)
}

func (H HTTPServer) validate(req client.CreateOrderRequest) error {
	for _, item := range req.Items {
		if item.Quantity <= 0 {
			return fmt.Errorf("quantity must be positive")
		}
	}
	return nil
}
