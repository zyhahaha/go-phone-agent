package adb

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"go-phone-agent/config"
)

// Tap 点击屏幕
func Tap(x, y int, deviceID string) error {
	cmdPrefix := buildADBPrefix(deviceID)
	cmd := exec.Command(cmdPrefix[0], append(cmdPrefix[1:], "shell", "input", "tap", strconv.Itoa(x), strconv.Itoa(y))...)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tap failed: %w", err)
	}

	time.Sleep(500 * time.Millisecond) // 默认延迟
	return nil
}

// DoubleTap 双击屏幕
func DoubleTap(x, y int, deviceID string) error {
	cmdPrefix := buildADBPrefix(deviceID)

	// 第一次点击
	cmd := exec.Command(cmdPrefix[0], append(cmdPrefix[1:], "shell", "input", "tap", strconv.Itoa(x), strconv.Itoa(y))...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("DoubleTap first failed: %w", err)
	}

	time.Sleep(100 * time.Millisecond) // 双击间隔

	// 第二次点击
	cmd = exec.Command(cmdPrefix[0], append(cmdPrefix[1:], "shell", "input", "tap", strconv.Itoa(x), strconv.Itoa(y))...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("DoubleTap second failed: %w", err)
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

// LongPress 长按屏幕
func LongPress(x, y int, durationMS int, deviceID string) error {
	cmdPrefix := buildADBPrefix(deviceID)
	cmd := exec.Command(cmdPrefix[0], append(cmdPrefix[1:], "shell", "input", "swipe",
		strconv.Itoa(x), strconv.Itoa(y), strconv.Itoa(x), strconv.Itoa(y), strconv.Itoa(durationMS))...)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("long press failed: %w", err)
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

// Swipe 滑动屏幕
func Swipe(startX, startY, endX, endY int, durationMS int, deviceID string) error {
	cmdPrefix := buildADBPrefix(deviceID)

	// 自动计算滑动时长
	if durationMS == 0 {
		distSquared := float64((startX-endX)*(startX-endX) + (startY-endY)*(startY-endY))
		durationMS = int(distSquared / 1000)
		if durationMS < 500 {
			durationMS = 500
		} else if durationMS > 2000 {
			durationMS = 2000
		}
	}

	cmd := exec.Command(cmdPrefix[0], append(cmdPrefix[1:], "shell", "input", "swipe",
		strconv.Itoa(startX), strconv.Itoa(startY),
		strconv.Itoa(endX), strconv.Itoa(endY),
		strconv.Itoa(durationMS))...)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("swipe failed: %w", err)
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

// Back 返回
func Back(deviceID string) error {
	cmdPrefix := buildADBPrefix(deviceID)
	cmd := exec.Command(cmdPrefix[0], append(cmdPrefix[1:], "shell", "input", "keyevent", "4")...)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("back failed: %w", err)
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

// Home 返回桌面
func Home(deviceID string) error {
	cmdPrefix := buildADBPrefix(deviceID)
	cmd := exec.Command(cmdPrefix[0], append(cmdPrefix[1:], "shell", "input", "keyevent", "KEYCODE_HOME")...)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("home failed: %w", err)
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

// LaunchApp 启动应用
func LaunchApp(appName, deviceID string) (bool, error) {
	packageName, ok := config.GetPackageName(appName)
	if !ok {
		return false, fmt.Errorf("app not found: %s", appName)
	}

	cmdPrefix := buildADBPrefix(deviceID)
	cmd := exec.Command(cmdPrefix[0], append(cmdPrefix[1:], "shell", "monkey", "-p", packageName,
		"-c", "android.intent.category.LAUNCHER", "1")...)

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("launch failed: %w", err)
	}

	time.Sleep(2000 * time.Millisecond) // 应用启动需要更长时间
	return true, nil
}

// GetCurrentApp 获取当前应用
func GetCurrentApp(deviceID string) string {
	cmdPrefix := buildADBPrefix(deviceID)
	cmd := exec.Command(cmdPrefix[0], append(cmdPrefix[1:], "shell", "dumpsys", "window")...)

	output, err := cmd.Output()
	if err != nil {
		return "System Home"
	}

	outputStr := string(output)

	// 查找当前焦点窗口
	for appName, packageName := range config.AppPackages {
		if strings.Contains(outputStr, packageName) {
			return appName
		}
	}

	return "System Home"
}

// buildADBPrefix 构建 ADB 命令前缀
func buildADBPrefix(deviceID string) []string {
	if deviceID != "" {
		return []string{"adb", "-s", deviceID}
	}
	return []string{"adb"}
}

// ListDevices 列出已连接的设备
func ListDevices() ([]string, error) {
	cmd := exec.Command("adb", "devices")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	devices := []string{}

	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "\tdevice") {
			parts := strings.Split(line, "\t")
			if len(parts) > 0 {
				devices = append(devices, parts[0])
			}
		}
	}

	return devices, nil
}

// ConnectDevice 连接远程设备
func ConnectDevice(address string) error {
	cmd := exec.Command("adb", "connect", address)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to connect device: %w", err)
	}
	return nil
}

// DisconnectDevice 断开设备连接
func DisconnectDevice(address string) error {
	cmd := exec.Command("adb", "disconnect", address)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disconnect device: %w", err)
	}
	return nil
}
