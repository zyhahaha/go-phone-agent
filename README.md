# Go Phone Agent

使用 Go 语言实现的手机自动化智能体,基于 ADB 和视觉语言模型。

## 功能特性

- 基于 ADB 的手机控制
- 集成视觉语言模型 (AutoGLM-Phone)
- 支持多种操作:点击、滑动、输入文本、启动应用等
- 支持 WiFi 远程连接
- 支持多设备管理

## 安装

```bash
cd go-phone-agent
go mod download
go build -o phone-agent cmd/main.go
```

## 使用

### 命令行模式

```bash
# 基础使用
./phone-agent --base-url http://localhost:8000/v1 --model autoglm-phone-9b "打开抖音搜索美食"

# 使用 API Key
go run ./cmd/main.go --base-url https://open.bigmodel.cn/api/paas/v4 --apikey your-key --model autoglm-phone "打开微信"

# 交互模式
./phone-agent --base-url http://localhost:8000/v1
```

### Go 代码调用

```go
package main

import (
    "go-phone-agent/agent"
    "go-phone-agent/model"
)

func main() {
    config := &model.ModelConfig{
        BaseURL:   "http://localhost:8000/v1",
        ModelName: "autoglm-phone-9b",
    }

    phoneAgent := agent.NewPhoneAgent(config, &agent.AgentConfig{
        MaxSteps: 100,
        DeviceID: "",
    })

    result := phoneAgent.Run("打开淘宝搜索iPhone")
    println(result)
}
```

## 架构

```
go-phone-agent/
├── cmd/main.go          # 命令行入口
├── agent/              # Agent 核心逻辑
├── adb/                # ADB 操作封装
├── model/              # 模型客户端
├── config/             # 配置文件
└── actions/            # 动作处理器
```

## 依赖

- Go 1.21+
- ADB (Android Debug Bridge)
- AutoGLM-Phone 模型服务
