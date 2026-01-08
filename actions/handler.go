package actions

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"go-phone-agent/adb"
)

// ActionResult 动作执行结果
type ActionResult struct {
	Success      bool   // 是否成功
	ShouldFinish bool   // 是否完成任务
	Message      string // 结果消息
}

// ActionHandler 动作处理器
type ActionHandler struct {
	deviceID             string
	confirmationCallback func(message string) bool
	takeoverCallback     func(message string)
}

// NewActionHandler 创建动作处理器
func NewActionHandler(deviceID string, confirmationCallback func(string) bool, takeoverCallback func(string)) *ActionHandler {
	if confirmationCallback == nil {
		confirmationCallback = defaultConfirmationCallback
	}
	if takeoverCallback == nil {
		takeoverCallback = defaultTakeoverCallback
	}

	return &ActionHandler{
		deviceID:             deviceID,
		confirmationCallback: confirmationCallback,
		takeoverCallback:     takeoverCallback,
	}
}

// Execute 执行动作
func (h *ActionHandler) Execute(action map[string]interface{}, screenWidth, screenHeight int) (*ActionResult, error) {
	metadata, _ := action["_metadata"].(string)

	// 处理完成动作
	if metadata == "finish" {
		message, _ := action["message"].(string)
		return &ActionResult{
			Success:      true,
			ShouldFinish: true,
			Message:      message,
		}, nil
	}

	// 处理执行动作
	if metadata != "do" {
		return &ActionResult{
			Success:      false,
			ShouldFinish: true,
			Message:      fmt.Sprintf("Unknown action type: %s", metadata),
		}, nil
	}

	actionName, _ := action["action"].(string)
	switch actionName {
	case "Launch":
		return h.handleLaunch(action)
	case "Tap":
		return h.handleTap(action, screenWidth, screenHeight)
	case "Type":
		return h.handleType(action)
	case "Swipe":
		return h.handleSwipe(action, screenWidth, screenHeight)
	case "Back":
		return h.handleBack()
	case "Home":
		return h.handleHome()
	case "Double Tap":
		return h.handleDoubleTap(action, screenWidth, screenHeight)
	case "Long Press":
		return h.handleLongPress(action, screenWidth, screenHeight)
	case "Wait":
		return h.handleWait(action)
	case "Take_over":
		return h.handleTakeover(action)
	default:
		return &ActionResult{
			Success:      false,
			ShouldFinish: false,
			Message:      fmt.Sprintf("Unknown action: %s", actionName),
		}, nil
	}
}

// handleLaunch 处理启动应用
func (h *ActionHandler) handleLaunch(action map[string]interface{}) (*ActionResult, error) {
	appName, _ := action["app"].(string)
	if appName == "" {
		return &ActionResult{
			Success:      false,
			ShouldFinish: false,
			Message:      "No app name specified",
		}, nil
	}

	success, err := adb.LaunchApp(appName, h.deviceID)
	if err != nil {
		return &ActionResult{
			Success:      false,
			ShouldFinish: false,
			Message:      err.Error(),
		}, nil
	}

	if success {
		return &ActionResult{
			Success:      true,
			ShouldFinish: false,
		}, nil
	}

	return &ActionResult{
		Success:      false,
		ShouldFinish: false,
		Message:      fmt.Sprintf("App not found: %s", appName),
	}, nil
}

// handleTap 处理点击
func (h *ActionHandler) handleTap(action map[string]interface{}, screenWidth, screenHeight int) (*ActionResult, error) {
	element := action["element"]
	if element == nil {
		return &ActionResult{
			Success:      false,
			ShouldFinish: false,
			Message:      "No element coordinates",
		}, nil
	}

	// 检查敏感操作
	if msg, ok := action["message"].(string); ok && msg != "" {
		if !h.confirmationCallback(msg) {
			return &ActionResult{
				Success:      false,
				ShouldFinish: true,
				Message:      "User cancelled sensitive operation",
			}, nil
		}
	}

	// 转换坐标
	coords, err := parseCoordinates(element)
	if err != nil {
		return &ActionResult{
			Success:      false,
			ShouldFinish: false,
			Message:      fmt.Sprintf("Failed to parse coordinates: %v", err),
		}, nil
	}
	x := int(float64(coords[0]) / 1000 * float64(screenWidth))
	y := int(float64(coords[1]) / 1000 * float64(screenHeight))

	if err := adb.Tap(x, y, h.deviceID); err != nil {
		return &ActionResult{
			Success:      false,
			ShouldFinish: false,
			Message:      err.Error(),
		}, nil
	}

	return &ActionResult{
		Success:      true,
		ShouldFinish: false,
	}, nil
}

