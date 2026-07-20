//go:build integration

package integration

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/choirulanwar/simple-bank/api"
	"github.com/choirulanwar/simple-bank/api/pb"
	"github.com/choirulanwar/simple-bank/db/sqlc"
	"github.com/choirulanwar/simple-bank/internal/middleware"
	"github.com/choirulanwar/simple-bank/internal/repository"
	"github.com/choirulanwar/simple-bank/pkg/token"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func readSQL(filename string) string {
	_, src, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot get source file path")
	}
	path := filepath.Join(filepath.Dir(src), "../../db/migrations", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("read migration %s: %v", filename, err))
	}
	return string(data)
}

type IntegrationTestSuite struct {
	suite.Suite
	pool   *pgxpool.Pool
	conn   *grpc.ClientConn
	client pb.SimpleBankClient
	server *grpc.Server
	addr   string
}

func TestIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupSuite() {
	os.Setenv("POSTGRES_HOST", "localhost")
	os.Setenv("POSTGRES_PORT", "5432")
	os.Setenv("POSTGRES_USER", "root")
	os.Setenv("POSTGRES_PASSWORD", "secret")
	os.Setenv("POSTGRES_DB", "simple_bank")
	os.Setenv("TOKEN_SYMMETRIC_KEY", "12345678901234567890123456789012")

	ctx := context.Background()

	dbURL := "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable"
	pool, err := pgxpool.New(ctx, dbURL)
	s.Require().NoError(err)
	s.pool = pool

	s.dropAll()

	_, err = pool.Exec(ctx, readSQL("000001_init_schema.up.sql"))
	s.Require().NoError(err)
	_, err = pool.Exec(ctx, readSQL("000002_audit_trigger.up.sql"))
	s.Require().NoError(err)

	store := sqlc.New(pool)
	repo := repository.NewAccountRepo(store, pool)
	custRepo := repository.NewCustomerRepo(store)

	tokenMaker, err := token.NewPasetoMaker("12345678901234567890123456789012")
	s.Require().NoError(err)

	customerHandler := api.NewCustomerHandler(custRepo)
	accountHandler := api.NewAccountHandler(repo)
	transactionHandler := api.NewTransactionHandler(repo, custRepo, tokenMaker)

	lis, err := net.Listen("tcp", "localhost:0")
	s.Require().NoError(err)
	s.addr = lis.Addr().String()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.ChainUnaryInterceptors(
			middleware.RecoveryInterceptor(),
			middleware.LoggingInterceptor(logger),
		)),
	)
	pb.RegisterSimpleBankServer(grpcServer, api.NewSimpleBankServer(
		customerHandler,
		accountHandler,
		transactionHandler,
	))

	s.server = grpcServer
	go func() {
		grpcServer.Serve(lis)
	}()

	conn, err := grpc.NewClient(s.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	s.Require().NoError(err)
	s.conn = conn
	s.client = pb.NewSimpleBankClient(conn)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	if s.server != nil {
		s.server.GracefulStop()
	}
	if s.conn != nil {
		s.conn.Close()
	}
	if s.pool != nil {
		s.cleanup()
		s.pool.Close()
	}
}

func (s *IntegrationTestSuite) SetupTest() {
	s.cleanup()
}

func (s *IntegrationTestSuite) TearDownTest() {
	s.cleanup()
}

func (s *IntegrationTestSuite) dropAll() {
	ctx := context.Background()
	s.pool.Exec(ctx, "DROP TABLE IF EXISTS audit_logs CASCADE")
	s.pool.Exec(ctx, "DROP TABLE IF EXISTS transactions CASCADE")
	s.pool.Exec(ctx, "DROP TABLE IF EXISTS transfers CASCADE")
	s.pool.Exec(ctx, "DROP TABLE IF EXISTS accounts CASCADE")
	s.pool.Exec(ctx, "DROP TABLE IF EXISTS customers CASCADE")
	s.pool.Exec(ctx, "DROP FUNCTION IF EXISTS audit_trigger_function CASCADE")
}

func (s *IntegrationTestSuite) cleanup() {
	ctx := context.Background()
	tables := []string{"transactions", "transfers", "accounts", "customers", "audit_logs"}
	for _, t := range tables {
		_, err := s.pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s", t))
		if err != nil {
			s.T().Logf("cleanup warning: DELETE FROM %s: %v", t, err)
		}
	}
}

