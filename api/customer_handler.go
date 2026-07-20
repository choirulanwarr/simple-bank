package api

import (
	"context"
	"strings"

	"github.com/choirulanwar/simple-bank/api/pb"
	"github.com/choirulanwar/simple-bank/db/sqlc"
	"github.com/choirulanwar/simple-bank/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CustomerHandler struct {
	repo *repository.CustomerRepo
	pb.UnimplementedSimpleBankServer
}

func NewCustomerHandler(repo *repository.CustomerRepo) *CustomerHandler {
	return &CustomerHandler{repo: repo}
}

func (h *CustomerHandler) CreateCustomer(ctx context.Context, req *pb.CreateCustomerRequest) (*pb.CreateCustomerResponse, error) {
	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "name is required")
	}
	if req.Email == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email is required")
	}
	if req.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "password is required")
	}

	customer, err := h.repo.CreateCustomer(ctx, repository.CreateCustomerParams{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "email already registered") {
			return nil, status.Errorf(codes.AlreadyExists, "email already registered")
		}
		if strings.Contains(errMsg, "invalid email format") {
			return nil, status.Errorf(codes.InvalidArgument, "invalid email format")
		}
		return nil, status.Errorf(codes.Internal, "failed to create customer: %v", err)
	}

	return &pb.CreateCustomerResponse{
		Customer: h.customerToProto(customer),
	}, nil
}

func (h *CustomerHandler) GetCustomer(ctx context.Context, req *pb.GetCustomerRequest) (*pb.GetCustomerResponse, error) {
	if req.Id <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid customer ID")
	}

	customer, err := h.repo.GetCustomer(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "customer not found")
	}

	return &pb.GetCustomerResponse{
		Customer: h.customerToProto(customer),
	}, nil
}

func (h *CustomerHandler) ListCustomers(ctx context.Context, req *pb.ListCustomersRequest) (*pb.ListCustomersResponse, error) {
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	customers, err := h.repo.ListCustomers(ctx, req.Limit, req.Offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list customers: %v", err)
	}

	protoCustomers := make([]*pb.Customer, len(customers))
	for i, c := range customers {
		protoCustomers[i] = h.customerToProto(c)
	}

	return &pb.ListCustomersResponse{
		Customers:  protoCustomers,
		TotalCount: int64(len(protoCustomers)),
		HasMore:    len(customers) == int(req.Limit),
	}, nil
}

func (h *CustomerHandler) customerToProto(c sqlc.Customer) *pb.Customer {
	return &pb.Customer{
		Id:        c.ID,
		Name:      c.Name,
		Email:     c.Email,
		IsActive:  c.IsActive,
		CreatedAt: timestamppb.New(c.CreatedAt),
		UpdatedAt: timestamppb.New(c.UpdatedAt),
	}
}

func (h *CustomerHandler) UpdateCustomer(ctx context.Context, req *pb.UpdateCustomerRequest) (*pb.UpdateCustomerResponse, error) {
	if req.Id <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid customer ID")
	}

	customer, err := h.repo.UpdateCustomer(ctx, repository.UpdateCustomerParams{
		ID:       req.Id,
		Name:     req.Name,
		Email:    req.Email,
		IsActive: req.IsActive,
	})
	if err != nil {
		if err.Error() == "email already registered" {
			return nil, status.Errorf(codes.AlreadyExists, "email already registered")
		}
		return nil, status.Errorf(codes.Internal, "failed to update customer: %v", err)
	}

	return &pb.UpdateCustomerResponse{
		Customer: h.customerToProto(customer),
	}, nil
}

func (h *CustomerHandler) DeleteCustomer(ctx context.Context, req *pb.DeleteCustomerRequest) (*pb.DeleteCustomerResponse, error) {
	if req.Id <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid customer ID")
	}

	err := h.repo.DeleteCustomer(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete customer: %v", err)
	}

	return &pb.DeleteCustomerResponse{
		Success: true,
	}, nil
}
