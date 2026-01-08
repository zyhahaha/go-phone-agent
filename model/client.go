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

// Client AI 模型客户端
type Client struct {
	config     *ModelConfig
	httpClient *http.Client
}

// NewClient 创建模型客户端
func NewClient(config *ModelConfig) *Client {
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Request 发送请求到模型
func (c *Client) Request(messages []Message) (*ModelResponse, error) {
	startTime := time.Now()
	var timeToFirstToken, timeToThinkingEnd float64

	// 构建请求
	req := &ChatCompletionRequest{
		Messages:        messages,
		Model:           c.config.ModelName,
		MaxTokens:       c.config.MaxTokens,
		Temperature:     c.config.Temperature,
		TopP:            c.config.TopP,
		FrequencyPenalty: c.config.FrequencyPenalty,
		Stream:          true,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequest("POST", c.config.BaseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	// 处理流式响应
	scanner := bufio.NewScanner(resp.Body)
	rawContent := ""
	buffer := ""
	inActionPhase := false
	firstTokenReceived := false
	actionMarkers := []string{"finish(message=", "do(action="}

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

		if len(streamResp.Choices) == 0 || streamResp.Choices[0].Delta.Content == "" {
			continue
		}

		content := streamResp.Choices[0].Delta.Content
		rawContent += content

		// 记录首字延迟
		if !firstTokenReceived {
			timeToFirstToken = time.Since(startTime).Seconds()
			firstTokenReceived = true
		}

		if inActionPhase {
			continue
		}

		buffer += content

		// 检查是否有动作标记
		markerFound := false
		for _, marker := range actionMarkers {
			if strings.Contains(buffer, marker) {
				// 打印思考部分
				thinkingPart := strings.Split(buffer, marker)[0]
				fmt.Print(thinkingPart)
				fmt.Println()
				inActionPhase = true
				markerFound = true

				if timeToThinkingEnd == 0 {
					timeToThinkingEnd = time.Since(startTime).Seconds()
				}
				break
			}
		}

		if markerFound {
			continue
		}

		// 检查是否可能是标记的前缀
		isPotentialMarker := false
		for _, marker := range actionMarkers {
			for i := 1; i < len(marker); i++ {
				if strings.HasSuffix(buffer, marker[:i]) {
					isPotentialMarker = true
					break
				}
			}
			if isPotentialMarker {
				break
			}
		}

		if !isPotentialMarker {
			fmt.Print(buffer)
			buffer = ""
		}
	}

	// 计算总时间
	totalTime := time.Since(startTime).Seconds()

	// 解析响应
	thinking, action := parseResponse(rawContent)

	// 打印性能指标
	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 48))
	fmt.Println("⏱️  性能指标:")
	fmt.Println("-" + strings.Repeat("-", 48))
	if timeToFirstToken > 0 {
		fmt.Printf("首字延迟: %.3fs\n", timeToFirstToken)
	}
	if timeToThinkingEnd > 0 {
		fmt.Printf("思考完成:     %.3fs\n", timeToThinkingEnd)
	}
	fmt.Printf("总推理时间:     %.3fs\n", totalTime)
	fmt.Println("=" + strings.Repeat("=", 48))

	return &ModelResponse{
		Thinking:          thinking,
		Action:            action,
		RawContent:        rawContent,
		TimeToFirstToken:  timeToFirstToken,
		TimeToThinkingEnd: timeToThinkingEnd,
		TotalTime:         totalTime,
	}, nil
}

// parseResponse 解析模型响应
func parseResponse(content string) (string, string) {
	// 规则1: 检查 finish(message=
	if strings.Contains(content, "finish(message=") {
		parts := strings.SplitN(content, "finish(message=", 2)
		if len(parts) == 2 {
			thinking := strings.TrimSpace(parts[0])
			action := "finish(message=" + parts[1]
			return thinking, action
		}
	}

	// 规则2: 检查 do(action=
	if strings.Contains(content, "do(action=") {
		parts := strings.SplitN(content, "do(action=", 2)
		if len(parts) == 2 {
			thinking := strings.TrimSpace(parts[0])
			action := "do(action=" + parts[1]
			return thinking, action
		}
	}

	// 规则3: 回退到 XML 标签解析
	if strings.Contains(content, "<answer>") {
		parts := strings.SplitN(content, "<answer>", 2)
		if len(parts) == 2 {
			thinking := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(parts[0], "", ""), "", ""))
			action := strings.TrimSpace(strings.ReplaceAll(parts[1], "</answer>", ""))
			return thinking, action
		}
	}

	// 规则4: 没有找到标记,全部作为动作
	return "", content
}

// CreateUserMessage 创建用户消息
func CreateUserMessage(text string, imageBase64 string) Message {
	content := []ImageContent{}

	if imageBase64 != "" {
		content = append(content, ImageContent{
			Type: "image_url",
		})
		content[len(content)-1].ImageURL.URL = "data:image/png;base64," + imageBase64
	}

	content = append(content, ImageContent{
		Type: "text",
		Text: text,
	})

	return Message{
		Role:    "user",
		Content: content,
	}
}

// CreateSystemMessage 创建系统消息
func CreateSystemMessage(content string) Message {
	return Message{
		Role:    "system",
		Content: content,
	}
}

// CreateAssistantMessage 创建助手消息
func CreateAssistantMessage(content string) Message {
	return Message{
		Role:    "assistant",
		Content: content,
	}
}
