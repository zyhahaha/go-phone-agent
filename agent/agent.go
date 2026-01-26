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
	modelClient     *model.Client
	actionHandler   *actions.ActionHandler
	config          *AgentConfig
	modelConfig     *model.ModelConfig
	scheduler       *model.SchedulerDeepSeek
	schedulerConfig *model.SchedulerConfig
	context         []model.Message
	stepCount       int
	actionHistory   []model.ActionHistory
	currentTask     string // 当前任务（调度器模式使用）
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
		actionHistory: []model.ActionHistory{},
		currentTask:   "",
	}
}

// NewPhoneAgentWithScheduler 创建带调度器的 PhoneAgent
func NewPhoneAgentWithScheduler(schedulerConfig *model.SchedulerConfig, agentConfig *AgentConfig, confirmationCallback func(string) bool, takeoverCallback func(string)) *PhoneAgent {
	if schedulerConfig == nil {
		schedulerConfig = model.DefaultSchedulerConfig()
	}
	if agentConfig == nil {
		agentConfig = DefaultAgentConfig()
	}

	return &PhoneAgent{
		modelClient:     model.NewClient(schedulerConfig.Vision),
		actionHandler:   actions.NewActionHandler(agentConfig.DeviceID, confirmationCallback, takeoverCallback),
		config:          agentConfig,
		modelConfig:     schedulerConfig.Vision,
		scheduler:       model.NewSchedulerDeepSeek(schedulerConfig.Scheduler),
		schedulerConfig: schedulerConfig,
		context:         []model.Message{},
		stepCount:       0,
		actionHistory:   []model.ActionHistory{},
		currentTask:     "",
	}
}

// Run 运行任务
func (a *PhoneAgent) Run(task string) string {
	a.context = []model.Message{}
	a.stepCount = 0
	a.currentTask = task // 保存当前任务

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

	// 如果是第一步，保存任务
	if isFirst && task != "" {
		a.currentTask = task
	}

	return a.executeStep(task, isFirst)
}

// Reset 重置 Agent 状态
func (a *PhoneAgent) Reset() {
	a.context = []model.Message{}
	a.stepCount = 0
	a.actionHistory = []model.ActionHistory{}
	a.currentTask = ""
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
	screenInfo := buildScreenInfo(currentApp)

	var action map[string]interface{}
	var thinking string
	var execErr error

	// 判断是否使用调度器模式
	if a.scheduler != nil && a.schedulerConfig != nil && a.schedulerConfig.Enabled {
		// 调度器模式：DeepSeek 规划，autoglm-phone 执行
		action, thinking, execErr = a.executeWithScheduler(userPrompt, screenInfo, screenshot)
	} else {
		// 原始模式：autoglm-phone 直接处理
		action, thinking, execErr = a.executeWithVisionModel(userPrompt, screenInfo, screenshot, isFirst)
	}

	if execErr != nil {
		if a.config.Verbose {
			fmt.Printf("Model error: %v\n", execErr)
		}
		return &StepResult{
			Success:  false,
			Finished: true,
			Message:  fmt.Sprintf("Model error: %v", execErr),
		}
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

	// 记录操作历史
	actionStr := ""
	if actionType, ok := action["action"].(string); ok {
		actionStr = actionType
	} else if actionType, ok := action["_metadata"].(string); ok {
		actionStr = actionType
	}
	reasonStr := thinking
	if len(thinking) > 100 {
		reasonStr = thinking[:100] + "..."
	}
	a.actionHistory = append(a.actionHistory, model.ActionHistory{
		Action:  actionStr,
		Reason:  reasonStr,
		Success: result.Success,
	})

	// 添加助手响应到上下文（仅原始模式）
	if a.scheduler == nil {
		assistantContent := fmt.Sprintf("<thinking>%s</thinking>\n<answer>%s</answer>", thinking, fmt.Sprintf("%v", action))
		a.context = append(a.context, model.CreateAssistantMessage(assistantContent))
	}

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
		fmt.Printf("✅ 任务完成: %s\n", msg)
	}

	return &StepResult{
		Success:  result.Success,
		Finished: finished,
		Action:   action,
		Thinking: thinking,
		Message:  result.Message,
	}
}

