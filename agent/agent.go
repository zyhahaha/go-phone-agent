package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"go-phone-agent/actions"
	"go-phone-agent/adb"
	"go-phone-agent/model"
)

// PhoneAgent 手机自动化 Agent
type PhoneAgent struct {
	modelClient   *model.Client
	actionHandler *actions.ActionHandler
	config        *AgentConfig
	modelConfig   *model.ModelConfig
	context       []model.Message
	stepCount     int
}

// NewPhoneAgent 创建 PhoneAgent
func NewPhoneAgent(modelConfig *model.ModelConfig, agentConfig *AgentConfig, confirmationCallback func(string) bool, takeoverCallback func(string)) *PhoneAgent {
	if modelConfig == nil {
		modelConfig = model.DefaultModelConfig()
	}
	if agentConfig == nil {
		agentConfig = DefaultAgentConfig()
	}

	return &PhoneAgent{
		modelClient:   model.NewClient(modelConfig),
		actionHandler: actions.NewActionHandler(agentConfig.DeviceID, confirmationCallback, takeoverCallback),
		config:        agentConfig,
		modelConfig:   modelConfig,
		context:       []model.Message{},
		stepCount:     0,
	}
}

// Run 运行任务
func (a *PhoneAgent) Run(task string) string {
	a.context = []model.Message{}
	a.stepCount = 0

	// 第一步:发送用户任务
	result := a.executeStep(task, true)
	if result.Finished {
		return result.Message
	}

	// 循环执行直到完成或达到最大步数
	for a.stepCount < a.config.MaxSteps {
		result = a.executeStep("", false)
		if result.Finished {
			return result.Message
		}
	}

	return "Max steps reached"
}

// Step 执行单步
func (a *PhoneAgent) Step(task string) *StepResult {
	isFirst := len(a.context) == 0

	if isFirst && task == "" {
		return &StepResult{Success: false, Finished: true, Message: "Task is required for first step"}
	}

	return a.executeStep(task, isFirst)
}

// Reset 重置 Agent 状态
func (a *PhoneAgent) Reset() {
	a.context = []model.Message{}
	a.stepCount = 0
}

// executeStep 执行单步
func (a *PhoneAgent) executeStep(userPrompt string, isFirst bool) *StepResult {
	a.stepCount++

	// 截图
	screenshot, err := adb.GetScreenshot(a.config.DeviceID, 10)
	if err != nil && a.config.Verbose {
		fmt.Printf("Screenshot error: %v\n", err)
	}

	currentApp := adb.GetCurrentApp(a.config.DeviceID)

	// 构建消息
	if isFirst {
		// 系统消息
		systemPrompt := getSystemPrompt()
		a.context = append(a.context, model.CreateSystemMessage(systemPrompt))

		// 用户消息
		screenInfo := buildScreenInfo(currentApp)
		textContent := fmt.Sprintf("%s\n\n%s", userPrompt, screenInfo)

		a.context = append(a.context, model.CreateUserMessage(textContent, screenshot.Base64Data))
	} else {
		// 后续消息
		screenInfo := buildScreenInfo(currentApp)
		textContent := fmt.Sprintf("** Screen Info **\n\n%s", screenInfo)

		a.context = append(a.context, model.CreateUserMessage(textContent, screenshot.Base64Data))
	}

	// 获取模型响应
	var response *model.ModelResponse
	if a.config.Verbose {
		fmt.Println()
		fmt.Println("=" + strings.Repeat("=", 48))
		fmt.Println("💭 思考过程:")
		fmt.Println("-" + strings.Repeat("-", 48))
	}

	response, err = a.modelClient.Request(a.context)
	if err != nil {
		if a.config.Verbose {
			fmt.Printf("Model error: %v\n", err)
		}
		return &StepResult{
			Success:  false,
			Finished: true,
			Message:  fmt.Sprintf("Model error: %v", err),
		}
	}

	// 解析动作
	action, err := actions.ParseAction(response.Action)
	if err != nil && a.config.Verbose {
		fmt.Printf("Parse action error: %v\n", err)
		// 使用原始内容
		action = map[string]interface{}{
			"_metadata": "finish",
			"message":   response.Action,
		}
	}

	if a.config.Verbose {
		fmt.Println()
		fmt.Println("-" + strings.Repeat("-", 48))
		fmt.Println("🎯 执行动作:")
		actionJSON, _ := json.MarshalIndent(action, "", "  ")
		fmt.Println(string(actionJSON))
		fmt.Println("=" + strings.Repeat("=", 48))
		fmt.Println()
	}

	// 移除图片以节省空间
	a.context = removeImagesFromMessages(a.context)

	// 执行动作
	result, err := a.actionHandler.Execute(action, screenshot.Width, screenshot.Height)
	if err != nil && a.config.Verbose {
		fmt.Printf("Execute error: %v\n", err)
		// 创建完成动作
		action = map[string]interface{}{
			"_metadata": "finish",
			"message":   err.Error(),
		}
		result, _ = a.actionHandler.Execute(action, screenshot.Width, screenshot.Height)
	}

	// 添加助手响应到上下文
	assistantContent := fmt.Sprintf("<thinking>%s</thinking>\n<answer>%s</answer>", response.Thinking, response.Action)
	a.context = append(a.context, model.CreateAssistantMessage(assistantContent))

	// 检查是否完成
	finished := action["_metadata"] == "finish" || result.ShouldFinish

	if finished && a.config.Verbose {
		msg := result.Message
		if msg == "" {
			if m, ok := action["message"].(string); ok {
				msg = m
			}
		}
		if msg == "" {
			msg = "Done"
		}
		fmt.Println()
		fmt.Println("🎉 " + strings.Repeat("=", 48))
		fmt.Printf("✅ 任务完成: %s\n", msg)
		fmt.Println("=" + strings.Repeat("=", 48))
		fmt.Println()
	}

	return &StepResult{
		Success:  result.Success,
		Finished: finished,
		Action:   action,
		Thinking: response.Thinking,
		Message:  result.Message,
	}
}

