package repository

import (
	"context"
	"fmt"
	"regexp"

	"github.com/choirulanwar/simple-bank/db/sqlc"
	"github.com/choirulanwar/simple-bank/pkg/password"
)

var adminEmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type AdminRepo struct {
	store sqlc.Querier
}

func NewAdminRepo(store sqlc.Querier) *AdminRepo {
	return &AdminRepo{store: store}
}

type CreateAdminParams struct {
	Name     string
	Email    string
	Password string
	Role     string
}

func (r *AdminRepo) CreateAdmin(ctx context.Context, arg CreateAdminParams) (sqlc.Admin, error) {
	if !adminEmailRegex.MatchString(arg.Email) {
		return sqlc.Admin{}, fmt.Errorf("invalid email format")
	}

	hashed, err := password.HashPassword(arg.Password)
	if err != nil {
		return sqlc.Admin{}, fmt.Errorf("hash password: %w", err)
	}

	role := arg.Role
	if role == "" {
		role = "admin"
	}
	if role != "superadmin" && role != "admin" && role != "viewer" {
		return sqlc.Admin{}, fmt.Errorf("invalid role: %s", role)
	}

	admin, err := r.store.CreateAdmin(ctx, sqlc.CreateAdminParams{
		Name:         arg.Name,
		Email:        arg.Email,
		PasswordHash: hashed,
		Role:         role,
	})
	if err != nil {
		return sqlc.Admin{}, fmt.Errorf("create admin: %w", err)
	}
	return admin, nil
}

func (r *AdminRepo) GetAdminByEmail(ctx context.Context, email string) (sqlc.Admin, error) {
	admin, err := r.store.GetAdminByEmail(ctx, email)
	if err != nil {
		return sqlc.Admin{}, fmt.Errorf("get admin by email: %w", err)
	}
	return admin, nil
}

func (r *AdminRepo) VerifyPassword(ctx context.Context, hashedPassword, plainPassword string) error {
	return password.VerifyPassword(hashedPassword, plainPassword)
}
