package model

// ModelConfig AI 模型配置
type ModelConfig struct {
	BaseURL          string  // 模型 API 地址
	APIKey           string  // API 密钥
	ModelName        string  // 模型名称
	MaxTokens        int     // 最大 token 数
	Temperature      float64 // 采样温度
	TopP             float64 // Top-P 采样
	FrequencyPenalty float64 // 频率惩罚
}

// DefaultModelConfig 返回默认配置
func DefaultModelConfig() *ModelConfig {
	return &ModelConfig{
		BaseURL:          "https://open.bigmodel.cn/api/paas/v4",
		APIKey:           "EMPTY",
		ModelName:        "autoglm-phone",
		MaxTokens:        3000,
		Temperature:      0.0,
		TopP:             0.85,
		FrequencyPenalty: 0.2,
	}
}

// ModelResponse 模型响应
type ModelResponse struct {
	Thinking          string  // 思考过程
	Action            string  // 动作指令
	RawContent        string  // 原始内容
	TimeToFirstToken  float64 // 首字延迟(秒)
	TimeToThinkingEnd float64 // 思考结束时间(秒)
	TotalTime         float64 // 总时间(秒)
}

// Message 对话消息
type Message struct {
	Role    string      `json:"role"`    // system/user/assistant
	Content interface{} `json:"content"` // text 或 [{type:"text",text:"..."},{type:"image_url",...}]
}

// ChatCompletionRequest 聊天完成请求
type ChatCompletionRequest struct {
	Messages         []Message `json:"messages"`
	Model            string    `json:"model"`
	MaxTokens        int       `json:"max_tokens,omitempty"`
	Temperature      float64   `json:"temperature,omitempty"`
	TopP             float64   `json:"top_p,omitempty"`
	FrequencyPenalty float64   `json:"frequency_penalty,omitempty"`
	Stream           bool      `json:"stream"`
}

// ImageContent 图片内容
type ImageContent struct {
	Type     string `json:"type"` // text 或 image_url
	Text     string `json:"text,omitempty"`
	ImageURL struct {
		URL string `json:"url"` // data:image/png;base64,xxx
	} `json:"image_url,omitempty"`
}
