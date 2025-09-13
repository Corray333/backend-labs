package httptransport

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/corray333/backend-labs/order/internal/service/models/order"
	"github.com/corray333/backend-labs/order/internal/service/models/orderitem"
	"github.com/corray333/backend-labs/order/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/gorilla/schema"
	"github.com/spf13/viper"
)

const (
	// Version is the version of the HTTPTransport.
	Version = "1.0.0"
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
	var orders []order.Order
	if err := json.NewDecoder(r.Body).Decode(&orders); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		slog.Error("Error decoding request body for batch insert", "error", err)
		return
	}

	insertedOrders, err := h.service.BatchInsert(r.Context(), orders)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("Error performing batch insert", "error", err)
		return
	}

	if err := json.NewEncoder(w).Encode(insertedOrders); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("Error sending response for batch insert", "error", err)
	}
}

type queryOrdersRequest struct {
	Ids         []int64 `schema:"ids,omitempty"`
	CustomerIds []int64 `schema:"customerIds,omitempty"`
	Limit       int     `schema:"limit,omitempty"`
	Offset      int     `schema:"offset,omitempty"`
}

func (q *queryOrdersRequest) ToModel() orderitem.QueryOrderItemsModel {
	return orderitem.QueryOrderItemsModel{
		Ids:         q.Ids,
		CustomerIds: q.CustomerIds,
		Limit:       q.Limit,
		Offset:      q.Offset,
	}
}

func (h *HTTPTransport) getOrders(w http.ResponseWriter, r *http.Request) {
	decoder := schema.NewDecoder()
	query := &queryOrdersRequest{}
	err := decoder.Decode(query, r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		slog.Error("Error decoding request", "error", err)
		return
	}

	orders, err := h.service.GetOrders(r.Context(), query.ToModel())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("Error getting orders", "error", err)
		return
	}

	if err := json.NewEncoder(w).Encode(orders); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("Error sending response", "error", err)
	}
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