// handleType 处理输入文本
func (h *ActionHandler) handleType(action map[string]interface{}) (*ActionResult, error) {
	text, _ := action["text"].(string)
	if text == "" {
		return &ActionResult{
			Success:      false,
			ShouldFinish: false,
			Message:      "No text specified",
		}, nil
	}

	if err := adb.TypeText(text, h.deviceID); err != nil {
		return &ActionResult{
			Success:      false,
			ShouldFinish: false,
			Message:      err.Error(),
		}, nil
	}

	return &ActionResult{
		Success:      true,
		ShouldFinish: false,
	}, nil
}

// handleSwipe 处理滑动
func (h *ActionHandler) handleSwipe(action map[string]interface{}, screenWidth, screenHeight int) (*ActionResult, error) {
	start := action["start"]
	end := action["end"]
	if start == nil || end == nil {
		return &ActionResult{
			Success:      false,
			ShouldFinish: false,
			Message:      "Missing swipe coordinates",
		}, nil
	}

	startCoords, err := parseCoordinates(start)
	if err != nil {
		return &ActionResult{
			Success:      false,
			ShouldFinish: false,
			Message:      fmt.Sprintf("Failed to parse start coordinates: %v", err),
		}, nil
	}
	endCoords, err := parseCoordinates(end)
	if err != nil {
		return &ActionResult{
			Success:      false,
			ShouldFinish: false,
			Message:      fmt.Sprintf("Failed to parse end coordinates: %v", err),
		}, nil
	}

	startX := int(float64(startCoords[0]) / 1000 * float64(screenWidth))
	startY := int(float64(startCoords[1]) / 1000 * float64(screenHeight))
	endX := int(float64(endCoords[0]) / 1000 * float64(screenWidth))
	endY := int(float64(endCoords[1]) / 1000 * float64(screenHeight))

	if err := adb.Swipe(startX, startY, endX, endY, 0, h.deviceID); err != nil {
		return &ActionResult{
			Success:      false,
			ShouldFinish: false,
			Message:      err.Error(),
		}, nil
	}

	return &ActionResult{
		Success:      true,
		ShouldFinish: false,
	}, nil
}

// handleBack 处理返回
func (h *ActionHandler) handleBack() (*ActionResult, error) {
	if err := adb.Back(h.deviceID); err != nil {
		return &ActionResult{
			Success:      false,
			ShouldFinish: false,
			Message:      err.Error(),
		}, nil
	}
	return &ActionResult{Success: true, ShouldFinish: false}, nil
}

// handleHome 处理返回桌面
func (h *ActionHandler) handleHome() (*ActionResult, error) {
	if err := adb.Home(h.deviceID); err != nil {
		return &ActionResult{
			Success:      false,
			ShouldFinish: false,
			Message:      err.Error(),
		}, nil
	}
	return &ActionResult{Success: true, ShouldFinish: false}, nil
}

