# 智能快递柜里程碑 2：Figma 设计基线实施计划

> 日期：2026-07-17  
> 分支：`codex/milestone-2-figma-baseline`  
> 范围：只建立可评审、可追踪、可供前端严格实现的 Figma 设计基线；不实现城市、网点、柜机、订单、支付等业务代码。

## 目标

创建一份蓝白专业风格的智能快递柜 Figma 设计文件，统一变量、样式、组件和关键业务状态，覆盖微信小程序用户端与 React 管理后台。设计通过截图和结构化元数据验收后，在仓库中固化 Figma 地址、文件标识、设计决策、状态台账和 `building-from-figma` 项目 Skill，作为后续前端实现的唯一视觉基准。

## 完成标准

- Figma 文件包含 `00 Foundations`、`01 Components`、`02 Mini App`、`03 Admin Web` 四个页面。
- 用户端画板基准为 `375 × 812`，管理后台画板基准为 `1440 × 900`。
- 所有颜色、间距、圆角、尺寸和语义状态通过 Figma Variables 或样式表达，不在页面中随意写死。
- 基础组件具有明确的变体轴、属性、交互状态、禁用状态和使用说明。
- 小程序与后台的核心页面及规定异常状态均有可评审画板。
- 每个完成阶段都有元数据检查与截图检查；不存在重叠、裁切、占位 shimmer 或未加载字体。
- 仓库记录 Figma URL、file key、页面与节点台账、设计决策、异常状态覆盖矩阵和验收结果。
- `building-from-figma` Skill 通过官方与仓库校验，并有基线/前向测试证据。
- 分支工作树干净，提交按阶段拆分。

## 明确不在本里程碑内

- 不创建或修改 Go Kratos 业务模块、Proto、数据库迁移、Redis、Worker。
- 不实现微信小程序或 React 管理后台代码。
- 不接入支付宝、真实柜机、通知渠道、Coze 或部署环境。
- 不在尚无对应前端组件代码时强行创建 Code Connect 映射；仅设置变量 WEB code syntax，并记录映射待办。

## 设计方向

- 主视觉：蓝白专业风格，强调可信、清晰、轻量和设备服务属性。
- 小程序：大留白、清晰的主操作、卡片式网点与订单信息、适合单手操作。
- 管理后台：深色侧边导航、浅色内容区、紧凑但不拥挤的数据表格和状态信息。
- 状态反馈：任何失败状态都提供原因、影响和可执行的下一步，而不是只显示“失败”。
- 金额展示：界面按人民币展示，设计注释明确后端以整数分保存。

## 仓库产物

- Create: `docs/design/figma-baseline.md`
- Create: `docs/design/figma-state-ledger.json`
- Create: `docs/design/figma-qa.md`
- Create: `docs/skill-evals/building-from-figma/baseline.md`
- Create: `docs/skill-evals/building-from-figma/with-skill.md`
- Create: `.agents/skills/building-from-figma/SKILL.md`
- Create: `.agents/skills/building-from-figma/agents/openai.yaml`
- Create: `.agents/skills/building-from-figma/references/implementation-contract.md`
- Modify: `README.md`

## Task 1：确认 Figma 身份、套餐与文件创建能力

1. 调用 Figma `whoami`，记录当前账号、团队和可用 `planKey`。
2. 确认能够创建 Design 文件，而不是 FigJam 或 Slides。
3. 若账号无创建权限，停止写入并记录明确阻塞信息；不得改用未授权团队或个人空间。
4. 创建名为 `Smart Parcel Locker / 智能快递柜` 的新 Design 文件。
5. 返回并保存 Figma URL、file key、editor type、plan key。

验收：文件可打开、可编辑，链接与 file key 已进入状态台账。

提交：本任务不单独提交，和 Task 2 的发现文档一起提交。

## Task 2：Phase 0 发现与需求差距分析

1. 阅读 10 份工单中与页面、字段、动作、状态和文案直接相关的内容；按用户端、后台端、跨端状态分类。
2. 检查新 Figma 文件的页面、变量、组件和字体初始状态。
3. 建立需求覆盖矩阵：工单编号 → 用户角色 → 页面 → 主操作 → 正常状态 → 异常状态。
4. 标注尚未确定但不阻塞视觉基线的内容，例如真实品牌 Logo、支付商户文案、柜机厂商字段。
5. 在 `docs/design/figma-baseline.md` 记录：
   - Figma URL 与 file key；
   - 设计范围和非范围；
   - 用户角色；
   - 关键业务流程；
   - 需求覆盖矩阵；
   - 暂定项和后续决策点。
