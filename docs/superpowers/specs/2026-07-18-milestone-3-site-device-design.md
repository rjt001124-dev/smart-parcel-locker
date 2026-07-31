# 智能快递柜里程碑 3：站点与设备领域设计

## 1. 目标

本里程碑在已有 Go Kratos 工程基线和已批准 Figma 设计基线上，建立城市、寄存网点、柜机、柜格和模拟设备网关的可运行后端基础。交付物必须能使用 MySQL 8.4 持久化业务数据，使用 Redis 7 提供可丢失的临时能力，并为下一里程碑的寄存订单、计费和开门流程提供稳定接口。

本里程碑不连接用户提供的线上生产 MySQL/Redis，不对生产数据执行迁移、写入或自动化测试。开发和 CI 只使用 Docker 中的隔离数据库。

## 2. 范围

### 2.1 本里程碑包含

- 城市、网点、柜机、柜格和设备指令的领域模型。
- 遵循 Kratos `service -> biz -> data` 依赖方向的模块化单体实现。
- MySQL 8.4 表结构、索引、约束、版本化 Migration 和演示种子数据。
- Redis 7 缓存、临时锁标记和内部接口限流边界。
- 公开查询 API、内部设备 API 和稳定业务错误码。
- 模拟设备网关，覆盖在线、离线、超时、失败和柜门未关五种场景。
- 柜格原子预留、释放、到期回收和幂等重试。
- Docker Compose 开发环境、就绪检查和 Navicat 连接说明。
- 单元测试、MySQL/Redis 集成测试、并发测试和 CI 验证。

### 2.2 明确不包含

- 微信小程序页面和 React 管理后台实现。
- 用户登录、RBAC 和完整管理端 CRUD。
- 订单状态机、计费、优惠券、支付宝和通知业务。
- 真实柜机厂商协议、远程重启、隔离和回滚。
- 生产数据库迁移、生产凭据验证和正式上线。

## 3. 技术决策

### 3.1 架构方式

采用模块化单体，不在当前阶段拆分微服务。各领域通过小型接口协作，运输层不直接访问 MySQL、Redis 或设备适配器。

```text
HTTP/Proto service
       |
       v
site / locker / device biz
       |
       v
repository and gateway interfaces
       |
       +---- MySQL repositories
       +---- Redis helpers
       +---- simulated device gateway
```

### 3.2 包边界

- `internal/site/service`：城市、网点请求校验和响应映射。
- `internal/site/biz`：城市启用规则、网点服务状态、附近网点查询用例和仓储接口。
- `internal/site/data`：MySQL 网点仓储和短时查询缓存。
- `internal/locker/service`：柜机、柜格和预留接口适配。
- `internal/locker/biz`：柜格状态、分配条件、预留、释放、过期回收和幂等规则。
- `internal/locker/data`：MySQL 柜机/柜格仓储和 Redis 临时加速标记。
- `internal/device/service`：心跳、指令发送和结果查询接口适配。
- `internal/device/biz`：设备在线判定、指令幂等、有效期和重试上限。
- `internal/device/data`：MySQL 指令仓储和模拟设备网关。
- `internal/platform/data`：MySQL、Redis 客户端创建、健康检查和共享事务工具。

## 4. 数据存储

### 4.1 MySQL 是业务事实来源

MySQL 8.4 保存所有需要恢复和审计的数据。应用正确性不得依赖 Redis 中的柜格锁或缓存仍然存在。即使 Redis 数据全部丢失，MySQL 中的柜格状态、预留关系和设备指令仍必须保持正确。

### 4.2 Redis 只保存临时数据

Redis 7 用于附近网点短时缓存、柜格预留快速标记、内部接口限流和将来的会话扩展。缓存未命中或 Redis 不可用时，查询和柜格预留可退化到 MySQL；降级期间仍不允许重复分配。

### 4.3 开发与生产隔离

- 开发使用 Docker Compose 命名 Volume，容器重启不删除数据。
- `docker compose down -v` 会删除本地开发数据，文档必须显式警告。
- 用户提供的线上凭据不复制到仓库、开发 `.env`、测试、日志或 CI。
- 生产部署使用服务器环境变量或密钥管理服务，且上线迁移必须经过备份和人工确认。
- Navicat 只是管理客户端，不是数据存储层。生产 Navicat 连接应使用只读或受限账号、IP 白名单以及 SSL/SSH 隧道。

## 5. 数据模型

所有业务时间以 UTC 保存。所有主表包含 `created_at` 和 `updated_at`。对外识别号使用不可推测的字符串编号，内部联表使用 `BIGINT UNSIGNED` 主键。

### 5.1 `cities`

- `id`、`code`、`name`、`province`、`enabled`、时间戳。
- `code` 全局唯一，禁止删除已被网点引用的城市。

### 5.2 `sites`

- `id`、`site_no`、`city_id`、`name`、`address`、`latitude`、`longitude`、`open_time`、`close_time`、`contact_phone`、`service_status`、时间戳。
- `site_no` 全局唯一。
- 经纬度分别使用 `DECIMAL(10,7)` 和 `DECIMAL(10,7)`，应用校验范围为 `[-90, 90]` 与 `[-180, 180]`。
- 为 `city_id + service_status` 和经纬度候选筛选建立索引。

