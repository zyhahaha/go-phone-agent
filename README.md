# Go Phone Agent

基于 Go 语言实现的开源手机自动化智能体框架，采用双模型架构（DeepSeek + 视觉模型），能够理解手机屏幕内容并通过 ADB 自动化操作完成用户任务。

## 核心原理

### 双模型架构工作流程

```
用户指令 → DeepSeek 调度器 → 任务规划 → 操作决策
                              ↓
                        判断是否需要视觉
                    ┌─────────┴─────────┐
                    ↓                   ↓
              无需视觉              需要视觉
              (Launch/Type)       (Tap/Swipe)
                    ↓                   ↓
              直接执行操作      视觉模型解析
                                    ↓
                              返回坐标 → 执行操作
```

### 技术栈

- **ADB (Android Debug Bridge)**: 底层设备控制
- **Go 语言**: 高性能、低内存占用
- **DeepSeek**: 任务规划和逻辑推理
- **视觉模型**: 屏幕识别和坐标解析
- **OpenAI 兼容 API**: 模型调用接口

### 架构优势

- 🔥 **智能规划**：DeepSeek 强大的逻辑推理能力
- ⚡ **性能优化**：减少视觉模型调用次数
- 🎯 **职责分离**：规划与执行分离，各司其职
- 💰 **成本控制**：按需调用视觉模型，降低成本
- 🛡️ **容错能力强**：基于屏幕元素而非应用名称决策
- 🔍 **识别准确**：视觉模型专注坐标识别，不受逻辑干扰

## 功能特性

### 支持的操作

| 操作 | 说明 |
|------|------|
| Launch | 启动应用 |
| Tap | 点击屏幕 |
| Type | 输入文本 |
| Swipe | 滑动屏幕 |
| Back | 返回上一页 |
| Home | 返回桌面 |
| DoubleTap | 双击 |
| Long Press | 长按 |
| Wait | 等待 |

## 快速开始

### 1. 环境准备

#### 在电脑上运行

安装 ADB:

```bash
# macOS
brew install android-platform-tools

# Linux
sudo apt install android-tools-adb

# Windows
# 下载并添加到 PATH: https://developer.android.com/tools/releases/platform-tools
```

连接设备:

```bash
adb devices
```

#### 在手机上独立运行

支持在 Android 手机上直接运行程序,无需依赖电脑。

**依赖软件:**

- **Termux**: Android 终端模拟器,提供 Linux 环境
  - 下载地址: https://github.com/termux/termux-app/releases

- **LADB**: Android 版本的 ADB 工具
  - 下载地址: https://github.com/yurikodesu/ladb-builds/releases
  - 注意: 需要在手机上启用 USB 调试或无线调试（Android 10及以下需要使用电脑开启无线调试）

**配置步骤:**

1. 安装 Termux 和 LADB
2. 在 Termux 中安装 Go:
```bash
pkg update
# 安装 Go 语言
pkg install golang

# 验证安装
go version

# 安装 ADB 工具
pkg install android-tools

# 连接到本地 ADB 服务器
adb connect localhost:5555

# 验证连接
adb devices
```
3. 克隆项目并编译:
```bash
git clone git@github.com:zyhahaha/go-phone-agent.git
cd go-phone-agent
go mod download
go build -o phone-agent cmd/main.go
```
4. 运行程序:
```bash
./phone-agent --base-url https://open.bigmodel.cn/api/paas/v4 --model "autoglm-phone" --apikey "key" "打开微信"
```

**注意:** 在手机上运行时,需要使用 LADB 提供的 ADB 服务,连接到本地设备。

### 2. 编译项目

```bash
cd go-phone-agent
go env -w GOPROXY=https://goproxy.cn,direct
go mod download
go build -o phone-agent cmd/main.go
```

```ps
$env:GOOS="windows"; $env:GOARCH="amd64"; go build -ldflags="-s -w" -o phone-agent-windows-amd64.exe cmd/main.go
```

