# Smart Parcel Locker

智能快递柜系统，后端采用 Go Kratos。当前包含工程基线、健康检查 API、项目专属 Skill、Docker/CI，以及不依赖付费 MCP 的本地 Figma 设计生成插件。

## Requirements

- Go 1.23+
- Python 3.13+
- Node.js 20+
- Docker with Compose
- Figma Desktop（生成设计时需要）

## Run API locally

```powershell
go run ./app/api
Invoke-RestMethod http://127.0.0.1:8000/healthz
```

## Verify repository

```powershell
gofmt -w app internal
go vet ./...
go test ./...
python scripts/test_agents_rules.py
python scripts/test_validate_skills.py
python scripts/validate_skills.py
```

## Build Figma generator

```powershell
Set-Location tools/figma-plugin
npm install
npm test -- --run
npm run typecheck
npm run build
```

导入和运行方法见 `tools/figma-plugin/README.md`。Figma 视觉验收以 `docs/design/figma-qa.md` 为准，批准前不得开始页面实现。

## Run with Docker

```powershell
docker compose -f deploy/docker-compose.yml up -d --build
Invoke-RestMethod http://127.0.0.1:8000/healthz
docker compose -f deploy/docker-compose.yml down
```

## Project guidance

- Engineering rules: `AGENTS.md`
- Project Skills: `.agents/skills/`
- Approved system design: `docs/superpowers/specs/2026-07-16-smart-parcel-locker-design.md`
- Local Figma plugin design: `docs/superpowers/specs/2026-07-17-local-figma-generator-plugin-design.md`
- Local Figma plugin plan: `docs/superpowers/plans/2026-07-17-local-figma-generator-plugin.md`
