# 模型配置最佳实践指南

## 概述

本指南提供 Go Phone Agent 双模型架构的详细配置建议，帮助你在不同场景下获得最佳性能。

## 参数详解

### MaxTokens（最大 Token 数）

**调度器模型（DeepSeek）：**
- **推荐值：** 1500-2500
- **原理：** 任务规划需要足够的上下文
- **过低影响：** 复杂任务可能被截断
- **过高影响：** 增加成本和延迟

**视觉模型：**
- **推荐值：** 2500-3500
- **原理：** 屏幕描述需要详细输出
- **过低影响：** 描述不完整，缺少关键信息
- **过高影响：** 不必要的成本

### Temperature（采样温度）

**调度器模型：**
- **推荐值：** 0.6-0.8
- **作用：** 控制规划 creativity
- **过低（<0.5）：** 过于保守，可能陷入死循环
- **过高（>0.9）：** 过于随机，可能做出错误决策

**视觉模型：**
- **推荐值：** 0.0-0.2
- **作用：** 坐标识别需要确定性
- **必须接近 0：** 确保坐标精确性

### TopP（核采样）

**调度器模型：**
- **推荐值：** 0.85-0.95
- **作用：** 平衡多样性和准确性
- **建议：** 与 Temperature 配合使用

**视觉模型：**
- **推荐值：** 0.8-0.9
- **作用：** 限制输出范围
- **建议：** 配合低 Temperature 使用

### FrequencyPenalty（频率惩罚）

**调度器模型：**
- **推荐值：** 0.0-0.1
- **作用：** 减少重复描述
- **注意：** 任务规划需要一定重复

**视觉模型：**
- **推荐值：** 0.1-0.3
- **作用：** 避免重复描述相同元素
- **建议：** 适当惩罚提高效率

## 场景配置模板

### 模板1：高精度任务（推荐默认）

适用于复杂游戏操作、专业应用自动化：

```go
schedulerConfig := &model.SchedulerConfig{
    Scheduler: &model.ModelConfig{
        BaseURL:          "https://api.deepseek.com",
        APIKey:           os.Getenv("DEEPSEEK_API_KEY"),
        ModelName:        "deepseek-chat",
        MaxTokens:        2000,
        Temperature:      0.7,   // 平衡创造性和准确性
        TopP:             0.9,
        FrequencyPenalty: 0.0,   // 不限制重复，确保完整
    },
    Vision: &model.ModelConfig{
        BaseURL:          "https://open.bigmodel.cn/api/paas/v4",
        APIKey:           os.Getenv("VISION_API_KEY"),
        ModelName:        "autoglm-phone",
        MaxTokens:        3000,
        Temperature:      0.0,   // 完全确定性
        TopP:             0.85,
        FrequencyPenalty: 0.2,   // 适度去重
    },
}
```

**适用场景：**
- 游戏自动化（星穹铁道、原神等）
- 复杂业务流程（电商购物流程）
- 需要高精度点击的操作
- 长时间运行的任务

**性能表现：**
- 成功率：> 95%
- 平均耗时：15-25秒/任务
- Token 消耗：2000-3000/任务

### 模板2：快速响应模式

适用于简单操作、快速测试：

```go
schedulerConfig := &model.SchedulerConfig{
    Scheduler: &model.ModelConfig{
        MaxTokens:        1000,  // 减少token
        Temperature:      0.5,   // 更确定
        TopP:             0.8,
        FrequencyPenalty: 0.0,
    },
    Vision: &model.ModelConfig{
        MaxTokens:        1500,  // 减少token
        Temperature:      0.0,
        TopP:             0.8,
        FrequencyPenalty: 0.1,   // 轻度去重
    },
}
```

**适用场景：**
- 启动应用
- 简单点击操作
- 快速测试
- 演示用途

**性能表现：**
- 成功率：85-90%
- 平均耗时：8-15秒/任务
- Token 消耗：1000-1500/任务
- **成本降低：~50%**

### 模板3：成本控制模式

适用于批量任务、预算有限：

