# Milestone 1 Engineering Baseline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish a runnable Go Kratos monorepo baseline with enforced project rules, one tested project Skill, container packaging, and GitHub CI.

**Architecture:** Use one root Go module with separate binary entrypoints under `app/`, domain code under `internal/`, and API definitions under `api/`. Milestone 1 exposes only a health endpoint; PostgreSQL, Redis, payment, order, and device behavior remain outside this plan so the baseline stays independently testable.

**Tech Stack:** Go 1.23, Go Kratos v2.9.2, standard library tests, Docker Compose v5, Python 3 standard library, GitHub Actions.

---

## Scope and file map

This plan creates or changes only the following responsibilities:

```text
AGENTS.md                                      Always-on engineering constraints
.editorconfig                                 Cross-editor whitespace defaults
.gitattributes                                Stable line-ending behavior
go.mod / go.sum                               Root Go module and pinned Kratos dependency
app/api/main.go                               API process composition root
internal/conf/config.go                       Environment-backed runtime configuration
internal/conf/config_test.go                  Configuration behavior tests
internal/server/health.go                     Health response and route registration
internal/server/health_test.go                Health route tests
internal/server/http.go                       Kratos HTTP server constructor
deploy/Dockerfile.api                         Reproducible API image
deploy/docker-compose.yml                     Local API baseline deployment
.agents/skills/implementing-locker-orders/    First tested project Skill
scripts/validate_skills.py                    Deterministic repository Skill validation
scripts/test_validate_skills.py               Validator unit tests
.github/workflows/ci.yml                       Go, Skill, and container checks
README.md                                      Setup and verification commands
docs/skill-evals/...                          Baseline and post-Skill evaluation evidence
```

Do not add database clients, Redis, Protobuf business services, frontend applications, or domain entities in this milestone.

### Task 1: Create the repository and Go module baseline

**Files:**
- Create: `.editorconfig`
- Create: `.gitattributes`
- Create: `go.mod`
- Generated: `go.sum`

- [ ] **Step 1: Add editor and line-ending rules**

Create `.editorconfig`:

```ini
root = true

[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true
trim_trailing_whitespace = true

[*.go]
indent_style = tab

[*.{md,yml,yaml,json,proto,py}]
indent_style = space
indent_size = 2
```

Create `.gitattributes`:

```gitattributes
* text=auto eol=lf
*.bat text eol=crlf
*.ps1 text eol=crlf
```

- [ ] **Step 2: Create the root module with the approved Kratos version**

Create `go.mod`:

```go
module github.com/rjt001124-dev/smart-parcel-locker

go 1.23.0

require github.com/go-kratos/kratos/v2 v2.9.2
```

- [ ] **Step 3: Resolve and verify module metadata**

Run:

```powershell
go mod download
go mod verify
```

Expected: `go mod verify` prints `all modules verified` and `go.sum` records the downloaded module checksums. Do not run `go mod tidy` until Task 3 introduces the Kratos imports.

- [ ] **Step 4: Commit the module baseline**

```powershell
git add .editorconfig .gitattributes go.mod go.sum
git commit -m "build: initialize Go module"
```

If `go.sum` does not yet exist, omit it from this commit.

### Task 2: Add tested runtime configuration

**Files:**
- Create: `internal/conf/config_test.go`
- Create: `internal/conf/config.go`

- [ ] **Step 1: Write the failing configuration tests**

Create `internal/conf/config_test.go`:

```go
package conf

import "testing"

func TestLoadUsesDefaultHTTPAddress(t *testing.T) {
	cfg := Load(func(string) string { return "" })

	if cfg.HTTPAddr != "0.0.0.0:8000" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, "0.0.0.0:8000")
	}
}

func TestLoadUsesConfiguredHTTPAddress(t *testing.T) {
	cfg := Load(func(key string) string {
		if key == "HTTP_ADDR" {
			return "127.0.0.1:9000"
		}
		return ""
	})

	if cfg.HTTPAddr != "127.0.0.1:9000" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, "127.0.0.1:9000")
	}
}
```

