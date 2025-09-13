package app

import (
	"github.com/corray333/backend-labs/order/internal/dal/postgres"
	"github.com/corray333/backend-labs/order/internal/service/services/ordersvc"
	httptransport "github.com/corray333/backend-labs/order/internal/transport/http"
)

type App struct {
	orderSvc  *ordersvc.OrderService
	transport *httptransport.HTTPTransport
}

func MustNewApp() *App {
	postgresClient := postgres.MustNewClient()

	orderSvc := ordersvc.MustNewOrderService(
		ordersvc.WithPostgresClient(postgresClient),
	)

	transport := httptransport.NewHTTPTransport(orderSvc)
	transport.RegisterRoutes()

	return &App{
		orderSvc:  orderSvc,
		transport: transport,
	}
}

func (a *App) Run() {
	err := a.transport.Run()
	if err != nil {
		panic(err)
	}
}
