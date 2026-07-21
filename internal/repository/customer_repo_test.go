package repository

import (
	"context"
	"testing"

	"github.com/choirulanwar/simple-bank/db/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
)

type CustomerRepoTestSuite struct {
	suite.Suite
	pool  *pgxpool.Pool
	repo  *CustomerRepo
	store sqlc.Querier
}

func (s *CustomerRepoTestSuite) SetupSuite() {
	dbURL := "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable"
	pool, err := pgxpool.New(context.Background(), dbURL)
	s.Require().NoError(err)
	s.pool = pool

	s.store = sqlc.New(pool)
	s.repo = NewCustomerRepo(s.store)

	// Clean up before tests
	s.cleanup()
}

func (s *CustomerRepoTestSuite) TearDownSuite() {
	s.pool.Close()
}

func (s *CustomerRepoTestSuite) SetupTest() {
	s.cleanup()
}

func (s *CustomerRepoTestSuite) TearDownTest() {
	s.cleanup()
}

func (s *CustomerRepoTestSuite) cleanup() {
	ctx := context.Background()
	_, err := s.pool.Exec(ctx, "DELETE FROM transactions")
	s.Require().NoError(err)
	_, err = s.pool.Exec(ctx, "DELETE FROM transfers")
	s.Require().NoError(err)
	_, err = s.pool.Exec(ctx, "DELETE FROM accounts")
	s.Require().NoError(err)
	_, err = s.pool.Exec(ctx, "DELETE FROM customers")
	s.Require().NoError(err)
}

func (s *CustomerRepoTestSuite) TestCreateCustomer_Success() {
	ctx := context.Background()

	customer, err := s.repo.CreateCustomer(ctx, CreateCustomerParams{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "SecureP@ss1",
	})

	s.Require().NoError(err)
	s.Require().NotZero(customer.ID)
	s.Require().Equal("John Doe", customer.Name)
	s.Require().Equal("john@example.com", customer.Email)
	s.Require().True(customer.IsActive)
	s.Require().NotEqual("SecureP@ss1", customer.PasswordHash) // Should be hashed
	s.Require().NotEmpty(customer.CreatedAt)
}

func (s *CustomerRepoTestSuite) TestCreateCustomer_DuplicateEmail() {
	ctx := context.Background()

	// Create first customer
	_, err := s.repo.CreateCustomer(ctx, CreateCustomerParams{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "SecureP@ss1",
	})
	s.Require().NoError(err)

	// Try to create with same email
	_, err = s.repo.CreateCustomer(ctx, CreateCustomerParams{
		Name:     "Jane Doe",
		Email:    "john@example.com",
		Password: "AnotherP@ss2",
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "email already registered")
}

func (s *CustomerRepoTestSuite) TestCreateCustomer_InvalidEmail() {
	ctx := context.Background()

	_, err := s.repo.CreateCustomer(ctx, CreateCustomerParams{
		Name:     "John Doe",
		Email:    "not-an-email",
		Password: "SecureP@ss1",
	})
	s.Require().Error(err)
}

func (s *CustomerRepoTestSuite) TestGetCustomer() {
	ctx := context.Background()

	// Create customer first
	created, err := s.repo.CreateCustomer(ctx, CreateCustomerParams{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "SecureP@ss1",
	})
	s.Require().NoError(err)

	// Get by ID
	customer, err := s.repo.GetCustomer(ctx, created.ID)
	s.Require().NoError(err)
	s.Require().Equal(created.ID, customer.ID)
	s.Require().Equal(created.Name, customer.Name)
	s.Require().Equal(created.Email, customer.Email)
}

func (s *CustomerRepoTestSuite) TestGetCustomerByEmail() {
	ctx := context.Background()

	// Create customer first
	created, err := s.repo.CreateCustomer(ctx, CreateCustomerParams{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "SecureP@ss1",
	})
	s.Require().NoError(err)

	// Get by email
	customer, err := s.repo.GetCustomerByEmail(ctx, "john@example.com")
	s.Require().NoError(err)
	s.Require().Equal(created.ID, customer.ID)
	s.Require().Equal(created.Email, customer.Email)
}

func (s *CustomerRepoTestSuite) TestListCustomers() {
	ctx := context.Background()

	// Create multiple customers
	for i := 1; i <= 5; i++ {
		_, err := s.repo.CreateCustomer(ctx, CreateCustomerParams{
			Name:     "User " + string(rune('A'+i)),
			Email:    "user" + string(rune('0'+i)) + "@example.com",
			Password: "SecureP@ss1",
		})
		s.Require().NoError(err)
	}

	// List with pagination
	customers, err := s.repo.ListCustomers(ctx, 3, 0)
	s.Require().NoError(err)
	s.Require().Len(customers, 3)

	// Second page
	customers, err = s.repo.ListCustomers(ctx, 3, 3)
	s.Require().NoError(err)
	s.Require().Len(customers, 2)
}

func TestCustomerRepoTestSuite(t *testing.T) {
	suite.Run(t, new(CustomerRepoTestSuite))
}
