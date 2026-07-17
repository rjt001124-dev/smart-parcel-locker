# Smart Parcel Locker Figma Generator

本地 Figma 开发插件，用于在 Starter 免费套餐中生成智能快递柜设计基线，不消耗 Figma MCP 调用额度。

## 环境

- Node.js 20+
- Figma Desktop
- 当前仓库分支的完整工作区

## 构建

```powershell
Set-Location tools/figma-plugin
npm install
npm test -- --run
npm run typecheck
npm run build
```

## 导入

在 Figma Desktop 中打开目标 Design 文件，然后选择：

```text
Plugins → Development → Import plugin from manifest...
```

选择当前目录中的 `manifest.json`。以后代码重新构建后不需要再次导入，只需关闭并重新打开插件。

## 运行

依次运行：

1. `生成 Foundations`
2. `生成 Components`
3. `生成 Mini App`
4. `生成 Admin Web`

插件适配 Starter 套餐的三页限制：

- `00 Design System`：Foundations 与 Components
- `01 Mini App`
- `02 Admin Web`

首次运行会复用默认的 `Page 1`。重复运行更新插件拥有的资源，不重复创建页面和稳定 key，不删除未知用户节点。

## 最终生成范围

- 原始/语义颜色、尺寸变量、文字和效果样式
- 含真实示例的 Foundations 文档
- 25 个可见业务组件
- 13 个小程序业务页面与 12 类状态矩阵
- 9 个管理后台页面、权限不足页和高风险操作确认页

## 故障排查

- `Run Foundations first`：按规定顺序运行阶段。
- `conflict`：存在同名但非插件拥有的资源；插件不会覆盖它。
- 字体错误：确认 Figma 可使用 Inter Regular、Medium、Semi Bold、Bold。
- 结果未更新：关闭插件窗口并从 Development 菜单重新打开。
- 页面数量不足：不要手工新增页面；插件会复用初始页面并控制在三页内。

## 验收

运行完成后依据 `docs/design/figma-qa.md` 检查。只有四阶段均成功、重复运行没有新增稳定资源且视觉检查通过后，才能批准前端实现。
