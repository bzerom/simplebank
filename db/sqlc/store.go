package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Store provides all functions to execute db queries and transactions
type Store interface {
	TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error)
	Querier
}

// SQLStore provides all functions to execute SQL queries and transactions
// Queries struct doesn't support transaction so we have to extend its functionality
type SQLStore struct {
	db       *sql.DB
	*Queries // composition
}

// NewStore creates a new Store
func NewStore(db *sql.DB) Store {
	return &SQLStore{
		db:      db,
		Queries: New(db),
	}
}

// execTx executes a function within a database transaction
// Takes a context and a call back function
func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
	// nil, use &sql.TxOptions{} to overwrite
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// This works because it accepts a db tx interface
	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}
	return tx.Commit()
}

// TransferTxParams contains the input parameters of the transfer transaction
type TransferTxParams struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	Amount        int64 `json:"amount"`
}

// TransferTxResult is the result of the transfer transaction
type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount   Account  `json:"to_account"`
	FromEntry   Entry    `json:"from_entry"`
	ToEntry     Entry    `json:"to_entry"`
}

//var txKey = struct{}{}

// TransferTx performs many transfer from one account to the other.
// it creates a transfer record, add account entries and update accounts balance
// within a single database transaction
func (store *SQLStore) TransferTx(
	ctx context.Context,
	arg TransferTxParams,
) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		//txName := ctx.Value(txKey)

		//fmt.Println(txName, "create transfer")
		// ------ transfer record ------

		// Accessing the result variable from the outer function. Similar
		// for the arg variable. This makes the callback function become
		// a closure.  Since Go lacks support for generic type,closure is
		// often used when we want to get the result from a callback
		// function.
		// https://betterprogramming.pub/closures-made-simple-with-golang-69db3017cd7b
		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: arg.FromAccountID,
			ToAccountID:   arg.ToAccountID,
			Amount:        arg.Amount,
		})

		if err != nil {
			return err
		}

		//fmt.Println(txName, "create entry 1")
		// ------ Remove money from entry ------
		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.FromAccountID,
			Amount:    -arg.Amount,
		})
		if err != nil {
			return err
		}

		//fmt.Println(txName, "create entry 2")
		// ------ Add money to entry ------
		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccountID,
			Amount:    arg.Amount,
		})

		if err != nil {
			return err
		}

		/*
			//fmt.Println(txName, "get account 1")
			// ------ Get and Update account balance------
			account1, err := q.GetAccountForUpdate(ctx, arg.FromAccountID)
			if err != nil {
				return err
			}

			//fmt.Println(txName, "update account 1")
			result.FromAccount, err = q.UpdateAccount(ctx, UpdateAccountParams{
				ID:      arg.FromAccountID,
				Balance: account1.Balance - arg.Amount,
			})
			if err != nil {
				return err
			}

			//fmt.Println(txName, "get account 2")
			account2, err := q.GetAccountForUpdate(ctx, arg.ToAccountID)
			if err != nil {
				return err
			}

			//fmt.Println(txName, "update account 2")
			result.ToAccount, err = q.UpdateAccount(ctx, UpdateAccountParams{
				ID:      arg.ToAccountID,
				Balance: account2.Balance + arg.Amount,
			})
			if err != nil {
				return err
			}
		*/

		// Avoid deadlock by changing the order of execution.
		if arg.FromAccountID < arg.ToAccountID {
			result.FromAccount, result.ToAccount, err = addMoney(
				ctx, q, arg.FromAccountID, -arg.Amount,
				arg.ToAccountID, arg.Amount)
		} else {
			result.ToAccount, result.FromAccount, err = addMoney(
				ctx, q, arg.ToAccountID, arg.Amount,
				arg.FromAccountID, -arg.Amount)
		}

		return nil
	})

	return result, err
}

func addMoney(
	ctx context.Context,
	q *Queries,
	accountID1 int64,
	amount1 int64,
	accountID2 int64,
	amount2 int64,
) (account1, account2 Account, err error) {
	account1, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:     accountID1,
		Amount: amount1,
	})
	if err != nil {
		//return accoutn1, account2, err
		return
	}

	account2, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:     accountID2,
		Amount: amount2,
	})
	return
}
