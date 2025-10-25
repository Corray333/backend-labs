package httptransport

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	v1 "github.com/corray333/backend-labs/order/pkg/api/v1"
	"github.com/corray333/backend-labs/order/pkg/http/middleware/trace"
	"github.com/corray333/backend-labs/order/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/spf13/viper"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// HTTPTransport represents the HTTP transport layer.
type HTTPTransport struct {
	server     *http.Server
	router     *chi.Mux
	grpcServer v1.OrderServiceServer
	gatewayMux *runtime.ServeMux
}

// NewHTTPTransport creates a new HTTPTransport.
func NewHTTPTransport(grpcServer v1.OrderServiceServer) *HTTPTransport {
	router := newRouter()
	gatewayMux := runtime.NewServeMux()
	server := newServer(router)

	return &HTTPTransport{
		server:     server,
		router:     router,
		grpcServer: grpcServer,
		gatewayMux: gatewayMux,
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
func (h *HTTPTransport) RegisterRoutes() {
	// Register grpc-gateway handlers
	if err := v1.RegisterOrderServiceHandlerServer(context.Background(), h.gatewayMux, h.grpcServer); err != nil {
		slog.Error("Failed to register grpc-gateway handler", "error", err)
		panic(err)
	}

	// Serve Swagger UI
	h.router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/v1/order.swagger.json"),
	))

	// Serve Swagger JSON file
	h.router.Handle(
		"/swagger/v1/*",
		http.StripPrefix("/swagger/", http.FileServer(http.Dir("./api/swagger"))),
	)

	// Mount the gateway mux to the router
	h.router.Mount("/", h.gatewayMux)
}

// newRouter creates a new router for the HTTPTransport.
func newRouter() *chi.Mux {
	router := chi.NewMux()
	router.Use(middleware.RequestID)
	router.Use(logger.NewLoggerMiddleware(slog.Default()))
	router.Use(trace.NewTraceMiddleware)

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
