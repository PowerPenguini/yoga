# Repository Guidelines

This file is the normative architecture guide for contributors and coding agents working in this service. The README may explain the same patterns with longer examples, but changes must follow the rules below.

## Template Intent

- This service follows the local composition-root architecture used by Frostbox/iqTMS.
- DI belongs to the service; do not extract the composition root into a shared DI framework.
- Keep domain and service boundaries explicit even when a shortcut would reduce the number of files.
- Treat the example `Item` domain as a reference vertical slice: transport mapping, logic actions, entity validation, persistence, read views, transactions, and outbox.

## Service Structure

- `cmd/server`: process entrypoint, process lifecycle, and route registration.
- `contract`: HTTP request and response DTOs.
- `di`: service-local dependency graph, transaction helpers, and wiring checks.
- `handlers`: HTTP transport boundary.
- `logic`: use-case actions, normalization, orchestration, and action validation.
- `migrations`: schema and indexes owned by this service.
- `models`: persistence/domain entities.
- `repos`: DBTX-backed writes and state-loading queries used by actions.
- `services`: external clients and shared utilities.
- `validators`: entity validators.
- `views`: read queries shaped for presentation.
- `workers`: scheduling and background-loop adapters when a service needs them.

Do not move behavior between these packages merely to avoid defining an explicit action, model, or mapping.

## Dependency Direction

The write path is:

```text
HTTP request
    -> contract request DTO
    -> handler mapping
    -> logic action
        -> entity validator
        -> repository
        -> nested logic action
        -> shared service
    -> logic result
    -> handler mapping
    -> contract response DTO
```

The read path is:

```text
HTTP read request
    -> handler transport parsing
    -> viewer query
    -> contract response
```

Enforce these dependency rules:

- `logic` must not import or depend on `contract`.
- `models` must not represent HTTP inputs or responses.
- `repos` must not accept `contract` DTOs.
- `validators` must not accept `contract` DTOs or logic action structs.
- Handlers may depend on `contract`, `logic`, `views`, and `di`.
- Logic may depend on `di`, `models`, typed errors, and domain helpers/services.
- Repositories and validators may depend on `models`, but must not depend on handlers.

## Handlers and Contracts

- A write handler must perform one top-level action:
  1. parse transport syntax;
  2. decode one request DTO;
  3. map the DTO to one logic action;
  4. execute the action;
  5. map its result to a response DTO;
  6. encode the response.
- A read handler parses path/query values and calls one viewer.
- Parse transport syntax at the HTTP edge: path IDs, query booleans, dates, UUIDs, enums encoded as strings, and JSON shape.
- Keep JSON names, optional request fields, and response serialization in `contract`.
- Models are never request DTOs.
- Do not pass a request DTO into logic or a repository.
- Do not return a logic result directly as an HTTP response; map it to a `contract` response.
- Do not perform domain normalization in handlers.
- Do not implement business workflow, entity state checks, or SQL-error interpretation in handlers.
- Use `errs.WriteError` for typed errors.
- Transport parsing/decoding failures belong to the handler and use `errs.BadRequestType`.

## Logic Actions

- Represent every use case as a concrete action struct such as `CreateItem`, `UpdateItem`, or `ArchiveItem`.
- Define action input and result structs in `logic`.
- Action inputs and results contain primitives, typed IDs, and logic-local types; they never contain `contract` DTOs.
- Do not accept a model as the public input to an action. Construct or merge models inside the action.
- Do not return `contract` types from logic.
- Keep orchestration in `Execute`.
- If an action returns data, define a logic-local `Result` struct instead of returning a contract or persistence model by convenience.

### Normalization

- `Normalize` methods and normalization helpers exist only in `logic`.
- Normalize inside the transaction closure when normalized data participates in state-dependent validation or writes.
- Normalization must be deterministic and idempotent.
- Normalization may trim whitespace, canonicalize case, normalize optional values, and preserve PATCH intent.
- Validators may trim into local variables for inspection, but must never mutate or canonicalize the supplied model.
- Handlers, repositories, validators, and views must not normalize domain input.

### Action Validation

- Every action with action-specific preconditions exposes a `Validate` method in `logic`.
- Action validation owns:
  - required raw/typed IDs;
  - the requirement that a PATCH contains at least one field;
  - entity existence for the action;
  - allowed state transitions;
  - permissions and policy checks;
  - uniqueness or availability specific to create/update behavior;
  - duplicate entries in action input;
  - assign-like raw identifier and target existence checks.
- When no complete model exists, validate the bare input directly in the action; do not invent a DTO validator.
- A `Validate` method may return locked/current state needed to build the final model.
- If validation depends on current database state, perform it inside the write transaction.
- Use explicit repository locking methods when the checked state must remain stable until the write.
- Do not add nil checks for required DI dependencies. DI wiring is a composition-root responsibility.

### Action Ordering

There is no single rigid order for all actions. Use the flow appropriate to the action while preserving transaction consistency.

Create flow:

