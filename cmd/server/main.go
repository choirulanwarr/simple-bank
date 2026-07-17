package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/choirulanwar/simple-bank/internal/config"
	"github.com/choirulanwar/simple-bank/internal/middleware"
	"github.com/choirulanwar/simple-bank/internal/repository"
	"github.com/choirulanwar/simple-bank/db/sqlc"
	"github.com/choirulanwar/simple-bank/api"
	"github.com/choirulanwar/simple-bank/api/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("✅ Connected to PostgreSQL")

	// Initialize layers
	store := sqlc.New(pool)
	repo := repository.NewAccountRepo(store, pool)
	custRepo := repository.NewCustomerRepo(store)

	// Create handlers
	customerHandler := api.NewCustomerHandler(custRepo)
	accountHandler := api.NewAccountHandler(repo)
	transactionHandler := api.NewTransactionHandler(repo)

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

	lis, err := net.Listen("tcp", cfg.GRPCServerAddress)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("🚀 gRPC server listening on %s", cfg.GRPCServerAddress)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	<-quit
	log.Println("🛑 Shutting down server...")
	grpcServer.GracefulStop()
}