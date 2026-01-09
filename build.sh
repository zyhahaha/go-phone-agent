#!/bin/bash

# Go Phone Agent 多平台构建脚本

echo "开始编译 Go Phone Agent..."

# 输出目录
OUTPUT_DIR="build"
mkdir -p $OUTPUT_DIR

# 编译函数
build() {
    GOOS=$1
    GOARCH=$2
    GOARM=$3
    OUTPUT=$4

    echo "编译: $GOOS/$GOARCH${GOARM:+/v$GOARM} -> $OUTPUT"

    if [ -n "$GOARM" ]; then
        GOOS=$GOOS GOARCH=$GOARCH GOARM=$GOARM go build -o $OUTPUT_DIR/$OUTPUT cmd/main.go
    else
        GOOS=$GOOS GOARCH=$GOARCH go build -o $OUTPUT_DIR/$OUTPUT cmd/main.go
    fi

    if [ $? -eq 0 ]; then
        echo "✓ 编译成功: $OUTPUT"
        # 显示文件大小
        ls -lh $OUTPUT_DIR/$OUTPUT | awk '{print "  文件大小: " $5}'
    else
        echo "✗ 编译失败: $OUTPUT"
    fi
    echo
}

# 编译各平台版本

# Linux
build linux amd64 "" phone-agent-linux-amd64
build linux arm64 "" phone-agent-linux-arm64
build linux arm 7 phone-agent-linux-armv7
build linux arm 6 phone-agent-linux-armv6
build linux arm 5 phone-agent-linux-armv5

# Windows
build windows amd64 "" phone-agent-windows-amd64.exe

# macOS
build darwin amd64 "" phone-agent-macos-amd64
build darwin arm64 "" phone-agent-macos-arm64

echo "========================================"
echo "编译完成! 输出目录: $OUTPUT_DIR"
echo ""
echo "常用平台:"
echo "  - Linux 64位: phone-agent-linux-amd64"
echo "  - Linux ARM64: phone-agent-linux-arm64"
echo "  - 树莓派: phone-agent-linux-armv7"
echo "========================================"
