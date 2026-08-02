# Repository Guidelines

## Service Structure

- `cmd/server`: process entrypoint and route registration.
- `contract`: HTTP request and response DTOs.
- `di`: service-local dependency graph, transactions, and wiring checks.
- `handlers`: HTTP transport boundary.
- `logic`: business actions, normalization, orchestration, and action validation.
- `migrations`: schema owned by this service.
- `models`: persistence/domain entities.
- `repos`: DBTX-backed write and state-loading repositories.
- `services`: external clients and shared utilities.
- `validators`: entity validators.
- `views`: read queries shaped for presentation.

## Architecture Conventions

- A write handler decodes/parses input, maps it to one logic action, calls that action, and maps its result to a response DTO.
- A read handler parses query/path input and calls one viewer.
- Parse transport syntax in handlers. Normalize domain values only in `logic`.
- Keep request/response types in `contract`; models are not API inputs.
- Logic must not import `contract`.
- Define action input and result structs in `logic`; do not accept or return contract DTOs.
- Logic actions do not accept models as public input. Construct or merge models inside the action.
- Every action with action-specific preconditions exposes `Validate` in `logic`.
- Action validation owns raw ID checks, existence, state transitions, permissions, and action-specific uniqueness.
- Use one concrete validator struct per entity. `Validate` and optional `ValidateFields`/`ValidateWithout` methods must accept that entity model.
- Validators inspect models and never normalize or mutate them.
- Do not create request/action validators such as `CreateItemValidator`.
- Validate every model before insert or update.
- For partial model checks, select fields on the model; do not create validation DTOs.
- Repositories accept models for inserts/updates and local `repos.DBTX` for execution.
- Keep SQL and `sql.ErrNoRows` handling in repositories.
- Use explicit `ForUpdate` repository methods when validation depends on state that must remain stable until write.
- Do not inspect SQLSTATE or driver-specific errors in logic. Wrap repository failures in typed domain/internal errors.
- Views may return `contract` response types when they are explicitly the response-building read layer.
- Views return typed errors and do not leak raw infrastructure failures.
- Workers remain thin: schedule/log and call one logic action per cycle.
- Perform external side effects after commit or write an outbox event inside the transaction.

## Transactions and DI

- Run normalize, state loading, validation, preparation, and related writes inside the transaction closure when consistency matters.
- Nested actions may use `ExecuteInTx`/`ExecuteInTxNoResult`; an existing transactional DI is reused.
- Rebuild transaction-sensitive repositories and validators from the transaction executor in `DI.WithTx`.
- Run `MustValidateWiring` for root and transactional DI graphs.
- Do not add nil checks for required DI dependencies inside handlers or logic.

## Errors

- Use `github.com/PowerPenguini/errs` for stable error codes and HTTP mapping.
- Handlers own bad request errors caused by transport parsing.
- Logic owns validation, not-found, forbidden, and internal action errors.
- Validators may aggregate independent entity errors with `errs.ErrorList`.
- Preserve infrastructure errors as internal causes, but never return raw SQL/driver errors to handlers.

## Verification

- Format Go code with `gofmt`.
- Run `go test ./...` after changes.
- Run `go vet ./...` for structural or cross-package changes.
- Add focused tests for normalization, entity validators, transaction reuse, and DI wiring.