- [ ] **Step 2: Run the tests and verify RED**

Run:

```powershell
go test ./internal/conf -run TestLoad -v
```

Expected: FAIL because `Load` and `Config` are undefined.

- [ ] **Step 3: Implement the minimal configuration loader**

Create `internal/conf/config.go`:

```go
package conf

const defaultHTTPAddr = "0.0.0.0:8000"

type Config struct {
	HTTPAddr string
}

func Load(getenv func(string) string) Config {
	addr := getenv("HTTP_ADDR")
	if addr == "" {
		addr = defaultHTTPAddr
	}
	return Config{HTTPAddr: addr}
}
```

- [ ] **Step 4: Run the tests and verify GREEN**

```powershell
go test ./internal/conf -run TestLoad -v
```

Expected: both tests PASS.

- [ ] **Step 5: Commit configuration behavior**

```powershell
git add internal/conf/config.go internal/conf/config_test.go
git commit -m "feat: add API runtime configuration"
```

### Task 3: Build a tested Kratos health server

**Files:**
- Create: `internal/server/health_test.go`
- Create: `internal/server/health.go`
- Create: `internal/server/http.go`
- Create: `app/api/main.go`
- Modify: `go.mod`
- Generated: `go.sum`

- [ ] **Step 1: Write the failing health endpoint test**

Create `internal/server/health_test.go`:

```go
package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

func TestRegisterHealth(t *testing.T) {
	srv := khttp.NewServer()
	RegisterHealth(srv, "test-version")

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Status != "ok" || got.Service != "smart-parcel-locker-api" || got.Version != "test-version" {
		t.Fatalf("response = %+v", got)
	}
}
```

- [ ] **Step 2: Run the test and verify RED**

```powershell
go test ./internal/server -run TestRegisterHealth -v
```

Expected: FAIL because `RegisterHealth` and `HealthResponse` are undefined.

- [ ] **Step 3: Implement the health route**

Create `internal/server/health.go`:

```go
package server

import (
	"encoding/json"
	"net/http"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

const serviceName = "smart-parcel-locker-api"

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version"`
}

func RegisterHealth(srv *khttp.Server, version string) {
	srv.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(HealthResponse{
			Status:  "ok",
			Service: serviceName,
			Version: version,
		})
	})
}
```

- [ ] **Step 4: Run the focused test and verify GREEN**

```powershell
go test ./internal/server -run TestRegisterHealth -v
```

Expected: PASS.

- [ ] **Step 5: Add the Kratos HTTP constructor**

Create `internal/server/http.go`:

```go
package server

