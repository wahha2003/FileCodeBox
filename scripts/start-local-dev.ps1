param(
    [switch]$SkipInstall
)

$ErrorActionPreference = 'Stop'

function Write-Info($msg) {
    Write-Host "[INFO] $msg" -ForegroundColor Cyan
}

function Write-WarnMsg($msg) {
    Write-Host "[WARN] $msg" -ForegroundColor Yellow
}

function Test-Command($name) {
    return $null -ne (Get-Command $name -ErrorAction SilentlyContinue)
}

function Test-PortInUse([int]$Port) {
    try {
        $conn = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
        return $null -ne $conn
    } catch {
        return $false
    }
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$backendDir = Join-Path $repoRoot 'backend'
$frontendDir = Join-Path $repoRoot 'frontend'

if (-not (Test-Path $backendDir)) { throw "未找到 backend 目录: $backendDir" }
if (-not (Test-Path $frontendDir)) { throw "未找到 frontend 目录: $frontendDir" }

if (-not (Test-Command 'go')) { throw '未检测到 go，请先安装 Go 1.21+' }
if (-not (Test-Command 'node')) { throw '未检测到 node，请先安装 Node.js 18+' }
if (-not (Test-Command 'npm')) { throw '未检测到 npm，请先安装 npm' }

$backendPort = 8888
$frontendPort = 3000

if (Test-PortInUse $backendPort) {
    Write-WarnMsg "端口 $backendPort 已被占用，后端可能启动失败。"
}
if (Test-PortInUse $frontendPort) {
    Write-WarnMsg "端口 $frontendPort 已被占用，前端可能启动失败。"
}

if (-not $SkipInstall) {
    Write-Info '检查并安装前端依赖...'
    if (-not (Test-Path (Join-Path $frontendDir 'node_modules'))) {
        Push-Location $frontendDir
        try {
            npm install
        } finally {
            Pop-Location
        }
    } else {
        Write-Info 'frontend/node_modules 已存在，跳过 npm install。'
    }

    Write-Info '下载后端 Go 依赖...'
    Push-Location $backendDir
    try {
        go mod download
    } finally {
        Pop-Location
    }
} else {
    Write-Info '已启用 -SkipInstall，跳过依赖安装。'
}

$backendCmd = "Set-Location '$backendDir'; go run ./cmd/server/main.go"
$frontendCmd = "Set-Location '$frontendDir'; npm run dev"

Write-Info '正在启动后端服务...'
Start-Process -FilePath 'powershell' -ArgumentList @('-NoExit', '-ExecutionPolicy', 'Bypass', '-Command', $backendCmd) | Out-Null

Write-Info '正在启动前端服务...'
Start-Process -FilePath 'powershell' -ArgumentList @('-NoExit', '-ExecutionPolicy', 'Bypass', '-Command', $frontendCmd) | Out-Null

Write-Host ''
Write-Host '已发起启动：' -ForegroundColor Green
Write-Host "- Backend: http://localhost:$backendPort" -ForegroundColor Green
Write-Host "- Frontend: http://localhost:$frontendPort" -ForegroundColor Green
Write-Host ''
Write-Host '关闭服务：直接关闭对应的两个终端窗口即可。' -ForegroundColor DarkGray