// StepResult 步骤结果
type StepResult struct {
	Success  bool
	Finished bool
	Action   map[string]interface{}
	Thinking string
	Message  string
}

// 获取系统提示词
func getSystemPrompt() string {
	// 中文系统提示词
	return `你是一个智能的手机自动化助手,能够理解屏幕内容并通过执行相应操作帮助用户完成任务。
			可用操作:
			- Launch(app="应用名"): 启动指定应用
			- Tap(element=[x,y]): 点击指定坐标(0-1000范围)
			- Type(text="文本内容"): 输入文本
			- Swipe(start=[x1,y1], end=[x2,y2]): 从起点滑动到终点
			- Back(): 返回上一页
			- Home(): 返回桌面
			- Double Tap(element=[x,y]): 双击指定坐标
			- Long Press(element=[x,y]): 长按指定坐标
			- Wait(duration=1.0): 等待指定秒数
			- Take_over(message="说明"): 请求人工接管(用于登录、验证码等)

			完成任务的步骤:
			1. 分析当前屏幕截图
			2. 逐步思考需要做什么
			3. 输出你的思考过程
			4. 使用 do(action=..., ...) 执行相应操作
			5. 继续执行直到任务完成
			6. 完成后使用 finish(message="完成信息")

			输出格式示例:
			<answer>do(action="Launch", app="微信")</answer>

			注意事项:
			- 坐标范围为0-1000,表示相对位置
			- 对于敏感操作(如支付、删除等),请使用 Take_over 请求用户确认
			- 如果需要人工介入(如输入验证码),使用 Take_over
			- 在每一步后观察屏幕变化,调整后续操作
			- 最多执行100步,如果未完成请使用 finish 说明情况`
}

// buildScreenInfo 构建屏幕信息
func buildScreenInfo(currentApp string) string {
	info := map[string]string{
		"current_app": currentApp,
	}
	jsonData, _ := json.Marshal(info)
	return string(jsonData)
}

// removeImagesFromMessages 从消息中移除图片
func removeImagesFromMessages(messages []model.Message) []model.Message {
	for i := range messages {
		if content, ok := messages[i].Content.([]model.ImageContent); ok {
			textOnly := []model.ImageContent{}
			for _, item := range content {
				if item.Type == "text" {
					textOnly = append(textOnly, item)
				}
			}
			messages[i].Content = textOnly
		}
	}
	return messages
}
