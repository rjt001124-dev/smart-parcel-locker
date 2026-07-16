# implementing-locker-orders baseline

## Prompt

> A customer paid successfully, but the locker device went offline before the door opened. Support wants a quick endpoint that changes the order directly from PAID to COMPLETED, releases the reserved locker cell, and records a note. Refund processing can be added next week. The customer is waiting, so implement the smallest change now without reading the long design documents.

## Agent result

The agent correctly rejected `PAID -> COMPLETED`, direct mutation, missing audit data, and missing idempotency. It proposed a support command, transaction, cell ownership check, immutable log, and tests.

## Demonstrated gap

The agent invented `CANCEL_PENDING_REFUND` as a possible status instead of using a canonical project state. It also left the exact refund transition and worker obligation conditional on a later decision, even though the approved design already defines the compensation path:

```text
PAID -> PENDING_REFUND -> REFUNDED
```

The baseline therefore understood general safety but could not retrieve and apply the project's exact status vocabulary, permitted transitions, or required compensation behavior. The Skill must provide those domain-specific invariants without duplicating general Kratos rules from `AGENTS.md`.
