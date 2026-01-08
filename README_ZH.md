# Go Phone Agent - 中文版

基于 Go 语言实现的开源手机自动化智能体框架,能够理解手机屏幕内容并通过 ADB 自动化操作完成用户任务。

## 核心原理

### 工作流程

```
用户指令 → ADB 截图 → 视觉模型分析 → 输出动作指令 → ADB 执行操作 → 循环直到任务完成
```

### 技术栈

- **ADB (Android Debug Bridge)**: 底层设备控制
- **Go 语言**: 高性能、低内存占用
- **视觉语言模型**: 屏幕理解和决策
- **OpenAI 兼容 API**: 模型调用接口

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
| Double Tap | 双击 |
| Long Press | 长按 |
| Wait | 等待 |

### 支持的应用

支持 50+ 主流中文应用,包括:
- 社交通讯: 微信、QQ、微博
- 电商购物: 淘宝、京东、拼多多
- 视频娱乐: 抖音、B站、爱奇艺
- 生活服务: 美团、大众点评、高德地图
- 等等...

## 快速开始

### 1. 环境准备

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

### 2. 编译项目

```bash
cd go-phone-agent
go mod download
go build -o phone-agent cmd/main.go
```

### 3. 运行示例

```bash
# 单次任务
./phone-agent --base-url http://localhost:8000/v1 --model autoglm-phone-9b "打开微信发消息给文件传输助手:测试"

# 交互模式
./phone-agent --base-url http://localhost:8000/v1
```

## 高级用法

### 多设备支持

```bash
# 连接远程设备
adb connect 192.168.1.100:5555

# 指定设备运行
./phone-agent --device-id 192.168.1.100:5555 "打开抖音"
```

## 许可证

MIT License

## 致谢

本项目基于 [Open-AutoGLM](https://github.com/zai-org/Open-AutoGLM) 项目重构实现。
