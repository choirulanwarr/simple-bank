package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/choirulanwar/simple-bank/api"
	"github.com/choirulanwar/simple-bank/api/pb"
	"github.com/choirulanwar/simple-bank/db/sqlc"
	"github.com/choirulanwar/simple-bank/internal/cache"
	"github.com/choirulanwar/simple-bank/internal/config"
	"github.com/choirulanwar/simple-bank/internal/middleware"
	"github.com/choirulanwar/simple-bank/internal/repository"
	"github.com/choirulanwar/simple-bank/pkg/token"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Setup logger with configurable level
	logLevel := new(slog.LevelVar)
	switch cfg.LogLevel {
	case "debug":
		logLevel.Set(slog.LevelDebug)
	case "info":
		logLevel.Set(slog.LevelInfo)
	case "warn":
		logLevel.Set(slog.LevelWarn)
	case "error":
		logLevel.Set(slog.LevelError)
	default:
		logLevel.Set(slog.LevelInfo)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL())
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to PostgreSQL")

	// Initialize cache
	redisAddr := fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort)
	cacheClient := cache.New(redisAddr)
	defer func() { _ = cacheClient.Close() }()
	if err := cacheClient.Ping(ctx); err != nil {
		slog.Warn("redis not available, caching disabled", "error", err)
	} else {
		slog.Info("redis connected, caching enabled", "addr", redisAddr)
	}

	// Initialize layers
	store := sqlc.New(pool)
	repo := repository.NewAccountRepo(store, pool)
	custRepo := repository.NewCustomerRepo(store)
	repo.SetCache(cacheClient)
	custRepo.SetCache(cacheClient)

	// Create token maker
	tokenMaker, err := token.NewPasetoMaker(cfg.TokenSymmetricKey)
	if err != nil {
		slog.Error("failed to create token maker", "error", err)
		os.Exit(1)
	}

	// Create handlers
	customerHandler := api.NewCustomerHandler(custRepo)
	accountHandler := api.NewAccountHandler(repo)
	transactionHandler := api.NewTransactionHandler(repo, custRepo, tokenMaker)

	// gRPC Server with interceptors
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.ChainUnaryInterceptors(
			middleware.RecoveryInterceptor(),
			middleware.LoggingInterceptor(logger),
		)),
	)

	// Register services
	pb.RegisterSimpleBankServer(grpcServer, api.NewSimpleBankServer(
		customerHandler,
		accountHandler,
		transactionHandler,
	))

	reflection.Register(grpcServer)

	// Prometheus metrics
	promRegistry := prometheus.NewRegistry()
	promRegistry.MustRegister(collectors.NewGoCollector())
	promRegistry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	grpcRequests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "grpc_requests_total",
		Help: "Total gRPC requests by method and status",
	}, []string{"method", "status"})
	promRegistry.MustRegister(grpcRequests)

	// Update logging interceptor to track metrics
	loggingInt := middleware.LoggingInterceptor(logger)
	withMetrics := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := loggingInt(ctx, req, info, handler)
		st := "ok"
		if err != nil {
			st = "error"
		}
		grpcRequests.WithLabelValues(info.FullMethod, st).Inc()
		return resp, err
	}

	grpcServer = grpc.NewServer(
		grpc.UnaryInterceptor(middleware.ChainUnaryInterceptors(
			middleware.RecoveryInterceptor(),
			withMetrics,
		)),
	)

	pb.RegisterSimpleBankServer(grpcServer, api.NewSimpleBankServer(
		customerHandler,
		accountHandler,
		transactionHandler,
	))
	reflection.Register(grpcServer)

	// Metrics HTTP server
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.HandlerFor(promRegistry, promhttp.HandlerOpts{}))
	metricsSrv := &http.Server{
		Addr:    cfg.MetricsServerAddress,
		Handler: metricsMux,
	}

	// Start gRPC-web server (handles both gRPC and gRPC-web on same port)
	grpcWebServer := grpcweb.WrapServer(grpcServer,
		grpcweb.WithOriginFunc(func(origin string) bool { return true }),
		grpcweb.WithWebsockets(true),
		grpcweb.WithWebsocketOriginFunc(func(req *http.Request) bool { return true }),
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
	)

	// Wrap with CORS handler for browser preflight requests
	srv := &http.Server{
		Addr: cfg.GRPCServerAddress,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Grpc-Web, X-User-Agent")
			w.Header().Set("Access-Control-Max-Age", "86400")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			grpcWebServer.ServeHTTP(w, r)
		}),
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("gRPC-web server starting", "address", cfg.GRPCServerAddress)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("gRPC-web server failed", "error", err)
			os.Exit(1)
		}
	}()

	go func() {
		slog.Info("metrics server starting", "address", cfg.MetricsServerAddress)
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server failed", "error", err)
			os.Exit(1)
		}
	}()

	sig := <-quit
	slog.Info("shutting down server", "signal", sig.String())
	_ = srv.Close()
	_ = metricsSrv.Close()
}