func (s *IntegrationTestSuite) TestFullFlow() {
	ctx := context.Background()

	createCustResp, err := s.client.CreateCustomer(ctx, &pb.CreateCustomerRequest{
		Name:     "John Doe",
		Email:    "john.doe@example.com",
		Password: "SecureP@ss1",
	})
	s.Require().NoError(err)
	s.Require().NotNil(createCustResp.Customer)

	createAccResp, err := s.client.CreateAccount(ctx, &pb.CreateAccountRequest{
		CustomerId: createCustResp.Customer.Id,
		Currency:   "IDR",
	})
	s.Require().NoError(err)
	s.Require().NotNil(createAccResp.Account)
	accountID := createAccResp.Account.Id

	depositResp, err := s.client.Deposit(ctx, &pb.DepositRequest{
		AccountId:   accountID,
		Amount:      "1000000.00",
		Reference:   "DEP-001",
		Description: "Initial deposit",
	})
	s.Require().NoError(err)
	s.Require().Equal("deposit", depositResp.Transaction.Type)
	s.Require().Equal("1000000.00", depositResp.Transaction.Amount)
	s.Require().Equal("1000000.00", depositResp.BalanceAfter)

	withdrawResp, err := s.client.Withdraw(ctx, &pb.WithdrawRequest{
		AccountId:   accountID,
		Amount:      "200000.00",
		Reference:   "WTH-001",
		Description: "Withdrawal",
	})
	s.Require().NoError(err)
	s.Require().Equal("withdrawal", withdrawResp.Transaction.Type)
	s.Require().Equal("200000.00", withdrawResp.Transaction.Amount)
	s.Require().Equal("800000.00", withdrawResp.BalanceAfter)

	createAccResp2, err := s.client.CreateAccount(ctx, &pb.CreateAccountRequest{
		CustomerId: createCustResp.Customer.Id,
		Currency:   "IDR",
	})
	s.Require().NoError(err)
	s.Require().NotNil(createAccResp2.Account)

	_, err = s.client.Deposit(ctx, &pb.DepositRequest{
		AccountId:   createAccResp2.Account.Id,
		Amount:      "500000.00",
		Reference:   "DEP-002",
		Description: "Second account deposit",
	})
	s.Require().NoError(err)

	transferResp, err := s.client.Transfer(ctx, &pb.TransferRequest{
		FromAccountId: accountID,
		ToAccountId:   createAccResp2.Account.Id,
		Amount:        "300000.00",
		Fee:           "5000.00",
		Reference:     "TRF-001",
		Description:   "Payment for invoice #123",
	})
	s.Require().NoError(err)
	s.Require().Equal("completed", transferResp.Status)

	acc1Resp, err := s.client.GetAccount(ctx, &pb.GetAccountRequest{Id: accountID})
	s.Require().NoError(err)
	s.Require().Equal("495000.00", acc1Resp.Account.Balance)

	acc2Resp, err := s.client.GetAccount(ctx, &pb.GetAccountRequest{Id: createAccResp2.Account.Id})
	s.Require().NoError(err)
	s.Require().Equal("800000.00", acc2Resp.Account.Balance)

	historyResp, err := s.client.GetTransactionHistory(ctx, &pb.GetTransactionHistoryRequest{
		AccountId: accountID,
		Limit:     10,
		Offset:    0,
	})
	s.Require().NoError(err)
	s.Require().Len(historyResp.Transactions, 3)
}

