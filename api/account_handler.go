package api

import (
	"context"

	"github.com/choirulanwar/simple-bank/db/sqlc"
	"github.com/choirulanwar/simple-bank/internal/repository"
	"github.com/choirulanwar/simple-bank/api/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AccountHandler struct {
	repo *repository.AccountRepo
	pb.UnimplementedSimpleBankServer
}

func NewAccountHandler(repo *repository.AccountRepo) *AccountHandler {
	return &AccountHandler{repo: repo}
}

func (h *AccountHandler) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	if req.CustomerId <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "customer_id is required")
	}
	if req.Currency == "" {
		req.Currency = "IDR"
	}

	account, err := h.repo.CreateAccount(ctx, repository.CreateAccountParams{
		CustomerID: req.CustomerId,
		Currency:   req.Currency,
	})
	if err != nil {
		if err.Error() == "customer not found" {
			return nil, status.Errorf(codes.NotFound, "customer not found")
		}
		if err.Error() == "customer is inactive" {
			return nil, status.Errorf(codes.FailedPrecondition, "customer is inactive")
		}
		return nil, status.Errorf(codes.Internal, "failed to create account: %v", err)
	}

	return &pb.CreateAccountResponse{
		Account: h.accountToProto(account),
	}, nil
}

func (h *AccountHandler) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
	if req.Id <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid account ID")
	}

	account, err := h.repo.GetAccount(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "account not found")
	}

	return &pb.GetAccountResponse{
		Account: h.accountToProto(account),
	}, nil
}

func (h *AccountHandler) ListAccounts(ctx context.Context, req *pb.ListAccountsRequest) (*pb.ListAccountsResponse, error) {
	if req.CustomerId <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "customer_id is required")
	}

	accounts, err := h.repo.ListAccountsByCustomer(ctx, req.CustomerId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list accounts: %v", err)
	}

	protoAccounts := make([]*pb.Account, len(accounts))
	for i, a := range accounts {
		protoAccounts[i] = h.accountToProto(a)
	}

	return &pb.ListAccountsResponse{
		Accounts: protoAccounts,
	}, nil
}

func (h *AccountHandler) UpdateAccountStatus(ctx context.Context, req *pb.UpdateAccountStatusRequest) (*pb.UpdateAccountStatusResponse, error) {
	if req.Id <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid account ID")
	}
	if req.Status == "" {
		return nil, status.Errorf(codes.InvalidArgument, "status is required")
	}

	account, err := h.repo.UpdateAccountStatus(ctx, req.Id, req.Status)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update account status: %v", err)
	}

	return &pb.UpdateAccountStatusResponse{
		Account: h.accountToProto(account),
	}, nil
}

func (h *AccountHandler) accountToProto(a sqlc.Account) *pb.Account {
	return &pb.Account{
		Id:            a.ID,
		CustomerId:    a.CustomerID,
		AccountNumber: a.AccountNumber,
		Currency:      a.Currency,
		Balance:       a.Balance.String(),
		Status:        a.Status,
		CreatedAt:     timestamppb.New(a.CreatedAt),
		UpdatedAt:     timestamppb.New(a.UpdatedAt),
	}
}