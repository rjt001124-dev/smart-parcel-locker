# Locker order state-machine reference

## Canonical states

```text
PENDING_PAYMENT
CANCELED
PAID
PENDING_STORE
DOOR_OPEN_FAILED
STORING
PENDING_PICKUP
OVERDUE_PAYMENT_REQUIRED
PENDING_REFUND
REFUNDED
COMPLETED
```

Do not introduce aliases such as `CANCEL_PENDING_REFUND`, `SUCCESS`, `DONE`, or `FORCE_COMPLETED`.

## Allowed transitions

| From | To | Required side effect |
|---|---|---|
| `PENDING_PAYMENT` | `CANCELED` | release temporary cell reservation |
| `PENDING_PAYMENT` | `PAID` | append verified payment transaction |
| `PAID` | `PENDING_STORE` | confirm reserved cell ownership |
| `PAID` | `PENDING_REFUND` | release owned reservation and enqueue refund |
| `PENDING_STORE` | `STORING` | record successful open and confirmed close |
| `PENDING_STORE` | `DOOR_OPEN_FAILED` | record device command failure |
| `DOOR_OPEN_FAILED` | `PENDING_STORE` | choose retry or verified replacement cell |
| `DOOR_OPEN_FAILED` | `PENDING_REFUND` | release owned cell and enqueue refund |
| `STORING` | `PENDING_PICKUP` | create pickup intent |
| `STORING` | `OVERDUE_PAYMENT_REQUIRED` | freeze pickup until payment succeeds |
| `OVERDUE_PAYMENT_REQUIRED` | `PENDING_PICKUP` | append verified overdue payment |
| `PENDING_PICKUP` | `COMPLETED` | confirm pickup door close and release cell |
| `PENDING_REFUND` | `REFUNDED` | append verified refund transaction |

## Invariants

- `PAID` never transitions directly to `COMPLETED` or `CANCELED`.
- `COMPLETED` means the pickup door opened and later closed successfully.
- Payment success never proves storage success.
- A locker cell may be released only when the order still owns it.
- Replayed payment, refund, device, and support requests return the existing result.
- Status log, cell mutation, and order mutation commit atomically.
- Refund enqueueing uses a transactional outbox when the payment provider call is asynchronous.

## Verification

```powershell
gofmt -w app internal
go vet ./...
go test ./...
```
