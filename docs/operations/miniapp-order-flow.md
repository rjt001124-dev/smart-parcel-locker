# 小程序订单流程与界面状态约定

本文说明 `frontend/apps/miniapp` 中订单相关的页面、状态机与不可违反的界面约定。
所有页面代码均位于 `frontend/apps/miniapp/src/pages`，订单数据访问位于
`frontend/packages/api-client`，纯展示模型位于 `frontend/packages/domain-ui`。

## 订单状态机

订单状态（`OrderStatus`，见 `domain-ui/src/order.ts`）由**后端唯一权威**驱动：

| 状态 | 含义 | 可用动作 |
| --- | --- | --- |
| `ORDER_STATUS_PENDING_PAYMENT` | 待支付 | 去支付 / 取消订单 |
| `ORDER_STATUS_PAID` | 已支付 | （无） |
| `ORDER_STATUS_AWAITING_DEPOSIT` | 待存入 | 开门存入 |
| `ORDER_STATUS_IN_STORAGE` | 寄存中 | 开门取件 |
| `ORDER_STATUS_AWAITING_PICKUP` | 待取件 | 开门取件 |
| `ORDER_STATUS_OVERDUE` | 已逾期 | 补缴逾期费 |
| `ORDER_STATUS_COMPLETED` | 已完成 | （无） |
| `ORDER_STATUS_CANCELLED` | 已取消 | （无） |

`availableOrderActions(status)` 始终在末尾追加 `contactSupport`（联系客服兜底入口）。

## 页面与职责

| 页面 | 路由 | 职责 |
| --- | --- | --- |
| 订单列表 | `pages/orders/index` | tabBar「订单」；`listOrders` 展示全部订单，点击进入详情 |
| 订单详情 | `pages/order-detail/index` | 状态胶囊 + 状态时间线 + 费用明细 + 由服务端状态派生的动作按钮 |
| 存入 | `pages/store/index` | `openDepositDoor`；柜门「已打开」仅来自开门接口成功返回 |
| 取件 | `pages/pickup/index` | `openPickupDoor`；同上 |
| 补缴 | `pages/overdue/index` | `settleOverdue`；仅在 `ORDER_STATUS_OVERDUE` 且开发模拟器开启时提供「补缴逾期费（开发环境）」 |
| 支付 | `pages/payment/index` | `confirmDevelopmentPayment`；仅在开发模拟器开启时提供「模拟支付（开发环境）」 |

`pages/orders/index` 与 `pages/order-detail/index` 通过 `navigateTo` 跳转；
`pages/orders/index` 同时是 tabBar 页面，需使用 `switchTab`（由 Taro 框架按 tabBar 配置自动处理）。

## 不可违反的界面约定

1. **动作只来自服务端状态。** `order-detail` 的动作按钮完全由
   `availableOrderActions(order.status)` 决定，前端绝不根据本地臆测展示「支付成功 /
   开门成功」。
2. **绝不伪造支付或开门结果。** 支付成功、柜门已打开等状态必须来自对应 API 的真实
   返回（`confirmDevelopmentPayment` / `openDepositDoor` / `openPickupDoor` /
   `settleOverdue`）。
3. **金额只用整数分。** 所有费用在 `fee_snapshot` 与 `overdue_fee_fen` 中以整数
   `fen`（分）传输与展示，前端通过 `domain-ui/src/money.ts` 的 `formatFen` 渲染为
   `¥X.XX`，不允许浮点计算。
4. **开发模拟器受环境变量门控。** `TARO_APP_DEV_PAYMENT_SIMULATOR=true` 只在本地
   `dev:h5` 等开发命令中注入，生产构建不会携带，因此不会暴露模拟支付/补缴入口。
5. **前端无密钥。** 小程序只调用公开 `/v1` 接口，绝不配置或发送 `X-Internal-Token`。

## 核心复用单元

- `features/orders/use-order.ts`
  - `useOrder(orderId)`：加载单个订单、暴露 `reload`，错误时回传 `code` / `traceId`。
  - `orderActionRoute(action, orderId)`：把服务端派生的动作映射到目标页面路由
    （`pay` / `openDepositDoor` / `openPickupDoor` / `settleOverdue`；
    `cancel` 与 `contactSupport` 为页内处理，返回 `null`）。
  - `devPaymentSimulatorEnabled()` / `statusPillTone(status)`：开发模拟器开关与状态胶囊色调。
- 组件：`FeeSummary`、`StatusTimeline`、`DoorStatusPanel`、`StatusPill`、
  `PrimaryButton`、`StatePanel` 均为纯展示组件，不持有订单业务状态。

## 本地预览

见 [小程序本地开发](./miniapp-local-development.md)。H5 预览默认在
`http://127.0.0.1:10086/index.html`，并把 `/api` 代理到本地 Kratos 服务的 `:8000`，
因此不需要修改后端 CORS，也不会把内部令牌发送到前端。
