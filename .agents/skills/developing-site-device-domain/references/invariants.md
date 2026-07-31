# Site and device invariants

## Persistence authority

- MySQL is the source of truth for cities, sites, devices, cells, reservations, and device commands.
- Redis is never authoritative. It may cache site queries and reservation markers, but cache loss or flush must not change MySQL correctness.
- Reserve with a MySQL transaction using conditional row locking (`FOR UPDATE SKIP LOCKED` where supported), then write the Redis marker after commit.
- Release only when `cell_id`, reservation key, `LOCKED` status, and unbound-order conditions still match.
- Reclaim only expired, unbound, `LOCKED` rows and include the current reservation key in the update condition.

## Idempotency and commands

- Reservation, release, device command, retry, callback, and externally repeated requests require idempotency keys.
- Replaying the same request returns the original result. Reusing a key with different device, action, cell, or semantic JSON payload returns `IDEMPOTENCY_CONFLICT`.
- Persist terminal command results even if the caller cancels after the device response.
- Use stable states and errors such as `CELL_NOT_AVAILABLE`, `CELL_RESERVATION_CONFLICT`, `DEVICE_OFFLINE`, `DEVICE_COMMAND_TIMEOUT`, and `DEVICE_COMMAND_FAILED`.

## Internal API and simulator

- Every `/v1/internal` request requires a constant-time checked `X-Internal-Token`.
- A missing or empty configured token must fail startup outside tests. Development is not an authentication bypass.
- The simulator is test/development tooling. Production must not register or expose its scenario route, even behind authentication.
- High-risk real-device operations require explicit human confirmation and an audit record.

## Time and secrets

- Normalize stored and compared time to UTC; parse external timestamps explicitly.
- Redact DSNs, passwords, tokens, reservation keys, phone numbers, device credentials, payment material, and production endpoints.
- Log stable error codes, request trace IDs, counts, and hashed/truncated identifiers only.
- Never read or use desktop production credential files for development, CI, migrations, smoke tests, or documentation.

## Verification

```powershell
gofmt -w app api internal tests
go mod verify
go vet ./...
go test ./...
go test -tags=integration ./tests/integration
python scripts/test_agents_rules.py
python scripts/test_validate_skills.py
python scripts/validate_skills.py
git diff --check
```
