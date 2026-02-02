package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"go-phone-agent/adb"
	"go-phone-agent/agent"
	"go-phone-agent/config"
	"go-phone-agent/model"
)

func main() {
	// 定义命令行参数
	configFile := flag.String("config", "", "Path to config file (default: ./config.yaml or ~/.phone-agent/config.yaml)")
	maxSteps := flag.Int("max-steps", 0, "Maximum steps per task (overrides config)")
	deviceID := flag.String("device-id", "", "ADB device ID (overrides config)")
	quiet := flag.Bool("quiet", false, "Suppress verbose output")
	logEnabled := flag.Bool("log", false, "Enable logging to file (default: disabled)")
	// listApps := flag.Bool("list-apps", false, "List supported apps and exit")
	listDevices := flag.Bool("list-devices", false, "List connected devices and exit")
	connect := flag.String("connect", "", "Connect to remote device (e.g., 192.168.1.100:5555)")
	disconnect := flag.String("disconnect", "", "Disconnect from remote device")
	// 决策模型模式参数（双模型架构）
	decisionURL := flag.String("decision-url", "", "Decision model API base URL (overrides config)")
	decisionKey := flag.String("decision-key", "", "Decision model API key (overrides config)")
	decisionModel := flag.String("decision-model", "", "Decision model model name (overrides config)")
	visionURL := flag.String("vision-url", "", "Vision model API base URL (overrides config)")
	visionKey := flag.String("vision-key", "", "Vision model API key (overrides config)")
	visionModel := flag.String("vision-model", "", "Vision model model name (overrides config)")

	flag.Parse()

	// 加载配置文件
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 合并命令行参数（命令行参数优先级更高）
	flags := &config.Flags{
		MaxSteps:       *maxSteps,
		DeviceID:       *deviceID,
		Quiet:          *quiet,
		LogEnabled:     *logEnabled,
		ListDevices:    *listDevices,
		Connect:        *connect,
		Disconnect:     *disconnect,
		DecisionURL:    *decisionURL,
		DecisionKey:    *decisionKey,
		DecisionModel:  *decisionModel,
		VisionURL:      *visionURL,
		VisionKey:      *visionKey,
		VisionModel:    *visionModel,
		ConfigFile:     *configFile,
	}
	cfg.MergeWithFlags(flags)

	// 从环境变量获取 API 密钥
	cfg.GetAPIKeysFromEnv()

	// 初始化日志系统（仅在 -log 参数启用时）
	if flags.LogEnabled {
		if err := model.InitLogger(); err != nil {
			fmt.Printf("Warning: Failed to initialize logger: %v\n", err)
		}
		defer model.CloseLogger()
	} else {
		// 禁用日志到文件，只输出到控制台
		model.SetConsoleOnly(true)
	}

	// 列出支持的应用
	// if *listApps {
	// 	fmt.Println("Supported apps:")
	// 	for _, app := range config.ListSupportedApps() {
	// 		fmt.Printf("  - %s\n", app)
	// 	}
	// 	return
	// }

	// 列出设备
	if flags.ListDevices {
		devices, err := adb.ListDevices()
		if err != nil {
			fmt.Printf("Failed to list devices: %v\n", err)
			os.Exit(1)
		}

		if len(devices) == 0 {
			fmt.Println("No devices connected.")
			fmt.Println("\nTroubleshooting:")
			fmt.Println("  1. Enable USB debugging on your Android device")
			fmt.Println("  2. Connect via USB and authorize the connection")
			fmt.Println("  3. Run: adb devices")
		} else {
			fmt.Println("Connected devices:")
			fmt.Println("-" + strings.Repeat("-", 58))
			for _, dev := range devices {
				fmt.Printf("  ✓ %s\n", dev)
			}
		}
		return
	}

	// 连接远程设备
	if flags.Connect != "" {
		fmt.Printf("Connecting to %s...\n", flags.Connect)
		if err := adb.ConnectDevice(flags.Connect); err != nil {
			fmt.Printf("✗ Failed to connect: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Connected to %s\n", flags.Connect)
		cfg.Agent.DeviceID = flags.Connect
	}

	// 断开设备
	if flags.Disconnect != "" {
		fmt.Printf("Disconnecting from %s...\n", flags.Disconnect)
		if err := adb.DisconnectDevice(flags.Disconnect); err != nil {
			fmt.Printf("✗ Failed to disconnect: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Disconnected from %s\n", flags.Disconnect)
		return
	}

	// 检查 ADB 连接
	devices, err := adb.ListDevices()
	if err != nil {
		fmt.Printf("❌ Failed to check ADB connection: %v\n", err)
		os.Exit(1)
	}

	if len(devices) == 0 {
		fmt.Println("❌ No devices connected.")
		fmt.Println("Please connect your Android device and enable USB debugging.")
		os.Exit(1)
	}

	if cfg.Agent.DeviceID == "" {
		cfg.Agent.DeviceID = devices[0]
	}

	// 检查 ADB Keyboard
	if !adb.CheckADBKeyboard(cfg.Agent.DeviceID) {
		fmt.Println("❌ ADB Keyboard is not installed on the device.")
		fmt.Println("Solution:")
		fmt.Println("  1. Download ADB Keyboard APK from:")
		fmt.Println("     https://github.com/senzhk/ADBKeyBoard/blob/master/ADBKeyboard.apk")
		fmt.Println("  2. Install it on your device: adb install ADBKeyboard.apk")
		fmt.Println("  3. Enable it in Settings > System > Languages & Input > Virtual Keyboard")
		os.Exit(1)
	}

	// 将 config.Config 转换为 agent.AgentConfig 和 model.DecisionConfig
	agentConfig := &agent.AgentConfig{
		MaxSteps: cfg.Agent.MaxSteps,
		DeviceID: cfg.Agent.DeviceID,
		SystemPrompt: cfg.Agent.SystemPrompt,
		Verbose: cfg.Agent.Verbose,
	}

	decisionConfig := &model.DecisionConfig{
		Decision: &model.ModelConfig{
			BaseURL:          cfg.Decision.Decision.BaseURL,
			APIKey:           cfg.Decision.Decision.APIKey,
			ModelName:        cfg.Decision.Decision.ModelName,
			MaxTokens:        cfg.Decision.Decision.MaxTokens,
			Temperature:      cfg.Decision.Decision.Temperature,
			TopP:             cfg.Decision.Decision.TopP,
			FrequencyPenalty: cfg.Decision.Decision.FrequencyPenalty,
		},
		Vision: &model.ModelConfig{
			BaseURL:          cfg.Decision.Vision.BaseURL,
			APIKey:           cfg.Decision.Vision.APIKey,
			ModelName:        cfg.Decision.Vision.ModelName,
			MaxTokens:        cfg.Decision.Vision.MaxTokens,
			Temperature:      cfg.Decision.Vision.Temperature,
			TopP:             cfg.Decision.Vision.TopP,
			FrequencyPenalty: cfg.Decision.Vision.FrequencyPenalty,
		},
	}

	phoneAgent := agent.NewPhoneAgentWithDecisionModel(decisionConfig, agentConfig, nil, nil)

	// 打印配置信息
	fmt.Println("=" + strings.Repeat("=", 48))
	fmt.Println("Phone Agent - Decision Model Mode (Decision Model + Vision Model)")
	fmt.Println("=" + strings.Repeat("=", 48))
	if *configFile != "" {
		fmt.Printf("Config: %s\n", *configFile)
	} else {
		fmt.Printf("Config: Using default or auto-detected config\n")
	}
	fmt.Printf("Decision Model: %s\n", decisionConfig.Decision.ModelName)
	fmt.Printf("Decision URL: %s\n", decisionConfig.Decision.BaseURL)
	fmt.Printf("Vision Model: %s\n", decisionConfig.Vision.ModelName)
	fmt.Printf("Vision URL: %s\n", decisionConfig.Vision.BaseURL)
	fmt.Printf("Max Steps: %d\n", agentConfig.MaxSteps)
	fmt.Printf("Device: %s\n", agentConfig.DeviceID)
	fmt.Println("=" + strings.Repeat("=", 48))

	// 获取任务
	task := ""
	args := flag.Args()
	if len(args) > 0 {
		task = strings.Join(args, " ")
	}

	if task == "" {
		// 交互模式
		fmt.Println("Entering interactive mode. Type 'quit' to exit.")

		for {
			fmt.Print("Enter your task: ")
			var input string
			fmt.Scanln(&input)

			input = strings.TrimSpace(input)
			if input == "" {
				continue
			}

			if strings.ToLower(input) == "quit" || strings.ToLower(input) == "exit" || strings.ToLower(input) == "q" {
				fmt.Println("Goodbye!")
				break
			}

			fmt.Println()
			result := phoneAgent.Run(input)
			fmt.Printf("\nResult: %s\n\n", result)

			phoneAgent.Reset()
		}
	} else {
		// 单次任务模式
		fmt.Printf("\nTask: %s\n\n", task)
		result := phoneAgent.Run(task)
		fmt.Printf("\nResult: %s\n", result)
	}
}
