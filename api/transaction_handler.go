package api

import (
	"context"
	"time"

	"github.com/choirulanwar/simple-bank/api/pb"
	"github.com/choirulanwar/simple-bank/db/sqlc"
	"github.com/choirulanwar/simple-bank/internal/repository"
	"github.com/choirulanwar/simple-bank/pkg/token"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TransactionHandler struct {
	repo       *repository.AccountRepo
	custRepo   *repository.CustomerRepo
	tokenMaker token.Maker
	pb.UnimplementedSimpleBankServer
}

func NewTransactionHandler(repo *repository.AccountRepo, custRepo *repository.CustomerRepo, tokenMaker token.Maker) *TransactionHandler {
	return &TransactionHandler{repo: repo, custRepo: custRepo, tokenMaker: tokenMaker}
}

func (h *TransactionHandler) Deposit(ctx context.Context, req *pb.DepositRequest) (*pb.DepositResponse, error) {
	if req.AccountId <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "account_id is required")
	}
	if req.Amount == "" {
		return nil, status.Errorf(codes.InvalidArgument, "amount is required")
	}
	if req.Reference == "" {
		return nil, status.Errorf(codes.InvalidArgument, "reference is required")
	}

	result, err := h.repo.Deposit(ctx, repository.DepositParams{
		AccountID:   req.AccountId,
		Amount:      req.Amount,
		Reference:   req.Reference,
		Description: req.Description,
	})
	if err != nil {
		if err.Error() == "account is not active" {
			return nil, status.Errorf(codes.FailedPrecondition, "account is not active")
		}
		if err.Error() == "amount must be greater than zero" {
			return nil, status.Errorf(codes.InvalidArgument, "amount must be greater than zero")
		}
		return nil, status.Errorf(codes.Internal, "failed to deposit: %v", err)
	}

	return &pb.DepositResponse{
		Transaction:  h.transactionToProto(result.Transaction),
		BalanceAfter: result.Balance,
	}, nil
}

func (h *TransactionHandler) Withdraw(ctx context.Context, req *pb.WithdrawRequest) (*pb.WithdrawResponse, error) {
	if req.AccountId <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "account_id is required")
	}
	if req.Amount == "" {
		return nil, status.Errorf(codes.InvalidArgument, "amount is required")
	}
	if req.Reference == "" {
		return nil, status.Errorf(codes.InvalidArgument, "reference is required")
	}

	result, err := h.repo.Withdraw(ctx, repository.WithdrawParams{
		AccountID:   req.AccountId,
		Amount:      req.Amount,
		Reference:   req.Reference,
		Description: req.Description,
	})
	if err != nil {
		if err.Error() == "account is not active" {
			return nil, status.Errorf(codes.FailedPrecondition, "account is not active")
		}
		if err.Error() == "amount must be greater than zero" {
			return nil, status.Errorf(codes.InvalidArgument, "amount must be greater than zero")
		}
		if err.Error() == "insufficient balance" {
			return nil, status.Errorf(codes.FailedPrecondition, "insufficient balance")
		}
		return nil, status.Errorf(codes.Internal, "failed to withdraw: %v", err)
	}

	return &pb.WithdrawResponse{
		Transaction:  h.transactionToProto(result.Transaction),
		BalanceAfter: result.Balance,
	}, nil
}

func (h *TransactionHandler) Transfer(ctx context.Context, req *pb.TransferRequest) (*pb.TransferResponse, error) {
	if req.FromAccountId <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "from_account_id is required")
	}
	if req.ToAccountId <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "to_account_id is required")
	}
	if req.Amount == "" {
		return nil, status.Errorf(codes.InvalidArgument, "amount is required")
	}
	if req.FromAccountId == req.ToAccountId {
		return nil, status.Errorf(codes.InvalidArgument, "cannot transfer to the same account")
	}

	result, err := h.repo.TransferTx(ctx, repository.TransferTxParams{
		FromAccountID: req.FromAccountId,
		ToAccountID:   req.ToAccountId,
		Amount:        req.Amount,
		Fee:           req.Fee,
		Reference:     req.Reference,
		Description:   req.Description,
	})
	if err != nil {
		if err.Error() == "cannot transfer to the same account" {
			return nil, status.Errorf(codes.InvalidArgument, "cannot transfer to the same account")
		}
		if err.Error() == "from account is not active" {
			return nil, status.Errorf(codes.FailedPrecondition, "from account is not active")
		}
		if err.Error() == "to account is not active" {
			return nil, status.Errorf(codes.FailedPrecondition, "to account is not active")
		}
		if err.Error() == "insufficient balance" {
			return nil, status.Errorf(codes.FailedPrecondition, "insufficient balance")
		}
		if err.Error() == "invalid fee" {
			return nil, status.Errorf(codes.InvalidArgument, "fee cannot be negative")
		}
		return nil, status.Errorf(codes.Internal, "failed to transfer: %v", err)
	}

	return &pb.TransferResponse{
		TransferId:  result.Transfer.ID,
		Status:      result.Transfer.Status,
		Amount:      result.Transfer.Amount.StringFixed(2),
		FromAccount: h.accountToProto(result.FromAccount),
		ToAccount:   h.accountToProto(result.ToAccount),
	}, nil
}

