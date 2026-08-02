# yoga

Go service project template with local dependency injection and explicit service-owned composition roots.

This is not an imported DI library. The DI code belongs inside each service so the composition root stays explicit and domain-owned. Use this template as a starting point, then rename the module and replace the example `Item` domain with your own.

## What This Template Gives You

- Local `di.DI` with `svc`, `repo`, `view`, and `val` groups.
- Local `repos.DBTX` so repositories work with both `*sql.DB` and `*sql.Tx`.
- `di.ExecuteInTx` and `di.ExecuteInTxNoResult`.
- A short local `DI.WithTx` that rebuilds transaction-sensitive dependencies with the transaction executor.
- Panic-only wiring guardrails via `DI.MustValidateWiring`.
- Minimal HTTP handlers, logic action, repo, validator, view, model, and contract examples.

## Layout

```text
cmd/server       HTTP entrypoint
contract         request/response DTOs
di               service-local dependency graph, transactions, wiring checks
handlers         HTTP edge: decode, call logic/view, encode response
logic            business actions
models           domain models
repos            DBTX and write repositories
services         external services/shared utilities
validators       entity validators
views            read/query layer
```

## Architecture Rules

Keep ownership of behavior explicit:

- `handlers` only translate transport data: decode a request, construct a logic action, call it, and encode the result.
- `logic` owns use-case orchestration. Normalization happens only here, before data is mapped to a model or validated.
- Every action that has action-specific preconditions exposes its own `Validate` method in `logic`. For example, checking that a name is available belongs to `CreateItem.Validate`, because the rule applies to creating an item rather than to every valid item.
- Every validator is a concrete struct named after exactly one entity, such as `ItemValidator`. Its `Validate` method accepts that entity and checks rules that are true for the entity regardless of the action that produced it. Do not create action-named validators such as `CreateItemValidator`.
- Validators inspect data; they do not normalize or mutate it. They must not contain create-, update-, delete-, or transport-specific rules.
- `repos` own persistence, not business workflow. `views` own reads shaped for presentation. Neither layer normalizes input or decides which business action is allowed.

The expected action order is:

```text
normalize action input
        ↓
build the entity
        ↓
validate entity-wide rules with EntityValidator
        ↓
validate action-specific rules with Action.Validate
        ↓
perform repository writes
```

For the example domain, whitespace trimming belongs to `logic.CreateItem.Normalize`, the required name belongs to `validators.ItemValidator`, and name availability belongs to `logic.CreateItem.Validate`.

## Start A New Service

1. Copy this repo or use it as a GitHub template.
2. Change the module path in `go.mod`.
3. Replace the example `Item` files with your service domain.
4. Add fields to `di.repo`, `di.val`, `di.view`, and `di.svc`.
5. Wire new dependencies in `buildRepos`, `buildValidators`, `buildViews`, and `buildServices`.
6. Keep `DI.WithTx` local and rebuild transaction-sensitive dependencies from `tx`.

Run checks:

```sh
gofmt -w $(find . -name '*.go')
go test ./...
```

## DI Pattern

`NewDI` opens the root database and creates shared dependencies. `buildDI` creates either the root graph or a transaction graph.

```go
func NewDI(connStr string) (*DI, error) {
	db, err := NewDatabase(connStr)
	if err != nil {
		return nil, err
	}

	logger := log.New(os.Stdout, "[service] ", log.LstdFlags|log.Lmicroseconds)
	shared := sharedDeps{
		Services: buildServices(),
		Views:    buildViews(db, logger),
		Logger:   logger,
	}

	return buildDI(db, nil, shared), nil
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
```

The important boundary is simple: services, views, and the logger are shared, while transaction-sensitive dependencies are rebuilt with the current SQL executor.

## Repository Pattern

Repositories accept local `repos.DBTX`, not concrete `*sql.DB`.

```go
type DBTX interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type ItemRepo struct {
	db DBTX
}

func NewItemRepo(db DBTX) *ItemRepo {
	return &ItemRepo{db: db}
}
```

That lets `buildRepos(db)` and `buildRepos(tx)` use the same constructors.

## Transactions

Use `ExecuteInTx` when the action returns a result:

```go
action := CreateItem{Name: inputName}.Normalize()
item := models.Item{Name: action.Name}

result, err := di.ExecuteInTx(deps, func(txDI *di.DI) (*CreateItemResult, error) {
	if err := txDI.ItemValidator.Validate(item); err != nil {
		return nil, err
	}
	if err := action.Validate(ctx, txDI); err != nil {
		return nil, err
	}
	id, err := txDI.ItemRepo.Insert(ctx, item)
	if err != nil {
		return nil, err
	}
	item.ID = id
	return &CreateItemResult{Item: item}, nil
})
```

Use `ExecuteInTxNoResult` when only an error matters:

```go
err := di.ExecuteInTxNoResult(deps, func(txDI *di.DI) error {
	if err := txDI.ItemRepo.Delete(ctx, id); err != nil {
		return err
	}
	return txDI.OutboxRepo.Insert(ctx, "item.deleted", id)
})
```

Nested calls reuse the current transaction. If `txDI.tx` is already set, the helper runs the callback directly.

## Wiring Guardrails

`DI.MustValidateWiring` panics during startup or transaction graph construction when a required repo/validator field is nil or not a pointer.

```go
func (d *DI) MustValidateWiring() {
	var problems []string
	problems = append(problems, missingNilPtrFields(reflect.ValueOf(d.repo), "repo")...)
	problems = append(problems, missingNilPtrFields(reflect.ValueOf(d.val), "val")...)
	if len(problems) > 0 {
		panic(fmt.Sprintf("invalid DI wiring: %s", strings.Join(problems, ", ")))
	}
}
```

Keep this strict. Missing DI wiring should fail immediately, not appear later as a nil pointer in business logic.