```text
Normalize action
    -> build model
    -> entity validator
    -> action Validate for create-specific state
    -> repository insert
    -> nested writes/outbox
    -> commit
```

Update flow:

```text
Normalize action
    -> action Validate and SelectByIDForUpdate
    -> merge input into locked model
    -> entity validator
    -> repository update
    -> nested writes/outbox
    -> commit
```

ID/state-only action flow:

```text
action Validate raw ID and load/lock state
    -> apply transition to model
    -> entity validator
    -> repository update
    -> nested writes/outbox
    -> commit
```

The invariant is that normalization, state loading, action validation, model validation, preparation, and related writes that must be consistent run in the same transaction.

### Partial Updates

- Pointer input fields represent PATCH presence: `nil` means omitted.
- Preserve omission intent during normalization.
- Define and document how a client clears an optional value.
- In the reference `UpdateItem`, omitted `description` means unchanged and an empty string clears the value.
- Load the existing entity with `FOR UPDATE`, merge supplied values, then validate the resulting model.
- If only selected entity fields can be constructed, call `ValidateFields` or `ValidateWithout` with the model. Never create `UpdateXValidationInput`.

### Nested Actions and Side Effects

- A parent action may call another logic action with its `txDI`.
- Nested `ExecuteInTx` or `ExecuteInTxNoResult` calls reuse an existing transaction and must not open a second transaction.
- Use nested actions for reusable business operations, not as a substitute for simple private helpers.
- Perform irreversible/external side effects only after commit.
- When an external notification must be atomic with a domain change, write an outbox event inside the transaction.
- Validate an outbox model before inserting it.
- A publisher worker processes unpublished outbox rows after commit; the write action does not publish directly.

## Entity Validators

- Use one concrete validator struct per entity, for example `ItemValidator` and `OutboxEventValidator`.
- Name validators after entities, never actions or requests.
- Do not create `CreateItemValidator`, `UpdateItemValidator`, request validators, or `*ValidationInput` DTOs.
- `Validate` accepts the complete model represented by the validator.
- Entity validators own rules that remain true regardless of the producing action:
  - required model fields;
  - length limits;
  - formats;
  - numeric/date ranges;
  - relationships between fields on the model;
  - action-independent entity relationships.
- Validators inspect models and never mutate them.
- Validators do not normalize models.
- Validators do not own create-, update-, delete-, archive-, transport-, permission-, or transition-specific rules.
- Validate every model before every insert or update.
- Use `ValidateFields(fields, model)` to check selected fields.
- Use `ValidateWithout(fields, model)` when most standard fields apply except an explicit subset.
- Field-selection methods must still accept the entity model.
- Reject unsupported field names instead of silently treating them as valid.
- Validators may return `errs.ErrorList` to aggregate independent entity failures.
- Validators assume constructor/DI dependencies are present; do not guard against missing wiring inside validation methods.

## Repositories

- Define a service-local `repos.DBTX` interface implemented by both `*sql.DB` and `*sql.Tx`.
- Repository constructors accept `repos.DBTX`, never only a concrete `*sql.DB`.
- Insert and update methods accept entity models.
- Do not accept HTTP DTOs or long lists of loose model fields in repository write methods.
- Repositories own SQL, scanning, affected-row checks, and driver details.
- Repositories may translate `sql.ErrNoRows` into `nil` or a boolean so logic stays driver-agnostic.
- Expose explicit state-loading methods such as `SelectByIDForUpdate`.
- Expose explicit locking existence checks such as `ExistsByCodeForUpdate` or `ExistsByCodeOtherForUpdate` when needed by an action.
- Return infrastructure errors to logic; do not map them to HTTP responses.
- Database constraints remain the final concurrency/integrity safety net even when logic performs a friendly pre-check.
- Logic must not inspect SQLSTATE, database-driver error codes, or `sql.ErrNoRows`.
- For generated identifiers that require an availability check, query through a repository and retry generation without branching on SQL error codes.

## Views

- Use viewers for read queries shaped for a screen or API response.
- Viewers may return `contract` response types when they are explicitly the response-building read layer.
- A read handler calls one viewer rather than reconstructing a read model through write repositories.
- Viewers use the root `*sql.DB`, not the write transaction graph.
- Viewers may define typed query structs for filters, pagination, and cursors.
- Viewers log infrastructure detail with the service logger.
- Viewers return typed domain/internal errors and never leak raw SQL/driver errors to handlers.
- Return stable empty collections when the API contract expects a collection rather than `null`.

## Services and Workers

- Services are shared clients/utilities such as clocks, object stores, mailers, publishers, or downstream API clients.
- Share services across root and transaction DI graphs unless they are explicitly transaction-sensitive.
- Workers stay thin like handlers:
  1. load/normalize worker configuration;
  2. invoke one logic action per cycle/item;
  3. own scheduling, cancellation, logging, and retry timing only.
- Business decisions performed by a worker still belong to logic actions.
- Workers must not bypass validators or repositories with ad hoc SQL writes.

