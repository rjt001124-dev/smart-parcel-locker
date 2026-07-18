# 智能快递柜 Figma 设计基线

- 状态：文件已创建，设计系统发现阶段受 Figma Starter MCP 调用额度限制
- 文件名：`Smart Parcel Locker / 智能快递柜`
- Figma URL：https://www.figma.com/design/ui9lT54QlghpCFiYxiB6WT
- File key：`ui9lT54QlghpCFiYxiB6WT`
- Editor type：Design
- 创建日期：2026-07-17
- Figma 账号：`rjt666`
- 团队：`Loera Maaske's team`
- 套餐：Starter
- 席位：View

## 设计范围

本基线负责建立智能快递柜的设计变量、组件库、微信小程序页面、React 管理后台页面及关键异常状态。批准后的 Figma 文件是前端布局、颜色、间距、排版、组件状态和交互行为的唯一视觉基准。

## 文件结构

计划创建以下页面：

1. `00 Foundations`
2. `01 Components`
3. `02 Mini App`
4. `03 Admin Web`

## 视觉方向

- 蓝白专业风格，强调可信、清晰和设备服务属性。
- 小程序采用充足留白、卡片式信息和单手可达的主操作。
- 管理后台采用深色侧边导航与浅色内容区。
- 失败状态必须说明原因、影响和下一步操作。

## 发现结果

新文件默认订阅了以下可用库：

- Material 3 Design Kit
- Simple Design System

本项目优先评估 Simple Design System 中可复用的通用组件和变量，再建立智能快递柜专属语义变量与业务组件。不得直接复制库组件后失去变量和实例关系。

## 当前阻塞

创建文件后，在搜索 Simple Design System 的组件、变量和样式时，Figma MCP 返回 Starter 套餐调用上限：

```text
You've reached the Figma MCP tool call limit on the Starter plan.
```

恢复后从设计系统搜索继续，不重复创建文件。升级入口：

https://www.figma.com/files/team/1646437779906104624/all-projects?upgrade=mcp_rate_limit_paywall

## 实施边界

- Figma 基线未批准前不实现前端页面。
- 当前里程碑不实现 Go Kratos 业务模块。
- Code Connect 等前端组件代码出现后再建立映射。
