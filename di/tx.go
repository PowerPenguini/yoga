package di

import "fmt"

func ExecuteInTx[T any](d *DI, fn func(txDI *DI) (T, error)) (T, error) {
	var zero T
	if d.tx != nil {
		return fn(d)
	}

	tx, err := d.db.Begin()
	if err != nil {
		return zero, err
	}

	txDI := d.WithTx(tx)
	result, err := fn(txDI)
	if err != nil {
		_ = tx.Rollback()
		return zero, err
	}

	if err := tx.Commit(); err != nil {
		return zero, err
	}
	return result, nil
}

func ExecuteInTxNoResult(d *DI, fn func(txDI *DI) error) error {
	if d.tx != nil {
		return fn(d)
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	txDI := d.WithTx(tx)
	if err := fn(txDI); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v, original error: %w", rbErr, err)
		}
		return err
	}

	return tx.Commit()
}
