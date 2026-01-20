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

	// 单步调试模式
	fmt.Println("=== 单步调试模式 ===")

	// 执行第一步
	task := "打开微信"
	fmt.Printf("任务: %s\n", task)
	stepResult := phoneAgent.Step(task)
	fmt.Printf("思考: %s\n", stepResult.Thinking)
	fmt.Printf("完成: %v\n", stepResult.Finished)

	// 继续执行
	for !stepResult.Finished && phoneAgent.StepCount() < 10 {
		fmt.Printf("\n执行第 %d 步...\n", phoneAgent.StepCount())
		stepResult = phoneAgent.Step("")
		fmt.Printf("思考: %s\n", stepResult.Thinking)
		fmt.Printf("完成: %v\n", stepResult.Finished)
	}

	fmt.Printf("\n最终结果: %s\n", stepResult.Message)
}
