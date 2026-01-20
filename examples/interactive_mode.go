package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"go-phone-agent/agent"
	"go-phone-agent/model"
)

func main() {
	// 创建配置
	modelConfig := &model.ModelConfig{
		BaseURL:   "https://open.bigmodel.cn/api/paas/v4",
		ModelName: "autoglm-phone",
		APIKey:    "EMPTY",
	}

	agentConfig := &agent.AgentConfig{
		MaxSteps: 100,
		DeviceID: "",
		Lang:     "cn",
		Verbose:  true,
	}

	phoneAgent := agent.NewPhoneAgent(modelConfig, agentConfig, nil, nil)

	// 交互模式
	fmt.Println("=== Go Phone Agent - 交互模式 ===")
	fmt.Println("输入任务描述,输入 'quit' 退出\n")

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("任务: ")
		if !scanner.Scan() {
			break
		}

		task := strings.TrimSpace(scanner.Text())
		if task == "" {
			continue
		}

		if strings.ToLower(task) == "quit" || strings.ToLower(task) == "exit" {
			fmt.Println("再见!")
			break
		}

		fmt.Println()
		result := phoneAgent.Run(task)
		fmt.Printf("\n结果: %s\n", result)

		// 重置 Agent 状态
		phoneAgent.Reset()

		fmt.Println()
		fmt.Println(strings.Repeat("-", 50))
		fmt.Println()
	}
}