6. 初始化 `docs/design/figma-state-ledger.json`，至少包含 `runId`、`fileKey`、`pages`、`collections`、`styles`、`components`、`screens`、`qa`。

验收：没有 `TODO`、示例 ID 或虚假 Figma 链接；每个核心流程都有页面落点。

提交：

```powershell
git add docs/design/figma-baseline.md docs/design/figma-state-ledger.json
git commit -m "docs: define Figma design baseline"
```

## Task 3：Phase 1 变量、字体与效果基础

在创建任何组件前完成变量基础，并在每次 Figma 写入后把返回的 collection、variable、style、node ID 写入状态台账。

### 3.1 变量集合

建立以下集合，并为变量设置明确 scope：

- `Primitives / Color`：蓝、灰、绿、橙、红及透明度基础色阶。
- `Semantic / Color`：`bg/*`、`text/*`、`border/*`、`action/*`、`status/*`、`overlay/*`。
- `Dimension`：`space/*`、`radius/*`、`size/*`、`stroke/*`。
- `Typography`：字号、行高、字重语义；字体家族仅使用实际可用并已验证的字体。

颜色语义至少覆盖：primary、secondary、success、warning、danger、info、disabled、offline、processing、selected、focus。

### 3.2 模式与代码语法

- 颜色集合建立 `Light` 模式；后台深色导航通过语义变量表达，不在本阶段扩展完整 Dark Mode。
- 为 Web 可复用变量设置统一 WEB code syntax，例如 `color/bg/primary` → `var(--color-bg-primary)`。
- 原始色只供语义变量 alias，组件优先绑定语义变量。

### 3.3 样式

- 创建文字样式：Display、Heading 1–3、Body、Label、Caption、Number。
- 创建效果样式：卡片阴影、悬浮阴影、弹窗阴影、焦点环。
- 建立图标尺寸和描边规则；图标先采用统一占位图标体系，不混用不同风格。

验收：变量数量、模式、scope、alias、code syntax、文字样式和效果样式通过结构检查；基础页截图无裁切和字体异常。

提交：

```powershell
git add docs/design/figma-state-ledger.json docs/design/figma-baseline.md
git commit -m "design: establish Figma foundations"
```

## Task 4：Phase 2 文件结构与 Foundations 文档页

1. 创建并按顺序排列：
   - `00 Foundations`
   - `01 Components`
   - `02 Mini App`
   - `03 Admin Web`
2. 在 `00 Foundations` 建立说明区：设计原则、色彩、排版、间距、圆角、阴影、图标、状态色和金额格式。
3. 使用 Auto Layout 组织说明内容；顶层区块先使用 placeholder，填充完成后必须关闭 placeholder。
4. 每个 token 示例展示变量名和用途，而不仅是视觉色块。
5. 截图 Foundations 页面并检查颜色层级、文本换行、间距和说明完整性。

验收：四页命名精确；Foundations 可独立指导前端实现；页面截图和元数据检查通过。

提交：

```powershell
git add docs/design/figma-state-ledger.json docs/design/figma-qa.md
git commit -m "design: document Figma foundations"
```

## Task 5：Phase 3 核心组件库

每次只完成一个组件族：创建 → 绑定变量 → 组合变体 → 添加组件属性 → 元数据检查 → 截图检查 → 更新台账。每个 Figma 写入调用控制在最多 10 个逻辑操作。

### 5.1 通用组件

- Button：Style × Size × State，支持 loading、disabled、icon、label。
- Icon Button：Size × Style × State。
- Input / Search / Textarea：Default、Focus、Filled、Error、Disabled。
- Select / Picker / City Selector。
- Radio、Checkbox、Switch、Stepper。
- Tag / Badge / Status Pill：订单、支付、设备、通知状态。
- Toast、Inline Alert、Result、Empty State、Error State、Skeleton。
- Modal、Bottom Sheet、Dialog、Confirm Dialog。
- Tabs、Segmented Control、Pagination。

### 5.2 业务组件

- Site Card：距离、营业状态、可用柜格数、地址、导航动作。
- Locker Size Card：尺寸、价格、剩余数量、选中/售罄/被抢占。
- Price Summary：租金、押金、优惠、超时费、合计。
- Order Card / Order Timeline / Payment Status。
- Pickup Code / Door Operation Panel / Door Not Closed Alert。
- Metric Card、Chart Container、Filter Bar、Data Table、Drawer Form。
- Device Status、Cell Status、Notification Delivery Status、Audit Entry。
- Mini App Top Bar、Bottom Tab Bar；Admin Sidebar、Header、Breadcrumb。

