package agent

// AgentConfig 配置 PhoneAgent 的行为
type AgentConfig struct {
	MaxSteps    int    // 每个任务最大步数
	DeviceID    string // ADB 设备 ID,为空则自动检测
	Lang        string // 语言: cn/en
	SystemPrompt string // 自定义系统提示词
	Verbose     bool   // 是否打印调试信息
}

// DefaultAgentConfig 返回默认配置
func DefaultAgentConfig() *AgentConfig {
	return &AgentConfig{
		MaxSteps:    100,
		DeviceID:    "",
		Lang:        "cn",
		SystemPrompt: "",
		Verbose:     true,
	}
}
