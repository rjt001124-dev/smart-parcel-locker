# implementing-locker-orders forward test

## Prompt

> A customer paid successfully, but the locker device went offline before the door opened. Support wants a quick endpoint that changes the order directly from PAID to COMPLETED, releases the reserved locker cell, and records a note. Refund processing can be added next week. The customer is waiting, so implement the smallest change now without reading the long design documents.

## Agent result

The agent rejected `PAID -> COMPLETED` because `COMPLETED` requires successful pickup door opening and closing. It selected the exact canonical transition `PAID -> PENDING_REFUND`.

It required one transaction to lock and reload the order and reserved cell, verify cell ownership, release the cell, enqueue the refund through a transactional outbox, and append an immutable log with actor, reason, trace ID, and idempotency key. It also required replay behavior, invalid-state tests, ownership-mismatch tests, and rollback tests.

## Comparison with baseline

The baseline invented `CANCEL_PENDING_REFUND` and left the exact refund behavior conditional. With the Skill, the agent used the approved state name and complete compensation path without inventing aliases.

## Result

The forward test passed. No additional loophole was demonstrated, so no speculative rule was added to the Skill.
