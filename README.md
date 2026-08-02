# yoga

Go service template based on the service architecture used in Frostbox/iqTMS. It keeps dependency injection local to the service, separates HTTP contracts from business actions and models, and makes transaction ownership explicit.

This repository is a cleaned reference slice, not a shared DI framework. Copy it, rename the module, and replace the example `Item` domain with the domain of the new service.

## What The Example Covers

The example is intentionally larger than a single create endpoint:

- create, partial update, archive, and list flows;
- request/response DTO mapping at the HTTP boundary;
- logic-local action and result types;
- normalization owned by logic;
- action-specific and entity-wide validation;
- typed domain errors and HTTP error mapping;
- repositories working with both `*sql.DB` and `*sql.Tx`;
- row locking for state-dependent actions;
- nested actions that reuse an active transaction;
- an outbox event written atomically with the domain change;
- a read-optimized viewer;
- DI wiring guardrails for both root and transactional graphs;
- repository-level `AGENTS.md` conventions for future contributors and coding agents;
- focused unit and transaction tests.

## Layout

```text
cmd/server       process entrypoint and route registration
contract         HTTP request and response DTOs
di               service-local composition root, transactions, wiring checks
handlers         HTTP edge: parse/decode, map, call one action or viewer, encode
logic            use-case actions, normalization, orchestration, action validation
migrations       schema owned by this service
models           persistence/domain entities
repos            DBTX-backed write and state-loading repositories
services         external clients and shared utilities
validators       entity validators
views            read/query layer shaped for presentation
```

## Dependency Direction

```text
HTTP request
    │
    ▼
contract DTO ──mapped by handler──▶ logic action
                                      │
                                      ├──▶ entity validator
                                      ├──▶ repository
                                      ├──▶ nested logic action
                                      └──▶ shared service

HTTP read request ──parsed by handler──▶ viewer ──▶ contract response
```

`logic` must not import `contract`. Repositories must not accept request DTOs. Models do not represent HTTP inputs.

## Architecture Rules

### Handlers and contracts

- A handler parses transport values, decodes one `contract` request, maps it to one logic action, invokes it, and maps the result to a `contract` response.
- Parse transport syntax at the edge: path IDs, query booleans, dates, UUIDs, and JSON shape belong to handlers.
- Domain normalization does not belong to handlers. Trimming, case normalization, optional-value cleanup, and canonical forms belong to the action in `logic`.
- Models are never request DTOs.
- A write handler calls one top-level logic action. A read handler calls one viewer.
- Handlers write typed errors with `errs.WriteError`; they do not translate raw SQL errors themselves.

### Logic actions

- Each use case is a concrete action struct, for example `CreateItem`, `UpdateItem`, or `ArchiveItem`.
- Action inputs and results are local to `logic`. They contain primitives and typed IDs, not `contract` DTOs.
- Actions do not accept models as public input. They construct or update models internally.
- `Normalize` exists only in `logic`. It should be deterministic and idempotent.
- Action-specific preconditions belong to `Action.Validate`: required raw IDs, entity existence, allowed state transitions, permissions, uniqueness for that action, and duplicate input entries.
- If validation depends on current state, load and lock that state in the same transaction used for the write.
- Map repository failures to precise domain/internal errors. Do not inspect SQLSTATE or driver-specific errors in logic.
- Return a logic-local result and let the handler build the response DTO.

### Entity validators

- Every validator is a concrete struct named after exactly one entity, such as `ItemValidator` or `OutboxEventValidator`.
- `Validate` accepts the complete model represented by that validator.
- Entity validators check rules that remain true regardless of the action: required model fields, lengths, formats, ranges, and relationships between model fields.
- Validators inspect data; they do not normalize or mutate it.
- Do not create action-named validators such as `CreateItemValidator`.
- For targeted validation, expose `ValidateFields(fields, model)` or `ValidateWithout(fields, model)`. These methods still accept the entity model; never introduce a `ValidationInput` DTO.
- Validate every model immediately before an insert or update.

### Repositories

