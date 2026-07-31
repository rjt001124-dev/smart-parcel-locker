---
name: developing-site-device-domain
description: Use when changing smart parcel locker cities, sites, devices, cells, reservations, heartbeats, device commands, simulators, MySQL repositories, Redis acceleration, internal APIs, workers, or related tests
---

# Developing Site and Device Domain

## Core rule

MySQL is the source of truth for cell allocation, reservation ownership, heartbeats, and device commands. Redis is never authoritative and its loss must not create duplicate allocation or corrupt persisted state.

## Workflow

1. Read `references/invariants.md`, the relevant migration, biz interface, use case, and repository before editing.
2. Write a failing test for the domain invariant or failure mode.
3. Keep Kratos dependencies `service -> biz -> data`; handlers only authenticate, validate, call use cases, and map responses.
4. Enforce reservation, release, command creation, and retry idempotency in MySQL transactions or conditional updates.
5. Treat Redis writes as best-effort acceleration after MySQL commits; Redis failure is degradation, not rollback.
6. Require `X-Internal-Token` on every internal route in every non-test runtime. Never use an empty token as a development bypass.
7. Never register or expose simulator behavior in production.
8. Store and compare business time in UTC. Redact DSNs, passwords, tokens, reservation keys, phone numbers, and device credentials.
9. Run focused tests, integration tests when persistence changes, then `gofmt`, `go vet ./...`, and `go test ./...`.

## Stop conditions

- Stop if Redis alone can allocate or release a cell.
- Stop if a retry can create a second reservation or command.
- Stop if an internal route can run with a missing or empty token.
- Stop if a production path can change simulator scenarios.
- Stop if logs, errors, fixtures, CI, or documentation expose connection values or replayable keys.

## Reference

Read `references/invariants.md` for transaction boundaries, stable errors, production restrictions, and required verification commands.
