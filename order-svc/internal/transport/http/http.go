package httptransport

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/corray333/backend-labs/order/internal/service/models/order"
	"github.com/corray333/backend-labs/order/internal/service/models/orderitem"
	createorder "github.com/corray333/backend-labs/order/internal/transport/http/create_order"
	listorders "github.com/corray333/backend-labs/order/internal/transport/http/list_orders"
	"github.com/corray333/backend-labs/order/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/spf13/viper"
)

type service interface {
	GetOrders(ctx context.Context, model orderitem.QueryOrderItemsModel) ([]order.Order, error)
	BatchInsert(ctx context.Context, orders []order.Order) ([]order.Order, error)
}

type HTTPTransport struct {
	server  *http.Server
	router  *chi.Mux
	service service
}

func NewHTTPTransport(service service) *HTTPTransport {
	router := newRouter()
	server := newServer(router)
	return &HTTPTransport{
		server:  server,
		router:  router,
		service: service,
	}
}

func (h *HTTPTransport) Run() error {
	return h.server.ListenAndServe()
}

// RegisterRoutes registers the routes for the HTTPTransport.
func (h *HTTPTransport) RegisterRoutes() {
	h.router.Route("/api", func(r chi.Router) {
		r.Get("/orders", h.getOrders)
		r.Post("/orders", h.batchInsert)
	})
}

func (h *HTTPTransport) batchInsert(w http.ResponseWriter, r *http.Request) {
	createorder.BatchInsert(w, r, h.service)
}

func (h *HTTPTransport) getOrders(w http.ResponseWriter, r *http.Request) {
	listorders.ListOrders(w, r, h.service)
}

func newRouter() *chi.Mux {
	router := chi.NewMux()
	router.Use(middleware.RequestID)
	router.Use(logger.NewLoggerMiddleware(slog.Default()))

	allowedOrigins := viper.GetStringSlice("server.http.cors.allowed_origins")
	allowedMethods := viper.GetStringSlice("server.http.cors.allowed_methods")
	allowedHeaders := viper.GetStringSlice("server.http.cors.allowed_headers")
	exposedHeaders := viper.GetStringSlice("server.http.cors.exposed_headers")
	allowCredentials := viper.GetBool("server.http.cors.allow_credentials")
	maxAge := viper.GetInt("server.http.cors.max_age")

	c := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   allowedMethods,
		AllowedHeaders:   allowedHeaders,
		ExposedHeaders:   exposedHeaders,
		AllowCredentials: allowCredentials,
		MaxAge:           maxAge,
	})

	router.Use(c.Handler)

	return router
}

func newServer(router http.Handler) *http.Server {
	return &http.Server{
		Addr:    "0.0.0.0:" + viper.GetString("server.http.port"),
		Handler: router,
	}
}