### 3. 运行示例

#### 原始模式（单一模型）

```bash
# 单次任务
./phone-agent --base-url https://open.bigmodel.cn/api/paas/v4 --apikey your-api-key --model autoglm-phone "打开微信发消息给文件传输助手:测试"

# 交互模式
./phone-agent --base-url https://open.bigmodel.cn/api/paas/v4 --apikey your-api-key --model autoglm-phone
```

#### 调度器模式（双模型 - 推荐）

```bash
# 启用调度器模式
./phone-agent \
  --scheduler \
  --scheduler-key your-deepseek-api-key \
  --vision-key your-autoglm-api-key \
  "打开微信发消息给文件传输助手:测试"

# 交互模式（自动使用调度器）
./phone-agent \
  --scheduler \
  --scheduler-key your-deepseek-api-key \
  --vision-key your-autoglm-api-key
```

**说明：** 双模型架构下，DeepSeek 负责任务规划和逻辑判断，视觉模型只负责屏幕解析和坐标识别。

## 高级用法

### 命令行选项

```bash
./phone-agent --scheduler-key <KEY> --vision-key <KEY> [OPTIONS] [TASK]
```

**模型参数：**
- `--scheduler-url`: 调度器 API 地址 (默认: `https://api.deepseek.com`)
- `--scheduler-key`: 调度器 API 密钥 (DeepSeek)
- `--scheduler-model`: 调度器模型名称 (默认: `deepseek-chat`)
- `--vision-url`: 视觉模型 API 地址 (默认: `https://open.bigmodel.cn/api/paas/v4`)
- `--vision-key`: 视觉模型 API 密钥 (视觉模型)
- `--vision-model`: 视觉模型名称 (默认: `autoglm-phone`)

**通用参数：**
- `--device-id`: ADB 设备 ID (不指定则自动检测)
- `--max-steps`: 每个任务最大步数 (默认: 100)
- `--quiet`: 抑制详细输出
- `--list-apps`: 列出支持的应用并退出
- `--list-devices`: 列出已连接的设备并退出
- `--connect <ADDRESS>`: 连接远程设备 (例如: `192.168.1.100:5555`)
- `--disconnect <ADDRESS>`: 断开远程设备

### 多设备支持

```bash
# 连接远程设备
adb connect 192.168.1.100:5555

# 指定设备运行
./phone-agent --device-id 192.168.1.100:5555 "打开抖音"
```

### 使用 API Key

```bash
./phone-agent \
  --base-url https://open.bigmodel.cn/api/paas/v4 \
  --apikey your-api-key \
  --model autoglm-phone \
  "打开微信"
```

## 代码示例

### 基础使用（双模型架构）

```go
package main

import (
    "go-phone-agent/agent"
    "go-phone-agent/model"
)

func main() {
    // 创建调度器配置（DeepSeek + 视觉模型）
    schedulerConfig := &model.SchedulerConfig{
        Scheduler: &model.ModelConfig{
            BaseURL:   "https://api.deepseek.com",
            ModelName: "deepseek-chat",
            APIKey:    "YOUR_DEEPSEEK_API_KEY",
        },
        Vision: &model.ModelConfig{
            BaseURL:   "https://open.bigmodel.cn/api/paas/v4",
            ModelName: "autoglm-phone",
            APIKey:    "YOUR_VISION_API_KEY",
        },
    }

    // 创建 Agent
    phoneAgent := agent.NewPhoneAgentWithScheduler(schedulerConfig, &agent.AgentConfig{
        MaxSteps: 100,
        DeviceID: "",
    }, nil, nil)

    // 执行任务
    result := phoneAgent.Run("打开淘宝搜索iPhone")
    println(result)
}
```

### 交互模式

