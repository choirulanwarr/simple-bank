package repository

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/choirulanwar/simple-bank/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type AccountRepo struct {
	store sqlc.Querier
	queries *sqlc.Queries
	pool  *pgxpool.Pool
}

func NewAccountRepo(store sqlc.Querier, pool *pgxpool.Pool) *AccountRepo {
	queries, ok := store.(*sqlc.Queries)
	if !ok {
		panic("store must be *sqlc.Queries")
	}
	return &AccountRepo{store: store, queries: queries, pool: pool}
}

func (r *AccountRepo) generateAccountNumber() string {
	max := big.NewInt(9999999999999999)
	n, _ := rand.Int(rand.Reader, max)
	return fmt.Sprintf("%016d", n.Int64())
}

type CreateAccountParams struct {
	CustomerID int64
	Currency   string
}

func (r *AccountRepo) CreateAccount(ctx context.Context, arg CreateAccountParams) (sqlc.Account, error) {
	customer, err := r.store.GetCustomer(ctx, arg.CustomerID)
	if err != nil {
		return sqlc.Account{}, fmt.Errorf("customer not found: %w", err)
	}
	if !customer.IsActive {
		return sqlc.Account{}, fmt.Errorf("customer is inactive")
	}

	if arg.Currency == "" {
		arg.Currency = "IDR"
	}

	var accountNumber string
	for i := 0; i < 10; i++ {
		accountNumber = r.generateAccountNumber()
		account, err := r.store.CreateAccount(ctx, sqlc.CreateAccountParams{
			CustomerID:    arg.CustomerID,
			AccountNumber: accountNumber,
			Currency:      arg.Currency,
		})
		if err == nil {
			return account, nil
		}
	}

	account, err := r.store.CreateAccount(ctx, sqlc.CreateAccountParams{
		CustomerID:    arg.CustomerID,
		AccountNumber: accountNumber,
		Currency:      arg.Currency,
	})
	if err != nil {
		return sqlc.Account{}, fmt.Errorf("create account: %w", err)
	}

	return account, nil
}

func (r *AccountRepo) GetAccount(ctx context.Context, id int64) (sqlc.Account, error) {
	account, err := r.store.GetAccount(ctx, id)
	if err != nil {
		return sqlc.Account{}, fmt.Errorf("get account: %w", err)
	}
	return account, nil
}

func (r *AccountRepo) GetAccountForUpdate(ctx context.Context, id int64) (sqlc.Account, error) {
	account, err := r.store.GetAccountForUpdate(ctx, id)
	if err != nil {
		return sqlc.Account{}, fmt.Errorf("get account for update: %w", err)
	}
	return account, nil
}

func (r *AccountRepo) ListAccountsByCustomer(ctx context.Context, customerID int64) ([]sqlc.Account, error) {
	accounts, err := r.store.ListAccountsByCustomer(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("list accounts by customer: %w", err)
	}
	return accounts, nil
}

type DepositParams struct {
	AccountID   int64
	Amount      string
	Reference   string
	Description string
}

type WithdrawParams struct {
	AccountID   int64
	Amount      string
	Reference   string
	Description string
}

type TransactionResult struct {
	Transaction sqlc.Transaction
	Balance     string
}

