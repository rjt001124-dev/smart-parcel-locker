# 智能快递柜本地 Figma 生成插件设计规格

- 日期：2026-07-17
- 仓库：`rjt001124-dev/smart-parcel-locker`
- Figma 文件：https://www.figma.com/design/ui9lT54QlghpCFiYxiB6WT
- 目标：在不升级 Figma Starter 套餐的前提下，通过本地开发插件生成并维护智能快递柜设计基线

## 1. 背景

正式 Figma Design 文件已经创建，但 Starter 套餐在设计系统搜索阶段触发 MCP 调用上限。项目不能依赖继续购买 MCP 调用额度，也不能通过批量注册账号规避限制。

解决方案是在仓库中维护一个本地 Figma 开发插件。插件直接运行在用户打开的 Figma 文件中，使用官方 Figma Plugin API 创建变量、样式、组件和页面，不经过 Figma MCP。用户只需在 Figma 桌面版中从 `manifest.json` 导入一次，后续可以重复运行各生成阶段。

## 2. 设计目标

- 不依赖 Figma MCP、外部服务或付费接口完成设计写入。
- 生成结果保存在现有 `Smart Parcel Locker / 智能快递柜` 文件中。
- 以四个可独立执行的阶段生成设计，失败时不必从头重建。
- 重复执行时更新插件拥有的节点，避免重复页面、变量和组件。
- 生成内容遵守里程碑 2 的页面、变量、组件、状态和画板尺寸要求。
- 插件源码、生成清单和测试可提交 GitHub，设计过程可重复、可审计。
- 插件只修改带有项目命名空间标识的资源，不删除或覆盖用户手工创建的未知节点。

## 3. 非目标

- 插件不实现微信小程序或 React 管理后台代码。
- 插件不连接订单、支付、柜机或通知后端。
- 插件不负责发布 Figma Library 或建立 Code Connect。
- 插件不尝试自动绕过 Figma 套餐、权限或调用限制。
- 首版不提供任意主题编辑器、拖拽页面编排器或通用低代码能力。

## 4. 用户操作流程

1. 安装并打开 Figma 桌面版。
2. 打开现有智能快递柜 Figma 文件；若团队权限导致只读，先复制到本人 Drafts。
3. 选择 `Plugins → Development → Import plugin from manifest...`。
4. 选择仓库中的 `tools/figma-plugin/manifest.json`。
5. 打开插件面板，按顺序执行四个阶段：
   - Foundations
   - Components
   - Mini App
   - Admin Web
6. 每个阶段完成后查看报告并检查对应页面。
7. 将异常信息或导出的截图交给 Codex进行修复和视觉 QA。

插件界面必须同时显示目标文件提示，避免用户在错误文件中运行。插件不能可靠校验 file key，因此以文件名、现有项目标识和显式确认作为防误操作措施。

## 5. 插件架构

```text
tools/figma-plugin/
├── manifest.json
├── package.json
├── tsconfig.json
├── src/
│   ├── code.ts                 # Figma 主线程入口与消息路由
│   ├── ui.html                 # 分阶段操作面板
│   ├── domain/
│   │   ├── catalog.ts          # 页面、组件、屏幕和状态清单
│   │   ├── tokens.ts           # 原始与语义 token 定义
│   │   └── run-report.ts       # 阶段结果与错误模型
│   ├── figma/
│   │   ├── ownership.ts        # 项目标识、查找与幂等更新
│   │   ├── variables.ts        # 变量集合、alias、scope、code syntax
│   │   ├── styles.ts           # 文字与效果样式
│   │   ├── nodes.ts            # Auto Layout、文本、图形等基础工厂
│   │   ├── components.ts       # 组件及变体生成
│   │   └── screens.ts          # 页面骨架与画板生成
│   └── stages/
│       ├── foundations.ts
│       ├── components.ts
│       ├── mini-app.ts
│       └── admin-web.ts
├── tests/
│   ├── catalog.test.ts
│   ├── tokens.test.ts
│   ├── ownership.test.ts
│   └── reports.test.ts
└── README.md
```

Figma 主线程仅负责接收 UI 消息、调用阶段函数、捕获错误并返回报告。各阶段通过小型服务模块创建资源，不把所有生成逻辑放进单个文件。

## 6. 分阶段生成

### 6.1 Foundations

创建或更新：

- `00 Foundations` 页面。
- 原始颜色、语义颜色、尺寸和排版变量集合。
- `Light` 模式、变量 scope、语义 alias 和 WEB code syntax。
- Display、Heading、Body、Label、Caption、Number 文字样式。
- 卡片、悬浮、弹窗和焦点环效果样式。
- Foundations 说明画板，展示 token 名称、用途和示例。

### 6.2 Components

创建或更新 `01 Components` 页面及首批组件族：

- Button、Icon Button。
- Input、Search、Select。
- Tag、Badge、Status Pill。
- Card、Site Card、Locker Size Card、Order Card。
- Alert、Result、Empty、Error、Skeleton。
- Modal、Bottom Sheet、Confirm Dialog。
- Mini App Top Bar、Bottom Tab Bar。
- Admin Sidebar、Header、Filter Bar、Data Table、Pagination。