func (h *TransactionHandler) GetTransactionHistory(ctx context.Context, req *pb.GetTransactionHistoryRequest) (*pb.GetTransactionHistoryResponse, error) {
	if req.AccountId <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "account_id is required")
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	transactions, err := h.repo.ListTransactionsByAccount(ctx, req.AccountId, req.Limit, req.Offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get transaction history: %v", err)
	}

	protoTransactions := make([]*pb.Transaction, len(transactions))
	for i, t := range transactions {
		protoTransactions[i] = h.transactionToProto(t)
	}

	return &pb.GetTransactionHistoryResponse{
		Transactions: protoTransactions,
		TotalCount:   int64(len(protoTransactions)),
		HasMore:      len(transactions) == int(req.Limit),
	}, nil
}

func (h *TransactionHandler) GetAuditLogs(ctx context.Context, req *pb.GetAuditLogsRequest) (*pb.GetAuditLogsResponse, error) {
	if req.TableName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "table_name is required")
	}
	if req.RecordId <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "record_id is required")
	}

	logs, err := h.repo.ListAuditLogsByRecord(ctx, req.TableName, req.RecordId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get audit logs: %v", err)
	}

	protoLogs := make([]*pb.AuditLog, len(logs))
	for i, log := range logs {
		protoLogs[i] = h.auditLogToProto(log)
	}

	return &pb.GetAuditLogsResponse{
		Logs: protoLogs,
	}, nil
}

func (h *TransactionHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.Email == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email is required")
	}
	if req.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "password is required")
	}

	// Find customer by email
	customer, err := h.custRepo.GetCustomerByEmail(ctx, req.Email)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
	}

	// Check if customer is active
	if !customer.IsActive {
		return nil, status.Errorf(codes.Unauthenticated, "account deactivated")
	}

	// Verify password
	err = h.custRepo.VerifyPassword(ctx, customer.PasswordHash, req.Password)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
	}

	// Generate token
	accessTokenDuration := time.Hour * 24 // 24 hours
	token, payload, err := h.tokenMaker.CreateToken(customer.ID, accessTokenDuration)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create token: %v", err)
	}

	return &pb.LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresAt:   timestamppb.New(payload.ExpiredAt),
	}, nil
}

func (h *TransactionHandler) accountToProto(a sqlc.Account) *pb.Account {
	return &pb.Account{
		Id:            a.ID,
		CustomerId:    a.CustomerID,
		AccountNumber: a.AccountNumber,
		Currency:      a.Currency,
		Balance:       a.Balance.StringFixed(2),
		Status:        a.Status,
		CreatedAt:     timestamppb.New(a.CreatedAt),
		UpdatedAt:     timestamppb.New(a.UpdatedAt),
	}
}

func (h *TransactionHandler) transactionToProto(t sqlc.Transaction) *pb.Transaction {
	return &pb.Transaction{
		Id:            t.ID,
		AccountId:     t.AccountID,
		Type:          t.Type,
		Amount:        t.Amount.StringFixed(2),
		BalanceBefore: t.BalanceBefore.StringFixed(2),
		BalanceAfter:  t.BalanceAfter.StringFixed(2),
		Reference:     derefString(t.Reference),
		Description:   derefString(t.Description),
		CreatedAt:     timestamppb.New(t.CreatedAt),
	}
}

func (h *TransactionHandler) auditLogToProto(a sqlc.AuditLog) *pb.AuditLog {
	return &pb.AuditLog{
		Id:        a.ID,
		TableName: a.TableName,
		RecordId:  a.RecordID,
		Operation: a.Operation,
		OldValues: string(a.OldValues),
		NewValues: string(a.NewValues),
		ChangedBy: derefString(a.ChangedBy),
		ChangedAt: timestamppb.New(a.ChangedAt),
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
