package main

import (
	"fmt"
	"go-phone-agent/agent"
	"go-phone-agent/model"
)

func main() {
	// 创建模型配置
	modelConfig := &model.ModelConfig{
		BaseURL:   "https://open.bigmodel.cn/api/paas/v4",
		ModelName: "autoglm-phone",
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
