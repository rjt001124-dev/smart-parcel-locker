# Smart Parcel Locker

智能快递柜项目，后端采用 Go 1.23、Kratos、MySQL 8.4 和 Redis 7。MySQL 保存城市、网点、柜机、柜格、预约与设备命令；Redis 仅提供缓存和临时加速。

## 本地启动

```powershell
Copy-Item .env.example .env
docker compose -f deploy/docker-compose.yml up -d mysql redis
make migrate-up
docker compose -f deploy/docker-compose.yml up -d --build api worker
Invoke-RestMethod http://127.0.0.1:8000/readyz
```

Windows 没有 `make` 时，可在 Git Bash 中执行迁移命令，或直接使用项目 `Makefile` 中对应的 `go run ... migrate` 命令。

## 数据表

- `cities`：城市
- `sites`：服务网点，关联城市
- `locker_devices`：柜机设备，关联网点
- `locker_cells`：柜格及预约状态，关联柜机
- `device_commands`：开门、状态查询等设备命令，关联柜机

Navicat 连接、本地 Volume、Redis 清空恢复及生产隔离说明见 [本地 MySQL/Redis 运维](docs/operations/local-mysql-redis.md)。API 示例见 [站点与设备 API](docs/operations/site-device-api.md)。

## 验证

```powershell
gofmt -w app api internal tests
go mod verify
go vet ./...
go test ./...
python scripts/test_agents_rules.py
python scripts/test_validate_skills.py
python scripts/validate_skills.py
git diff --check
```

完整本地烟测：

```powershell
powershell -ExecutionPolicy Bypass -File scripts/smoke_site_device.ps1
```

## Frontend

小程序使用 Taro + React + TypeScript，后端继续使用 Go Kratos。详细启动步骤见 [小程序本地开发](docs/operations/miniapp-local-development.md)。

提交前必须运行：

```powershell
Set-Location frontend
npm run typecheck
npm test
npm run build:miniapp
```

## Figma

- 原型：[智能快递柜 Figma](https://www.figma.com/design/ui9lT54QlghpCFiYxiB6WT)
- 本地生成插件：`tools/figma-plugin`
- 设计基线：`docs/design/figma-baseline.md`
- 视觉验收：`docs/design/figma-qa.md`

## 项目规则

- 工程规则：`AGENTS.md`
- 项目 Skills：`.agents/skills/`
- 里程碑设计：`docs/superpowers/specs/2026-07-18-milestone-3-site-device-design.md`
- 实施计划：`docs/superpowers/plans/2026-07-18-milestone-3-site-device.md`

仓库不提交 `.env`、生产数据库凭据、Token、支付证书或客户数据。开发、CI 和烟测不得读取桌面上的线上凭据文件。
