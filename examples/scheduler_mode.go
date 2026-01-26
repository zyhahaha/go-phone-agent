package main

import (
	"fmt"
	"go-phone-agent/agent"
	"go-phone-agent/model"
)

func main() {
	// 调度器模式示例：DeepSeek 负责任务规划，autoglm-phone 负责视觉解析
	schedulerConfig := &model.SchedulerConfig{
		Enabled: true,
		Scheduler: &model.ModelConfig{
			BaseURL:          "https://api.deepseek.com",
			APIKey:           "YOUR_DEEPSEEK_API_KEY",
			ModelName:        "deepseek-chat",
			MaxTokens:        2000,
			Temperature:      0.7,
			TopP:             0.9,
			FrequencyPenalty: 0.0,
		},
		Vision: &model.ModelConfig{
			BaseURL:          "https://open.bigmodel.cn/api/paas/v4",
			APIKey:           "YOUR_AUTOGLM_API_KEY",
			ModelName:        "autoglm-phone",
			MaxTokens:        3000,
			Temperature:      0.0,
			TopP:             0.85,
			FrequencyPenalty: 0.2,
		},
		VisionOnly: true,
	}

	agentConfig := &agent.AgentConfig{
		MaxSteps: 100,
		DeviceID: "",
		Verbose:  true,
	}

	// 创建带调度器的 Agent
	phoneAgent := agent.NewPhoneAgentWithScheduler(schedulerConfig, agentConfig, nil, nil)

	// 运行任务
	result := phoneAgent.Run("打开微信并发送消息给文件传输助手：测试")
	fmt.Printf("任务结果: %s\n", result)

	// 交互模式示例
	fmt.Println("\n=== 交互模式 ===")
	phoneAgent.Reset()

	tasks := []string{
		"打开抖音",
		"搜索：人工智能",
		"点击第一个视频",
	}

	for _, task := range tasks {
		fmt.Printf("\n执行任务: %s\n", task)
		result := phoneAgent.Run(task)
		fmt.Printf("结果: %s\n", result)
		phoneAgent.Reset()
	}
}
