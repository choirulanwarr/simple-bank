package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
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
	defer cacheClient.Close()
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

	lis, err := net.Listen("tcp", cfg.GRPCServerAddress)
	if err != nil {
		slog.Error("failed to listen gRPC", "address", cfg.GRPCServerAddress, "error", err)
		os.Exit(1)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("gRPC server starting", "address", cfg.GRPCServerAddress)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server failed", "error", err)
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
	grpcServer.GracefulStop()
	_ = metricsSrv.Close()
}