func (r *AccountRepo) Deposit(ctx context.Context, arg DepositParams) (TransactionResult, error) {
	var result TransactionResult

	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.queries.WithTx(tx)

		account, err := q.GetAccountForUpdate(ctx, arg.AccountID)
		if err != nil {
			return fmt.Errorf("get account: %w", err)
		}

		if account.Status != "active" {
			return fmt.Errorf("account is not active")
		}

		amount, err := decimal.NewFromString(arg.Amount)
		if err != nil {
			return fmt.Errorf("invalid amount: %w", err)
		}
		if amount.LessThanOrEqual(decimal.Zero) {
			return fmt.Errorf("amount must be greater than zero")
		}

		balanceBefore := account.Balance

		updatedAccount, err := q.AddAccountBalance(ctx, sqlc.AddAccountBalanceParams{
			ID:     arg.AccountID,
			Amount: amount,
		})
		if err != nil {
			return fmt.Errorf("update balance: %w", err)
		}

		balanceAfter := updatedAccount.Balance

		var ref, desc *string
		if arg.Reference != "" {
			ref = &arg.Reference
		}
		if arg.Description != "" {
			desc = &arg.Description
		}

		transaction, err := q.CreateTransaction(ctx, sqlc.CreateTransactionParams{
			AccountID:     arg.AccountID,
			Type:          "deposit",
			Amount:        amount,
			BalanceBefore: balanceBefore,
			BalanceAfter:  balanceAfter,
			Reference:     ref,
			Description:   desc,
		})
		if err != nil {
			return fmt.Errorf("create transaction: %w", err)
		}

		result.Transaction = transaction
		result.Balance = balanceAfter.StringFixed(2)

		return nil
	})

	if err != nil {
		return TransactionResult{}, err
	}

	return result, nil
}

func (r *AccountRepo) Withdraw(ctx context.Context, arg WithdrawParams) (TransactionResult, error) {
	var result TransactionResult

	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.queries.WithTx(tx)

		account, err := q.GetAccountForUpdate(ctx, arg.AccountID)
		if err != nil {
			return fmt.Errorf("get account: %w", err)
		}

		if account.Status != "active" {
			return fmt.Errorf("account is not active")
		}

		amount, err := decimal.NewFromString(arg.Amount)
		if err != nil {
			return fmt.Errorf("invalid amount: %w", err)
		}
		if amount.LessThanOrEqual(decimal.Zero) {
			return fmt.Errorf("amount must be greater than zero")
		}

		if account.Balance.LessThan(amount) {
			return fmt.Errorf("insufficient balance: have %s, need %s", account.Balance, amount)
		}

		balanceBefore := account.Balance

		updatedAccount, err := q.AddAccountBalance(ctx, sqlc.AddAccountBalanceParams{
			ID:     arg.AccountID,
			Amount: amount.Neg(),
		})
		if err != nil {
			return fmt.Errorf("update balance: %w", err)
		}

		balanceAfter := updatedAccount.Balance

		var ref, desc *string
		if arg.Reference != "" {
			ref = &arg.Reference
		}
		if arg.Description != "" {
			desc = &arg.Description
		}

		transaction, err := q.CreateTransaction(ctx, sqlc.CreateTransactionParams{
			AccountID:     arg.AccountID,
			Type:          "withdrawal",
			Amount:        amount,
			BalanceBefore: balanceBefore,
			BalanceAfter:  balanceAfter,
			Reference:     ref,
			Description:   desc,
		})
		if err != nil {
			return fmt.Errorf("create transaction: %w", err)
		}

		result.Transaction = transaction
		result.Balance = balanceAfter.StringFixed(2)

		return nil
	})

	if err != nil {
		return TransactionResult{}, err
	}

	return result, nil
}
type TransferTxParams struct {
	FromAccountID int64
	ToAccountID   int64
	Amount        string
	Fee           string
	Reference     string
	Description   string
}

type TransferTxResult struct {
	Transfer         sqlc.Transfer
	FromAccount      sqlc.Account
	ToAccount        sqlc.Account
	FromTransaction  sqlc.Transaction
	ToTransaction    sqlc.Transaction
}