- Repositories accept the local `repos.DBTX` interface, never a concrete `*sql.DB`.
- Insert and update methods accept models. They do not accept HTTP DTOs or large lists of loose arguments.
- Repositories own SQL and driver details. They may translate `sql.ErrNoRows` into `nil` or a boolean so logic stays driver-agnostic.
- Expose explicit locking methods such as `SelectByIDForUpdate` and `ExistsByCodeOtherForUpdate` when an action makes a state-dependent decision before writing.
- Return infrastructure errors to logic; logic wraps them as domain/internal errors.
- Database constraints remain the final concurrency safety net even when an action performs a friendly pre-check.

### Views

- Viewers own read queries shaped for a screen or API response.
- A viewer may return `contract` response types when it is explicitly the response-building layer.
- Viewers use the root `*sql.DB`, not the write transaction graph.
- Viewers log infrastructure detail and return typed domain/internal errors. They never leak raw database errors to handlers.

### Services and workers

- Services are shared clients or utilities such as a clock, object store, mailer, or message publisher.
- Workers should stay thin like handlers: normalize worker configuration, invoke one logic action per cycle, and own only scheduling/logging.
- External side effects should happen after commit or through an outbox written inside the transaction.

## Where A Rule Belongs

| Rule | Owner | Example |
| --- | --- | --- |
| JSON field names and optional request fields | `contract` | `UpdateItemRequest` |
| Path ID must parse as a positive integer | `handler` | `parseItemID` |
| Trim a name and uppercase a code | `logic.Normalize` | `CreateItem.Normalize` |
| At least one PATCH field is present | `logic.Validate` | `UpdateItem.Validate` |
| Item must exist and not be archived before update | `logic.Validate` | `UpdateItem.Validate` |
| Code must be available for this create/update action | `logic.Validate` | `CreateItem.Validate`, `UpdateItem.Validate` |
| Item name is required and code has a valid format | entity validator | `ItemValidator.Validate` |
| Only selected entity fields should be checked | entity validator | `ItemValidator.ValidateFields` |
| SQL and `sql.ErrNoRows` handling | repository | `ItemRepo.SelectByIDForUpdate` |
| Response-shaped list query | viewer | `ItemViewer.List` |
| Domain change and integration event must be atomic | logic + transaction | `CreateItem` + `EnqueueItemEvent` |

## Typed Errors

The template uses the same `github.com/PowerPenguini/errs` package as Frostbox:

| Error type | Typical owner | HTTP status |
| --- | --- | --- |
| `BadRequestType` | handler parsing/decoding | 400 |
| `ValidationType` | logic or entity validator | 422 |
| `NotFoundType` | logic after a repository lookup | 404 |
| `InternalType` | logic/view wrapping infrastructure failure | 500 |

Validators may return `errs.ErrorList` to report several independent entity problems in one response. Logic should usually return one precise action failure. Preserve the infrastructure error as the cause of an internal error for logs and debugging, but expose only the stable code/message through the HTTP layer.

## Reference Flows

There is no single rigid ordering for every action. Creation can build a new entity immediately, while update must first load existing state. The invariant is that normalization, state-dependent checks, model validation, and writes that belong together run in one transaction.

### Create

```text
handler maps CreateItemRequest to CreateItem
    ↓
ExecuteInTx
    ↓
CreateItem.Normalize
    ↓
build models.Item
    ↓
ItemValidator.Validate
    ↓
CreateItem.Validate (code availability)
    ↓
ItemRepo.Insert
    ↓
EnqueueItemEvent.Execute(txDI)
    ↓
OutboxEventValidator.Validate + OutboxEventRepo.Insert
    ↓
commit
    ↓
handler maps CreateItemResult to CreateItemResponse
```

### Partial update

```text
handler parses item_id and maps UpdateItemRequest to UpdateItem
    ↓
ExecuteInTx
    ↓
UpdateItem.Normalize (nil still means omitted)
    ↓
UpdateItem.Validate
    ├── require at least one patch field
    ├── SelectByIDForUpdate
    ├── reject archived items
    └── verify changed code against other rows
    ↓
merge patch into the locked models.Item
    ↓
ItemValidator.Validate
    ↓
ItemRepo.Update + nested outbox action
    ↓
commit
```