### 5.3 组件验收

- 变体命名可预测，例如 `Size=Medium, Style=Primary, State=Default`。
- 文本、布尔和 instance swap 属性连接到真实子节点。
- 状态色和间距绑定变量，组件变体不堆叠、不重叠。
- 每个组件族有用途、禁用场景和状态说明。
- 每个组件族完成后保存截图证据和节点 ID。

提交：按 3–5 个组件族拆分，不做一个超大提交，例如：

```powershell
git add docs/design/figma-state-ledger.json docs/design/figma-qa.md
git commit -m "design: add Figma form components"
git commit -m "design: add Figma feedback components"
git commit -m "design: add smart locker components"
```

## Task 6：Phase 4 小程序页面（375 × 812）

页面使用组件实例和变量组装，不复制散落的视觉元素。先创建每个页面骨架，再逐区填充和截图。

### 6.1 核心页面

1. 首页：定位城市、附近寄存点、搜索、快捷入口、当前订单摘要。
2. 城市/定位：定位授权、当前城市、热门城市、城市列表、定位失败。
3. 附近网点：列表、地图切换、筛选、距离、营业状态。
4. 网点详情：地址、营业时间、柜格库存、导航、选择柜格。
5. 柜格与时长：尺寸选择、剩余数、寄存时长、价格预估。
6. 订单确认：计费明细、押金、优惠券、协议、提交支付。
7. 支付：支付宝跳转提示、处理中、成功、失败、取消。
8. 存件开门：柜机/柜格信息、开门进度、失败重试、门未关闭。
9. 订单列表：当前订单、历史订单、筛选和空状态。
10. 订单详情：状态时间线、费用、网点、柜格、可执行动作。
11. 超时补缴：超时费用、补缴支付、补缴成功。
12. 取件：取件码、开门、门未关闭、完成。
13. 个人中心：头像、账户入口、优惠券、帮助、客服、退出。

### 6.2 必须显式覆盖的状态

- Loading、Empty、Network failure。
- Payment processing、Payment failure、Payment success。
- Cell contention / `LOCKER_CELL_OCCUPIED`。
- Device offline / `LOCKER_DEVICE_OFFLINE`。
- Door-open failure / `DEVICE_COMMAND_TIMEOUT`。
- Door not closed。
- Overdue payment required。
- Permission denied。

验收：核心寄存链路从首页到完成可按画板顺序审阅；所有失败态都有返回、重试、重选、补缴或联系客服等明确动作。

提交：

```powershell
git add docs/design/figma-state-ledger.json docs/design/figma-qa.md
git commit -m "design: create mini app Figma flows"
```

## Task 7：Phase 5 管理后台页面（1440 × 900）

### 7.1 核心页面

1. 登录与权限不足。
2. Dashboard：订单、收入、柜格使用率、设备在线率、告警摘要。
3. 网点管理：列表、筛选、详情、创建/编辑抽屉、空状态。
4. 柜机管理：在线状态、心跳、柜格占用、门状态、异常告警。
5. 订单管理：列表、详情、状态时间线、费用和关联设备事件。
6. 支付管理：支付单、退款单、流水、金额不一致告警。
7. 通知管理：模板、投递记录、失败重试、人工补发。
8. 权限管理：管理员、角色、权限矩阵。
9. 审计日志：操作者、动作、对象、结果、trace ID、时间。

### 7.2 高风险操作设计

- 远程开门、重启、隔离、回滚等操作必须经过二次确认。
- 确认框显示影响对象、风险、原因输入和审计提示。
- 权限不足时隐藏或禁用高风险操作，并提供申请权限指引。

验收：数据表格在 1440 宽度下不发生关键字段裁切；筛选、分页、抽屉、弹窗和状态反馈均使用组件实例；设备离线、告警、空数据、加载失败和权限不足有专门状态。

提交：

```powershell
git add docs/design/figma-state-ledger.json docs/design/figma-qa.md
git commit -m "design: create admin web Figma flows"
```

## Task 8：Phase 6 全量 QA 与用户评审

