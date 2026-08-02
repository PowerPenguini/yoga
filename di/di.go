package di

import (
	"database/sql"
	"log"
	"os"

	"main/repos"
	"main/services"
	"main/validators"
	"main/views"
)

type svc struct {
	Clock *services.Clock
}

type repo struct {
	ItemRepo        *repos.ItemRepo
	OutboxEventRepo *repos.OutboxEventRepo
}

type view struct {
	ItemViewer *views.ItemViewer
}

type val struct {
	ItemValidator        *validators.ItemValidator
	OutboxEventValidator *validators.OutboxEventValidator
}

type sharedDeps struct {
	Services svc
	Views    view
	Logger   *log.Logger
}

type DI struct {
	svc
	repo
	view
	val
	Logger *log.Logger
	db     *sql.DB
	tx     *sql.Tx
}

func NewDI(connStr string) (*DI, error) {
	db, err := NewDatabase(connStr)
	if err != nil {
		return nil, err
	}

	logger := log.New(os.Stdout, "[yoga] ", log.LstdFlags|log.Lmicroseconds)
	shared := sharedDeps{
		Services: buildServices(),
		Views:    buildViews(db, logger),
		Logger:   logger,
	}

	return buildDI(db, nil, shared), nil
}

func buildDI(db *sql.DB, tx *sql.Tx, shared sharedDeps) *DI {
	var executor repos.DBTX = db
	if tx != nil {
		executor = tx
	}

	repos := buildRepos(executor)
	di := &DI{
		svc:    shared.Services,
		repo:   repos,
		view:   shared.Views,
		val:    buildValidators(),
		Logger: shared.Logger,
		db:     db,
		tx:     tx,
	}
	di.MustValidateWiring()
	return di
}

func buildServices() svc {
	return svc{
		Clock: services.NewClock(),
	}
}

func buildRepos(db repos.DBTX) repo {
	return repo{
		ItemRepo:        repos.NewItemRepo(db),
		OutboxEventRepo: repos.NewOutboxEventRepo(db),
	}
}

func buildViews(db *sql.DB, logger *log.Logger) view {
	return view{
		ItemViewer: views.NewItemViewer(db, logger),
	}
}

func buildValidators() val {
	return val{
		ItemValidator:        validators.NewItemValidator(),
		OutboxEventValidator: validators.NewOutboxEventValidator(),
	}
}

func (d *DI) WithTx(tx *sql.Tx) *DI {
	if tx == nil {
		return d
	}
	return buildDI(d.db, tx, sharedDeps{
		Services: d.svc,
		Views:    d.view,
		Logger:   d.Logger,
	})
}

func (d *DI) Close() error {
	if d.db == nil {
		return nil
	}
	return d.db.Close()
}
