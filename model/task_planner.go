package model

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DecisionModel 决策模型，负责任务规划和逻辑处理
type DecisionModel struct {
	client *Client // 复用 AI API 客户端
}

// NewDecisionModel 创建决策模型
func NewDecisionModel(config *ModelConfig) *DecisionModel {
	client := NewClientWithSystemPrompt(config, DecisionModelPrompt)
	return &DecisionModel{
		client: client,
	}
}

// PlanStep 计划下一步操作
func (m *DecisionModel) PlanStep(task string, screenInfo string, currentStep int, maxSteps int, history []ActionHistory) (*PlanResult, error) {
	// 构建任务上下文
	taskContext := m.buildTaskContext(task, screenInfo, currentStep, maxSteps, history)
	messages := []Message{
		CreateUserMessage(taskContext, ""),
	}

	// 记录日志（包含系统提示词）
	LogStart("决策模型提示词")
	LogContent(*m.client.SystemPrompt)
	LogContent(messages[0])
	LogEnd("决策模型提示词")

	// 调用决策模型（系统提示词已缓存在client中）
	response, err := m.client.Request(messages)
	if err != nil {
		return nil, fmt.Errorf("Decision model error: %w", err)
	}

	LogStart("决策模型输出")
	LogContent(response)
	LogEnd("决策模型输出")

	// 直接从流式响应的RawContent解析计划
	plan := m.parsePlan(response.RawContent)
	return plan, nil
}

// buildTaskContext 构建任务上下文（优化：只保留最近5条历史）
func (m *DecisionModel) buildTaskContext(task string, screenInfo string, currentStep int, maxSteps int, history []ActionHistory) string {
	context := fmt.Sprintf("任务: %s\n", task)
	context += fmt.Sprintf("步骤: %d/%d\n", currentStep, maxSteps)
	context += fmt.Sprintf("屏幕:\n%s\n", screenInfo)

	// 只保留最近5条历史记录
	if len(history) > 0 {
		recent := history
		if len(history) > 5 {
			recent = history[len(history)-5:]
		}
		context += "历史: "
		for _, h := range recent {
			context += fmt.Sprintf("%s→", h.Action)
		}
		context += "\n"
	}

	return context
}

// parsePlan 解析决策模型返回的计划
func (m *DecisionModel) parsePlan(content string) *PlanResult {
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

// PlanResult 决策模型计划结果
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
