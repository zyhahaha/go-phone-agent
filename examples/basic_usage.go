package main

import (
	"fmt"
	"go-phone-agent/agent"
	"go-phone-agent/model"
)

func main() {
	// 创建模型配置
	modelConfig := &model.ModelConfig{
		BaseURL:   "http://localhost:8000/v1",
		ModelName: "autoglm-phone-9b",
		APIKey:    "EMPTY",
	}

	// 创建 Agent 配置
	agentConfig := &agent.AgentConfig{
		MaxSteps: 100,
		DeviceID: "",
		Lang:     "cn",
		Verbose:  true,
	}

	// 创建 Agent
	phoneAgent := agent.NewPhoneAgent(modelConfig, agentConfig, nil, nil)

	// 执行任务
	result := phoneAgent.Run("打开微信,对文件传输助手发送消息:你好,Go版本!")

	fmt.Printf("\n结果: %s\n", result)
}
