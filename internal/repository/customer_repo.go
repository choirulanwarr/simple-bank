package repository

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/choirulanwar/simple-bank/internal/cache"
	"github.com/choirulanwar/simple-bank/db/sqlc"
	"github.com/choirulanwar/simple-bank/pkg/password"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type CustomerRepo struct {
	store sqlc.Querier
	cache *cache.Cache
}

func NewCustomerRepo(store sqlc.Querier) *CustomerRepo {
	return &CustomerRepo{store: store}
}

func (r *CustomerRepo) SetCache(c *cache.Cache) {
	r.cache = c
}

type CreateCustomerParams struct {
	Name     string
	Email    string
	Password string
}

func (r *CustomerRepo) CreateCustomer(ctx context.Context, arg CreateCustomerParams) (sqlc.Customer, error) {
	// Validate email format
	if !emailRegex.MatchString(arg.Email) {
		return sqlc.Customer{}, fmt.Errorf("invalid email format")
	}

	// Check if email already exists
	existing, err := r.store.GetCustomerByEmail(ctx, arg.Email)
	if err == nil && existing.ID > 0 {
		return sqlc.Customer{}, fmt.Errorf("email already registered: %s", arg.Email)
	}

	// Hash password
	hashedPassword, err := password.HashPassword(arg.Password)
	if err != nil {
		return sqlc.Customer{}, fmt.Errorf("hash password: %w", err)
	}

	// Create customer
	customer, err := r.store.CreateCustomer(ctx, sqlc.CreateCustomerParams{
		Name:         arg.Name,
		Email:        arg.Email,
		PasswordHash: hashedPassword,
	})
	if err != nil {
		return sqlc.Customer{}, fmt.Errorf("create customer: %w", err)
	}

	return customer, nil
}

func (r *CustomerRepo) GetCustomer(ctx context.Context, id int64) (sqlc.Customer, error) {
	// Try cache first
	if r.cache != nil {
		var customer sqlc.Customer
		if err := r.cache.Get(ctx, cache.CustomerKey(id), &customer); err == nil {
			return customer, nil
		}
	}

	customer, err := r.store.GetCustomer(ctx, id)
	if err != nil {
		return sqlc.Customer{}, fmt.Errorf("get customer: %w", err)
	}

	// Populate cache (longer TTL — customer info changes infrequently)
	if r.cache != nil {
		_ = r.cache.Set(ctx, cache.CustomerKey(id), &customer, 5*time.Minute)
	}

	return customer, nil
}

func (r *CustomerRepo) GetCustomerByEmail(ctx context.Context, email string) (sqlc.Customer, error) {
	customer, err := r.store.GetCustomerByEmail(ctx, email)
	if err != nil {
		return sqlc.Customer{}, fmt.Errorf("get customer by email: %w", err)
	}
	return customer, nil
}

func (r *CustomerRepo) ListCustomers(ctx context.Context, limit, offset int32) ([]sqlc.Customer, error) {
	customers, err := r.store.ListCustomers(ctx, sqlc.ListCustomersParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}
	return customers, nil
}

func (r *CustomerRepo) UpdateCustomer(ctx context.Context, arg UpdateCustomerParams) (sqlc.Customer, error) {
	customer, err := r.store.UpdateCustomer(ctx, sqlc.UpdateCustomerParams{
		ID:       arg.ID,
		Name:     arg.Name,
		Email:    arg.Email,
		IsActive: arg.IsActive,
	})
	if err != nil {
		return sqlc.Customer{}, fmt.Errorf("update customer: %w", err)
	}

	// Invalidate cache after update
	if r.cache != nil {
		_ = r.cache.Del(ctx, cache.CustomerKey(arg.ID))
	}

	return customer, nil
}

func (r *CustomerRepo) DeleteCustomer(ctx context.Context, id int64) error {
	err := r.store.DeleteCustomer(ctx, id)
	if err != nil {
		return fmt.Errorf("delete customer: %w", err)
	}
	return nil
}

func (r *CustomerRepo) VerifyPassword(ctx context.Context, hashedPassword, plainPassword string) error {
	return password.VerifyPassword(hashedPassword, plainPassword)
}

type UpdateCustomerParams struct {
	ID       int64
	Name     string
	Email    string
	IsActive bool
}
