package model

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SchedulerDeepSeek DeepSeek 调度器，负责任务规划和逻辑处理
type SchedulerDeepSeek struct {
	config     *ModelConfig
	httpClient *http.Client
}

// NewSchedulerDeepSeek 创建 DeepSeek 调度器
func NewSchedulerDeepSeek(config *ModelConfig) *SchedulerDeepSeek {
	return &SchedulerDeepSeek{
		config: config,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// PlanStep 计划下一步操作
func (s *SchedulerDeepSeek) PlanStep(task string, screenInfo string, currentStep int, maxSteps int, history []ActionHistory) (*PlanResult, error) {
	messages := []Message{}

	// 系统消息
	systemMsg := CreateSystemMessage(s.getSystemPrompt())
	messages = append(messages, systemMsg)

	// 构建任务上下文
	taskContext := s.buildTaskContext(task, screenInfo, currentStep, maxSteps, history)
	userMsg := CreateUserMessage(taskContext, "")
	messages = append(messages, userMsg)

	fmt.Println(strings.Repeat("=", 50), "决策模型提示词 Start", strings.Repeat("=", 50))
	fmt.Println(messages)
	fmt.Println(strings.Repeat("=", 50), "决策模型提示词 End", strings.Repeat("=", 50))

	// 调用 DeepSeek
	response, err := s.request(messages)
	if err != nil {
		return nil, fmt.Errorf("DeepSeek planning error: %w", err)
	}

	// 解析计划
	var content string
	if len(response.Choices) > 0 {
		content = response.Choices[0].Message.Content
	}
	plan := s.parsePlan(content)
	return plan, nil
}

// request 发送请求到 DeepSeek
func (s *SchedulerDeepSeek) request(messages []Message) (*ChatCompletionResponse, error) {
	req := &ChatCompletionRequest{
		Messages:         messages,
		Model:            s.config.ModelName,
		MaxTokens:        s.config.MaxTokens,
		Temperature:      s.config.Temperature,
		TopP:             s.config.TopP,
		FrequencyPenalty: s.config.FrequencyPenalty,
		Stream:           false,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", s.config.BaseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.config.APIKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var chatResp ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &chatResp, nil
}

// getSystemPrompt 获取调度器系统提示词
func (s *SchedulerDeepSeek) getSystemPrompt() string {
	return `
		你是一个手机自动化的任务调度器。你的职责是分析用户任务并规划操作步骤。

		**你的工作模式：**
		1. 理解用户意图，将复杂任务拆解为可执行的步骤
		2. 视觉模型已经为你分析了当前屏幕内容，提供了客观描述
		3. 根据屏幕描述中的实际内容（文字、元素、布局）和用户任务，判断当前需要做什么操作
		4. 指挥视觉模型执行具体的屏幕操作

		**重要说明：**

		关于屏幕信息的理解：
		- 你不会看到实际的屏幕图片，只能看到视觉模型提供的屏幕描述
		- 屏幕描述是客观的：包含可见的文字、数字、UI元素、布局等
		- **屏幕描述中可能包含应用名称，但这个名称可能不准确，不要完全依赖**
		- 你的决策应该基于屏幕上实际显示的内容，而不是应用名称

		决策原则：
		- 根据屏幕上的**文字、数字、按钮、图标**来判断当前状态
		- 即使应用名称识别错误，只要能看到关键元素就能正确操作
		- 例如：即使不认识这是什么游戏，只要看到"进攻"按钮就知道该点击

		**可用操作类型：**
		- Launch(app="应用名"): 启动应用
		- Tap: 点击屏幕（具体坐标由视觉模型确定）
		- Type: 输入文本
		- Swipe: 滑动屏幕
		- Back: 返回
		- Home: 返回桌面
		- DoubleTap: 双击
		- LongPress: 长按
		- Wait: 等待
		- Take_over: 请求人工接管

		**输出格式：**
		<thought>你的思考过程</thought>
		<action>操作类型</action>
		<parameters>{"param": "value"}</parameters>
		<reason>操作原因</reason>

		**示例：**

		示例1 - 点击图标（只依赖可见元素）：
		<thought>用户要求打开某个应用，屏幕描述显示底部有多个应用图标，其中包括一个带有绿色图标和红点通知的图标</thought>
		<action>Tap</action>
		<parameters>{"target": "底部绿色的应用图标"}</parameters>
		<reason>点击绿色的应用图标</reason>

		示例2 - 点击按钮（不依赖应用名）：
		<thought>用户要求进入个人页面，屏幕描述显示底部有四个导航图标，最右侧一个显示文字"我"</thought>
		<action>Tap</action>
		<parameters>{"target": "底部最右侧显示'我'的按钮"}</parameters>
		<reason>点击"我"按钮进入个人页面</reason>

		示例3 - 游戏操作（完全基于可见内容）：
		<thought>用户要求点击进攻按钮，屏幕描述显示底部有一个绿色的"进击！"按钮，带有两把剑图标</thought>
		<action>Tap</action>
		<parameters>{"target": "底部绿色的'进击！'按钮，带有两把剑图标"}</parameters>
		<reason>根据描述匹配，点击"进击！"按钮</reason>

		示例4 - 任务完成：
		<thought>用户要求查看个人资料，屏幕描述已显示个人信息和头像，任务已完成</thought>
		<action>finish</action>
		<parameters>{}</parameters>
		<reason>已成功显示个人资料信息</reason>

		**注意事项：**
		- 仔细阅读屏幕描述，识别其中的文字、数字、按钮
		- 优先基于**可见元素**而非应用名称做决策
		- 对于复杂任务，分步骤执行
		- 每次只执行一个操作
		- 需要点击或滑动时，action设为Tap或Swipe，在parameters中描述目标元素（使用屏幕描述中的文字和特征）
		- 任务完成后使用finish标记
		- 如果屏幕描述不清晰，可以优先使用Back返回或使用更保守的策略`
}

// buildTaskContext 构建任务上下文
func (s *SchedulerDeepSeek) buildTaskContext(task string, screenInfo string, currentStep int, maxSteps int, history []ActionHistory) string {
	context := fmt.Sprintf("用户任务: %s\n", task)
	context += fmt.Sprintf("当前步骤: %d/%d\n", currentStep, maxSteps)
	context += fmt.Sprintf("屏幕信息:\n%s\n", screenInfo)

	if len(history) > 0 {
		context += "\n历史操作:\n"
		for i, h := range history {
			context += fmt.Sprintf("%d. Action: %s, Reason: %s\n", i+1, h.Action, h.Reason)
		}
	}

	return context
}

// parsePlan 解析调度器返回的计划
func (s *SchedulerDeepSeek) parsePlan(content string) *PlanResult {
	result := &PlanResult{
		ActionType: "",
		Parameters: map[string]interface{}{},
		Reason:     "",
		Thought:    "",
	}

	// 解析 thought
	if strings.Contains(content, "<thought>") {
		parts := strings.Split(content, "<thought>")
		if len(parts) > 1 {
			thoughtPart := strings.Split(parts[1], "</thought>")
			if len(thoughtPart) > 0 {
				result.Thought = strings.TrimSpace(thoughtPart[0])
			}
		}
	}

	// 解析 action
	if strings.Contains(content, "<action>") {
		parts := strings.Split(content, "<action>")
		if len(parts) > 1 {
			actionPart := strings.Split(parts[1], "</action>")
			if len(actionPart) > 0 {
				result.ActionType = strings.TrimSpace(actionPart[0])
			}
		}
	}

	// 解析 parameters
	if strings.Contains(content, "<parameters>") {
		parts := strings.Split(content, "<parameters>")
		if len(parts) > 1 {
			paramPart := strings.Split(parts[1], "</parameters>")
			if len(paramPart) > 0 {
				var params map[string]interface{}
				if err := json.Unmarshal([]byte(paramPart[0]), &params); err == nil {
					result.Parameters = params
				}
			}
		}
	}

	// 解析 reason
	if strings.Contains(content, "<reason>") {
		parts := strings.Split(content, "<reason>")
		if len(parts) > 1 {
			reasonPart := strings.Split(parts[1], "</reason>")
			if len(reasonPart) > 0 {
				result.Reason = strings.TrimSpace(reasonPart[0])
			}
		}
	}

	// 检查是否完成
	if strings.Contains(content, "finish") || result.ActionType == "finish" {
		result.Finished = true
	}

	return result
}

// PlanResult 调度器计划结果
type PlanResult struct {
	ActionType string                 // 操作类型
	Parameters map[string]interface{} // 操作参数
	Reason     string                 // 操作原因
	Thought    string                 // 思考过程
	Finished   bool                   // 是否完成
}

// ActionHistory 操作历史记录
type ActionHistory struct {
	Action  string // 操作类型
	Reason  string // 操作原因
	Success bool   // 是否成功
}

// ChatCompletionResponse 聊天完成响应（非流式）
type ChatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// StreamChatCompletion 流式聊天完成（用于DeepSeek流式响应，备用）
func (s *SchedulerDeepSeek) StreamChatCompletion(messages []Message, callback func(string)) error {
	req := &ChatCompletionRequest{
		Messages:         messages,
		Model:            s.config.ModelName,
		MaxTokens:        s.config.MaxTokens,
		Temperature:      s.config.Temperature,
		TopP:             s.config.TopP,
		FrequencyPenalty: s.config.FrequencyPenalty,
		Stream:           true,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", s.config.BaseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.config.APIKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var streamResp struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			continue
		}

		if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
			callback(streamResp.Choices[0].Delta.Content)
		}
	}

	return nil
}