func (r *AccountRepo) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.queries.WithTx(tx)

		// Validate from != to
		if arg.FromAccountID == arg.ToAccountID {
			return fmt.Errorf("cannot transfer to the same account")
		}

		// Consistent lock ordering: always lock the account with smaller ID first
		firstID, secondID := arg.FromAccountID, arg.ToAccountID
		if firstID > secondID {
			firstID, secondID = secondID, firstID
		}

		// Lock both accounts in consistent order
		_, err := q.GetAccountForUpdate(ctx, firstID)
		if err != nil {
			return fmt.Errorf("get first account for update: %w", err)
		}
		_, err = q.GetAccountForUpdate(ctx, secondID)
		if err != nil {
			return fmt.Errorf("get second account for update: %w", err)
		}

		// Now get accounts with proper roles
		fromAccount, err := q.GetAccount(ctx, arg.FromAccountID)
		if err != nil {
			return fmt.Errorf("get from account: %w", err)
		}
		toAccount, err := q.GetAccount(ctx, arg.ToAccountID)
		if err != nil {
			return fmt.Errorf("get to account: %w", err)
		}

		// Validate accounts are active
		if fromAccount.Status != "active" {
			return fmt.Errorf("from account is not active")
		}
		if toAccount.Status != "active" {
			return fmt.Errorf("to account is not active")
		}

		// Parse amount and fee
		amount, err := decimal.NewFromString(arg.Amount)
		if err != nil {
			return fmt.Errorf("invalid amount: %w", err)
		}
		fee, err := decimal.NewFromString(arg.Fee)
		if err != nil {
			return fmt.Errorf("invalid fee: %w", err)
		}
		if amount.LessThanOrEqual(decimal.Zero) {
			return fmt.Errorf("amount must be greater than zero")
		}
		if fee.LessThan(decimal.Zero) {
			return fmt.Errorf("fee cannot be negative")
		}

		totalAmount := amount.Add(fee)

		// Check sufficient balance
		if fromAccount.Balance.LessThan(totalAmount) {
			return fmt.Errorf("insufficient balance: have %s, need %s", fromAccount.Balance, totalAmount)
		}

		balanceBeforeFrom := fromAccount.Balance
		balanceBeforeTo := toAccount.Balance

		// Debit from account
		updatedFromAccount, err := q.AddAccountBalance(ctx, sqlc.AddAccountBalanceParams{
			ID:     arg.FromAccountID,
			Amount: totalAmount.Neg(),
		})
		if err != nil {
			return fmt.Errorf("debit from account: %w", err)
		}

		// Credit to account
		updatedToAccount, err := q.AddAccountBalance(ctx, sqlc.AddAccountBalanceParams{
			ID:     arg.ToAccountID,
			Amount: amount,
		})
		if err != nil {
			return fmt.Errorf("credit to account: %w", err)
		}

		balanceAfterFrom := updatedFromAccount.Balance
		balanceAfterTo := updatedToAccount.Balance

		// Create withdrawal transaction for from account
		var ref, desc *string
		if arg.Reference != "" {
			ref = &arg.Reference
		}
		if arg.Description != "" {
			desc = &arg.Description
		}

		fromTransaction, err := q.CreateTransaction(ctx, sqlc.CreateTransactionParams{
			AccountID:     arg.FromAccountID,
			Type:          "withdrawal",
			Amount:        amount,
			BalanceBefore: balanceBeforeFrom,
			BalanceAfter:  balanceAfterFrom,
			Reference:     ref,
			Description:   desc,
		})
		if err != nil {
			return fmt.Errorf("create from transaction: %w", err)
		}

		// Create deposit transaction for to account
		toTransaction, err := q.CreateTransaction(ctx, sqlc.CreateTransactionParams{
			AccountID:     arg.ToAccountID,
			Type:          "deposit",
			Amount:        amount,
			BalanceBefore: balanceBeforeTo,
			BalanceAfter:  balanceAfterTo,
			Reference:     ref,
			Description:   desc,
		})
		if err != nil {
			return fmt.Errorf("create to transaction: %w", err)
		}

		// Create transfer record
		transfer, err := q.CreateTransfer(ctx, sqlc.CreateTransferParams{
			FromAccountID: arg.FromAccountID,
			ToAccountID:   arg.ToAccountID,
			Amount:        amount,
			Fee:           fee,
			Status:        "completed",
			Reference:     ref,
			Description:   desc,
		})
		if err != nil {
			return fmt.Errorf("create transfer: %w", err)
		}

		result.Transfer = transfer
		result.FromAccount = updatedFromAccount
		result.ToAccount = updatedToAccount
		result.FromTransaction = fromTransaction
		result.ToTransaction = toTransaction

		return nil
	})

	if err != nil {
		return TransferTxResult{}, err
	}

	return result, nil
}