1. 对四个页面分别运行结构检查：页面名、顶层节点、组件/实例关系、变量绑定、变体数量、画板尺寸。
2. 对 Foundations、Components、Mini App、Admin Web 分别生成最终截图。
3. 检查：
   - 无重叠、无裁切、无零宽文本、无遗留 placeholder；
   - 字体均已加载且风格名称真实存在；
   - 颜色、间距、圆角、效果使用变量或样式；
   - 关键状态覆盖矩阵全部通过；
   - 页面组件使用实例，不出现无理由复制；
   - 小程序触控目标和后台信息密度合理。
4. 将结果写入 `docs/design/figma-qa.md`，每项包含证据节点 ID、截图说明、结果和遗留问题。
5. 向用户展示四页总览和核心流程截图，获得设计批准后才允许进入前端实现。
6. 若用户要求调整，先更新 Figma，再重新执行受影响页面的结构和截图验收。

验收：所有阻塞级问题关闭；用户明确批准设计基线。

提交：

```powershell
git add docs/design/figma-baseline.md docs/design/figma-state-ledger.json docs/design/figma-qa.md
git commit -m "test: verify Figma design baseline"
```

## Task 9：创建并验证 `building-from-figma` 项目 Skill

1. 在创建 Skill 前，用一个固定压力提示记录无 Skill 基线：要求开发者绕过 Figma、临时修改颜色和省略失败状态。
2. 保存完整结果到 `docs/skill-evals/building-from-figma/baseline.md`。
3. 使用官方 Skill 生成器初始化 `.agents/skills/building-from-figma`。
4. Skill 必须要求：
   - 先读取仓库中的 Figma URL、file key 和状态台账；
   - 设计批准前不实现页面；
   - 复用变量、组件和交互状态；
   - 不擅自改变布局、颜色、间距、层级、文案和流程；
   - 金额、订单状态、设备错误码与后端契约一致；
   - 实现 Loading、Empty、失败、权限不足等设计状态；
   - 保存设计节点与代码组件的追踪关系；
   - 视觉回归与浏览器测试通过后才宣称完成。
5. 在 `references/implementation-contract.md` 固化画板尺寸、页面清单、状态矩阵和变更审批规则。
6. 运行官方校验器、仓库校验器和测试。
7. 使用与基线相同的压力提示前向测试；Skill 必须拒绝绕过批准设计并指出正确流程。
8. 保存结果到 `docs/skill-evals/building-from-figma/with-skill.md`；只修复实际暴露的漏洞。

提交：

```powershell
git add .agents/skills/building-from-figma docs/skill-evals/building-from-figma
git commit -m "feat: add Figma implementation skill"
```

## Task 10：文档收口与分支验证

1. 在 `README.md` 增加设计入口：Figma 链接、设计状态、验收文档、前端实现前置条件。
2. 确认 Code Connect 当前状态：
   - 若尚无前端组件，记录为“待前端组件创建后映射”；
   - 后续优先为 Button、Input、Status、SiteCard、OrderCard 建立简单映射。
3. 检查所有文档不存在令牌、账号隐私、临时链接、示例 file key 和占位内容。
4. 运行：

```powershell
python scripts/test_agents_rules.py
python scripts/test_validate_skills.py
python scripts/validate_skills.py
python "D:\gowork\.codex\skills\.system\skill-creator\scripts\quick_validate.py" .agents/skills/building-from-figma
git diff --check
git status --short --branch
```

5. 检查提交历史按“发现 → Foundations → Components → Mini App → Admin → QA → Skill → README”组织。

提交：

```powershell
git add README.md docs/design
git commit -m "docs: publish Figma design baseline"
```

## 执行规则

- 每次 Figma 写入前都加载并遵守 `figma-use`；新建文件前加载 `figma-create-new-file`。
- 组件建设同时遵守 `figma-generate-library`；页面组装同时遵守 `figma-generate-design`。
- 任何 Figma 写入错误都停止，先分析错误；失败写入视为原子失败，修正后再重试。
- 每次调用最多切换一次页面；跨页面任务拆分为每页一个调用。
- 所有写入必须返回创建/修改的节点 ID，并同步到状态台账。
- 组件和页面都使用 Auto Layout 表达结构关系。
- 组件完成后必须截图；阶段完成后必须提交阶段总结和 QA 证据。
- 用户批准 Figma 基线前，不开始前端代码实现。

## 后续里程碑边界

里程碑 2 批准后，下一步进入城市、网点、柜机和柜格的 Kratos 领域基线；前端工程骨架可并行准备，但任何页面实现必须读取并严格遵循本里程碑批准的 Figma 设计与 `building-from-figma` Skill。
