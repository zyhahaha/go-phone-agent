package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"go-phone-agent/adb"
	"go-phone-agent/agent"
	"go-phone-agent/model"
)

func main() {
	// 定义命令行参数
	maxSteps := flag.Int("max-steps", 100, "Maximum steps per task")
	deviceID := flag.String("device-id", os.Getenv("PHONE_AGENT_DEVICE_ID"), "ADB device ID")
	quiet := flag.Bool("quiet", false, "Suppress verbose output")
	logEnabled := flag.Bool("log", false, "Enable logging to file (default: disabled)")
	// listApps := flag.Bool("list-apps", false, "List supported apps and exit")
	listDevices := flag.Bool("list-devices", false, "List connected devices and exit")
	connect := flag.String("connect", "", "Connect to remote device (e.g., 192.168.1.100:5555)")
	disconnect := flag.String("disconnect", "", "Disconnect from remote device")
	// 调度器模式参数（双模型架构）
	schedulerURL := flag.String("scheduler-url", "https://api.deepseek.com", "Scheduler (DeepSeek) API base URL")
	schedulerKey := flag.String("scheduler-key", os.Getenv("SCHEDULER_API_KEY"), "Scheduler (DeepSeek) API key")
	schedulerModel := flag.String("scheduler-model", "deepseek-chat", "Scheduler (DeepSeek) model name")
	visionURL := flag.String("vision-url", "https://open.bigmodel.cn/api/paas/v4", "Vision (autoglm-phone) API base URL")
	visionKey := flag.String("vision-key", os.Getenv("VISION_API_KEY"), "Vision (autoglm-phone) API key")
	visionModel := flag.String("vision-model", "autoglm-phone", "Vision (autoglm-phone) model name")

	flag.Parse()

	// 初始化日志系统（仅在 -log 参数启用时）
	if *logEnabled {
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
	if *listDevices {
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
	if *connect != "" {
		fmt.Printf("Connecting to %s...\n", *connect)
		if err := adb.ConnectDevice(*connect); err != nil {
			fmt.Printf("✗ Failed to connect: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Connected to %s\n", *connect)
		*deviceID = *connect
	}

	// 断开设备
	if *disconnect != "" {
		fmt.Printf("Disconnecting from %s...\n", *disconnect)
		if err := adb.DisconnectDevice(*disconnect); err != nil {
			fmt.Printf("✗ Failed to disconnect: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Disconnected from %s\n", *disconnect)
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

	if *deviceID == "" {
		*deviceID = devices[0]
	}

	// 检查 ADB Keyboard
	if !adb.CheckADBKeyboard(*deviceID) {
		fmt.Println("❌ ADB Keyboard is not installed on the device.")
		fmt.Println("Solution:")
		fmt.Println("  1. Download ADB Keyboard APK from:")
		fmt.Println("     https://github.com/senzhk/ADBKeyBoard/blob/master/ADBKeyboard.apk")
		fmt.Println("  2. Install it on your device: adb install ADBKeyboard.apk")
		fmt.Println("  3. Enable it in Settings > System > Languages & Input > Virtual Keyboard")
		os.Exit(1)
	}

	// 创建调度器模式配置（双模型架构）
	schedulerConfig := &model.SchedulerConfig{
		Scheduler: &model.ModelConfig{
			BaseURL:          *schedulerURL,
			APIKey:           *schedulerKey,
			ModelName:        *schedulerModel,
			MaxTokens:        2000,
			Temperature:      0.7,
			TopP:             0.9,
			FrequencyPenalty: 0.0,
		},
		Vision: &model.ModelConfig{
			BaseURL:          *visionURL,
			APIKey:           *visionKey,
			ModelName:        *visionModel,
			MaxTokens:        3000,
			Temperature:      0.0,
			TopP:             0.85,
			FrequencyPenalty: 0.2,
		},
	}

	agentConfig := &agent.AgentConfig{
		MaxSteps: *maxSteps,
		DeviceID: *deviceID,
		Verbose:  !*quiet,
	}

	phoneAgent := agent.NewPhoneAgentWithScheduler(schedulerConfig, agentConfig, nil, nil)

	// 打印头部
	fmt.Println("=" + strings.Repeat("=", 48))
	fmt.Println("Phone Agent - Scheduler Mode (DeepSeek + autoglm-phone)")
	fmt.Println("=" + strings.Repeat("=", 48))
	fmt.Printf("Scheduler Model: %s\n", *schedulerModel)
	fmt.Printf("Scheduler URL: %s\n", *schedulerURL)
	fmt.Printf("Vision Model: %s\n", *visionModel)
	fmt.Printf("Vision URL: %s\n", *visionURL)
	fmt.Printf("Max Steps: %d\n", *maxSteps)
	fmt.Printf("Device: %s\n", *deviceID)
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
