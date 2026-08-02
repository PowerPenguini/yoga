package di

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"main/services"
	"main/views"
)

const testDriverName = "yoga_template_test"

var (
	registerTestDriverOnce sync.Once
	testDriverStats        sync.Map
	testDSNSeq             atomic.Int64
)

type txStats struct {
	begins    atomic.Int64
	commits   atomic.Int64
	rollbacks atomic.Int64
}

type testDriver struct{}

func (d testDriver) Open(name string) (driver.Conn, error) {
	stats, _ := testDriverStats.LoadOrStore(name, &txStats{})
	return &testConn{stats: stats.(*txStats)}, nil
}

type testConn struct {
	stats *txStats
}

func (c *testConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not implemented by test driver")
}

func (c *testConn) Close() error {
	return nil
}

func (c *testConn) Begin() (driver.Tx, error) {
	c.stats.begins.Add(1)
	return &testTx{stats: c.stats}, nil
}

type testTx struct {
	stats *txStats
	done  atomic.Bool
}

func (t *testTx) Commit() error {
	if t.done.Swap(true) {
		return errors.New("transaction already closed")
	}
	t.stats.commits.Add(1)
	return nil
}

func (t *testTx) Rollback() error {
	if t.done.Swap(true) {
		return errors.New("transaction already closed")
	}
	t.stats.rollbacks.Add(1)
	return nil
}

func openTestDB(t *testing.T) (*sql.DB, *txStats) {
	t.Helper()
	registerTestDriverOnce.Do(func() {
		sql.Register(testDriverName, testDriver{})
	})
	dsn := fmt.Sprintf("%s-%d", strings.ReplaceAll(t.Name(), "/", "-"), testDSNSeq.Add(1))
	stats := &txStats{}
	testDriverStats.Store(dsn, stats)

	db, err := sql.Open(testDriverName, dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		testDriverStats.Delete(dsn)
	})
	return db, stats
}

func newTestDI(t *testing.T) (*DI, *txStats) {
	t.Helper()
	db, stats := openTestDB(t)
	logger := testLogger(t)
	deps := buildDI(db, nil, sharedDeps{
		Services: svc{Clock: services.NewClock()},
		Views:    view{ItemViewer: views.NewItemViewer(db, logger)},
		Logger:   logger,
	})
	return deps, stats
}

func TestExecuteInTxCommitsAndReturnsResult(t *testing.T) {
	deps, stats := newTestDI(t)

	result, err := ExecuteInTx(deps, func(txDI *DI) (int, error) {
		if txDI.tx == nil {
			t.Fatal("callback DI is not in tx")
		}
		if _, ok := txDI.ItemRepo.Executor().(*sql.Tx); !ok {
			t.Fatalf("repo executor = %T, want *sql.Tx", txDI.ItemRepo.Executor())
		}
		return 42, nil
	})
	if err != nil {
		t.Fatalf("ExecuteInTx: %v", err)
	}
	if result != 42 {
		t.Fatalf("result = %d, want 42", result)
	}
	assertTxStats(t, stats, 1, 1, 0)
}

func TestExecuteInTxRollsBackOnError(t *testing.T) {
	deps, stats := newTestDI(t)
	wantErr := errors.New("fail")

	_, err := ExecuteInTx(deps, func(txDI *DI) (int, error) {
		if txDI.tx == nil {
			t.Fatal("callback DI is not in tx")
		}
		return 0, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	assertTxStats(t, stats, 1, 0, 1)
}

func TestExecuteInTxNoResultCommitsAndRollsBack(t *testing.T) {
	deps, stats := newTestDI(t)

	if err := ExecuteInTxNoResult(deps, func(txDI *DI) error {
		if txDI.tx == nil {
			t.Fatal("callback DI is not in tx")
		}
		return nil
	}); err != nil {
		t.Fatalf("ExecuteInTxNoResult success: %v", err)
	}

	wantErr := errors.New("fail")
	err := ExecuteInTxNoResult(deps, func(txDI *DI) error {
		if txDI.tx == nil {
			t.Fatal("callback DI is not in tx")
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	assertTxStats(t, stats, 2, 1, 1)
}

func TestExecuteInTxNestedDoesNotBeginNewTx(t *testing.T) {
	deps, stats := newTestDI(t)

	err := ExecuteInTxNoResult(deps, func(txDI *DI) error {
		return ExecuteInTxNoResult(txDI, func(nested *DI) error {
			if nested.tx == nil {
				t.Fatal("nested DI is not in tx")
			}
			return nil
		})
	})
	if err != nil {
		t.Fatalf("nested ExecuteInTxNoResult: %v", err)
	}
	assertTxStats(t, stats, 1, 1, 0)
}

func TestWithTxRebuildsReposWithTxExecutor(t *testing.T) {
	deps, _ := newTestDI(t)

	tx, err := deps.db.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	txDI := deps.WithTx(tx)
	if txDI == deps {
		t.Fatal("WithTx returned original DI")
	}
	if _, ok := txDI.ItemRepo.Executor().(*sql.Tx); !ok {
		t.Fatalf("repo executor = %T, want *sql.Tx", txDI.ItemRepo.Executor())
	}
}

func assertTxStats(t *testing.T, stats *txStats, begins, commits, rollbacks int64) {
	t.Helper()
	if stats.begins.Load() != begins || stats.commits.Load() != commits || stats.rollbacks.Load() != rollbacks {
		t.Fatalf(
			"stats begin/commit/rollback = %d/%d/%d, want %d/%d/%d",
			stats.begins.Load(),
			stats.commits.Load(),
			stats.rollbacks.Load(),
			begins,
			commits,
			rollbacks,
		)
	}
}