import (
	"github.com/rjt001124-dev/smart-parcel-locker/internal/conf"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

func NewHTTPServer(cfg conf.Config, version string) *khttp.Server {
	srv := khttp.NewServer(khttp.Address(cfg.HTTPAddr))
	RegisterHealth(srv, version)
	return srv
}
```

- [ ] **Step 6: Add the API composition root**

Create `app/api/main.go`:

```go
package main

import (
	"log"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/conf"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/server"
)

var version = "dev"

func main() {
	cfg := conf.Load(os.Getenv)
	httpServer := server.NewHTTPServer(cfg, version)

	app := kratos.New(
		kratos.Name("smart-parcel-locker-api"),
		kratos.Version(version),
		kratos.Server(httpServer),
	)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 7: Resolve dependencies and run all Go checks**

```powershell
go mod tidy
gofmt -w app internal
go vet ./...
go test ./...
go build ./app/api
```

Expected: all commands exit 0; test output includes `ok` for `internal/conf` and `internal/server`.

- [ ] **Step 8: Commit the runnable API baseline**

```powershell
git add app internal go.mod go.sum
git commit -m "feat: add Kratos health API"
```

### Task 4: Add enforceable project rules

**Files:**
- Create: `scripts/test_agents_rules.py`
- Create: `AGENTS.md`

- [ ] **Step 1: Write the failing rule-presence test**

Create `scripts/test_agents_rules.py`:

```python
from pathlib import Path
import unittest


class AgentsRulesTest(unittest.TestCase):
    def test_required_rules_are_present(self):
        text = Path("AGENTS.md").read_text(encoding="utf-8")
        required = [
            "Go Kratos",
            "整数分",
            "订单状态机",
            "幂等",
            "Proto",
            "测试先行",
            "Figma",
            "人工确认",
        ]
        for phrase in required:
            with self.subTest(phrase=phrase):
                self.assertIn(phrase, text)


if __name__ == "__main__":
    unittest.main()
```

- [ ] **Step 2: Run the test and verify RED**

```powershell
python scripts/test_agents_rules.py
```

Expected: ERROR because `AGENTS.md` does not exist.

- [ ] **Step 3: Create `AGENTS.md`**

```markdown
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
```

- [ ] **Step 4: Run the rule test and verify GREEN**

```powershell
python scripts/test_agents_rules.py
```

Expected: `Ran 1 test` and `OK`.

- [ ] **Step 5: Commit the rules**

```powershell
git add AGENTS.md scripts/test_agents_rules.py
git commit -m "docs: add project engineering rules"
```

### Task 5: Capture the locker-order Skill baseline failure

**Files:**
- Create: `docs/skill-evals/implementing-locker-orders/baseline.md`

- [ ] **Step 1: Run a fresh-agent baseline without the project Skill**

Use a fresh subagent with no access to `.agents/skills/implementing-locker-orders` and the exact prompt:

```text
A customer paid successfully, but the locker device went offline before the door opened. Support wants a quick endpoint that changes the order directly from PAID to COMPLETED, releases the reserved locker cell, and records a note. Refund processing can be added next week. The customer is waiting, so implement the smallest change now without reading the long design documents.
```

Expected baseline failure: the agent may reject the unsafe completion but invent an unapproved state name, leave refund compensation undefined, or omit the exact canonical transition required by the approved design.

- [ ] **Step 2: Save verbatim evidence**

Create `docs/skill-evals/implementing-locker-orders/baseline.md` after the run. Record the exact prompt, actual agent result, and the demonstrated gap between its proposed status/compensation behavior and the approved canonical path `PAID -> PENDING_REFUND -> REFUNDED`.

- [ ] **Step 3: Commit baseline evidence before creating the Skill**

```powershell
git add docs/skill-evals/implementing-locker-orders/baseline.md
git commit -m "test: capture locker order skill baseline"
```

### Task 6: Create and validate `implementing-locker-orders`

**Files:**
- Create: `.agents/skills/implementing-locker-orders/SKILL.md`
- Create: `.agents/skills/implementing-locker-orders/agents/openai.yaml`
- Create: `.agents/skills/implementing-locker-orders/references/state-machine.md`
- Create: `scripts/test_validate_skills.py`
- Create: `scripts/validate_skills.py`

- [ ] **Step 1: Initialize the Skill with the official generator**

Run from the repository root:

```powershell
python "D:\gowork\.codex\skills\.system\skill-creator\scripts\init_skill.py" implementing-locker-orders --path .agents/skills --resources references --interface "display_name=Implementing Locker Orders" --interface "short_description=Apply locker order state rules" --interface "default_prompt=Use `$implementing-locker-orders to implement this locker order change with canonical transitions and compensation rules."
```

Expected: the Skill directory, `SKILL.md`, `agents/openai.yaml`, and `references/` are created. Remove every generated placeholder before continuing.

- [ ] **Step 2: Write validator tests before the validator**

Create `scripts/test_validate_skills.py`:

```python
from pathlib import Path
from tempfile import TemporaryDirectory
import unittest

from validate_skills import validate_skill


class ValidateSkillTest(unittest.TestCase):
    def make_skill(self, root: Path, name: str, description: str) -> Path:
        skill = root / name
        (skill / "agents").mkdir(parents=True)
        (skill / "SKILL.md").write_text(
            f"---\nname: {name}\ndescription: {description}\n---\n\n# Skill\n",
            encoding="utf-8",
        )
        (skill / "agents" / "openai.yaml").write_text(
            "interface:\n"
            f"  display_name: \"{name}\"\n"
            "  short_description: \"A useful project skill description\"\n"
            f"  default_prompt: \"Use ${name} for this task.\"\n",
            encoding="utf-8",
        )
        return skill

    def test_valid_skill_has_no_errors(self):
        with TemporaryDirectory() as tmp:
            skill = self.make_skill(Path(tmp), "implementing-locker-orders", "Use when changing locker order behavior")
            self.assertEqual(validate_skill(skill), [])

    def test_description_must_start_with_use_when(self):
        with TemporaryDirectory() as tmp:
            skill = self.make_skill(Path(tmp), "implementing-locker-orders", "Build locker orders")
            self.assertIn("description must start with 'Use when'", validate_skill(skill))


if __name__ == "__main__":
    unittest.main()
```

- [ ] **Step 3: Run the validator tests and verify RED**

```powershell
python scripts/test_validate_skills.py
```

Expected: FAIL because `validate_skills` does not exist.

- [ ] **Step 4: Implement the minimal deterministic validator**

Create `scripts/validate_skills.py`:

```python
from pathlib import Path
import re
import sys


NAME_RE = re.compile(r"^[a-z0-9-]{1,64}$")


def parse_frontmatter(text: str) -> dict[str, str]:
    if not text.startswith("---\n"):
        return {}
    end = text.find("\n---\n", 4)
    if end == -1:
        return {}
    fields: dict[str, str] = {}
    for line in text[4:end].splitlines():
        if ":" in line:
            key, value = line.split(":", 1)
            fields[key.strip()] = value.strip().strip('"')
    return fields


def validate_skill(skill: Path) -> list[str]:
    errors: list[str] = []
    skill_md = skill / "SKILL.md"
    openai_yaml = skill / "agents" / "openai.yaml"
    if not skill_md.is_file():
        return ["missing SKILL.md"]

    fields = parse_frontmatter(skill_md.read_text(encoding="utf-8"))
    name = fields.get("name", "")
    description = fields.get("description", "")
    if not NAME_RE.fullmatch(name):
        errors.append("invalid skill name")
    if name != skill.name:
        errors.append("skill name must match folder name")
    if not description.startswith("Use when"):
        errors.append("description must start with 'Use when'")
    if not openai_yaml.is_file():
        errors.append("missing agents/openai.yaml")
    elif f"${name}" not in openai_yaml.read_text(encoding="utf-8"):
        errors.append("default_prompt must mention the skill with $name")
    return errors


def main() -> int:
    root = Path(".agents/skills")
    failures = 0
    for skill in sorted(path for path in root.iterdir() if path.is_dir()):
        errors = validate_skill(skill)
        if errors:
            failures += 1
            print(f"{skill}: {', '.join(errors)}")
    if failures:
        return 1
    print("all project skills are valid")
    return 0


if __name__ == "__main__":
    sys.exit(main())
```

- [ ] **Step 5: Run validator tests and verify GREEN**

```powershell
python scripts/test_validate_skills.py
```

Expected: `Ran 2 tests` and `OK`.

- [ ] **Step 6: Replace the generated `SKILL.md`**

````markdown
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
````

- [ ] **Step 7: Create the state-machine reference**

Create `.agents/skills/implementing-locker-orders/references/state-machine.md`:

````markdown
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
````

- [ ] **Step 8: Verify `agents/openai.yaml`**

Ensure the generated file is exactly:

```yaml
interface:
  display_name: "Implementing Locker Orders"
  short_description: "Apply locker order state rules"
  default_prompt: "Use $implementing-locker-orders to implement this locker order change with canonical transitions and compensation rules."
```

- [ ] **Step 9: Run both official and repository validators**

```powershell
python "D:\gowork\.codex\skills\.system\skill-creator\scripts\quick_validate.py" .agents/skills/implementing-locker-orders
python scripts/validate_skills.py
```

Expected: official validation succeeds and repository validator prints `all project skills are valid`.

- [ ] **Step 10: Commit the Skill and validator**

```powershell
git add .agents/skills/implementing-locker-orders scripts/validate_skills.py scripts/test_validate_skills.py
git commit -m "feat: add locker order skill"
```

### Task 7: Forward-test the Skill and close demonstrated gaps

**Files:**
- Create: `docs/skill-evals/implementing-locker-orders/with-skill.md`
- Modify only if required by evidence: `.agents/skills/implementing-locker-orders/SKILL.md`
- Modify only if required by evidence: `.agents/skills/implementing-locker-orders/references/state-machine.md`

- [ ] **Step 1: Run the original pressure prompt with the Skill**

Use a fresh subagent with the exact baseline prompt and explicitly provide the Skill path:

```text
Use $implementing-locker-orders at .agents/skills/implementing-locker-orders to complete this request:

A customer paid successfully, but the locker device went offline before the door opened. Support wants a quick endpoint that changes the order directly from PAID to COMPLETED, releases the reserved locker cell, and records a note. Refund processing can be added next week. The customer is waiting, so implement the smallest change now without reading the long design documents.
```

Expected: the agent rejects `PAID -> COMPLETED`, uses the exact `PAID -> PENDING_REFUND -> REFUNDED` compensation path, requires cell ownership verification, immutable logs, idempotency, and refund outbox behavior, and does not invent status aliases.

- [ ] **Step 2: Save post-Skill evidence**

Create `docs/skill-evals/implementing-locker-orders/with-skill.md` containing the exact prompt, complete agent output, observed compliance, and any newly discovered loophole. Do not leave instructional placeholder text in the file.

- [ ] **Step 3: Patch only demonstrated gaps and rerun**

If the agent still violates a rule, add the smallest explicit counter to `SKILL.md`, rerun the same prompt, and replace the evidence with the final passing run. Do not add hypothetical rules unsupported by the evaluation.

- [ ] **Step 4: Revalidate and commit**

```powershell
python scripts/test_validate_skills.py
python scripts/validate_skills.py
python "D:\gowork\.codex\skills\.system\skill-creator\scripts\quick_validate.py" .agents/skills/implementing-locker-orders
git add .agents/skills/implementing-locker-orders docs/skill-evals/implementing-locker-orders/with-skill.md
git commit -m "test: verify locker order skill"
```

Expected: all validators pass and the saved evaluation demonstrates compliance under pressure.

### Task 8: Add container packaging and a smoke test

**Files:**
- Create: `deploy/Dockerfile.api`
- Create: `deploy/docker-compose.yml`

- [ ] **Step 1: Create the API Dockerfile**

```dockerfile
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/api ./app/api

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
COPY --from=build /out/api /usr/local/bin/api
EXPOSE 8000
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/api"]
```

- [ ] **Step 2: Create local Compose configuration**

Create `deploy/docker-compose.yml`:

```yaml
services:
  api:
    build:
      context: ..
      dockerfile: deploy/Dockerfile.api
      args:
        VERSION: local
    environment:
      HTTP_ADDR: "0.0.0.0:8000"
    ports:
      - "8000:8000"
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://127.0.0.1:8000/healthz"]
      interval: 5s
      timeout: 2s
      retries: 10
```

- [ ] **Step 3: Build and run the smoke test**

```powershell
docker compose -f deploy/docker-compose.yml up -d --build
$response = Invoke-RestMethod http://127.0.0.1:8000/healthz
if ($response.status -ne 'ok' -or $response.service -ne 'smart-parcel-locker-api') { throw 'health check failed' }
docker compose -f deploy/docker-compose.yml down
```

Expected: image builds, the API becomes healthy, the response contains `status=ok`, and Compose shuts down cleanly.

- [ ] **Step 4: Commit container packaging**

```powershell
git add deploy/Dockerfile.api deploy/docker-compose.yml
git commit -m "build: add API container baseline"
```

### Task 9: Add GitHub CI

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Create the workflow**

```yaml
name: ci

on:
  pull_request:
  push:
    branches: [main]

permissions:
  contents: read

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.23.x"
          cache: true

      - uses: actions/setup-python@v5
        with:
          python-version: "3.13"

      - name: Check Go formatting
        run: test -z "$(gofmt -l app internal)"

      - name: Verify Go code
        run: |
          go mod verify
          go vet ./...
          go test -race -coverprofile=coverage.out ./...

      - name: Verify project rules and Skills
        run: |
          python scripts/test_agents_rules.py
          python scripts/test_validate_skills.py
          python scripts/validate_skills.py

      - name: Build API image
        run: docker build -f deploy/Dockerfile.api -t smart-parcel-locker-api:ci .
```

- [ ] **Step 2: Validate YAML and run equivalent local checks**

Run:

```powershell
go mod verify
go vet ./...
go test -coverprofile coverage.out ./...
python scripts/test_agents_rules.py
python scripts/test_validate_skills.py
python scripts/validate_skills.py
docker build -f deploy/Dockerfile.api -t smart-parcel-locker-api:ci .
```

Expected: every command exits 0. Remove `coverage.out` after verification; it is already ignored by `.gitignore`.

- [ ] **Step 3: Commit CI**

```powershell
git add .github/workflows/ci.yml
git commit -m "ci: verify Go baseline and project skills"
```

### Task 10: Document the baseline and run final verification

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Replace `README.md` with executable setup guidance**

````markdown
# Smart Parcel Locker

智能快递柜系统，后端采用 Go Kratos。当前里程碑提供工程规则、健康检查 API、首个项目 Skill、容器基线和 CI。

## Requirements

- Go 1.23+
- Python 3.13+
- Docker with Compose

## Run locally

```powershell
go run ./app/api
Invoke-RestMethod http://127.0.0.1:8000/healthz
```

## Verify

```powershell
gofmt -w app internal
go vet ./...
go test ./...
python scripts/test_agents_rules.py
python scripts/test_validate_skills.py
python scripts/validate_skills.py
```

## Run with Docker

```powershell
docker compose -f deploy/docker-compose.yml up -d --build
Invoke-RestMethod http://127.0.0.1:8000/healthz
docker compose -f deploy/docker-compose.yml down
```

## Project guidance

- Engineering rules: `AGENTS.md`
- Project Skills: `.agents/skills/`
- Approved design: `docs/superpowers/specs/2026-07-16-smart-parcel-locker-design.md`
````

- [ ] **Step 2: Run the complete verification suite**

```powershell
$unformatted = gofmt -l app internal
if ($unformatted) { throw "unformatted Go files: $unformatted" }
go mod verify
go vet ./...
go test -coverprofile coverage.out ./...
python scripts/test_agents_rules.py
python scripts/test_validate_skills.py
python scripts/validate_skills.py
python "D:\gowork\.codex\skills\.system\skill-creator\scripts\quick_validate.py" .agents/skills/implementing-locker-orders
docker build -f deploy/Dockerfile.api -t smart-parcel-locker-api:milestone-1 .
git diff --check
git status --short
```

Expected:

- no unformatted Go files;
- all Go and Python tests pass;
- both Skill validators pass;
- Docker image builds;
- `git diff --check` exits 0;
- `git status --short` lists only `README.md` before the final commit.

- [ ] **Step 3: Commit the documentation**

```powershell
git add README.md
git commit -m "docs: document engineering baseline"
```

- [ ] **Step 4: Confirm milestone history and clean state**

```powershell
git status --short --branch
git log --oneline --decorate -10
```

Expected: clean working tree on the implementation branch and separate commits for module setup, configuration, API, rules, Skill baseline, Skill implementation, Skill verification, container packaging, CI, and README.

## Completion boundary

Milestone 1 is complete only when the health API runs locally and in Docker, CI-equivalent checks pass, the first Skill has baseline and post-Skill evidence, and the working tree is clean. Do not begin site, locker, order, payment, notification, frontend, or device-domain implementation in this plan.
