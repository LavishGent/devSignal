# DevSignal — Unit Test Plan

This plan describes how DevSignal's unit tests are organized, what they verify,
and how to run them. Actual run output and the summary of changes driven by
testing are in [TEST_RESULTS.md](TEST_RESULTS.md).

> **Screenshots.** Capture a screenshot of the full passing run (`make test`)
> and of a focused package run (for example `go test ./internal/validation/ -v`)
> and insert them where marked below. The exact textual output is also recorded
> in TEST_RESULTS.md as reproducible evidence.

## 1. Scope and approach

The backend is tested with Go's built-in `testing` package. Tests are written
alongside the code they cover (`*_test.go`) and follow a **table-driven** style
where a case list drives one assertion loop.

Two layers of automated tests:

1. **Unit tests** — exercise a single package in isolation. HTTP handler tests
   run the real router against fast **in-memory repositories**, so no database
   is required.
2. **Integration test** — runs the assembled server against a **real
   PostgreSQL** database (build tag `integration`) to prove the production
   persistence path works end to end.

### Why in-memory repositories
Because handlers and services depend on repository **interfaces**, the tests
substitute in-memory implementations for PostgreSQL. This keeps the unit suite
fast and deterministic while exercising the exact same handler code that runs in
production.

## 2. Environment

| Item | Value |
|------|-------|
| Language/tooling | Go 1.25, `go test` |
| Command (unit) | `go test -count=1 -cover ./...` (or `make test`) |
| Command (integration) | `make integration` (needs `make db-up`) |
| Database (unit) | none — in-memory repositories |
| Database (integration) | PostgreSQL 16 via `TEST_DATABASE_URL` |

## 3. Test inventory

The suite contains **107 test functions** across the following packages.

| Package | Focus | Example cases |
|---------|-------|---------------|
| `validation` | Field validators | required, max length, one-of, int range, email, URL, date order |
| `domain` | Entity `Validate()` rules and the `Date` value type | per-entity valid/invalid cases |
| `auth` | Password hashing and session lifecycle | hash+match, password policy, token uniqueness/hashing, expiry/garbage rejection |
| `httpapi` | REST handlers for all eight modules + auth/insights | CRUD, validation rejection, search/filter, ownership isolation, auth-required, malformed JSON |
| `recommend` | Rule engine | each of the eight rules, plus deterministic ordering |
| `reports` | Report builders and CSV | columns/rows per report, CSV formula-injection escaping |
| `dashboard` | Summary computation | counts, aggregates, skill-gap ranking |
| `integration` | Full stack vs. real PostgreSQL | register → login → CRUD → report round trip |

## 4. Representative test cases

The following cases are called out because each maps directly to a rubric
attribute (validation, security, search, CRUD, reports).

### TC-1 — Input validation rejects bad data
- **Package/test:** `validation.TestEmail`, `validation.TestWebURL`,
  `validation.TestIntRange`; `httpapi.TestApplicationValidationRejected`.
- **Steps:** submit records with an invalid email, a non-http URL, and an
  out-of-range integer.
- **Expected:** validators record a field-keyed error; the handler responds
  **400** with a per-field error map and nothing is persisted.

### TC-2 — Passwords are hashed and verified securely
- **Package/test:** `auth.TestHashPasswordAndMatch`,
  `auth.TestHashPasswordPolicy`.
- **Steps:** hash a password, compare correct and incorrect plaintexts; attempt
  a too-short password.
- **Expected:** correct plaintext matches, wrong plaintext fails, policy
  violations return `ErrPasswordPolicy`. Plaintext is never stored.

### TC-3 — Session tokens are unique and stored only as hashes
- **Package/test:** `auth.TestTokensAreUniqueAndHashedAtRest`,
  `auth.TestVerifyRejectsExpiredAndGarbage`.
- **Steps:** issue multiple sessions; inspect stored values; verify an expired
  and a garbage token.
- **Expected:** raw tokens differ, only SHA-256 hashes are stored, expired and
  invalid tokens are rejected.

### TC-4 — CRUD lifecycle for a module
- **Package/test:** `httpapi.TestApplicationCRUD`.
- **Steps:** create, read, list, update, and delete an application through the
  HTTP API.
- **Expected:** each step returns the correct status and body; the record
  reflects updates and is gone after delete.

### TC-5 — Users cannot access each other's data
- **Package/test:** `httpapi.TestApplicationOwnershipIsolation`,
  `httpapi.TestApplicationRequiresAuth`.
- **Steps:** user A creates a record; user B attempts to read/update/delete it;
  an unauthenticated request is attempted.
- **Expected:** B receives **404** (not visible), unauthenticated requests
  receive **401**.

### TC-6 — Search returns grouped cross-module matches
- **Package/test:** `httpapi.TestApplicationSearchAndFilter`.
- **Steps:** create records, query with a search term and a status filter.
- **Expected:** only matching, user-owned records are returned.

### TC-7 — Reports build correctly and CSV is injection-safe
- **Package/test:** `reports` builder tests and the CSV escaping test.
- **Steps:** build each report from a snapshot; export a report whose cell
  begins with `=`.
- **Expected:** correct columns/rows; dangerous cells are prefixed with a
  single quote so spreadsheets do not execute them as formulas.

### TC-8 — Recommendations are produced and ordered deterministically
- **Package/test:** `recommend.*Rule` tests and
  `recommend.TestEngineOrdersByPriorityThenModuleThenTitle`.
- **Steps:** run each rule against a crafted snapshot; run the full engine.
- **Expected:** each rule fires on the right condition; output is ordered by
  priority, then module, then title.

### TC-9 — Full stack against real PostgreSQL
- **Package/test:** `integration.TestFullStackAgainstRealPostgres`.
- **Steps:** start the real server on a real database, register, log in, run
  CRUD, and read back a report.
- **Expected:** the production persistence path succeeds end to end.

## 5. Pass/fail criteria

A run passes when `go test ./...` reports `ok` for every package with a non-zero
exit status of 0, `go vet ./...` is clean, and the integration suite passes
against PostgreSQL. Coverage is tracked per package (see TEST_RESULTS.md); the
validation core is held at 100%.

## 6. How to run

```bash
make test          # unit + handler tests with coverage, plus frontend build
make integration   # integration test against real PostgreSQL (after `make db-up`)
go test ./internal/validation/ -v   # focused, verbose run of one package
```