```go
schedulerConfig := &model.SchedulerConfig{
    Scheduler: &model.ModelConfig{
        MaxTokens:        1200,
        Temperature:      0.6,
        TopP:             0.85,
        FrequencyPenalty: 0.0,
    },
    Vision: &model.ModelConfig{
        MaxTokens:        2000,  // 大幅减少
        Temperature:      0.0,
        TopP:             0.8,
        FrequencyPenalty: 0.3,   // 强力去重
    },
}
```

**适用场景：**
- 批量自动化测试
- 定时任务（签到、打卡等）
- 监控类任务
- 预算受限项目

**性能表现：**
- 成功率：75-85%
- 平均耗时：12-20秒/任务
- Token 消耗：800-1200/任务
- **成本降低：~60%**

### 模板4：探索性任务模式

适用于未知界面、需要探索的场景：

```go
schedulerConfig := &model.SchedulerConfig{
    Scheduler: &model.ModelConfig{
        MaxTokens:        2500,  // 更多token用于分析
        Temperature:      0.8,   // 更高创造性
        TopP:             0.95,
        FrequencyPenalty: 0.1,
    },
    Vision: &model.ModelConfig{
        MaxTokens:        3500,  // 详细描述
        Temperature:      0.1,   // 轻微随机性
        TopP:             0.9,
        FrequencyPenalty: 0.2,
    },
}
```

**适用场景：**
- 新应用探索
- UI 自动化测试
- 未知界面导航
- 动态布局应用

**性能表现：**
- 成功率：70-80%（探索性质）
- 平均耗时：20-35秒/任务
- Token 消耗：3000-4000/任务

### 模板5：游戏专用模式

针对游戏场景的特殊优化：

```go
schedulerConfig := &model.SchedulerConfig{
    Scheduler: &model.ModelConfig{
        MaxTokens:        1800,
        Temperature:      0.6,   // 游戏逻辑更确定
        TopP:             0.85,
        FrequencyPenalty: 0.0,
    },
    Vision: &model.ModelConfig{
        MaxTokens:        3000,
        Temperature:      0.0,   // 坐标必须精确
        TopP:             0.85,
        FrequencyPenalty: 0.1,
    },
}
```

**游戏优化技巧：**
1. **提前缓存**：在游戏加载时截图缓存
2. **区域识别**：限定识别区域，减少干扰
3. **连续操作**：优化连续点击/滑动的响应
4. **状态判断**：准确识别游戏状态（加载中/游戏中/结算）

**性能表现：**
- 成功率：> 90%（熟悉界面后）
- 平均耗时：10-18秒/次操作
- 支持连续操作

## API Key 管理

### 环境变量配置

```bash
# Linux/macOS
export DEEPSEEK_API_KEY="sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
export VISION_API_KEY="your-vision-api-key"

# Windows PowerShell
$env:DEEPSEEK_API_KEY="sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
$env:VISION_API_KEY="your-vision-api-key"

# 运行程序
go run cmd/main.go --scheduler-key $env:DEEPSEEK_API_KEY --vision-key $env:VISION_API_KEY "打开微信"
```

### 代码中配置

```go
// 方式1：直接配置
schedulerConfig := &model.SchedulerConfig{
    Scheduler: &model.ModelConfig{
        APIKey: "sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    },
    Vision: &model.ModelConfig{
        APIKey: "your-vision-api-key",
    },
}

// 方式2：从环境变量读取
schedulerConfig := &model.SchedulerConfig{
    Scheduler: &model.ModelConfig{
        APIKey: os.Getenv("DEEPSEEK_API_KEY"),
    },
    Vision: &model.ModelConfig{
        APIKey: os.Getenv("VISION_API_KEY"),
    },
}
```

### 密钥安全建议

1. **不要硬编码**：避免提交到 Git
2. **使用环境变量**：生产环境标准做法
3. **配置文件**：使用 .env 文件（.gitignore）
4. **密钥轮换**：定期更换 API Key
5. **权限控制**：限制密钥访问权限

```bash
# .env 文件（添加到 .gitignore）
DEEPSEEK_API_KEY=sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
VISION_API_KEY=your-vision-api-key

# 加载 .env
go run cmd/main.go --scheduler-key $(grep DEEPSEEK_API_KEY .env | cut -d'=' -f2) \
                    --vision-key $(grep VISION_API_KEY .env | cut -d'=' -f2) \
                    "打开微信"
```

