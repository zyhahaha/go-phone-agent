@echo off
REM Go Phone Agent Windows 构建脚本

echo 开始编译 Go Phone Agent...
echo.

REM 创建输出目录
if not exist build mkdir build

REM 编译 Linux ARM64
echo 编译: Linux ARM64
set GOOS=linux
set GOARCH=arm64
go build -o build/phone-agent-linux-arm64 cmd/main.go
if %errorlevel% equ 0 (
    echo [成功] phone-agent-linux-arm64
) else (
    echo [失败] phone-agent-linux-arm64
)
echo.

REM 编译 Linux ARM v7
echo 编译: Linux ARM v7
set GOOS=linux
set GOARCH=arm
set GOARM=7
go build -o build/phone-agent-linux-armv7 cmd/main.go
if %errorlevel% equ 0 (
    echo [成功] phone-agent-linux-armv7
) else (
    echo [失败] phone-agent-linux-armv7
)
echo.

REM 编译 Linux AMD64
echo 编译: Linux AMD64
set GOOS=linux
set GOARCH=amd64
go build -o build/phone-agent-linux-amd64 cmd/main.go
if %errorlevel% equ 0 (
    echo [成功] phone-agent-linux-amd64
) else (
    echo [失败] phone-agent-linux-amd64
)
echo.

REM 编译 Windows
echo 编译: Windows AMD64
set GOOS=windows
set GOARCH=amd64
go build -o build/phone-agent-windows-amd64.exe cmd/main.go
if %errorlevel% equ 0 (
    echo [成功] phone-agent-windows-amd64.exe
) else (
    echo [失败] phone-agent-windows-amd64.exe
)
echo.

echo ========================================
echo 编译完成! 输出目录: build\
echo.
echo 常用平台:
echo   - Linux 64位: phone-agent-linux-amd64
echo   - Linux ARM64: phone-agent-linux-arm64
echo   - 树莓派: phone-agent-linux-armv7
echo ========================================
pause
