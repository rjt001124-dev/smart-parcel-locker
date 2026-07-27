# 小程序本地开发

## 启动后端依赖与服务

在仓库根目录执行：

```powershell
docker compose -f deploy/docker-compose.yml up -d mysql redis api worker
```

首次使用空数据库时，先按仓库根目录 `README.md` 的说明执行数据库迁移。

## 启动微信小程序构建

```powershell
Set-Location frontend
npm ci
$env:TARO_APP_API_BASE_URL = "http://127.0.0.1:8000"
npm run dev:weapp -w @spl/miniapp
```

构建进程会持续监听源码，并把微信小程序产物写入
`frontend/apps/miniapp/dist`。

## 导入微信开发者工具

1. 打开微信开发者工具，选择“导入项目”。
2. 项目目录选择 `frontend/apps/miniapp`。
3. 本地预览可以使用仓库中提交的 `touristappid`。
4. 真正发布时，在开发者工具本地配置中选择公司的微信小程序 AppID。

真实 AppID 的使用权限、密钥、上传私钥和线上接口凭据不得提交到仓库。小程序只调用公开 `/v1` 接口，绝不能配置或发送 `X-Internal-Token`。

## 提交前检查

```powershell
Set-Location frontend
npm run typecheck
npm test
npm run build:miniapp
```
