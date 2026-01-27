package adb

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/disintegration/imaging"
)

// Screenshot 截图信息
type Screenshot struct {
	Base64Data   string // base64 编码的图片数据
	Width        int    // 屏幕宽度
	Height       int    // 屏幕高度
	IsSensitive  bool   // 是否为敏感页面
}

// GetScreenshot 获取设备截图
func GetScreenshot(deviceID string, timeout int) (*Screenshot, error) {
	// 构建命令前缀
	cmdPrefix := buildADBPrefix(deviceID)

	// 执行截图到设备
	tempPath := "/sdcard/tmp.png"
	cmd := exec.Command(cmdPrefix[0], append(cmdPrefix[1:], "shell", "screencap", "-p", tempPath)...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// 检查是否是敏感页面
		if strings.Contains(stderr.String(), "Status: -1") || strings.Contains(stderr.String(), "Failed") {
			return createFallbackScreenshot(true), nil
		}
		return nil, fmt.Errorf("screenshot failed: %w, stderr: %s", err, stderr.String())
	}

	// 拉取截图到本地
	localTempPath := fmt.Sprintf("%s%s%d.png", os.TempDir(), string(os.PathSeparator), time.Now().UnixNano())
	cmd = exec.Command(cmdPrefix[0], append(cmdPrefix[1:], "pull", tempPath, localTempPath)...)

	if err := cmd.Run(); err != nil {
		return createFallbackScreenshot(false), nil
	}

	// 读取图片
	img, err := imaging.Open(localTempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer os.Remove(localTempPath)

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 转换为 base64
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, img, imaging.PNG); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	base64Data := base64.StdEncoding.EncodeToString(buf.Bytes())

	return &Screenshot{
		Base64Data:  base64Data,
		Width:       width,
		Height:      height,
		IsSensitive: false,
	}, nil
}

// createFallbackScreenshot 创建黑色占位图
func createFallbackScreenshot(isSensitive bool) *Screenshot {
	width, height := 1080, 2400

	img := imaging.New(width, height, image.Black)
	var buf bytes.Buffer
	imaging.Encode(&buf, img, imaging.PNG)

	base64Data := base64.StdEncoding.EncodeToString(buf.Bytes())

	return &Screenshot{
		Base64Data:  base64Data,
		Width:       width,
		Height:      height,
		IsSensitive:  isSensitive,
	}
}

// GetScreenSize 获取屏幕分辨率
func GetScreenSize(deviceID string) (int, int, error) {
	cmdPrefix := buildADBPrefix(deviceID)
	cmd := exec.Command(cmdPrefix[0], append(cmdPrefix[1:], "shell", "wm", "size")...)

	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get screen size: %w", err)
	}

	// 解析输出: Physical size: 1080x2400
	strOutput := string(output)
	var width, height int
	_, err = fmt.Sscanf(strOutput, "Physical size: %dx%d", &width, &height)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse screen size: %w", err)
	}

	return width, height, nil
}