## Typed Errors

Use `github.com/PowerPenguini/errs` consistently:

- `errs.BadRequestType`: malformed transport input; normally owned by handlers; HTTP 400.
- `errs.ValidationType`: invalid entity or action precondition; owned by logic/validators; HTTP 422.
- `errs.NotFoundType`: missing entity established by an action lookup; HTTP 404.
- `errs.UnauthorizedType`: missing/invalid authentication; HTTP 401.
- `errs.ForbiddenType`: authenticated caller lacks permission; HTTP 403.
- `errs.InternalType`: wrapped infrastructure or unexpected action/view failure; HTTP 500.

Additional error rules:

- Use stable, specific error codes and concise user-facing messages.
- Use field errors when a failure belongs to a particular input/model field.
- Logic generally returns one precise action error.
- Validators may aggregate independent entity errors with `errs.ErrorList`.
- Preserve the original infrastructure error as the cause of an internal error.
- Never expose raw SQL, driver, filesystem, network, or downstream-client errors to handlers/clients.
- Handlers call `errs.WriteError`; they do not duplicate status-selection switches.
- Views and logic must wrap raw infrastructure failures before returning them.

## Transactions

- Use `ExecuteInTx` when the action returns a result.
- Use `ExecuteInTxNoResult` when only an error matters.
- If `d.tx` already exists, transaction helpers execute the callback directly with the existing DI.
- On action error, roll back and return the original typed error.
- On commit failure, return the commit failure as an internal/infrastructure failure at the appropriate boundary.
- Keep all state-dependent checks and related writes in the same transaction.
- Do not start manual nested SQL transactions inside logic actions.
- Do not use a root repository from inside a transactional action.

## DI and Wiring

- Keep `DI`, `svc`, `repo`, `view`, and `val` groups local to the service.
- `NewDI` creates the root database and shared dependencies.
- `buildDI` composes both root and transaction graphs.
- `WithTx` rebuilds every transaction-sensitive repository using the `*sql.Tx` executor.
- Rebuild validators with transaction-scoped repository dependencies when validators have such dependencies.
- Share services, viewers, and the logger when they are not transaction-sensitive.
- Register every dependency in its DI group and constructor builder.
- Run `MustValidateWiring` during both root DI creation and transaction graph construction.
- Required repo and validator fields must be non-nil pointers.
- Missing wiring must fail immediately; do not defer it to handler/logic execution.
- Tests must prove transaction-scoped repositories use `*sql.Tx`.
- Tests must prove nested transaction helpers do not begin a second transaction.

## Ownership Examples

- JSON field names and optional request fields -> `contract`.
- Path ID parsing -> handler.
- Name trimming and code uppercasing -> `Action.Normalize` in `logic`.
- At least one PATCH field -> `UpdateAction.Validate`.
- Entity existence and non-archived state before update -> `UpdateAction.Validate`.
- Code availability for create/update -> the corresponding action `Validate`.
- Required item name and code format -> `ItemValidator.Validate`.
- Selected entity-field validation -> `ItemValidator.ValidateFields`.
- SQL and `sql.ErrNoRows` handling -> repository.
- Response-shaped list query -> viewer.
- Atomic domain write and event record -> logic transaction plus outbox.

## Database and Migrations

- Store schema changes in `migrations` using paired up/down files.
- Keep schema, constraints, and indexes owned by this service.
- Add database constraints for final integrity even when actions perform validation pre-checks.
- Add indexes required by repository and worker access patterns.
- The reference outbox keeps `published_at` for an external publisher worker.
- Do not run ad hoc schema mutations from handlers, logic actions, repositories, or application startup unless the service explicitly adopts a migration runner.

## Adding A Domain Slice

When adding or replacing a domain:

1. define request/response DTOs in `contract`;
2. define logic-local action and result structs;
3. add `Normalize`, action `Validate`, and `Execute`;
4. define the persisted model;
5. add exactly one validator per model;
6. add DBTX-backed repositories;
7. add a viewer for response-shaped reads;
8. add typed errors at each boundary;
9. add migrations and database constraints;
10. register routes;
11. register repos, validators, views, and services in DI;
12. ensure `WithTx` rebuilds transaction-sensitive dependencies;
13. add focused normalization, validation, repository/logic, transaction, and wiring tests.

## Verification

- Format all Go code with `gofmt`.
- Run `git diff --check`.
- Run `go test ./...` after every behavior or architecture change.
- Run `go vet ./...` for structural or cross-package changes.
- Run `go test -race ./...` for transaction, concurrency, worker, or shared-state changes.
- Check that `logic` does not import `contract`.
- Check that handlers and validators do not contain SQL/driver-specific logic.
- Check that repositories do not accept `contract` DTOs.
- Check that every inserted/updated model is validated first.
- Check root and transactional DI wiring.
- Check nested actions reuse the active transaction.
- Check typed errors reach handlers instead of raw infrastructure errors.
- Keep unrelated worktree changes out of the commit.
