package app

import "github.com/hmmm42/gorder-v2/order/app/query"

type Application struct {
	Commands Commands
	Queries  Queries
}

type Commands struct{}

type Queries struct {
	GetCustomerOrder query.GetCustomerOrderHandler
}
