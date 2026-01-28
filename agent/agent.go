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
	scheduler       *model.SchedulerDeepSeek
	schedulerConfig *model.SchedulerConfig
	context         []model.Message
	stepCount       int
	actionHistory   []model.ActionHistory
	currentTask     string // 当前任务
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

	// 执行调度器模式：DeepSeek 规划，autoglm-phone 执行
	action, thinking, execErr = a.executeWithScheduler(userPrompt, screenshot)

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

// executeWithScheduler 使用调度器模式执行
func (a *PhoneAgent) executeWithScheduler(userPrompt string, screenshot *adb.Screenshot) (map[string]interface{}, string, error) {
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
		model.CreateUserMessage("请分析屏幕并返回操作坐标。", screenshot.Base64Data),
	}

	// 打印发送给视觉模型的指令
	if a.config.Verbose {
		fmt.Println("📥 DeepSeek → autoglm-phone (视觉指令):")
		fmt.Printf("操作类型: %s\n", plan.ActionType)
		fmt.Printf("目标描述: %s\n", plan.Reason)
		fmt.Println()
	}

	// 调用视觉模型获取坐标
	response, err := a.modelClient.Request(visionContext)
	if err != nil {
		return nil, "", err
	}

	// 打印视觉模型的原始响应
	if a.config.Verbose {
		fmt.Println("📤 autoglm-phone → DeepSeek (坐标响应):")
		fmt.Printf("%s\n", response.RawContent)
		fmt.Println()
	}

	// 解析视觉模型的响应（纯坐标格式）
	coordinates, err := parseVisionCoordinates(response.RawContent, a.config.Verbose)
	if err != nil {
		return nil, "", err
	}

	// 构建完整的操作：DeepSeek 的操作类型 + 视觉模型的坐标
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
	// 构建屏幕分析的提示词
	screenAnalysisPrompt := `你是一个屏幕内容分析助手。请仔细分析屏幕截图，客观描述屏幕上可见的内容。

**重要原则：**
- 专注于描述屏幕上实际可见的内容
- 不要猜测或推断不可见的信息
- 应用名称可能无法准确识别，不要强制判断

**描述要点：**
1. 屏幕上可见的文字和数字（优先级最高）
2. 图标、按钮、输入框等UI元素的位置和样式
3. 页面布局结构（顶部、中间、底部等）
4. 颜色、背景、特殊标记等视觉特征
5. 任何弹出窗口、对话框、加载提示等
6. 如果能从标题栏或状态栏看到应用名称，可以提及，但标记为"可能"

**格式要求：**
- 先描述关键文字和数字
- 再描述UI元素和布局
- 最后提可能的页面状态
- 总共不超过200字

**示例：**
"屏幕显示游戏界面。左上角显示数字'32'和'10'，中间有倒计时'2分50秒'，底部有红色'结束战斗'按钮。"

**避免：**
- 不要说"这是微信界面"，而应该说"底部有四个导航图标（微信、通讯录、发现、我）"
- 不要说"这是设置应用"，而要说"顶部显示'Settings'标题，下方有多个设置项"`

	visionContext := []model.Message{
		model.CreateSystemMessage(screenAnalysisPrompt),
		model.CreateUserMessage("请分析这张图片", screenshot.Base64Data),
	}

	response, err := a.modelClient.Request(visionContext)
	if err != nil {
		return "", err
	}

	// 屏幕分析应该返回纯文本，直接使用原始响应内容
	// 不要经过 parseResponse 解析，避免被误解析为 finish 格式
	return response.RawContent, nil
}

// getVisionPrompt 获取视觉模型的提示词
func (a *PhoneAgent) getVisionPrompt(plan *model.PlanResult) string {
	basePrompt := `你是一个纯视觉坐标识别助手。你的唯一职责是分析屏幕截图并返回坐标。

**重要说明：**
- 你只负责识别屏幕上的元素位置，返回坐标
- 不需要分析操作逻辑或决定下一步做什么
- **只返回坐标数据，使用XML标签包裹，不要返回任何动作指令或解释文字**

**必须严格遵守的输出格式：**

如果描述提到"点击"、"点"或"tap"：
- 返回点击位置的坐标
- **唯一正确的输出格式**：<answer>[x,y]</answer>
- 示例：<answer>[500,200]</answer>

如果描述提到"滑动"、"划"或"swipe"：
- 返回起点和终点的坐标
- **唯一正确的输出格式**：<answer>[x1,y1],[x2,y2]</answer>
- 其中 [x1,y1] 是起点，[x2,y2] 是终点
- 示例：<answer>[500,800],[500,200]</answer>

如果描述提到"双击"：
- 返回双击位置的坐标
- **唯一正确的输出格式**：<answer>[x,y]</answer>
- 示例：<answer>[300,400]</answer>

如果描述提到"长按"：
- 返回长按位置的坐标
- **唯一正确的输出格式**：<answer>[x,y]</answer>
- 示例：<answer>[600,300]</answer>

坐标范围：0-1000，表示相对位置（左上角为[0,0]，右下角为[1000,1000]）。

**错误示例（绝对不要这样输出）：**
❌ [103,470] - 缺少XML标签
❌ 坐标是[103,470] - 添加了文字说明
❌ 点击位置：<answer>[103,470]</answer> - 添加了前缀文字

**正确示例（唯一正确的输出方式）：**
✅ <answer>[103,470]</answer>
✅ <answer>[500,800],[500,200]</answer>

**记住：整个响应只包含<answer>标签和坐标，不能有其他任何内容！**

`

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
	// 去除所有换行符和空格
	content = strings.ReplaceAll(content, "\n", "")
	content = strings.ReplaceAll(content, "\r", "")
	content = strings.TrimSpace(content)

	// 移除可能的 XML 标签
	content = strings.ReplaceAll(content, "<answer>", "")
	content = strings.ReplaceAll(content, "</answer>", "")
	content = strings.TrimSpace(content)

	// 尝试提取坐标（支持多种格式）
	// 格式：[x,y] 或 [x,y],[x2,y2]
	var coordinates [][]float64

	// 查找所有 [xxx,xxx] 格式的坐标
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
