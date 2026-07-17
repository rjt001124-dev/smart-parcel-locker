---
name: implementing-locker-orders
description: Use when adding or changing smart locker order creation, payment transitions, locker-cell reservation, storage, door-open failure, overdue payment, pickup, cancellation, refund, or completion behavior
---

# Implementing Locker Orders

## Core rule

Use only canonical order states and transitions. Payment success is not storage success, and every state change must preserve order, money, and locker-cell consistency.

## Workflow

1. Read `references/state-machine.md` before proposing or changing a status.
2. Write a failing test for the requested transition and its invalid-source-state case.
3. Lock and reload the order and owned locker cell inside one transaction.
4. Apply one canonical transition through the order use case; never update status directly in a handler or repository helper.
5. Append an immutable status log with actor, reason, trace ID, and idempotency key.
6. Execute the required compensation action in the same transaction or enqueue it through a transactional outbox.
7. Run focused tests, then `gofmt`, `go vet ./...`, and `go test ./...`.

## Stop conditions

- Stop if a requested state name is not in the canonical list.
- Stop if a transition is not explicitly allowed.
- Stop if payment succeeded but a failure path does not end in retry, cell-safe recovery, or `PENDING_REFUND`.
- Stop if a cell is released without verifying that the order still owns it.
- Stop if a transition lacks an immutable status log or idempotency key.
- Stop if `COMPLETED` would be used when storage or pickup never completed.

## References

Read `references/state-machine.md` for canonical states, transitions, ownership rules, and compensation behavior.
