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
	baseURL := flag.String("base-url", os.Getenv("PHONE_AGENT_BASE_URL"), "Model API base URL")
	modelName := flag.String("model", os.Getenv("PHONE_AGENT_MODEL"), "Model name")
	apiKey := flag.String("apikey", os.Getenv("PHONE_AGENT_API_KEY"), "API key")
	maxSteps := flag.Int("max-steps", 100, "Maximum steps per task")
	deviceID := flag.String("device-id", os.Getenv("PHONE_AGENT_DEVICE_ID"), "ADB device ID")
	lang := flag.String("lang", "cn", "Language (cn/en)")
	quiet := flag.Bool("quiet", false, "Suppress verbose output")
	listApps := flag.Bool("list-apps", false, "List supported apps and exit")
	listDevices := flag.Bool("list-devices", false, "List connected devices and exit")
	connect := flag.String("connect", "", "Connect to remote device (e.g., 192.168.1.100:5555)")
	disconnect := flag.String("disconnect", "", "Disconnect from remote device")

	flag.Parse()

	// 列出支持的应用
	if *listApps {
		fmt.Println("Supported apps:")
		for _, app := range config.ListSupportedApps() {
			fmt.Printf("  - %s\n", app)
		}
		return
	}

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

	// 创建配置
	modelConfig := &model.ModelConfig{
		BaseURL:          *baseURL,
		APIKey:           *apiKey,
		ModelName:        *modelName,
		MaxTokens:        3000,
		Temperature:      0.0,
		TopP:             0.85,
		FrequencyPenalty: 0.2,
	}

	agentConfig := &agent.AgentConfig{
		MaxSteps: *maxSteps,
		DeviceID: *deviceID,
		Lang:     *lang,
		Verbose:  !*quiet,
	}

	// 创建 Agent
	phoneAgent := agent.NewPhoneAgent(modelConfig, agentConfig, nil, nil)

	// 打印头部
	fmt.Println("=" + strings.Repeat("=", 48))
	fmt.Println("Phone Agent - AI-powered phone automation")
	fmt.Println("=" + strings.Repeat("=", 48))
	fmt.Printf("Model: %s\n", *modelName)
	fmt.Printf("Base URL: %s\n", *baseURL)
	fmt.Printf("Max Steps: %d\n", *maxSteps)
	fmt.Printf("Language: %s\n", *lang)
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
		fmt.Println("\nEntering interactive mode. Type 'quit' to exit.\n")

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
