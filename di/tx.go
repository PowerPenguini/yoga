package di

import (
	"fmt"

	"github.com/PowerPenguini/errs"
)

func ExecuteInTx[T any](d *DI, fn func(txDI *DI) (T, error)) (T, error) {
	var zero T
	if d.tx != nil {
		return fn(d)
	}

	tx, err := d.db.Begin()
	if err != nil {
		return zero, errs.NewError("transaction_begin_failed", "failed to begin transaction", errs.InternalType, err)
	}

	txDI := d.WithTx(tx)
	result, err := fn(txDI)
	if err != nil {
		_ = tx.Rollback()
		return zero, err
	}

	if err := tx.Commit(); err != nil {
		return zero, errs.NewError("transaction_commit_failed", "failed to commit transaction", errs.InternalType, err)
	}
	return result, nil
}

func ExecuteInTxNoResult(d *DI, fn func(txDI *DI) error) error {
	if d.tx != nil {
		return fn(d)
	}

	tx, err := d.db.Begin()
	if err != nil {
		return errs.NewError("transaction_begin_failed", "failed to begin transaction", errs.InternalType, err)
	}

	txDI := d.WithTx(tx)
	if err := fn(txDI); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			cause := fmt.Errorf("rollback failed: %v, original error: %w", rbErr, err)
			return errs.NewError("transaction_rollback_failed", "failed to rollback transaction", errs.InternalType, cause)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return errs.NewError("transaction_commit_failed", "failed to commit transaction", errs.InternalType, err)
	}
	return nil
}
