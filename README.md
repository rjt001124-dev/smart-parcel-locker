# Smart Parcel Locker

智能快递柜系统，后端采用 Go Kratos。当前里程碑提供工程规则、健康检查 API、首个订单状态机 Skill、容器基线和 CI。

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
- Locker order Skill: `.agents/skills/implementing-locker-orders/`
- Approved design: `docs/superpowers/specs/2026-07-16-smart-parcel-locker-design.md`
- Milestone 1 plan: `docs/superpowers/plans/2026-07-16-milestone-1-engineering-baseline.md`