// executeWithScheduler 使用调度器模式执行
func (a *PhoneAgent) executeWithScheduler(userPrompt string, screenInfo string, screenshot *adb.Screenshot) (map[string]interface{}, string, error) {
	// 使用保存的当前任务
	task := a.currentTask

	// 如果是第一步且 userPrompt 不为空，更新任务
	if a.stepCount == 1 && userPrompt != "" {
		task = userPrompt
		a.currentTask = userPrompt
	}

	// 第一步：先调用视觉模型获取屏幕描述
	screenDescription := ""
	screenDesc, err := a.analyzeScreen(screenInfo, screenshot)
	if err != nil {
		screenDescription = "屏幕分析失败"
	} else {
		screenDescription = screenDesc
	}

	// 打印视觉模型 → DeepSeek 的交互内容
	if a.config.Verbose {
		fmt.Println()
		fmt.Println("📤 autoglm-phone → DeepSeek (屏幕描述):")
		fmt.Printf("%s\n", screenDescription)
		fmt.Println()
	}

	// 第二步：调用 DeepSeek 调度器，基于屏幕描述做决策
	plan, err := a.scheduler.PlanStep(task, screenDescription, a.stepCount, a.config.MaxSteps, a.actionHistory)
	if err != nil {
		return nil, "", err
	}

	// 打印 DeepSeek → autoglm-phone 的交互内容
	if a.config.Verbose {
		fmt.Println("📥 DeepSeek → autoglm-phone (操作指令):")
		fmt.Printf("操作类型: %s\n", plan.ActionType)
		fmt.Printf("操作原因: %s\n", plan.Reason)
		if len(plan.Parameters) > 0 {
			params, _ := json.MarshalIndent(plan.Parameters, "  ", "")
			fmt.Printf("操作参数: %s\n", string(params))
		}
		fmt.Println()
	}

	// 检查是否完成
	if plan.Finished || plan.ActionType == "finish" {
		return map[string]interface{}{
			"_metadata": "finish",
			"message":   plan.Reason,
		}, plan.Thought, nil
	}

	// 根据计划构建操作
	var action map[string]interface{}

	// 不需要视觉解析的操作
	if plan.ActionType == "Launch" {
		appName := ""
		if app, ok := plan.Parameters["app"].(string); ok {
			appName = app
		}
		action = map[string]interface{}{
			"action":    "Launch",
			"app":       appName,
			"_metadata": "do",
		}
		return action, plan.Thought, nil
	}

	if plan.ActionType == "Type" {
		text := ""
		if t, ok := plan.Parameters["text"].(string); ok {
			text = t
		}
		action = map[string]interface{}{
			"action":    "Type",
			"text":      text,
			"_metadata": "do",
		}
		return action, plan.Thought, nil
	}

	if plan.ActionType == "Back" {
		action = map[string]interface{}{
			"action":    "Back",
			"_metadata": "do",
		}
		return action, plan.Thought, nil
	}

	if plan.ActionType == "Home" {
		action = map[string]interface{}{
			"action":    "Home",
			"_metadata": "do",
		}
		return action, plan.Thought, nil
	}

	if plan.ActionType == "Wait" {
		duration := 1.0
		if d, ok := plan.Parameters["duration"].(float64); ok {
			duration = d
		}
		action = map[string]interface{}{
			"action":    "Wait",
			"duration":  duration,
			"_metadata": "do",
		}
		return action, plan.Thought, nil
	}

	// 需要视觉解析的操作（Tap, Swipe, DoubleTap, LongPress）
	// 构建视觉模型的系统提示（仅获取坐标）
	visionPrompt := a.getVisionPrompt(plan)
	visionContext := []model.Message{
		model.CreateSystemMessage(visionPrompt),
		model.CreateUserMessage(screenInfo+"\n\n请分析屏幕并返回操作坐标。", screenshot.Base64Data),
	}

	// 调用视觉模型获取坐标
	response, err := a.modelClient.Request(visionContext)
	if err != nil {
		return nil, "", err
	}

	// 打印视觉模型的原始响应
	if a.config.Verbose {
		fmt.Println("📤 autoglm-phone → DeepSeek (坐标响应):")
		fmt.Printf("%s\n", response.Action)
		fmt.Println()
	}

	// 解析视觉模型的响应（纯坐标格式）
	coordinates, err := parseVisionCoordinates(response.Action, a.config.Verbose)
	if err != nil {
		return nil, "", err
	}

	// 构建完整的操作：DeepSeek 的操作类型 + 视觉模型的坐标
	visionAction := map[string]interface{}{
		"action":    plan.ActionType,
		"_metadata": "do",
	}

	fmt.Println(visionAction)
	fmt.Println(coordinates)

	// 根据操作类型添加坐标
	switch plan.ActionType {
	case "Tap", "DoubleTap", "LongPress":
		if len(coordinates) == 0 {
			return nil, "", fmt.Errorf("未返回任何坐标")
		}
		visionAction["element"] = coordinates[0]
	case "Swipe":
		if len(coordinates) == 0 {
			return nil, "", fmt.Errorf("未返回任何坐标")
		}
		// 如果只返回了一个坐标，根据描述推断另一个坐标
		if len(coordinates) == 1 {
			startCoord := coordinates[0]
			var endCoord []float64

			// 根据描述推断滑动方向
			reason := strings.ToLower(plan.Reason)
			if strings.Contains(reason, "从右向左") || strings.Contains(reason, "向左滑") {
				// 从右向左：终点 x 减小
				endCoord = []float64{startCoord[0] - 300, startCoord[1]}
			} else if strings.Contains(reason, "从左向右") || strings.Contains(reason, "向右滑") {
				// 从左向右：终点 x 增大
				endCoord = []float64{startCoord[0] + 300, startCoord[1]}
			} else if strings.Contains(reason, "从下往上") || strings.Contains(reason, "向上滑") {
				// 从下往上：终点 y 减小
				endCoord = []float64{startCoord[0], startCoord[1] - 300}
			} else if strings.Contains(reason, "从上往下") || strings.Contains(reason, "向下滑") {
				// 从上往下：终点 y 增大
				endCoord = []float64{startCoord[0], startCoord[1] + 300}
			} else {
				// 默认：从右向左滑动
				endCoord = []float64{startCoord[0] - 300, startCoord[1]}
			}

			// 确保坐标在有效范围内
			for i := 0; i < 2; i++ {
				if endCoord[i] < 0 {
					endCoord[i] = 0
				} else if endCoord[i] > 1000 {
					endCoord[i] = 1000
				}
			}

			coordinates = append(coordinates, endCoord)
		}
		visionAction["start"] = coordinates[0]
		visionAction["end"] = coordinates[1]
	}

	return visionAction, plan.Thought, nil
}

