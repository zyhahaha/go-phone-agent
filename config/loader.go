package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// AgentConfig Agent 配置结构（从 agent 包移过来，避免循环导入）
type AgentConfig struct {
	MaxSteps     int    `yaml:"max-steps"`
	DeviceID     string `yaml:"device-id"`
	SystemPrompt string `yaml:"system-prompt"`
	Verbose      bool   `yaml:"verbose"`
}

// ModelConfig AI 模型配置（从 model 包移过来，避免循环导入）
type ModelConfig struct {
	BaseURL          string  `yaml:"base-url"`
	APIKey           string  `yaml:"api-key"`
	ModelName        string  `yaml:"model-name"`
	MaxTokens        int     `yaml:"max-tokens"`
	Temperature      float64 `yaml:"temperature"`
	TopP             float64 `yaml:"top-p"`
	FrequencyPenalty float64 `yaml:"frequency-penalty"`
}

// DecisionConfig 决策模型配置（从 model 包移过来，避免循环导入）
type DecisionConfig struct {
	Decision *ModelConfig `yaml:"decision"`
	Vision   *ModelConfig `yaml:"vision"`
}

// Config 总配置结构
type Config struct {
	Agent    *AgentConfig    `yaml:"agent"`
	Decision *DecisionConfig `yaml:"decision"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Agent: &AgentConfig{
			MaxSteps:     100,
			DeviceID:     "",
			SystemPrompt: "",
			Verbose:      true,
		},
		Decision: &DecisionConfig{
			Decision: &ModelConfig{
				BaseURL:          "https://api.deepseek.com",
				APIKey:           "EMPTY",
				ModelName:        "deepseek-chat",
				MaxTokens:        2000,
				Temperature:      0.7,
				TopP:             0.9,
				FrequencyPenalty: 0.0,
			},
			Vision: &ModelConfig{
				BaseURL:          "https://open.bigmodel.cn/api/paas/v4",
				APIKey:           "EMPTY",
				ModelName:        "autoglm-phone",
				MaxTokens:        3000,
				Temperature:      0.0,
				TopP:             0.85,
				FrequencyPenalty: 0.2,
			},
		},
	}
}

// ConvertToAgentConfig 将 Config 中的 AgentConfig 转换为 agent.AgentConfig
func (c *Config) ConvertToAgentConfig() interface{} {
	if c.Agent == nil {
		return map[string]interface{}{
			"MaxSteps":     100,
			"DeviceID":     "",
			"SystemPrompt": "",
			"Verbose":      true,
		}
	}
	return map[string]interface{}{
		"MaxSteps":     c.Agent.MaxSteps,
		"DeviceID":     c.Agent.DeviceID,
		"SystemPrompt": c.Agent.SystemPrompt,
		"Verbose":      c.Agent.Verbose,
	}
}

// ConvertToDecisionConfig 将 Config 中的 DecisionConfig 转换为 model.DecisionConfig
func (c *Config) ConvertToDecisionConfig() interface{} {
	if c.Decision == nil {
		c.Decision = &DecisionConfig{}
	}

	decision := map[string]interface{}{}
	if c.Decision.Decision != nil {
		decision = map[string]interface{}{
			"BaseURL":          c.Decision.Decision.BaseURL,
			"APIKey":           c.Decision.Decision.APIKey,
			"ModelName":        c.Decision.Decision.ModelName,
			"MaxTokens":        c.Decision.Decision.MaxTokens,
			"Temperature":      c.Decision.Decision.Temperature,
			"TopP":             c.Decision.Decision.TopP,
			"FrequencyPenalty": c.Decision.Decision.FrequencyPenalty,
		}
	}

	vision := map[string]interface{}{}
	if c.Decision.Vision != nil {
		vision = map[string]interface{}{
			"BaseURL":          c.Decision.Vision.BaseURL,
			"APIKey":           c.Decision.Vision.APIKey,
			"ModelName":        c.Decision.Vision.ModelName,
			"MaxTokens":        c.Decision.Vision.MaxTokens,
			"Temperature":      c.Decision.Vision.Temperature,
			"TopP":             c.Decision.Vision.TopP,
			"FrequencyPenalty": c.Decision.Vision.FrequencyPenalty,
		}
	}

	return map[string]interface{}{
		"Decision": decision,
		"Vision":   vision,
	}
}

// LoadConfig 从文件加载配置
func LoadConfig(configPath string) (*Config, error) {
	// 如果未指定配置文件，尝试查找默认配置文件
	if configPath == "" {
		configPath = FindConfigFile()
		if configPath == "" {
			// 返回默认配置
			return DefaultConfig(), nil
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return config, nil
}

// SaveConfig 保存配置到文件
func SaveConfig(config *Config, configPath string) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// FindConfigFile 查找配置文件
func FindConfigFile() string {
	// 1. 当前目录
	if _, err := os.Stat("config.yaml"); err == nil {
		return "config.yaml"
	}

	// 2. 用户主目录
	homeDir, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(homeDir, ".phone-agent", "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	// 3. 可执行文件所在目录
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		configPath := filepath.Join(exeDir, "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	return ""
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Decision != nil {
		if c.Decision.Decision != nil {
			if c.Decision.Decision.BaseURL == "" {
				return fmt.Errorf("decision.base-url is required")
			}
		}
		if c.Decision.Vision != nil {
			if c.Decision.Vision.BaseURL == "" {
				return fmt.Errorf("vision.base-url is required")
			}
		}
	}
	return nil
}

// MergeWithFlags 将配置与命令行参数合并
func (c *Config) MergeWithFlags(flags *Flags) {
	if flags == nil {
		return
	}

	// Agent 配置
	if c.Agent == nil {
		c.Agent = &AgentConfig{
			MaxSteps: 100,
			Verbose:  true,
		}
	}
	if flags.MaxSteps > 0 {
		c.Agent.MaxSteps = flags.MaxSteps
	}
	if flags.DeviceID != "" {
		c.Agent.DeviceID = flags.DeviceID
	}
	if flags.Quiet {
		c.Agent.Verbose = false
	}

	// Decision 配置
	if c.Decision == nil {
		c.Decision = &DecisionConfig{}
	}
	if c.Decision.Decision == nil {
		c.Decision.Decision = &ModelConfig{}
	}
	if c.Decision.Vision == nil {
		c.Decision.Vision = &ModelConfig{}
	}

	if flags.DecisionURL != "" {
		c.Decision.Decision.BaseURL = flags.DecisionURL
	}
	if flags.DecisionKey != "" {
		c.Decision.Decision.APIKey = flags.DecisionKey
	}
	if flags.DecisionModel != "" {
		c.Decision.Decision.ModelName = flags.DecisionModel
	}

	if flags.VisionURL != "" {
		c.Decision.Vision.BaseURL = flags.VisionURL
	}
	if flags.VisionKey != "" {
		c.Decision.Vision.APIKey = flags.VisionKey
	}
	if flags.VisionModel != "" {
		c.Decision.Vision.ModelName = flags.VisionModel
	}
}

// Flags 命令行参数结构
type Flags struct {
	MaxSteps       int
	DeviceID       string
	Quiet          bool
	LogEnabled     bool
	ListDevices    bool
	Connect        string
	Disconnect     string
	DecisionURL    string
	DecisionKey    string
	DecisionModel  string
	VisionURL      string
	VisionKey      string
	VisionModel    string
	ConfigFile     string
}

// GetAPIKeysFromEnv 从环境变量获取 API 密钥
func (c *Config) GetAPIKeysFromEnv() {
	// Decision API Key
	if c.Decision != nil && c.Decision.Decision != nil {
		if c.Decision.Decision.APIKey == "" || c.Decision.Decision.APIKey == "EMPTY" {
			if apiKey := os.Getenv("DECISION_API_KEY"); apiKey != "" {
				c.Decision.Decision.APIKey = apiKey
			}
		}
	}

	// Vision API Key
	if c.Decision != nil && c.Decision.Vision != nil {
		if c.Decision.Vision.APIKey == "" || c.Decision.Vision.APIKey == "EMPTY" {
			if apiKey := os.Getenv("VISION_API_KEY"); apiKey != "" {
				c.Decision.Vision.APIKey = apiKey
			}
		}
	}

	// Device ID from env
	if c.Agent != nil {
		if c.Agent.DeviceID == "" {
			if deviceID := os.Getenv("PHONE_AGENT_DEVICE_ID"); deviceID != "" {
				c.Agent.DeviceID = deviceID
			}
		}
	}
}

// RedactSensitiveInfo 隐藏敏感信息（用于显示）
func (c *Config) RedactSensitiveInfo() string {
	redacted := *c

	if redacted.Decision != nil {
		if redacted.Decision.Decision != nil && redacted.Decision.Decision.APIKey != "" {
			redacted.Decision.Decision.APIKey = maskAPIKey(redacted.Decision.Decision.APIKey)
		}
		if redacted.Decision.Vision != nil && redacted.Decision.Vision.APIKey != "" {
			redacted.Decision.Vision.APIKey = maskAPIKey(redacted.Decision.Vision.APIKey)
		}
	}

	data, _ := yaml.Marshal(redacted)
	return string(data)
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}
