# Go Phone Agent Windows PowerShell 构建脚本

Write-Host "开始编译 Go Phone Agent..." -ForegroundColor Green
Write-Host ""

# 创建输出目录
if (-not (Test-Path "build")) {
    New-Item -ItemType Directory -Path "build" | Out-Null
}

# 编译函数
function Build-Target {
    param(
        [string]$OS,
        [string]$ARCH,
        [string]$ARM_VERSION,
        [string]$OUTPUT
    )

    Write-Host "编译: $OS/$ARCH" -NoNewline
    if ($ARM_VERSION -ne "") {
        Write-Host " (ARM$ARM_VERSION)"
    } else {
        Write-Host ""
    }

    $env:GOOS = $OS
    $env:GOARCH = $ARCH
    if ($ARM_VERSION -ne "") {
        $env:GOARM = $ARM_VERSION
    } else {
        Remove-Item Env:\GOARM -ErrorAction SilentlyContinue
    }

    go build -o "build/$OUTPUT" cmd/main.go

    if ($LASTEXITCODE -eq 0) {
        Write-Host "  [成功] $OUTPUT" -ForegroundColor Green
        # 显示文件大小
        $size = (Get-Item "build/$OUTPUT").Length / 1KB
        Write-Host "  文件大小: $([math]::Round($size, 2)) KB"
    } else {
        Write-Host "  [失败] $OUTPUT" -ForegroundColor Red
    }
    Write-Host ""
}

# 编译各平台版本

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Linux 平台" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Build-Target -OS "linux" -ARCH "amd64" -OUTPUT "phone-agent-linux-amd64"
Build-Target -OS "linux" -ARCH "arm64" -OUTPUT "phone-agent-linux-arm64"
Build-Target -OS "linux" -ARCH "arm" -ARM_VERSION "7" -OUTPUT "phone-agent-linux-armv7"
Build-Target -OS "linux" -ARCH "arm" -ARM_VERSION "6" -OUTPUT "phone-agent-linux-armv6"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Windows 平台" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Build-Target -OS "windows" -ARCH "amd64" -OUTPUT "phone-agent-windows-amd64.exe"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "macOS 平台" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Build-Target -OS "darwin" -ARCH "amd64" -OUTPUT "phone-agent-macos-amd64"
Build-Target -OS "darwin" -ARCH "arm64" -OUTPUT "phone-agent-macos-arm64"

Write-Host "========================================" -ForegroundColor Green
Write-Host "编译完成! 输出目录: build\" -ForegroundColor Green
Write-Host ""
Write-Host "常用平台:" -ForegroundColor Yellow
Write-Host "  - Linux 64位:    phone-agent-linux-amd64"
Write-Host "  - Linux ARM64:  phone-agent-linux-arm64"
Write-Host "  - 树莓派:       phone-agent-linux-armv7"
Write-Host "========================================" -ForegroundColor Green