// analyzeScreen 使用视觉模型分析屏幕，返回屏幕描述
func (a *PhoneAgent) analyzeScreen(screenInfo string, screenshot *adb.Screenshot) (string, error) {
	// 构建屏幕分析的提示词
	screenAnalysisPrompt := `你是一个屏幕内容分析助手。请仔细分析屏幕截图，用简洁的语言描述屏幕上显示的内容。

描述要点：
1. 当前应用名称（如果顶部有应用名或图标）
2. 屏幕上显示的主要内容
3. 可见的按钮、输入框、图标等关键元素
4. 任何弹出窗口、对话框、提示信息等
5. 当前页面的状态（如：列表页、详情页、设置页等）

请用简洁的中文描述，不要超过200字。`

	visionContext := []model.Message{
		model.CreateSystemMessage(screenAnalysisPrompt),
		model.CreateUserMessage(screenInfo, screenshot.Base64Data),
	}

	response, err := a.modelClient.Request(visionContext)
	if err != nil {
		return "", err
	}

	// 屏幕分析应该返回纯文本，直接使用原始响应内容
	// 不要经过 parseResponse 解析，避免被误解析为 finish 格式
	return response.RawContent, nil
}

// executeWithVisionModel 使用原始模式执行
func (a *PhoneAgent) executeWithVisionModel(userPrompt string, screenInfo string, screenshot *adb.Screenshot, isFirst bool) (map[string]interface{}, string, error) {
	// 构建消息
	if isFirst {
		// 系统消息
		systemPrompt := getSystemPrompt()
		a.context = append(a.context, model.CreateSystemMessage(systemPrompt))

		// 用户消息
		textContent := fmt.Sprintf("%s\n\n%s", userPrompt, screenInfo)
		a.context = append(a.context, model.CreateUserMessage(textContent, screenshot.Base64Data))
	} else {
		// 后续消息
		textContent := fmt.Sprintf("** Screen Info **\n\n%s", screenInfo)
		a.context = append(a.context, model.CreateUserMessage(textContent, screenshot.Base64Data))
	}

	response, err := a.modelClient.Request(a.context)
	if err != nil {
		return nil, "", err
	}

	// 解析动作
	action, err := actions.ParseAction(response.Action)
	if err != nil {
		action = map[string]interface{}{
			"_metadata": "finish",
			"message":   response.Action,
		}
	}

	return action, response.Thinking, nil
}

