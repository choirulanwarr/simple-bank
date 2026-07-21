package api

import (
	"context"

	"github.com/choirulanwar/simple-bank/api/pb"
)

// SimpleBankServer implements the gRPC service
type SimpleBankServer struct {
	pb.UnimplementedSimpleBankServer

	customerHandler    *CustomerHandler
	accountHandler     *AccountHandler
	transactionHandler *TransactionHandler
	adminHandler       *AdminHandler
}

func NewSimpleBankServer(
	customerHandler *CustomerHandler,
	accountHandler *AccountHandler,
	transactionHandler *TransactionHandler,
	adminHandler *AdminHandler,
) *SimpleBankServer {
	return &SimpleBankServer{
		customerHandler:    customerHandler,
		accountHandler:     accountHandler,
		transactionHandler: transactionHandler,
		adminHandler:       adminHandler,
	}
}

// Customer methods
func (s *SimpleBankServer) CreateCustomer(ctx context.Context, req *pb.CreateCustomerRequest) (*pb.CreateCustomerResponse, error) {
	return s.customerHandler.CreateCustomer(ctx, req)
}

func (s *SimpleBankServer) GetCustomer(ctx context.Context, req *pb.GetCustomerRequest) (*pb.GetCustomerResponse, error) {
	return s.customerHandler.GetCustomer(ctx, req)
}

func (s *SimpleBankServer) ListCustomers(ctx context.Context, req *pb.ListCustomersRequest) (*pb.ListCustomersResponse, error) {
	return s.customerHandler.ListCustomers(ctx, req)
}

func (s *SimpleBankServer) UpdateCustomer(ctx context.Context, req *pb.UpdateCustomerRequest) (*pb.UpdateCustomerResponse, error) {
	return s.customerHandler.UpdateCustomer(ctx, req)
}

func (s *SimpleBankServer) DeleteCustomer(ctx context.Context, req *pb.DeleteCustomerRequest) (*pb.DeleteCustomerResponse, error) {
	return s.customerHandler.DeleteCustomer(ctx, req)
}

// Account methods
func (s *SimpleBankServer) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	return s.accountHandler.CreateAccount(ctx, req)
}

func (s *SimpleBankServer) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
	return s.accountHandler.GetAccount(ctx, req)
}

func (s *SimpleBankServer) ListAccounts(ctx context.Context, req *pb.ListAccountsRequest) (*pb.ListAccountsResponse, error) {
	return s.accountHandler.ListAccounts(ctx, req)
}

func (s *SimpleBankServer) UpdateAccountStatus(ctx context.Context, req *pb.UpdateAccountStatusRequest) (*pb.UpdateAccountStatusResponse, error) {
	return s.accountHandler.UpdateAccountStatus(ctx, req)
}

// Transaction methods
func (s *SimpleBankServer) Deposit(ctx context.Context, req *pb.DepositRequest) (*pb.DepositResponse, error) {
	return s.transactionHandler.Deposit(ctx, req)
}

func (s *SimpleBankServer) Withdraw(ctx context.Context, req *pb.WithdrawRequest) (*pb.WithdrawResponse, error) {
	return s.transactionHandler.Withdraw(ctx, req)
}

func (s *SimpleBankServer) Transfer(ctx context.Context, req *pb.TransferRequest) (*pb.TransferResponse, error) {
	return s.transactionHandler.Transfer(ctx, req)
}

func (s *SimpleBankServer) GetTransactionHistory(ctx context.Context, req *pb.GetTransactionHistoryRequest) (*pb.GetTransactionHistoryResponse, error) {
	return s.transactionHandler.GetTransactionHistory(ctx, req)
}

// Audit methods
func (s *SimpleBankServer) GetAuditLogs(ctx context.Context, req *pb.GetAuditLogsRequest) (*pb.GetAuditLogsResponse, error) {
	return s.transactionHandler.GetAuditLogs(ctx, req)
}

// Auth methods
func (s *SimpleBankServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	return s.adminHandler.Login(ctx, req)
}
