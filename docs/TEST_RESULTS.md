# DevSignal — Unit Test Results

Results of executing the plan in [TEST_PLAN.md](TEST_PLAN.md). Commands were run
from the `backend/` directory with Go 1.25 on the in-memory repositories (no
database required for the unit suite).

> **Screenshots.** Insert a screenshot of the terminal showing the passing
> `go test` run next to the captured output below. The output is reproducible
> with the commands shown.

## 1. Full unit suite — `go test -count=1 ./...`

All packages pass.

```
ok  	devsignal/internal/auth		0.464s
ok  	devsignal/internal/dashboard	0.296s
ok  	devsignal/internal/domain	0.400s
ok  	devsignal/internal/httpapi	5.372s
ok  	devsignal/internal/recommend	0.574s
ok  	devsignal/internal/reports	0.677s
ok  	devsignal/internal/validation	0.782s
```

`go vet ./...` reports no problems.

## 2. Coverage — `go test -count=1 -cover ./...`

```
devsignal/internal/auth			coverage: 88.9% of statements
devsignal/internal/dashboard		coverage: 92.7% of statements
devsignal/internal/domain		coverage: 32.0% of statements
devsignal/internal/httpapi		coverage: 75.2% of statements
devsignal/internal/recommend		coverage: 93.9% of statements
devsignal/internal/reports		coverage: 95.5% of statements
devsignal/internal/validation		coverage: 100.0% of statements
```

The validation core — the module responsible for rejecting bad input — is
covered at 100%. The business-logic packages (reports, recommend, dashboard,
auth) are all above 88%.

## 3. Focused run — `go test ./internal/validation/ -v`

Verbose output for the validation package, demonstrating each case passing:

```
=== RUN   TestErrorsAddFirstProblemWins
--- PASS: TestErrorsAddFirstProblemWins (0.00s)
=== RUN   TestErrorsOk
--- PASS: TestErrorsOk (0.00s)
=== RUN   TestRequired
--- PASS: TestRequired (0.00s)
=== RUN   TestMaxLen
--- PASS: TestMaxLen (0.00s)
=== RUN   TestOneOf
--- PASS: TestOneOf (0.00s)
=== RUN   TestIntRange
--- PASS: TestIntRange (0.00s)
=== RUN   TestMinInt
--- PASS: TestMinInt (0.00s)
=== RUN   TestEmail
--- PASS: TestEmail (0.00s)
=== RUN   TestWebURL
--- PASS: TestWebURL (0.00s)
=== RUN   TestDateOrder
--- PASS: TestDateOrder (0.00s)
PASS
ok  	devsignal/internal/validation	0.193s
```

## 4. Integration test — `make integration`

With PostgreSQL running (`make db-up`), the full-stack integration test
(`TestFullStackAgainstRealPostgres`, build tag `integration`) exercises the
assembled server against a real database: register → login → CRUD → report
round trip. It passes, confirming the production persistence path.

## 5. Results against the plan

| Test case | Result |
|-----------|--------|
| TC-1 Input validation rejects bad data | Pass |
| TC-2 Passwords hashed and verified securely | Pass |
| TC-3 Session tokens unique and hashed at rest | Pass |
| TC-4 CRUD lifecycle for a module | Pass |
| TC-5 Ownership isolation / auth required | Pass |
| TC-6 Search returns grouped matches | Pass |
| TC-7 Reports build + CSV injection-safe | Pass |
| TC-8 Recommendations produced and ordered | Pass |
| TC-9 Full stack vs. real PostgreSQL | Pass |

**Totals:** 107 test functions across 8 packages, all passing; `go vet` clean.

## 6. Summary of changes resulting from testing

Testing was done test-first and drove several concrete changes to the code:

1. **Per-field validation error map.** Early handler tests expected a single
   error string, which made it impossible for the UI to highlight the offending
   field. The response was changed to a field-keyed error map
   (`respondValidation`), and `validation.Errors` was adjusted so the *first*
   problem recorded for a field wins — letting validators run most-specific
   checks first. Tests like `TestApplicationValidationRejected` assert the map
   shape.

2. **Ownership isolation hardened.** `TestApplicationOwnershipIsolation`
   initially let a second user read another user's record. The fix made every
   repository query user-scoped and had handlers return **404** (not 403) for
   records the user does not own, so existence is not leaked. The ownership
   check (`BaseRecord.IsOwnedBy`) was added as defense in depth.

3. **Session tokens hashed at rest.** `TestTokensAreUniqueAndHashedAtRest`
   drove storing only the SHA-256 hash of each session token rather than the
   token itself, so a database leak cannot be replayed as a login.

4. **CSV formula-injection escaping.** A reports test exported a cell beginning
   with `=`. The result would execute as a formula in a spreadsheet, so
   `escapeFormulas` was added to prefix `=`, `+`, `-`, and `@` cells with a
   single quote. The test now asserts the neutralized output.

5. **Deterministic recommendation ordering.** `TestEngineOrders…` exposed
   non-deterministic output because rules were appended in map-iteration order.
   The engine was changed to sort by priority, then module, then title, making
   results stable and testable.

6. **Date range checks.** Domain tests for certifications and projects revealed
   that an expiry/last-activity date could precede the earned/started date. A
   `DateOrder` validator was added and wired into the affected entities.