For `description`, an omitted field means “leave unchanged”, while an empty string means “clear the optional value”. That intent is preserved during normalization and resolved while constructing the final model.

### Archive

`ArchiveItem` demonstrates validation of a bare ID when no new model input exists. Its `Validate` method checks the ID, locks and loads the entity, verifies the transition, and returns the current model. The action then sets `ArchivedAt`, validates the resulting entity, updates it, and enqueues `item.archived`.

### Nested actions and outbox

`EnqueueItemEvent.Execute` uses `ExecuteInTxNoResult` even when called by another transactional action. Transaction helpers detect `txDI.tx` and invoke the nested callback directly, so no second transaction is opened.

This makes the item write and outbox write atomic. A real service can add a worker that publishes unpublished outbox rows after commit.

## DI Pattern

`NewDI` creates the root graph. `buildDI` creates either the root graph or a transaction graph:

```go
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
```

Services, views, and the logger are shared. Repositories are rebuilt with the current SQL executor. Validators are reconstructed as part of the graph; validators that need transaction-sensitive dependencies must receive the transaction-scoped repository instances.

`DI.MustValidateWiring` runs for both `NewDI` and `WithTx`. Missing repository or validator wiring fails immediately rather than becoming a nil-pointer failure inside business logic.

## Transaction Helpers

Use `ExecuteInTx` when an action returns a result and `ExecuteInTxNoResult` when only an error matters.

```go
return di.ExecuteInTx(deps, func(txDI *di.DI) (*CreateItemResult, error) {
	action.Normalize()
	item := &models.Item{
		Name: action.Name,
		Code: action.Code,
	}

	if err := txDI.ItemValidator.Validate(item); err != nil {
		return nil, err
	}
	if err := action.Validate(ctx, txDI); err != nil {
		return nil, err
	}

	id, err := txDI.ItemRepo.Insert(ctx, item)
	if err != nil {
		return nil, errs.NewError(
			"item_create_failed",
			"failed to create item",
			errs.InternalType,
			err,
		)
	}
	item.ID = id

	event := EnqueueItemEvent{
		Topic:  ItemCreatedTopic,
		ItemID: item.ID,
		Data:   map[string]any{"code": item.Code},
	}
	if err := event.Execute(ctx, txDI); err != nil {
		return nil, err
	}

	return &CreateItemResult{
		ID:   item.ID,
		Name: item.Name,
		Code: item.Code,
	}, nil
})
```

Normalize, validate, prepare, and write inside the closure whenever they depend on data that must remain consistent with the write.

## HTTP Endpoints

```text
POST   /items             create an item
GET    /items             list active items
GET    /items?include_archived=true
PATCH  /items/{item_id}   partially update an active item
DELETE /items/{item_id}   archive an item
GET    /health
```

## Database

Apply the example migration to PostgreSQL:

```sh
psql "$DATABASE_URL" -f migrations/000001_items.up.sql
```

The outbox table intentionally contains `published_at` even though this template does not include a publisher worker. It marks the boundary for a service-specific worker implementation.

## Start A New Service

1. Copy this repository or use it as a GitHub template.
2. Change the module path in `go.mod`.
3. Replace the example `Item` and `OutboxEvent` domain files.
4. Define request/response DTOs in `contract`.
5. Add logic-local action and result structs.
6. Add one validator per persisted model.
7. Add DBTX-backed repositories and read viewers.
8. Register dependencies in `di.repo`, `di.val`, `di.view`, and `di.svc`.
9. Wire constructors in `buildRepos`, `buildValidators`, `buildViews`, and `buildServices`.
10. Add migrations, routes, typed errors, and focused tests.
11. Confirm both root and transaction DI graphs pass wiring validation.

Run checks:

```sh
gofmt -w $(find . -name '*.go')
go test ./...
```