// handleDoubleTap 处理双击
func (h *ActionHandler) handleDoubleTap(action map[string]interface{}, screenWidth, screenHeight int) (*ActionResult, error) {
	element := action["element"]
	if element == nil {
		return &ActionResult{
			Success:      false,
			ShouldFinish: false,
			Message:      "No element coordinates",
		}, nil
	}

	coords, err := parseCoordinates(element)
	if err != nil {
		return &ActionResult{
			Success:      false,
			ShouldFinish: false,
			Message:      fmt.Sprintf("Failed to parse coordinates: %v", err),
		}, nil
	}
	x := int(float64(coords[0]) / 1000 * float64(screenWidth))
	y := int(float64(coords[1]) / 1000 * float64(screenHeight))

	if err := adb.DoubleTap(x, y, h.deviceID); err != nil {
		return &ActionResult{
			Success:      false,
			ShouldFinish: false,
			Message:      err.Error(),
		}, nil
	}

	return &ActionResult{
		Success:      true,
		ShouldFinish: false,
	}, nil
}

// handleLongPress 处理长按
func (h *ActionHandler) handleLongPress(action map[string]interface{}, screenWidth, screenHeight int) (*ActionResult, error) {
	element := action["element"]
	if element == nil {
		return &ActionResult{
			Success:      false,
			ShouldFinish: false,
			Message:      "No element coordinates",
		}, nil
	}

	coords, err := parseCoordinates(element)
	if err != nil {
		return &ActionResult{
			Success:      false,
			ShouldFinish: false,
			Message:      fmt.Sprintf("Failed to parse coordinates: %v", err),
		}, nil
	}
	x := int(float64(coords[0]) / 1000 * float64(screenWidth))
	y := int(float64(coords[1]) / 1000 * float64(screenHeight))

	if err := adb.LongPress(x, y, 3000, h.deviceID); err != nil {
		return &ActionResult{
			Success:      false,
			ShouldFinish: false,
			Message:      err.Error(),
		}, nil
	}

	return &ActionResult{
		Success:      true,
		ShouldFinish: false,
	}, nil
}

// handleWait 处理等待
func (h *ActionHandler) handleWait(action map[string]interface{}) (*ActionResult, error) {
	durationStr, _ := action["duration"].(string)
	if durationStr == "" {
		durationStr = "1 seconds"
	}

	durationStr = strings.TrimSuffix(durationStr, "seconds")
	durationStr = strings.TrimSpace(durationStr)

	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		duration = 1.0
	}

	// 等待指定秒数
	time.Sleep(time.Duration(duration * float64(time.Second)))

	return &ActionResult{
		Success:      true,
		ShouldFinish: false,
	}, nil
}

// handleTakeover 处理人工接管
func (h *ActionHandler) handleTakeover(action map[string]interface{}) (*ActionResult, error) {
	message, _ := action["message"].(string)
	if message == "" {
		message = "User intervention required"
	}

	h.takeoverCallback(message)

	return &ActionResult{
		Success:      true,
		ShouldFinish: false,
	}, nil
}

// parseCoordinates 解析坐标
func parseCoordinates(coords interface{}) ([2]float64, error) {
	result := [2]float64{0, 0}

	switch v := coords.(type) {
	case []interface{}:
		// 列表格式: [500, 500]
		for i := 0; i < len(v) && i < 2; i++ {
			switch num := v[i].(type) {
			case float64:
				result[i] = num
			case int:
				result[i] = float64(num)
			case string:
				if f, err := strconv.ParseFloat(num, 64); err == nil {
					result[i] = f
				}
			}
		}
	case string:
		// 字符串格式: "[500, 500]" 或 "500, 500"
		// 去除方括号
		v = strings.ReplaceAll(v, "[", "")
		v = strings.ReplaceAll(v, "]", "")
		parts := strings.Split(v, ",")
		for i := 0; i < len(parts) && i < 2; i++ {
			if f, err := strconv.ParseFloat(strings.TrimSpace(parts[i]), 64); err == nil {
				result[i] = f
			}
		}
	default:
		return result, fmt.Errorf("unsupported coordinates type: %T", coords)
	}

	return result, nil
}

// defaultConfirmationCallback 默认确认回调
func defaultConfirmationCallback(message string) bool {
	var response string
	fmt.Printf("敏感操作: %s\n确认? (Y/N): ", message)
	fmt.Scanln(&response)
	return strings.ToUpper(response) == "Y"
}

