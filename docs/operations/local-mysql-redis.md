# 本地 MySQL、Redis 与 Navicat

## 启动开发依赖

```powershell
Copy-Item .env.example .env
docker compose -f deploy/docker-compose.yml up -d mysql redis
make migrate-up
docker compose -f deploy/docker-compose.yml up -d --build api worker
```

`.env` 仅用于本机开发，不得提交。项目不会读取或使用桌面上的线上 MySQL/Redis 凭据文件。

## Navicat 连接

在 Navicat 新建 MySQL 连接，使用本地 `.env` 中的值：

| 字段 | 本地值 |
|---|---|
| 主机 | `127.0.0.1` |
| 端口 | `3306` |
| 数据库 | `smart_parcel_locker` |
| 用户名 | `.env` 的 `MYSQL_USER` |
| 密码 | `.env` 的 `MYSQL_PASSWORD` |

连接成功后可查看 `cities`、`sites`、`locker_devices`、`locker_cells`、`device_commands`。Navicat 是管理客户端，数据实际保存在 Docker 的 MySQL named volume 中。

## Volume 与重启

- `docker compose ... down`：停止容器，保留 `mysql_data` 和 `redis_data`。
- `docker compose ... up -d`：重启后 MySQL 数据仍存在。
- `docker compose ... down -v`：删除 named volume 和全部本地开发数据，属于破坏性操作，执行前必须人工确认。

Redis 可以清空并重建，因为它不是事实来源。Redis 丢失后，网点查询从 MySQL 回源；预约正确性仍由 MySQL 条件更新保证。

## Migration

```powershell
make migrate-up
$env:CONFIRM_LOCAL_RESET = "reset-local"
make migrate-cycle
```

迁移保护只允许 `127.0.0.1`/`localhost` 和 `development`/`test` 环境。应用启动不会自动修改数据库结构。

## 生产隔离清单

未来部署生产环境前必须完成：

- 数据库备份、Migration 预检和人工批准；
- MySQL 使用非 root、最小权限账号；
- Redis 使用 ACL、独立账号和私有网络；
- 使用 TLS 或 SSH 隧道，并配置 IP 白名单；
- 凭据放入服务器环境变量或密钥管理服务；
- 上线后轮换密码，删除桌面明文凭据；
- 生产禁用模拟器路由；
- Navicat 生产连接优先使用只读或受限账号。
