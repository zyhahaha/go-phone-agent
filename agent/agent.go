package agent

import (
	"fmt"
	"strings"

	"go-phone-agent/actions"
	"go-phone-agent/adb"
	"go-phone-agent/model"
)

// PhoneAgent 手机自动化 Agent
type PhoneAgent struct {
	visionClient    *model.Client      // 屏幕分析客户端
	coordClient     *model.Client      // 坐标识别客户端
	actionHandler   *actions.ActionHandler
	config          *AgentConfig
	decisionModel   *model.DecisionModel // 决策模型
	decisionConfig  *model.DecisionConfig
	context         []model.Message
	stepCount       int
	actionHistory   []model.ActionHistory
	currentTask     string // 当前任务
}

// NewPhoneAgentWithDecisionModel 创建带决策模型的 PhoneAgent
func NewPhoneAgentWithDecisionModel(decisionConfig *model.DecisionConfig, agentConfig *AgentConfig, confirmationCallback func(string) bool, takeoverCallback func(string)) *PhoneAgent {
	if decisionConfig == nil {
		decisionConfig = model.DefaultDecisionConfig()
	}
	if agentConfig == nil {
		agentConfig = DefaultAgentConfig()
	}

	// 创建两个专门的视觉客户端：屏幕分析和坐标识别
	visionClient := model.NewClientWithSystemPrompt(decisionConfig.Vision, model.ScreenAnalysisPrompt)
	coordClient := model.NewClientWithSystemPrompt(decisionConfig.Vision, model.VisionCoordPrompt)

	return &PhoneAgent{
		visionClient:     visionClient,
		coordClient:      coordClient,
		actionHandler:    actions.NewActionHandler(agentConfig.DeviceID, confirmationCallback, takeoverCallback),
		config:           agentConfig,
		decisionModel:    model.NewDecisionModel(decisionConfig.Decision),
		decisionConfig:   decisionConfig,
		context:           []model.Message{},
		stepCount:         0,
		actionHistory:     []model.ActionHistory{},
		currentTask:       "",
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

// GetStepCount 获取当前步数
func (a *PhoneAgent) GetStepCount() int {
	return a.stepCount
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

	var action map[string]interface{}
	var thinking string
	var execErr error

	// 执行决策模型模式：决策模型规划，视觉模型执行
	action, thinking, execErr = a.executeWithDecisionModel(userPrompt, screenshot)

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

// executeWithDecisionModel 使用决策模型模式执行
func (a *PhoneAgent) executeWithDecisionModel(userPrompt string, screenshot *adb.Screenshot) (map[string]interface{}, string, error) {
	// 使用保存的当前任务
	task := a.currentTask

	// 如果是第一步且 userPrompt 不为空，更新任务
	if a.stepCount == 1 && userPrompt != "" {
		task = userPrompt
		a.currentTask = userPrompt
	}

	// 第一步：先调用视觉模型获取屏幕描述
	screenDescription := ""
	screenDesc, err := a.analyzeScreen(screenshot)
	if err != nil {
		screenDescription = "屏幕分析失败"
	} else {
		screenDescription = screenDesc
	}

	// 打印视觉模型 → 决策模型的交互内容
	// if a.config.Verbose {
	// 	fmt.Println()
	// 	fmt.Println("📤 视觉模型 → 决策模型 (屏幕描述):")
	// 	fmt.Printf("%s\n", screenDescription)
	// 	fmt.Println()
	// }

	// 第二步：调用决策模型，基于屏幕描述做决策
	plan, err := a.decisionModel.PlanStep(task, screenDescription, a.stepCount, a.config.MaxSteps, a.actionHistory)
	if err != nil {
		return nil, "", err
	}

	// 打印决策模型发出的操作指令
	// if a.config.Verbose {
	// 	fmt.Println("操作指令:")
	// 	fmt.Printf("操作类型: %s\n", plan.ActionType)
	// 	fmt.Printf("操作原因: %s\n", plan.Reason)
	// 	if len(plan.Parameters) > 0 {
	// 		params, _ := json.MarshalIndent(plan.Parameters, "  ", "")
	// 		fmt.Printf("操作参数: %s\n", string(params))
	// 	}
	// 	fmt.Println()
	// }

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
	// 使用专门的坐标识别客户端
	description := a.getVisionDescription(plan)
	visionContext := []model.Message{
		model.CreateUserMessage(description, screenshot.Base64Data),
	}
	model.LogStart("视觉坐标分析提示词")
	model.LogContent(*a.coordClient.SystemPrompt)
	model.LogContent(visionContext[0])
	model.LogEnd("视觉坐标分析提示词")

	// 调用视觉模型获取坐标
	response, err := a.coordClient.Request(visionContext)
	if err != nil {
		return nil, "", err
	}

	model.LogStart("视觉坐标模型输出")
	model.LogContent(response)
	model.LogEnd("视觉坐标模型输出")

	// 解析视觉模型的响应（纯坐标格式）
	coordinates, err := parseVisionCoordinates(response.RawContent, a.config.Verbose)
	if err != nil {
		return nil, "", err
	}

	// 构建完整的操作：决策模型的操作类型 + 视觉模型的坐标
	visionAction := map[string]interface{}{
		"action":    plan.ActionType,
		"_metadata": "do",
	}

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
func (a *PhoneAgent) analyzeScreen(screenshot *adb.Screenshot) (string, error) {
	// 使用专门的屏幕分析客户端（系统提示词已缓存）
	messages := []model.Message{
		model.CreateUserMessage("描述屏幕内容", screenshot.Base64Data),
	}

	model.LogStart("屏幕内容分析提示词")
	model.LogContent(*a.visionClient.SystemPrompt)
	model.LogEnd("屏幕内容分析提示词")

	response, err := a.visionClient.Request(messages)
	if err != nil {
		return "", err
	}

	return response.RawContent, nil
}

// getVisionDescription 获取视觉模型的目标描述
func (a *PhoneAgent) getVisionDescription(plan *model.PlanResult) string {
	// 根据操作类型和原因构建具体的描述
	switch plan.ActionType {
	case "Tap":
		return fmt.Sprintf("需要点击：%s", plan.Reason)
	case "Swipe":
		return fmt.Sprintf("需要滑动：%s", plan.Reason)
	case "DoubleTap":
		return fmt.Sprintf("需要双击：%s", plan.Reason)
	case "LongPress":
		return fmt.Sprintf("需要长按：%s", plan.Reason)
	default:
		return plan.Reason
	}
}

// parseVisionCoordinates 解析视觉模型返回的纯坐标
func parseVisionCoordinates(content string, verbose bool) ([][]float64, error) {
	// 去除所有换行符和空格
	content = strings.ReplaceAll(content, "\n", "")
	content = strings.ReplaceAll(content, "\r", "")
	content = strings.TrimSpace(content)

	// 移除可能的 XML 标签
	content = strings.ReplaceAll(content, "<answer>", "")
	content = strings.ReplaceAll(content, "</answer>", "")
	content = strings.TrimSpace(content)

	// 尝试提取坐标（支持多种格式）
	// 格式1：[x,y] 或 [x,y],[x2,y2] - 点坐标
	// 格式2：[[x1,y1,x2,y2]] - 边界框，自动转换为中心点
	var coordinates [][]float64

	// 查找所有 [xxx,xxx] 或 [xxx,xxx,xxx,xxx] 格式的坐标
	openBracket := -1 // 使用 -1 表示未找到 [
	for i := 0; i < len(content); i++ {
		char := content[i]
		if char == '[' {
			openBracket = i
		} else if char == ']' && openBracket >= 0 {
			// 提取括号内的内容
			coordStr := content[openBracket+1 : i]
			coord, err := parseSingleCoord(coordStr)
			if err == nil {
				coordinates = append(coordinates, coord)
			}
			openBracket = -1 // 重置为 -1
		}
	}

	if len(coordinates) > 0 {
		return coordinates, nil
	}

	return nil, fmt.Errorf("无法解析坐标: %s", content)
}

// parseSingleCoord 解析单个坐标，支持点坐标和边界框格式
func parseSingleCoord(s string) ([]float64, error) {
	parts := strings.Split(s, ",")
	var coords []float64

	// 解析所有数值
	for _, part := range parts {
		val, err := parseFloat(strings.TrimSpace(part))
		if err != nil {
			continue // 跳过无效值
		}
		coords = append(coords, val)
	}

	// 根据数值数量判断格式
	if len(coords) == 2 {
		// 格式1：[x,y] - 点坐标
		return []float64{coords[0], coords[1]}, nil
	} else if len(coords) == 4 {
		// 格式2：[x1,y1,x2,y2] - 边界框，转换为中心点
		centerX := (coords[0] + coords[2]) / 2
		centerY := (coords[1] + coords[3]) / 2
		return []float64{centerX, centerY}, nil
	}

	return nil, fmt.Errorf("坐标格式错误：期望2或4个数值，实际得到%d个", len(coords))
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
