package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/choirulanwar/simple-bank/db/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

// decStrEq compares a decimal.Decimal with an expected string
func decStrEq(actual decimal.Decimal, expected string) bool {
	exp, _ := decimal.NewFromString(expected)
	return actual.Equal(exp)
}

type AccountRepoTestSuite struct {
	suite.Suite
	pool          *pgxpool.Pool
	repo          *AccountRepo
	repoCust      *CustomerRepo
	store         sqlc.Querier
	custID        int64
	fromAccountID int64
}

func (s *AccountRepoTestSuite) SetupSuite() {
	dbURL := "postgresql://postgres:root@localhost:5432/simple_bank?sslmode=disable"
	pool, err := pgxpool.New(context.Background(), dbURL)
	s.Require().NoError(err)
	s.pool = pool

	s.store = sqlc.New(pool)
	s.repo = NewAccountRepo(s.store, s.pool)
	s.repoCust = NewCustomerRepo(s.store)

	// Clean up before tests
	s.cleanup()
}

func (s *AccountRepoTestSuite) TearDownSuite() {
	s.pool.Close()
}

func (s *AccountRepoTestSuite) SetupTest() {
	s.cleanup()

	// Create a from account for each test with unique email
	uniqueEmail := fmt.Sprintf("test_%d@example.com", time.Now().UnixNano())
	cust, err := s.repoCust.CreateCustomer(context.Background(), CreateCustomerParams{
		Name:     "Test Customer",
		Email:    uniqueEmail,
		Password: "SecureP@ss1",
	})
	s.Require().NoError(err)
	s.custID = cust.ID

	// Create a from account for each test
	fromAccount, err := s.repo.CreateAccount(context.Background(), CreateAccountParams{
		CustomerID: s.custID,
		Currency:   "IDR",
	})
	s.Require().NoError(err)
	s.fromAccountID = fromAccount.ID
}

func (s *AccountRepoTestSuite) TearDownTest() {
	s.cleanup()
}