```go
package main

import (
    "fmt"
    "go-phone-agent/agent"
    "go-phone-agent/model"
)

func main() {
    schedulerConfig := &model.SchedulerConfig{
        Scheduler: &model.ModelConfig{
            BaseURL:   "https://api.deepseek.com",
            ModelName: "deepseek-chat",
            APIKey:    "YOUR_DEEPSEEK_API_KEY",
        },
        Vision: &model.ModelConfig{
            BaseURL:   "https://open.bigmodel.cn/api/paas/v4",
            ModelName: "autoglm-phone",
            APIKey:    "YOUR_VISION_API_KEY",
        },
    }

    phoneAgent := agent.NewPhoneAgentWithScheduler(schedulerConfig, &agent.AgentConfig{
        MaxSteps: 100,
        Verbose:  true,
    }, nil, nil)

    fmt.Println("输入任务 (输入 'quit' 退出):")
    for {
        var task string
        fmt.Print("> ")
        fmt.Scanln(&task)

        if task == "quit" {
            break
        }

        result := phoneAgent.Run(task)
        fmt.Printf("结果: %s\n", result)
        phoneAgent.Reset()
    }
}
```

### 自定义回调

```go
confirmationCallback := func(message string) bool {
    fmt.Printf("确认操作: %s (Y/N): ", message)
    var response string
    fmt.Scanln(&response)
    return strings.ToUpper(response) == "Y"
}

takeoverCallback := func(message string) {
    fmt.Printf("需要人工干预: %s\n", message)
    fmt.Println("完成后按回车继续...")
    fmt.Scanln(new(string))
}

schedulerConfig := &model.SchedulerConfig{
    Scheduler: &model.ModelConfig{
        BaseURL:   "https://api.deepseek.com",
        ModelName: "deepseek-chat",
        APIKey:    "YOUR_DEEPSEEK_API_KEY",
    },
    Vision: &model.ModelConfig{
        BaseURL:   "https://open.bigmodel.cn/api/paas/v4",
        ModelName: "autoglm-phone",
        APIKey:    "YOUR_VISION_API_KEY",
    },
}

phoneAgent := agent.NewPhoneAgentWithScheduler(
    schedulerConfig,
    &agent.AgentConfig{},
    confirmationCallback,
    takeoverCallback,
)
```

## 项目结构

```
go-phone-agent/
├── cmd/main.go              # 命令行入口
├── agent/                   # Agent 核心逻辑
│   ├── agent.go             # 主 Agent 实现（双模型架构）
│   └── config.go            # Agent 配置
├── adb/                     # ADB 操作封装
│   ├── device.go            # 设备控制函数
│   ├── input.go             # 输入处理
│   └── screenshot.go        # 截图函数
├── model/                   # 模型客户端
│   ├── client.go            # API 客户端
│   ├── scheduler.go         # DeepSeek 调度器实现
│   └── config.go            # 模型配置
├── actions/                 # 动作处理器
│   └── handler.go           # 执行各种动作
├── config/                  # 配置文件
│   └── apps.go              # 应用包名映射
├── examples/                # 使用示例
│   ├── basic_usage.go       # 基础使用
│   ├── interactive_mode.go  # 交互模式
│   ├── custom_callbacks.go  # 自定义回调
│   ├── step_by_step.go      # 单步调试
│   └── scheduler_mode.go    # 双模型示例
├── ARCHITECTURE.md          # 双模型架构详解
├── MODEL_CONFIG_GUIDE.md    # 模型配置最佳实践
└── README.md                # 项目文档
```

## 相关文档

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - 双模型架构详细说明
- **[MODEL_CONFIG_GUIDE.md](MODEL_CONFIG_GUIDE.md)** - 模型配置最佳实践和成本优化指南

## 依赖

- Go 1.21+
- ADB (Android Debug Bridge)
- AutoGLM-Phone 模型服务

## 许可证

MIT License

## 致谢

本项目基于 [Open-AutoGLM](https://github.com/zai-org/Open-AutoGLM) 项目重构实现。
