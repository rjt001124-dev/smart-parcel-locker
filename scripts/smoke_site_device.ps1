[CmdletBinding()]
param(
    [string]$BaseUrl = "http://127.0.0.1:8000",
    [string]$EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

if (Test-Path -LiteralPath $EnvFile) {
    foreach ($line in Get-Content -LiteralPath $EnvFile) {
        if ($line -match '^\s*#' -or $line -notmatch '=') { continue }
        $name, $value = $line -split '=', 2
        if ($name.Trim()) { [Environment]::SetEnvironmentVariable($name.Trim(), $value.Trim(), 'Process') }
    }
}

$token = $env:INTERNAL_API_TOKEN
if ([string]::IsNullOrWhiteSpace($token)) { $token = "change-this-local-token" }
$headers = @{ "X-Internal-Token" = $token }

for ($attempt = 0; $attempt -lt 60; $attempt++) {
    try {
        $ready = Invoke-RestMethod -Uri "$BaseUrl/readyz" -TimeoutSec 2
        if ($ready.status -in @('ok', 'degraded')) { break }
    } catch {}
    Start-Sleep -Seconds 2
}
if ($attempt -ge 60) { throw "API readiness timeout" }

$cities = Invoke-RestMethod -Uri "$BaseUrl/v1/cities"
$sites = Invoke-RestMethod -Uri "$BaseUrl/v1/sites?city_code=310100&latitude=31.2304&longitude=121.4737&radius_m=5000"
if (-not $sites.sites -or $sites.sites.Count -eq 0) { throw "No nearby site returned" }

$heartbeat = @{ deviceNo = 'DEV-SH-001'; firmwareVersion = 'smoke'; reportedAt = [DateTime]::UtcNow.ToString('o'); online = $true } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "$BaseUrl/v1/internal/devices/DEV-SH-001/heartbeat" -Headers $headers -ContentType 'application/json' -Body $heartbeat | Out-Null

$key = "smoke-reserve-$([Guid]::NewGuid().ToString('N'))"
$reserve = @{ siteId = [string]$sites.sites[0].id; size = 'CELL_SIZE_MEDIUM'; ttlSeconds = 120; idempotencyKey = $key } | ConvertTo-Json
$reservation = Invoke-RestMethod -Method Post -Uri "$BaseUrl/v1/internal/cells/reserve" -Headers $headers -ContentType 'application/json' -Body $reserve
$replay = Invoke-RestMethod -Method Post -Uri "$BaseUrl/v1/internal/cells/reserve" -Headers $headers -ContentType 'application/json' -Body $reserve
if ($reservation.cellId -ne $replay.cellId) { throw "Reservation replay mismatch" }

$commandBody = @{ deviceNo = $reservation.deviceNo; action = 'DEVICE_ACTION_OPEN_DOOR'; cellNo = $reservation.cellNo; ttlSeconds = 30; idempotencyKey = "smoke-command-$([Guid]::NewGuid().ToString('N'))" } | ConvertTo-Json
$command = Invoke-RestMethod -Method Post -Uri "$BaseUrl/v1/internal/device-commands" -Headers $headers -ContentType 'application/json' -Body $commandBody

Write-Host "City count: $($cities.cities.Count)"
Write-Host "Site ID: $($sites.sites[0].id)"
Write-Host "Reserved cell ID: $($reservation.cellId)"
Write-Host "Device command: $($command.commandNo) status=$($command.status)"
