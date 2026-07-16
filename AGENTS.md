# Smart Parcel Locker Engineering Rules

## Scope

These rules apply to the entire repository.

## Architecture

- Use Go Kratos and preserve the `service -> biz -> data` dependency direction.
- Keep transport handlers limited to authentication, validation, use-case calls, and response mapping.
- Do not access databases, Redis, Alipay, or locker SDKs directly from transport handlers.
- Keep each domain independently testable through interfaces.

## Business integrity

- Represent money as integer cents（整数分）; never use floating-point money.
- Change order status only through the approved 订单状态机 and append a status log for every transition.
- Require 幂等 keys for payment callbacks, locker commands, retries, and externally repeated requests.
- Never edit or delete posted account transactions; corrections use compensating entries.

## API compatibility

- Define public contracts in Proto and preserve field-number compatibility.
- Never reuse removed Proto field numbers; reserve them.
- Return stable business error codes and a request trace ID. Never expose internal stack traces or secrets.

## Quality

- Use 测试先行 for features and bug fixes: write a failing test, verify the failure, implement minimally, then refactor.
- Run `gofmt`, `go vet ./...`, and `go test ./...` locally before claiming Go work is complete; CI must additionally pass `go test -race ./...` on Linux.
- Keep generated code separate from handwritten code and do not manually edit generated files.

## Security and operations

- Never commit credentials, private keys, payment certificates, access tokens, or production customer data.
- Redact phone numbers, tokens, device credentials, and payment data from logs.
- Require 人工确认 and an audit record before restart, isolation, rollback, remote opening, or other high-risk device actions.

## Frontend and design

- Treat approved Figma designs, variables, components, and states as the implementation source of truth.
- Do not change layout, color, spacing, text hierarchy, or interaction behavior without updating and approving Figma first.
