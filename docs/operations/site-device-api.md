# 站点与设备 API

默认本地地址：`http://127.0.0.1:8000`。所有 `/v1/internal` 请求必须携带 `X-Internal-Token`，Token 从本地 `.env` 读取，不得打印到终端日志或文档。

## 公共查询

```powershell
Invoke-RestMethod http://127.0.0.1:8000/v1/cities
Invoke-RestMethod "http://127.0.0.1:8000/v1/sites?city_code=310100&latitude=31.2304&longitude=121.4737&radius_m=5000"
Invoke-RestMethod http://127.0.0.1:8000/v1/sites/1
Invoke-RestMethod "http://127.0.0.1:8000/v1/sites/1/cells?size=CELL_SIZE_MEDIUM"
```

## 内部预约

```powershell
$headers = @{ "X-Internal-Token" = $env:INTERNAL_API_TOKEN }
$body = @{
  siteId = "1"
  size = "CELL_SIZE_MEDIUM"
  ttlSeconds = 120
  idempotencyKey = "example-$([Guid]::NewGuid().ToString('N'))"
} | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri http://127.0.0.1:8000/v1/internal/cells/reserve -Headers $headers -ContentType application/json -Body $body
```

相同请求与相同幂等键返回原预约；同一键绑定不同参数返回 `IDEMPOTENCY_CONFLICT`。

## 心跳与设备命令

```powershell
$heartbeat = @{ deviceNo = "DEV-SH-001"; firmwareVersion = "local"; reportedAt = [DateTime]::UtcNow.ToString("o"); online = $true } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri http://127.0.0.1:8000/v1/internal/devices/DEV-SH-001/heartbeat -Headers $headers -ContentType application/json -Body $heartbeat

$command = @{ deviceNo = "DEV-SH-001"; action = "DEVICE_ACTION_OPEN_DOOR"; cellNo = "B01"; ttlSeconds = 30; idempotencyKey = "command-$([Guid]::NewGuid().ToString('N'))" } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri http://127.0.0.1:8000/v1/internal/device-commands -Headers $headers -ContentType application/json -Body $command
```

## 模拟器场景

开发/测试环境支持 `ONLINE`、`OFFLINE`、`TIMEOUT`、`FAIL`、`DOOR_LEFT_OPEN`。生产环境必须返回 404，不允许暴露场景切换能力。

```powershell
$scenario = @{ deviceNo = "DEV-SH-001"; scenario = "SIMULATOR_SCENARIO_DOOR_LEFT_OPEN" } | ConvertTo-Json
Invoke-RestMethod -Method Put -Uri http://127.0.0.1:8000/v1/internal/simulator/devices/DEV-SH-001/scenario -Headers $headers -ContentType application/json -Body $scenario
```

完整流程使用 `scripts/smoke_site_device.ps1`，脚本只输出业务 ID 和状态，不输出 Token 或密码。