func (s *AccountRepoTestSuite) cleanup() {
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

func (s *AccountRepoTestSuite) TestCreateAccount_Success() {
	ctx := context.Background()

	account, err := s.repo.CreateAccount(ctx, CreateAccountParams{
		CustomerID: s.custID,
		Currency:   "IDR",
	})

	s.Require().NoError(err)
	s.Require().NotZero(account.ID)
	s.Require().Equal(s.custID, account.CustomerID)
	s.Require().Len(account.AccountNumber, 16)
	s.Require().Equal("IDR", account.Currency)
	s.Require().Equal("0", account.Balance.String())
	s.Require().Equal("active", account.Status)
	s.Require().NotEmpty(account.CreatedAt)
}

func (s *AccountRepoTestSuite) TestCreateAccount_DefaultCurrency() {
	ctx := context.Background()

	account, err := s.repo.CreateAccount(ctx, CreateAccountParams{
		CustomerID: s.custID,
		Currency:   "",
	})

	s.Require().NoError(err)
	s.Require().Equal("IDR", account.Currency)
}

func (s *AccountRepoTestSuite) TestCreateAccount_CustomerNotFound() {
	ctx := context.Background()

	_, err := s.repo.CreateAccount(ctx, CreateAccountParams{
		CustomerID: 999999,
		Currency:   "IDR",
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "customer not found")
}

func (s *AccountRepoTestSuite) TestCreateAccount_CustomerInactive() {
	ctx := context.Background()

	// Create inactive customer
	inactiveCust, err := s.repoCust.CreateCustomer(ctx, CreateCustomerParams{
		Name:     "Inactive User",
		Email:    "inactive@example.com",
		Password: "SecureP@ss1",
	})
	s.Require().NoError(err)

	// Deactivate customer
	_, err = s.pool.Exec(ctx, "UPDATE customers SET is_active = false WHERE id = $1", inactiveCust.ID)
	s.Require().NoError(err)

	// Try to create account
	_, err = s.repo.CreateAccount(ctx, CreateAccountParams{
		CustomerID: inactiveCust.ID,
		Currency:   "IDR",
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "customer is inactive")
}

func (s *AccountRepoTestSuite) TestGetAccount() {
	ctx := context.Background()

	// Create account first
	created, err := s.repo.CreateAccount(ctx, CreateAccountParams{
		CustomerID: s.custID,
		Currency:   "IDR",
	})
	s.Require().NoError(err)

	// Get by ID
	account, err := s.repo.GetAccount(ctx, created.ID)
	s.Require().NoError(err)
	s.Require().Equal(created.ID, account.ID)
	s.Require().Equal(created.AccountNumber, account.AccountNumber)
}

func (s *AccountRepoTestSuite) TestGetAccountForUpdate() {
	ctx := context.Background()

	// Create account first
	created, err := s.repo.CreateAccount(ctx, CreateAccountParams{
		CustomerID: s.custID,
		Currency:   "IDR",
	})
	s.Require().NoError(err)

	// Get with lock
	account, err := s.repo.GetAccountForUpdate(ctx, created.ID)
	s.Require().NoError(err)
	s.Require().Equal(created.ID, account.ID)
}

func (s *AccountRepoTestSuite) TestListAccountsByCustomer() {
	ctx := context.Background()

	// Create multiple accounts
	for i := 0; i < 3; i++ {
		_, err := s.repo.CreateAccount(ctx, CreateAccountParams{
			CustomerID: s.custID,
			Currency:   "IDR",
		})
		s.Require().NoError(err)
	}

	accounts, err := s.repo.ListAccountsByCustomer(ctx, s.custID)
	s.Require().NoError(err)
	s.Require().Len(accounts, 4) // 1 from SetupTest + 3 created in test

	// Verify all belong to the customer
	for _, acc := range accounts {
		s.Require().Equal(s.custID, acc.CustomerID)
	}
}

func (s *AccountRepoTestSuite) TestAccountNumberUniqueness() {
	ctx := context.Background()

	// Create many accounts and verify uniqueness
	accountNumbers := make(map[string]bool)
	for i := 0; i < 20; i++ {
		account, err := s.repo.CreateAccount(ctx, CreateAccountParams{
			CustomerID: s.custID,
			Currency:   "IDR",
		})
		s.Require().NoError(err)
		s.Require().False(accountNumbers[account.AccountNumber], "duplicate account number: %s", account.AccountNumber)
		accountNumbers[account.AccountNumber] = true
	}
}

func (s *AccountRepoTestSuite) TestDeposit_Success() {
	ctx := context.Background()

	// Create account with initial balance via deposit
	account, err := s.repo.CreateAccount(ctx, CreateAccountParams{
		CustomerID: s.custID,
		Currency:   "IDR",
	})
	s.Require().NoError(err)

	// Deposit
	result, err := s.repo.Deposit(ctx, DepositParams{
		AccountID:   account.ID,
		Amount:      "100000.00",
		Reference:   "DEP-001",
		Description: "Test deposit",
	})
	s.Require().NoError(err)

	s.Require().Equal("deposit", result.Transaction.Type)
	s.Require().True(decStrEq(result.Transaction.Amount, "100000.00"))
	s.Require().True(decStrEq(result.Transaction.BalanceBefore, "0"))
	s.Require().True(decStrEq(result.Transaction.BalanceAfter, "100000.00"))
	s.Require().Equal("DEP-001", *result.Transaction.Reference)
	s.Require().Equal("Test deposit", *result.Transaction.Description)
	s.Require().Equal("100000.00", result.Balance)

	// Verify account balance updated
	acc, err := s.repo.GetAccount(ctx, account.ID)
	s.Require().NoError(err)
	s.Require().True(decStrEq(acc.Balance, "100000.00"))
}

func (s *AccountRepoTestSuite) TestDeposit_ZeroAmount() {
	ctx := context.Background()

	account, err := s.repo.CreateAccount(ctx, CreateAccountParams{
		CustomerID: s.custID,
		Currency:   "IDR",
	})
	s.Require().NoError(err)

	_, err = s.repo.Deposit(ctx, DepositParams{
		AccountID: account.ID,
		Amount:    "0.00",
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "amount must be greater than zero")
}

func (s *AccountRepoTestSuite) TestDeposit_NegativeAmount() {
	ctx := context.Background()

	account, err := s.repo.CreateAccount(ctx, CreateAccountParams{
		CustomerID: s.custID,
		Currency:   "IDR",
	})
	s.Require().NoError(err)

	_, err = s.repo.Deposit(ctx, DepositParams{
		AccountID: account.ID,
		Amount:    "-50000.00",
	})
	s.Require().Error(err)
}

func (s *AccountRepoTestSuite) TestDeposit_InactiveAccount() {
	ctx := context.Background()

	account, err := s.repo.CreateAccount(ctx, CreateAccountParams{
		CustomerID: s.custID,
		Currency:   "IDR",
	})
	s.Require().NoError(err)

	// Deactivate account
	_, err = s.pool.Exec(ctx, "UPDATE accounts SET status = 'inactive' WHERE id = $1", account.ID)
	s.Require().NoError(err)

	_, err = s.repo.Deposit(ctx, DepositParams{
		AccountID: account.ID,
		Amount:    "100000.00",
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "account is not active")
}

func (s *AccountRepoTestSuite) TestWithdraw_Success() {
	ctx := context.Background()

	// Create account and deposit first
	account, err := s.repo.CreateAccount(ctx, CreateAccountParams{
		CustomerID: s.custID,
		Currency:   "IDR",
	})
	s.Require().NoError(err)

	_, err = s.repo.Deposit(ctx, DepositParams{
		AccountID: account.ID,
		Amount:    "500000.00",
	})
	s.Require().NoError(err)

	// Withdraw
	result, err := s.repo.Withdraw(ctx, WithdrawParams{
		AccountID:   account.ID,
		Amount:      "200000.00",
		Reference:   "WTH-001",
		Description: "Test withdrawal",
	})
	s.Require().NoError(err)

	s.Require().Equal("withdrawal", result.Transaction.Type)
	s.Require().True(decStrEq(result.Transaction.Amount, "200000.00"))
	s.Require().True(decStrEq(result.Transaction.BalanceBefore, "500000.00"))
	s.Require().True(decStrEq(result.Transaction.BalanceAfter, "300000.00"))
	s.Require().Equal("300000.00", result.Balance)

	// Verify account balance updated
	acc, err := s.repo.GetAccount(ctx, account.ID)
	s.Require().NoError(err)
	s.Require().True(decStrEq(acc.Balance, "300000.00"))
}

func (s *AccountRepoTestSuite) TestWithdraw_InsufficientBalance() {
	ctx := context.Background()

	account, err := s.repo.CreateAccount(ctx, CreateAccountParams{
		CustomerID: s.custID,
		Currency:   "IDR",
	})
	s.Require().NoError(err)

	// Deposit small amount
	_, err = s.repo.Deposit(ctx, DepositParams{
		AccountID: account.ID,
		Amount:    "100000.00",
	})
	s.Require().NoError(err)

	// Try to withdraw more than balance
	_, err = s.repo.Withdraw(ctx, WithdrawParams{
		AccountID: account.ID,
		Amount:    "200000.00",
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "insufficient balance")

	// Verify balance unchanged
	acc, err := s.repo.GetAccount(ctx, account.ID)
	s.Require().NoError(err)
	s.Require().True(decStrEq(acc.Balance, "100000.00"))
}

func (s *AccountRepoTestSuite) TestWithdraw_ZeroAmount() {
	ctx := context.Background()

	account, err := s.repo.CreateAccount(ctx, CreateAccountParams{
		CustomerID: s.custID,
		Currency:   "IDR",
	})
	s.Require().NoError(err)

	_, err = s.repo.Deposit(ctx, DepositParams{
		AccountID: account.ID,
		Amount:    "100000.00",
	})
	s.Require().NoError(err)

	_, err = s.repo.Withdraw(ctx, WithdrawParams{
		AccountID: account.ID,
		Amount:    "0.00",
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "amount must be greater than zero")
}

func (s *AccountRepoTestSuite) TestWithdraw_NegativeAmount() {
	ctx := context.Background()

	account, err := s.repo.CreateAccount(ctx, CreateAccountParams{
		CustomerID: s.custID,
		Currency:   "IDR",
	})
	s.Require().NoError(err)

	_, err = s.repo.Deposit(ctx, DepositParams{
		AccountID: account.ID,
		Amount:    "100000.00",
	})
	s.Require().NoError(err)

	_, err = s.repo.Withdraw(ctx, WithdrawParams{
		AccountID: account.ID,
		Amount:    "-50000.00",
	})
	s.Require().Error(err)
}

func (s *AccountRepoTestSuite) TestWithdraw_InactiveAccount() {
	ctx := context.Background()

	account, err := s.repo.CreateAccount(ctx, CreateAccountParams{
		CustomerID: s.custID,
		Currency:   "IDR",
	})
	s.Require().NoError(err)

	_, err = s.repo.Deposit(ctx, DepositParams{
		AccountID: account.ID,
		Amount:    "100000.00",
	})
	s.Require().NoError(err)

	// Deactivate account
	_, err = s.pool.Exec(ctx, "UPDATE accounts SET status = 'inactive' WHERE id = $1", account.ID)
	s.Require().NoError(err)

	_, err = s.repo.Withdraw(ctx, WithdrawParams{
		AccountID: account.ID,
		Amount:    "50000.00",
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "account is not active")
}

func (s *AccountRepoTestSuite) TestConcurrentDeposit_NoRaceCondition() {
	ctx := context.Background()

	account, err := s.repo.CreateAccount(ctx, CreateAccountParams{
		CustomerID: s.custID,
		Currency:   "IDR",
	})
	s.Require().NoError(err)

	// Deposit initial amount
	_, err = s.repo.Deposit(ctx, DepositParams{
		AccountID: account.ID,
		Amount:    "100000.00",
	})
	s.Require().NoError(err)

	// Concurrent deposits
	n := 10
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			_, err := s.repo.Deposit(ctx, DepositParams{
				AccountID: account.ID,
				Amount:    "10000.00",
			})
			errCh <- err
		}()
	}

	// Wait for all to complete
	for i := 0; i < n; i++ {
		err := <-errCh
		s.Require().NoError(err)
	}

	// Final balance should be 100000 + 10*10000 = 200000
	acc, err := s.repo.GetAccount(ctx, account.ID)
	s.Require().NoError(err)
	s.Require().True(decStrEq(acc.Balance, "200000.00"))
}

func (s *AccountRepoTestSuite) TestTransferTx_Success() {
	ctx := context.Background()

	// Create second account
	toAccount, err := s.repo.CreateAccount(ctx, CreateAccountParams{
		CustomerID: s.custID,
		Currency:   "IDR",
	})
	s.Require().NoError(err)

	// Deposit to from account
	_, err = s.repo.Deposit(ctx, DepositParams{
		AccountID: s.fromAccountID,
		Amount:    "1000000.00",
	})
	s.Require().NoError(err)

	// Deposit to to account
	_, err = s.repo.Deposit(ctx, DepositParams{
		AccountID: toAccount.ID,
		Amount:    "500000.00",
	})
	s.Require().NoError(err)

	// Transfer
	result, err := s.repo.TransferTx(ctx, TransferTxParams{
		FromAccountID: s.fromAccountID,
		ToAccountID:   toAccount.ID,
		Amount:        "300000.00",
		Fee:           "5000.00",
		Reference:     "TRF-001",
		Description:   "Payment for invoice #123",
	})
	s.Require().NoError(err)

	s.Require().Equal("completed", result.Transfer.Status)
	s.Require().True(decStrEq(result.Transfer.Amount, "300000.00"))
	s.Require().True(decStrEq(result.Transfer.Fee, "5000.00"))

	// from account: 1000000 - 300000 - 5000 = 695000
	s.Require().True(decStrEq(result.FromAccount.Balance, "695000.00"))
	// to account: 500000 + 300000 = 800000
	s.Require().True(decStrEq(result.ToAccount.Balance, "800000.00"))

	s.Require().Equal("withdrawal", result.FromTransaction.Type)
	s.Require().Equal("deposit", result.ToTransaction.Type)
}

func (s *AccountRepoTestSuite) TestTransferTx_InsufficientBalance() {
	ctx := context.Background()

	// Create second account
	toAccount, err := s.repo.CreateAccount(ctx, CreateAccountParams{
		CustomerID: s.custID,
		Currency:   "IDR",
	})
	s.Require().NoError(err)

	// Transfer more than balance
	_, err = s.repo.TransferTx(ctx, TransferTxParams{
		FromAccountID: s.fromAccountID,
		ToAccountID:   toAccount.ID,
		Amount:        "200000.00",
		Fee:           "0.00",
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "insufficient balance")

	// Verify balances unchanged
	fromAcc, err := s.repo.GetAccount(ctx, s.fromAccountID)
	s.Require().NoError(err)
	s.Require().True(decStrEq(fromAcc.Balance, "0"))

	toAcc, err := s.repo.GetAccount(ctx, toAccount.ID)
	s.Require().NoError(err)
	s.Require().True(decStrEq(toAcc.Balance, "0"))
}

func (s *AccountRepoTestSuite) TestTransferTx_DeadlockPrevention() {
	ctx := context.Background()

	// Create second account
	acc2, err := s.repo.CreateAccount(ctx, CreateAccountParams{
		CustomerID: s.custID,
		Currency:   "IDR",
	})
	s.Require().NoError(err)

	// Deposit to both accounts
	_, err = s.repo.Deposit(ctx, DepositParams{
		AccountID: s.fromAccountID,
		Amount:    "1000000.00",
	})
	s.Require().NoError(err)

	_, err = s.repo.Deposit(ctx, DepositParams{
		AccountID: acc2.ID,
		Amount:    "1000000.00",
	})
	s.Require().NoError(err)

	// Run 10 concurrent transfers A->B and 10 B->A
	n := 10
	errCh := make(chan error, n*2)

	for i := 0; i < n; i++ {
		go func() {
			_, err := s.repo.TransferTx(ctx, TransferTxParams{
				FromAccountID: s.fromAccountID,
				ToAccountID:   acc2.ID,
				Amount:        "10000.00",
				Fee:           "0.00",
			})
			errCh <- err
		}()

		go func() {
			_, err := s.repo.TransferTx(ctx, TransferTxParams{
				FromAccountID: acc2.ID,
				ToAccountID:   s.fromAccountID,
				Amount:        "10000.00",
				Fee:           "0.00",
			})
			errCh <- err
		}()
	}

	// All should succeed without deadlock
	for i := 0; i < n*2; i++ {
		err := <-errCh
		s.Require().NoError(err, "transfer %d failed", i)
	}

	// Verify final balances (each direction: 10 * 10000 = 100000 movement)
	// A final = 1000000 - 100000 + 100000 = 1000000
	// B final = 1000000 - 100000 + 100000 = 1000000
	finalA, err := s.repo.GetAccount(ctx, s.fromAccountID)
	s.Require().NoError(err)
	finalB, err := s.repo.GetAccount(ctx, acc2.ID)
	s.Require().NoError(err)
	s.Require().True(decStrEq(finalA.Balance, "1000000.00"))
	s.Require().True(decStrEq(finalB.Balance, "1000000.00"))
}

func (s *AccountRepoTestSuite) TestTransferTx_SameAccount() {
	ctx := context.Background()

	_, err := s.repo.TransferTx(ctx, TransferTxParams{
		FromAccountID: s.fromAccountID,
		ToAccountID:   s.fromAccountID,
		Amount:        "100000.00",
		Fee:           "0.00",
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "cannot transfer to the same account")
}

func (s *AccountRepoTestSuite) TestTransferTx_InactiveAccount() {
	ctx := context.Background()

	// Create second account
	toAccount, err := s.repo.CreateAccount(ctx, CreateAccountParams{
		CustomerID: s.custID,
		Currency:   "IDR",
	})
	s.Require().NoError(err)

	// Deposit to from account
	_, err = s.repo.Deposit(ctx, DepositParams{
		AccountID: s.fromAccountID,
		Amount:    "100000.00",
	})
	s.Require().NoError(err)

	// Deactivate to account
	_, err = s.pool.Exec(ctx, "UPDATE accounts SET status = 'inactive' WHERE id = $1", toAccount.ID)
	s.Require().NoError(err)

	// Try transfer
	_, err = s.repo.TransferTx(ctx, TransferTxParams{
		FromAccountID: s.fromAccountID,
		ToAccountID:   toAccount.ID,
		Amount:        "50000.00",
		Fee:           "0.00",
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "to account is not active")
}

func TestAccountRepoTestSuite(t *testing.T) {
	suite.Run(t, new(AccountRepoTestSuite))
}
