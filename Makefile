# Go Phone Agent Makefile

.PHONY: all build clean test help

# 默认目标
all: build

# 编译当前平台
build:
	go build -o phone-agent cmd/main.go

# 编译 Linux ARM64 (常用)
build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -o phone-agent-linux-arm64 cmd/main.go
	@echo "✓ Linux ARM64 编译完成: phone-agent-linux-arm64"

# 编译 Linux ARM v7 (树莓派)
build-linux-armv7:
	GOOS=linux GOARCH=arm GOARM=7 go build -o phone-agent-linux-armv7 cmd/main.go
	@echo "✓ Linux ARM v7 编译完成: phone-agent-linux-armv7"

# 编译 Linux 所有架构
build-linux-all: build-linux-amd64 build-linux-arm64 build-linux-armv7

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -o phone-agent-linux-amd64 cmd/main.go

# 编译所有平台
build-all:
	@echo "编译所有平台..."
	@$(MAKE) build-linux-amd64
	@$(MAKE) build-linux-arm64
	@$(MAKE) build-linux-armv7
	GOOS=windows GOARCH=amd64 go build -o phone-agent-windows-amd64.exe cmd/main.go
	GOOS=darwin GOARCH=amd64 go build -o phone-agent-macos-amd64 cmd/main.go
	GOOS=darwin GOARCH=arm64 go build -o phone-agent-macos-arm64 cmd/main.go
	@echo "✓ 所有平台编译完成"

# 下载依赖
deps:
	go mod download
	go mod tidy

# 运行测试
test:
	go test -v ./...

# 清理构建文件
clean:
	rm -f phone-agent
	rm -f phone-agent-*
	rm -rf build/

# 显示帮助信息
help:
	@echo "Go Phone Agent - 可用命令:"
	@echo ""
	@echo "  make build            - 编译当前平台"
	@echo "  make build-linux-arm64- 编译 Linux ARM64 (推荐用于 ARM 设备)"
	@echo "  make build-linux-armv7- 编译 Linux ARM v7 (树莓派)"
	@echo "  make build-linux-all   - 编译所有 Linux 架构"
	@echo "  make build-all         - 编译所有平台"
	@echo "  make deps              - 下载依赖"
	@echo "  make test              - 运行测试"
	@echo "  make clean             - 清理构建文件"
	@echo "  make help              - 显示帮助信息"