// getVisionPrompt 获取视觉模型的提示词
func (a *PhoneAgent) getVisionPrompt(plan *model.PlanResult) string {
	basePrompt := `你是一个纯视觉坐标识别助手。你的唯一职责是分析屏幕截图并返回坐标。

重要说明：
- 你只负责识别屏幕上的元素位置，返回坐标
- 不需要分析操作逻辑或决定下一步做什么
- 只返回坐标数据，不要返回任何动作指令

根据描述识别屏幕上的目标元素：

如果描述提到"点击"、"点"或"tap"：
- 返回点击位置的坐标
- 格式：<answer>[x,y]</answer>

如果描述提到"滑动"、"划"或"swipe"：
- 返回起点和终点的坐标
- 格式：<answer>[x1,y1],[x2,y2]</answer>
  其中 [x1,y1] 是起点，[x2,y2] 是终点

如果描述提到"双击"：
- 返回双击位置的坐标
- 格式：<answer>[x,y]</answer>

如果描述提到"长按"：
- 返回长按位置的坐标
- 格式：<answer>[x,y]</answer>

坐标范围：0-1000，表示相对位置（左上角为[0,0]，右下角为[1000,1000]）。

示例：
- "点击搜索按钮" → <answer>[500,200]</answer>
- "从下往上滑动" → <answer>[500,800],[500,200]</answer>
- "双击图片" → <answer>[300,400]</answer>
- "长按图标" → <answer>[600,300]</answer>

请直接返回坐标，不要添加任何解释。`

	// 根据操作类型和原因构建具体的描述
	var description string
	switch plan.ActionType {
	case "Tap":
		description = fmt.Sprintf("需要点击：%s", plan.Reason)
	case "Swipe":
		description = fmt.Sprintf("需要滑动：%s", plan.Reason)
	case "DoubleTap":
		description = fmt.Sprintf("需要双击：%s", plan.Reason)
	case "LongPress":
		description = fmt.Sprintf("需要长按：%s", plan.Reason)
	default:
		description = plan.Reason
	}

	return basePrompt + "\n\n目标描述：" + description
}

// parseVisionCoordinates 解析视觉模型返回的纯坐标
func parseVisionCoordinates(content string, verbose bool) ([][]float64, error) {
	content = strings.TrimSpace(content)

	// 移除可能的 XML 标签
	content = strings.ReplaceAll(content, "<answer>", "")
	content = strings.ReplaceAll(content, "</answer>", "")
	content = strings.TrimSpace(content)

	// 尝试提取坐标（支持多种格式）
	// 格式：[x,y] 或 [x,y],[x2,y2]
	var coordinates [][]float64

	// 查找所有 [xxx,xxx] 格式的坐标
	openBracket := 0
	for i := 0; i < len(content); i++ {
		if content[i] == '[' {
			openBracket = i
		} else if content[i] == ']' && openBracket > 0 {
			// 提取括号内的内容
			coordStr := content[openBracket+1 : i]
			coord, err := parseSingleCoord(coordStr)
			if err == nil {
				coordinates = append(coordinates, coord)
			}
			openBracket = 0
		}
	}

	if len(coordinates) > 0 {
		return coordinates, nil
	}

	return nil, fmt.Errorf("无法解析坐标: %s", content)
}

// parseSingleCoord 解析单个坐标
func parseSingleCoord(s string) ([]float64, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return nil, fmt.Errorf("坐标格式错误")
	}

	x, err1 := parseFloat(strings.TrimSpace(parts[0]))
	y, err2 := parseFloat(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return nil, fmt.Errorf("坐标值错误")
	}

	return []float64{x, y}, nil
}

// parseFloat 解析浮点数
func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	var result float64
	_, err := fmt.Sscanf(s, "%f", &result)
	return result, err
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
			- DoubleTap(element=[x,y]): 双击指定坐标
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
