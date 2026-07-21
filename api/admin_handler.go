package api

import (
	"context"
	"time"

	"github.com/choirulanwar/simple-bank/api/pb"
	"github.com/choirulanwar/simple-bank/internal/repository"
	"github.com/choirulanwar/simple-bank/pkg/token"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AdminHandler struct {
	adminRepo  *repository.AdminRepo
	tokenMaker token.Maker
}

func NewAdminHandler(adminRepo *repository.AdminRepo, tokenMaker token.Maker) *AdminHandler {
	return &AdminHandler{adminRepo: adminRepo, tokenMaker: tokenMaker}
}

func (h *AdminHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.Email == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email is required")
	}
	if req.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "password is required")
	}

	admin, err := h.adminRepo.GetAdminByEmail(ctx, req.Email)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
	}

	if !admin.IsActive {
		return nil, status.Errorf(codes.Unauthenticated, "account deactivated")
	}

	if err := h.adminRepo.VerifyPassword(ctx, admin.PasswordHash, req.Password); err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
	}

	accessTokenDuration := time.Hour * 24
	tokenStr, payload, err := h.tokenMaker.CreateToken(admin.ID, accessTokenDuration)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create token: %v", err)
	}

	return &pb.LoginResponse{
		AccessToken: tokenStr,
		TokenType:   "Bearer",
		ExpiresAt:   timestamppb.New(payload.ExpiredAt),
	}, nil
}