func (s *IntegrationTestSuite) TestConcurrentTransfer_Deadlock() {
	ctx := context.Background()

	custResp, err := s.client.CreateCustomer(ctx, &pb.CreateCustomerRequest{
		Name:     "Deadlock Test",
		Email:    "deadlock.test@example.com",
		Password: "SecureP@ss1",
	})
	s.Require().NoError(err)

	acc1Resp, err := s.client.CreateAccount(ctx, &pb.CreateAccountRequest{
		CustomerId: custResp.Customer.Id,
		Currency:   "IDR",
	})
	s.Require().NoError(err)

	acc2Resp, err := s.client.CreateAccount(ctx, &pb.CreateAccountRequest{
		CustomerId: custResp.Customer.Id,
		Currency:   "IDR",
	})
	s.Require().NoError(err)

	acc1ID := acc1Resp.Account.Id
	acc2ID := acc2Resp.Account.Id

	_, err = s.client.Deposit(ctx, &pb.DepositRequest{
		AccountId: acc1ID,
		Amount:    "1000000.00",
		Reference: "DEP-DL-01",
	})
	s.Require().NoError(err)

	_, err = s.client.Deposit(ctx, &pb.DepositRequest{
		AccountId: acc2ID,
		Amount:    "1000000.00",
		Reference: "DEP-DL-02",
	})
	s.Require().NoError(err)

	n := 10
	var wg sync.WaitGroup
	errCh := make(chan error, n*2)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.client.Transfer(ctx, &pb.TransferRequest{
				FromAccountId: acc1ID,
				ToAccountId:   acc2ID,
				Amount:        "10000.00",
				Fee:           "0",
			})
			errCh <- err
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.client.Transfer(ctx, &pb.TransferRequest{
				FromAccountId: acc2ID,
				ToAccountId:   acc1ID,
				Amount:        "10000.00",
				Fee:           "0",
			})
			errCh <- err
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		s.Require().NoError(err, "concurrent transfer should not deadlock")
	}

	finalA, err := s.client.GetAccount(ctx, &pb.GetAccountRequest{Id: acc1ID})
	s.Require().NoError(err)
	finalB, err := s.client.GetAccount(ctx, &pb.GetAccountRequest{Id: acc2ID})
	s.Require().NoError(err)

	expected := "1000000.00"
	s.Require().Equal(expected, finalA.Account.Balance,
		"acc1 balance should be %s (10 A->B + 10 B->A cancel out)", expected)
	s.Require().Equal(expected, finalB.Account.Balance,
		"acc2 balance should be %s (10 A->B + 10 B->A cancel out)", expected)
}

func (s *IntegrationTestSuite) TestConcurrentFailedTransfer_NegativeBalance() {
	ctx := context.Background()

	custResp, err := s.client.CreateCustomer(ctx, &pb.CreateCustomerRequest{
		Name:     "Negative Test",
		Email:    "negative.test@example.com",
		Password: "SecureP@ss1",
	})
	s.Require().NoError(err)

	acc1Resp, err := s.client.CreateAccount(ctx, &pb.CreateAccountRequest{
		CustomerId: custResp.Customer.Id,
		Currency:   "IDR",
	})
	s.Require().NoError(err)

	acc2Resp, err := s.client.CreateAccount(ctx, &pb.CreateAccountRequest{
		CustomerId: custResp.Customer.Id,
		Currency:   "IDR",
	})
	s.Require().NoError(err)

	acc1ID := acc1Resp.Account.Id
	acc2ID := acc2Resp.Account.Id

	_, err = s.client.Deposit(ctx, &pb.DepositRequest{
		AccountId: acc1ID,
		Amount:    "100000.00",
		Reference: "DEP-NG-01",
	})
	s.Require().NoError(err)

	n := 20
	var wg sync.WaitGroup
	errCh := make(chan error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := s.client.Transfer(ctx, &pb.TransferRequest{
				FromAccountId: acc1ID,
				ToAccountId:   acc2ID,
				Amount:        "50000.00",
				Fee:           "0",
			})
			errCh <- err
		}(i)
	}

	wg.Wait()
	close(errCh)

	successes := 0
	failures := 0
	for err := range errCh {
		if err != nil {
			failures++
		} else {
			successes++
		}
	}

	s.Require().LessOrEqual(successes, 2, "at most 2 of 20 transfers should succeed (2 * 50000 = 100000)")
	s.Require().GreaterOrEqual(failures, 18, "at least 18 transfers should fail due to insufficient balance")

	finalA, err := s.client.GetAccount(ctx, &pb.GetAccountRequest{Id: acc1ID})
	s.Require().NoError(err)
	balanceA, err := decimal.NewFromString(finalA.Account.Balance)
	s.Require().NoError(err)
	s.Require().True(balanceA.GreaterThanOrEqual(decimal.Zero),
		"balance should never be negative, got: %s", finalA.Account.Balance)

	finalB, err := s.client.GetAccount(ctx, &pb.GetAccountRequest{Id: acc2ID})
	s.Require().NoError(err)
	balanceB, err := decimal.NewFromString(finalB.Account.Balance)
	s.Require().NoError(err)

	totalExpected := decimal.NewFromInt(100000)
	totalActual := balanceA.Add(balanceB)
	s.Require().True(totalActual.Equal(totalExpected),
		"total balance conserved: %s + %s = %s, expected %s",
		finalA.Account.Balance, finalB.Account.Balance, totalActual, totalExpected)
}