// defaultTakeoverCallback 默认接管回调
func defaultTakeoverCallback(message string) {
	fmt.Printf("请手动完成: %s\n", message)
	fmt.Println("完成后按回车继续...")
	var discard string
	fmt.Scanln(&discard)
}

// ParseAction 解析动作字符串
func ParseAction(response string) (map[string]interface{}, error) {
	response = strings.TrimSpace(response)

	// 处理 finish(message=xxx)
	if strings.HasPrefix(response, "finish(message=") {
		end := strings.LastIndex(response, ")")
		if end > 0 {
			message := response[16:end] // len("finish(message=") = 16
			return map[string]interface{}{
				"_metadata": "finish",
				"message":   message,
			}, nil
		}
	}

	// 处理 do(action="Type", text=xxx)
	if strings.HasPrefix(response, "do(action=\"Type\"") || strings.HasPrefix(response, "do(action=\"Type_Name\"") {
		parts := strings.SplitN(response, "text=", 2)
		if len(parts) == 2 {
			text := strings.Trim(strings.TrimSuffix(parts[1], ")"), "\"")
			return map[string]interface{}{
				"_metadata": "do",
				"action":    "Type",
				"text":      text,
			}, nil
		}
	}

	// 处理 do(action="xxx", ...)
	if strings.HasPrefix(response, "do(action=") {
		// 替换特殊字符以进行解析
		response = strings.ReplaceAll(response, "\n", "\\n")
		response = strings.ReplaceAll(response, "\r", "\\r")
		response = strings.ReplaceAll(response, "\t", "\\t")

		// 解析函数调用格式: do(action="Launch", app="QQ")
		action := make(map[string]interface{})
		action["_metadata"] = "do"

		// 跳过 "do(" 前缀
		start := strings.Index(response, "(")
		if start == -1 {
			return action, nil
		}
		content := response[start+1:]

		// 去掉结尾的 ")"
		if end := strings.LastIndex(content, ")"); end > 0 {
			content = content[:end]
		}

		// 解析所有 key="value" 参数
		for i := 0; i < len(content); {
			// 跳过空白和逗号
			for i < len(content) && (content[i] == ' ' || content[i] == ',' || content[i] == '\t') {
				i++
			}
			if i >= len(content) {
				break
			}

			// 查找 key
			keyStart := i
			equalPos := strings.Index(content[keyStart:], "=")
			if equalPos == -1 {
				break
			}
			key := strings.TrimSpace(content[keyStart : keyStart+equalPos])

			// 跳过 = 和可能的引号
			valueStart := keyStart + equalPos + 1
			if valueStart >= len(content) {
				break
			}

			// 跳过 = 后面的空格
			for valueStart < len(content) && content[valueStart] == ' ' {
				valueStart++
			}

			var value string
			var valueEnd int

			// 检查值是否用引号包围
			if valueStart < len(content) && (content[valueStart] == '"' || content[valueStart] == '\'') {
				// 带引号的值: "xxx"
				quoteChar := content[valueStart]
				valueStart++ // 跳过开始引号
				quotePos := strings.Index(content[valueStart:], string(quoteChar))
				if quotePos == -1 {
					break
				}
				value = content[valueStart : valueStart+quotePos]
				valueEnd = valueStart + quotePos + 1 // 包含结束引号
			} else {
				// 不带引号的值: [500, 500]
				// 查找下一个逗号或结尾,但要跳过方括号内的内容
				depth := 0 // 用于跟踪括号嵌套
				valueEnd = valueStart
				for valueEnd < len(content) {
					ch := content[valueEnd]
					if ch == '[' {
						depth++
					} else if ch == ']' {
						if depth > 0 {
							depth--
						}
					} else if ch == ',' && depth == 0 {
						// 遇到逗号且不在括号内,结束
						break
					}
					valueEnd++
				}
				value = strings.TrimSpace(content[valueStart:valueEnd])
			}

			// 保存参数
			action[key] = value

			// 移动到值的后面
			i = valueEnd + 1
		}

		return action, nil
	}

	return nil, fmt.Errorf("failed to parse action")
}
