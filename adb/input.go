package adb

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
)

// TypeText 输入文本
func TypeText(text, deviceID string) error {
	// 切换到 ADB Keyboard
	originalIME, err := detectAndSetADBKeyboard(deviceID)
	if err != nil {
		return fmt.Errorf("failed to switch keyboard: %w", err)
	}
	defer restoreKeyboard(originalIME, deviceID)

	// 清空文本框
	if err := ClearText(deviceID); err != nil {
		return fmt.Errorf("failed to clear text: %w", err)
	}

	// 输入文本 - 使用 base64 编码
	encodedText := base64.StdEncoding.EncodeToString([]byte(text))
	cmdPrefix := buildADBPrefix(deviceID)
	args := append(cmdPrefix[1:], "shell", "am", "broadcast", "-a", "ADB_INPUT_B64", "--es", "msg", encodedText)
	cmd := exec.Command(cmdPrefix[0], args...)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("type text failed: %w", err)
	}

	return nil
}

// ClearText 清空输入框
func ClearText(deviceID string) error {
	cmdPrefix := buildADBPrefix(deviceID)
	args := append(cmdPrefix[1:], "shell", "am", "broadcast", "-a", "ADB_CLEAR_TEXT")
	cmd := exec.Command(cmdPrefix[0], args...)
	return cmd.Run()
}

// detectAndSetADBKeyboard 检测并设置 ADB Keyboard
func detectAndSetADBKeyboard(deviceID string) (string, error) {
	cmdPrefix := buildADBPrefix(deviceID)

	// 获取当前输入法
	args := append(cmdPrefix[1:], "shell", "settings", "get", "secure", "default_input_method")
	cmd := exec.Command(cmdPrefix[0], args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	currentIME := strings.TrimSpace(string(output))

	// 检查是否已经是 ADB Keyboard
	if strings.Contains(currentIME, "com.android.adbkeyboard/.AdbIME") {
		return currentIME, nil
	}

	// 切换到 ADB Keyboard
	args = append(cmdPrefix[1:], "shell", "ime", "set", "com.android.adbkeyboard/.AdbIME")
	cmd = exec.Command(cmdPrefix[0], args...)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to set ADB keyboard: %w", err)
	}

	return currentIME, nil
}

// restoreKeyboard 恢复原始输入法
func restoreKeyboard(originalIME, deviceID string) error {
	if originalIME == "" {
		return nil
	}

	cmdPrefix := buildADBPrefix(deviceID)
	args := append(cmdPrefix[1:], "shell", "ime", "set", originalIME)
	cmd := exec.Command(cmdPrefix[0], args...)
	return cmd.Run()
}

// CheckADBKeyboard 检查 ADB Keyboard 是否已安装
func CheckADBKeyboard(deviceID string) bool {
	cmdPrefix := buildADBPrefix(deviceID)
	args := append(cmdPrefix[1:], "shell", "ime", "list", "-s")
	cmd := exec.Command(cmdPrefix[0], args...)

	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "com.android.adbkeyboard/.AdbIME")
}