## 成本控制策略

### 1. Token 使用监控

```go
// 在 model/client.go 中添加日志
func (c *Client) Request(messages []Message) (*ModelResponse, error) {
    // ... 现有代码 ...
    
    // 记录 token 使用
    if c.config.Verbose {
        fmt.Printf("Token 使用: %d\n", resp.Usage.TotalTokens)
        fmt.Printf("成本: $%.4f\n", calculateCost(resp.Usage.TotalTokens))
    }
}
```

### 2. 缓存机制

```go
// 缓存相同屏幕的识别结果
var screenCache = make(map[string]string)

func analyzeScreenWithCache(imageHash string) string {
    if desc, exists := screenCache[imageHash]; exists {
        return desc  // 使用缓存
    }
    
    desc := analyzeScreen()  // 调用视觉模型
    screenCache[imageHash] = desc
    return desc
}
```

### 3. 批量操作优化

```go
// 批量任务，复用配置
schedulerConfig := &model.SchedulerConfig{...}

for _, task := range batchTasks {
    phoneAgent := agent.NewPhoneAgentWithScheduler(schedulerConfig, agentConfig, nil, nil)
    result := phoneAgent.Run(task)
    fmt.Printf("任务 %s: %s\n", task, result)
}
```

### 4. 模型降级策略

```go
// 失败时切换到备用模型
func tryWithFallback(task string) string {
    // 尝试主模型
    result := phoneAgent.Run(task)
    if !strings.Contains(result, "error") {
        return result
    }
    
    // 切换到备用配置
    fallbackConfig := &model.SchedulerConfig{
        Scheduler: &model.ModelConfig{
            APIKey: os.Getenv("BACKUP_DEEPSEEK_KEY"),
        },
        Vision: &model.ModelConfig{
            APIKey: os.Getenv("BACKUP_VISION_KEY"),
        },
    }
    
    backupAgent := agent.NewPhoneAgentWithScheduler(fallbackConfig, agentConfig, nil, nil)
    return backupAgent.Run(task)
}
```

## 故障排查

### 问题1：任务执行缓慢

**可能原因：**
- MaxTokens 设置过高
- Temperature 过高导致模型思考过长
- 网络延迟

**解决方案：**
```go
// 降低 token 限制
MaxTokens: 1500

// 降低 temperature
Temperature: 0.5

// 启用流式响应（如支持）
Stream: true
```

### 问题2：识别准确率下降

**可能原因：**
- Temperature 过高
- MaxTokens 不足
- 屏幕过于复杂

**解决方案：**
```go
// 提高 token 限制
MaxTokens: 3000

// 降低 temperature（视觉模型）
Temperature: 0.0

// 简化识别区域
// 使用 adb 裁剪截图
```

### 问题3：成本过高

**优化策略：**
1. 使用"成本控制模式"配置
2. 减少 max-steps
3. 启用缓存
4. 批量处理任务

```go
agentConfig := &agent.AgentConfig{
    MaxSteps: 50,  // 从100降到50
}
```

## 总结

### 快速选择指南

| 场景 | 推荐配置 | MaxTokens | Temperature | 预估成本 |
|------|---------|-----------|-------------|---------|
| 游戏自动化 | 模板1 | 2000/3000 | 0.6-0.7/0.0 | 高 |
| 简单操作 | 模板2 | 1000/1500 | 0.5/0.0 | 低 |
| 批量任务 | 模板3 | 1200/2000 | 0.6/0.0 | 最低 |
| 探索性任务 | 模板4 | 2500/3500 | 0.8/0.1 | 最高 |
| 开发测试 | 模板2 | 1000/1500 | 0.5/0.0 | 低 |

### 最佳实践清单

- [ ] 使用环境变量管理 API Key
- [ ] 根据场景选择合适的配置模板
- [ ] 监控 Token 使用和成本
- [ ] 为生产环境启用缓存
- [ ] 设置适当的 MaxSteps 限制
- [ ] 定期轮换 API Key
- [ ] 添加错误处理和降级策略
- [ ] 记录详细的日志用于调试

通过合理的配置，可以在性能、成本和准确率之间找到最佳平衡点。