### 5.3 `locker_devices`

- `id`、`device_no`、`site_id`、`protocol_type`、`network_status`、`operational_status`、`last_heartbeat_at`、`firmware_version`、时间戳。
- `device_no` 全局唯一。
- 当前时间超过 `last_heartbeat_at + offline_threshold` 时，业务层将设备视为离线，不仅依赖存储的 `network_status`。

### 5.4 `locker_cells`

- `id`、`device_id`、`cell_no`、`size`、`occupancy_status`、`door_status`、`reservation_key`、`lock_expires_at`、`current_order_id`、`version`、时间戳。
- `(device_id, cell_no)` 唯一。
- `reservation_key` 可空且非空时唯一，用于幂等返回原预留结果。
- 本里程碑的 `current_order_id` 保留为空，下一里程碑创建订单表后再增加外键。
- 分配查询按网点、设备在线状态、柜格尺寸、占用状态和锁定到期时间建立复合索引。

### 5.5 `device_commands`

- `id`、`command_no`、`device_id`、`action`、`payload_json`、`idempotency_key`、`status`、`expires_at`、`attempt_count`、`result_json`、`error_code`、时间戳。
- `command_no` 和 `idempotency_key` 分别全局唯一。
- 动作限定为明确枚举；本里程碑实现模拟 `OPEN_DOOR` 和 `QUERY_STATUS`。
- 指令有效期过后不得再执行，重试不得绕过幂等键。

## 6. 核心 API

公共合约定义在 Proto 中，并生成 Kratos HTTP 路由。公开接口和内部接口使用不同路由组。

### 6.1 公开查询

- `GET /v1/cities`：返回已启用城市。
- `GET /v1/sites`：使用 `city_code`、`latitude`、`longitude`、`radius_m` 查询附近服务中网点，按距离升序。
- `GET /v1/sites/{site_id}`：返回网点详情、柜机在线概要和各尺寸可用柜格数。
- `GET /v1/sites/{site_id}/cells`：按 `size` 和 `status` 查询柜格，不向公开响应暴露设备凭据或内部锁标识。

`radius_m` 默认为 5000，最大为 50000。附近查询使用 MySQL 数学距离公式，当前不引入空间扩展。

### 6.2 内部接口

- `POST /v1/internal/devices/{device_no}/heartbeat`：更新心跳和设备状态。
- `POST /v1/internal/cells/reserve`：使用网点、柜格尺寸、预留时长和幂等键原子锁定柜格。
- `POST /v1/internal/cells/{cell_id}/release`：只在幂等键与当前预留匹配时释放。
- `POST /v1/internal/device-commands`：创建并发送模拟设备指令。
- `GET /v1/internal/device-commands/{command_no}`：查询指令状态与可披露结果。
- `PUT /v1/internal/simulator/devices/{device_no}/scenario`：仅开发/测试环境可用，切换模拟场景。

所有 `/v1/internal` 路由要求 `X-Internal-Token`。模拟场景路由在生产模式下不注册，不仅是返回禁止访问。

## 7. 柜格预留一致性

1. 先在 MySQL 查询 `reservation_key`；如已存在，返回原柜格和原到期时间。
2. 开启 MySQL 事务，选择在线、可运营柜机下尺寸匹配的空闲柜格。
3. 使用 `SELECT ... FOR UPDATE SKIP LOCKED` 隔离并发候选行。
4. 仅当柜格仍为空闲或原锁已过期时，写入 `LOCKED`、`reservation_key` 和 `lock_expires_at`。
5. 提交 MySQL 事务后再写 Redis 加速标记；Redis 写入失败不回滚已成功的 MySQL 预留。
6. 回收任务只释放未绑定订单且已过期的预留，每次更新必须带当前 `reservation_key` 条件。

MySQL 是并发正确性的最终保障；Redis 锁不得替代数据库条件更新。

## 8. 模拟设备网关

业务层依赖 `DeviceGateway` 接口，不依赖模拟适配器的具体实现。模拟适配器保存每台设备的测试场景：

- `ONLINE`：指令在配置延迟后成功。
- `OFFLINE`：发送前返回 `DEVICE_OFFLINE`。
- `TIMEOUT`：超过指令截止时间并返回可重试超时。
- `FAIL`：设备回复明确失败代码。
- `DOOR_LEFT_OPEN`：开门成功，但柜门在监测窗口内未关闭。

模拟场景配置不进入生产业务数据。每条设备指令的业务状态仍写入 MySQL，便于下一里程碑关联订单和审计。

## 9. 错误和安全

稳定错误码至少包含：

- `CITY_NOT_FOUND`
- `SITE_NOT_FOUND`
- `DEVICE_NOT_FOUND`
- `DEVICE_OFFLINE`
- `CELL_NOT_AVAILABLE`
- `CELL_RESERVATION_CONFLICT`
- `INVALID_COORDINATES`
- `DEVICE_COMMAND_TIMEOUT`
- `DEVICE_COMMAND_FAILED`
- `IDEMPOTENCY_CONFLICT`