组件优先绑定语义变量，变体使用可预测的属性命名。首版业务组件允许以结构完整的基础变体开始，后续按截图评审结果扩充细节。

### 6.3 Mini App

创建或更新 `02 Mini App` 页面，画板统一为 `375 × 812`。首版生成以下核心页面骨架：

- 首页、城市定位、附近网点、网点详情。
- 柜格与时长、订单确认、支付。
- 存件开门、订单列表、订单详情。
- 超时补缴、取件、个人中心。

页面必须使用已生成组件的实例。首版同时生成关键状态矩阵画板，覆盖 Loading、Empty、网络失败、支付处理中/失败/成功、柜格被抢占、设备离线、开门失败、柜门未关闭、超时补缴和权限不足。

### 6.4 Admin Web

创建或更新 `03 Admin Web` 页面，画板统一为 `1440 × 900`。首版生成：

- 登录/权限不足、Dashboard。
- 网点、柜机、订单、支付、通知管理。
- 权限管理、审计日志。

管理后台使用深色侧边导航、浅色内容区。远程开门、重启、隔离和回滚等高风险操作包含二次确认、原因输入和审计提示。

## 7. 幂等与所有权策略

插件使用 `setSharedPluginData` 写入以下标识：

- namespace：`smart-parcel-locker`
- `owner`：`local-figma-generator`
- `key`：稳定资源键，例如 `page/foundations`、`component/button`、`screen/miniapp/home`
- `schema_version`：插件生成结构版本

每次运行先通过共享插件数据查找资源：

- 找到同 key 且由插件拥有的资源：原位更新或重建其受控子树。
- 未找到：创建资源并写入标识。
- 找到同名但没有所有权标识的资源：不覆盖，报告冲突并停止该资源。
- 插件不得批量删除页面中的未知节点。

升级导致结构不兼容时，只替换带有相同稳定 key 的受控子树。任何删除动作都必须限制在插件拥有的节点内。

## 8. 插件界面与状态

界面提供：

- 当前文件名和运行环境提示。
- 四个阶段按钮及推荐执行顺序。
- 每阶段 `未运行 / 运行中 / 成功 / 部分失败 / 失败` 状态。
- 创建、更新、跳过、冲突和错误数量。
- 可复制的错误详情。
- “我确认当前是智能快递柜设计文件”确认框。

阶段运行期间禁用重复点击。插件关闭后不依赖 UI 内存恢复状态，真实状态始终以 Figma 文件中的共享插件数据为准。

## 9. 错误处理

- 字体不可用：列出实际可用字体，停止涉及文字的阶段，不生成半成品文本。
- 同名资源冲突：不覆盖用户资源，返回冲突 key 和节点名。
- 变量或组件创建失败：记录具体资源，阶段标记为部分失败。
- 前置阶段缺失：Components 要求 Foundations；页面阶段要求 Components。缺失时给出明确操作提示。
- 用户在错误文件运行：未确认时禁止生成；发现已有不同项目 owner 时停止。
- 单个资源失败不吞掉异常；报告必须保留阶段、资源 key 和错误消息。

## 10. 测试策略

纯数据和决策逻辑在 Node 测试环境中测试，Figma API 写入通过薄适配层隔离。

测试至少验证：

- 页面、组件、屏幕和关键状态清单完整且稳定 key 唯一。
- token 名称、类型、scope、alias 目标和 WEB code syntax 合法。
- 所有权判断只允许修改插件拥有的资源。
- 同名未知节点产生冲突而不是覆盖。
- 阶段前置依赖正确。
- 运行报告正确汇总创建、更新、跳过、冲突和错误。
- 构建产物包含有效 `manifest.json`、主线程代码和 UI。

Figma 内人工验收验证：

- 四个页面名称和顺序正确。
- 变量、样式、组件、画板尺寸和异常状态存在。
- 重复运行同一阶段不会增加重复页面或重复稳定 key。
- 无文字裁切、组件重叠、遗留占位效果或错误字体。
- 插件不会修改手工创建且无所有权标识的节点。

## 11. 构建与交付

插件使用 TypeScript 开发，构建为 Figma 可直接加载的 JavaScript 和内嵌 UI。仓库提交源代码、锁定依赖、测试、构建脚本和使用说明，不提交凭据或个人访问令牌。

推荐命令：

```powershell
npm install
npm test
npm run build
```

最终 `README.md` 提供 Figma 桌面版导入步骤、阶段顺序、重复运行规则、故障排查和截图验收方式。

## 12. 成功标准

- 用户无需升级 Figma 套餐即可在现有文件中运行插件。
- 四个阶段可以独立执行并产生结构化报告。
- 重复执行保持稳定 key 唯一且不覆盖未知用户节点。
- 自动化测试和构建通过。
- Figma 文件至少形成可评审的 Foundations、核心组件、小程序页面骨架、后台页面骨架与状态矩阵。
- 插件代码和文档推送到 `codex/milestone-2-figma-baseline` 分支。
