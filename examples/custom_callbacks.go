package main

import (
	"fmt"
	"strings"

	"go-phone-agent/agent"
	"go-phone-agent/model"
)

func main() {
	// 自定义确认回调
	confirmationCallback := func(message string) bool {
		fmt.Printf("\n⚠️  敏感操作: %s\n", message)
		fmt.Print("确认执行? (Y/N): ")
		var response string
		fmt.Scanln(&response)
		return strings.ToUpper(response) == "Y"
	}

	// 自定义接管回调
	takeoverCallback := func(message string) {
		fmt.Printf("\n👤 人工接管: %s\n", message)
		fmt.Println("请手动完成操作...")
		fmt.Print("完成后按回车继续...")
		var discard string
		fmt.Scanln(&discard)
		fmt.Println("\n继续自动执行...")
	}

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

	// 创建 Agent,传入自定义回调
	phoneAgent := agent.NewPhoneAgent(modelConfig, agentConfig, confirmationCallback, takeoverCallback)

	// 执行任务
	result := phoneAgent.Run("打开淘宝,搜索iPhone")
	fmt.Printf("\n结果: %s\n", result)
}