统一错误响应包含稳定业务码、用户可读消息、`retryable` 和 `trace_id`。不得返回 SQL、DSN、Redis 密码、内部 Token、堆栈或设备凭据。日志中对联系电话、Token 和连接信息脱敏。

生产 MySQL 应用账号不使用 `root`，只授予目标库所需权限。生产 Redis 使用 ACL/独立账号、私网访问和 TLS（若托管服务支持）。当前桌面明文凭据文件在安全迁移到生产密钥管理后应更换密码并删除。

## 10. 配置与运行

仓库只提交占位的 `.env.example`，不提交 `.env`。应用使用独立环境变量，避免将密码嵌入一个可能被记录的 DSN：

- `MYSQL_HOST`、`MYSQL_PORT`、`MYSQL_DATABASE`、`MYSQL_USER`、`MYSQL_PASSWORD`
- `REDIS_HOST`、`REDIS_PORT`、`REDIS_USERNAME`、`REDIS_PASSWORD`、`REDIS_DATABASE`
- `INTERNAL_API_TOKEN`、`APP_ENV`、`DEVICE_OFFLINE_THRESHOLD`

Docker Compose 新增 `mysql` 和 `redis` 服务、命名 Volume 和健康检查。API 的 `/healthz` 只表示进程存活，`/readyz` 检查 MySQL 和 Redis；Redis 失败可根据降级策略标记为部分就绪，MySQL 失败必须为未就绪。

Navicat 本地连接目标为 `127.0.0.1:3306`，具体数据库名、用户名和密码由本地 `.env` 决定。

## 11. Migration 和种子数据

- SQL Migration 保存在 `migrations/mysql`，每次结构修改包含可审查的 up/down 脚本。
- 开发迁移由显式命令执行，API 启动时不自动修改数据库结构。
- CI 在空库上执行 up，验证结构，再执行 down/up 验证可重建性。
- 种子数据仅用于开发和测试，包含一个城市、两个网点、两台柜机和小/中/大柜格。
- 生产迁移在部署阶段单独设计，必须先备份、再预检、最后人工确认。

## 12. 测试与 CI

### 12.1 单元测试

- 城市和网点启用/服务状态过滤。
- 经纬度与查询半径边界。
- 设备离线时间窗口判定。
- 柜格尺寸、占用状态和锁定过期规则。
- 指令有效期、重试上限和五种模拟场景。

### 12.2 集成和并发测试

- MySQL Migration 在空库可完整执行。
- 仓储查询、外键、唯一索引和事务行为正确。
- 多个 goroutine 同时预留同一尺寸最后一个柜格时，只能一个成功。
- 相同幂等键并发重试返回同一柜格。
- Redis 数据被清空或暂时不可用后，MySQL 中的预留仍正确。

### 12.3 CI 门禁

- `gofmt` 无差异。
- `go vet ./...` 通过。
- `go test ./...` 和 `go test -race ./...` 通过。
- 项目规则与 Skills 校验通过。
- MySQL/Redis 服务容器健康，Migration 和集成测试通过。
- API Docker 镜像构建通过。

## 13. 专属 Skill

本里程碑新增站点与设备领域 Skill，至少固化以下规则：

- MySQL 是柜格分配和设备指令的事实来源。
- Redis 只是加速层，不得单独决定柜格是否已分配。
- 柜格预留、释放、指令发送和重试必须使用幂等键。
- 内部接口不得无 Token 暴露，模拟器不得在生产注册。
- 连接信息、电话、Token 和设备凭据必须脱敏。
- 所有业务时间使用 UTC，状态和错误码使用稳定枚举。

Skill 必须包含基线对照与使用 Skill 后的对照测试，并纳入现有验证脚本。

## 14. 验收标准

- Docker Compose 可启动 API、MySQL 8.4 和 Redis 7，并使用命名 Volume。
- Navicat 可使用本地开发账号连接 MySQL，查看五张本里程碑业务表和种子数据。
- 公开 API 可查询已启用城市、附近网点、网点详情和柜格可用性。
- 内部 API 在缺少或使用错误 Token 时被拒绝。
- 柜格并发预留没有重复分配，幂等重试返回原结果。
- 模拟网关可稳定复现在线、离线、超时、失败和柜门未关。
- MySQL 和 Redis 容器重启后，MySQL 业务数据保留；清空 Redis 不破坏 MySQL 中的业务正确性。
- 生产 MySQL/Redis 凭据不出现在 Git 差异、测试输出、构建产物或应用日志中。
- 完整 CI 通过，工作树干净。

## 15. 后续里程碑边界

里程碑 3 验收后，里程碑 4 在当前预留接口上实现计费快照、订单状态机、存入、取件和状态日志。当订单绑定柜格后，过期回收任务不得释放该柜格。真实柜机协议、完整鉴权、生产部署和管理前端仍在后续里程碑实现。
