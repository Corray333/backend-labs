package httptransport

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/corray333/backend-labs/order/internal/service/models/order"
	"github.com/corray333/backend-labs/order/internal/service/models/orderitem"
	createorder "github.com/corray333/backend-labs/order/internal/transport/http/v1/create_order"
	listorders "github.com/corray333/backend-labs/order/internal/transport/http/v1/list_orders"
	"github.com/corray333/backend-labs/order/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/spf13/viper"
)

// service is an interface for the service layer.
type service interface {
	GetOrders(ctx context.Context, model orderitem.QueryOrderItemsModel) ([]order.Order, error)
	BatchInsert(ctx context.Context, orders []order.Order) ([]order.Order, error)
}

// HTTPTransport represents the HTTP transport layer.
type HTTPTransport struct {
	server  *http.Server
	router  *chi.Mux
	service service
}

// NewHTTPTransport creates a new HTTPTransport.
func NewHTTPTransport(service service) *HTTPTransport {
	router := newRouter()
	server := newServer(router)

	return &HTTPTransport{
		server:  server,
		router:  router,
		service: service,
	}
}

// Run starts the HTTP server.
func (h *HTTPTransport) Run() error {
	return h.server.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server.
func (h *HTTPTransport) Shutdown(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}

// RegisterRoutes registers the routes for the HTTPTransport.

// RegisterRoutes registers the routes for the HTTPTransport.
func (h *HTTPTransport) RegisterRoutes() {
	h.router.Get("/v1/api/orders", h.getOrders)
	h.router.Post("/v1/api/orders", h.batchInsert)
}

// batchInsert handles the batch insert request.
func (h *HTTPTransport) batchInsert(w http.ResponseWriter, r *http.Request) {
	createorder.BatchInsert(w, r, h.service)
}

// getOrders handles the get orders request.
func (h *HTTPTransport) getOrders(w http.ResponseWriter, r *http.Request) {
	listorders.ListOrders(w, r, h.service)
}

// newRouter creates a new router for the HTTPTransport.
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

const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 120 * time.Second
)

// newServer creates a new HTTP server.
func newServer(router http.Handler) *http.Server {
	return &http.Server{
		Addr:              "0.0.0.0:" + viper.GetString("server.http.port"),
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}
