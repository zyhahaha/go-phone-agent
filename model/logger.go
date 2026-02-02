package model

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	logFile     *os.File
	logOnce     sync.Once
	logMutex    sync.Mutex
	logWriter   *os.File
	consoleOnly bool = false // 是否只输出到控制台（不写入文件）
)

// InitLogger 初始化日志文件
func InitLogger() error {
	var initErr error
	logOnce.Do(func() {
		// 获取执行文件所在目录
		execPath, err := os.Executable()
		if err != nil {
			initErr = fmt.Errorf("failed to get executable path: %w", err)
			return
		}

		// 使用 filepath.Abs 确保获取到绝对路径
		execAbsPath, err := filepath.Abs(execPath)
		if err != nil {
			initErr = fmt.Errorf("failed to get absolute path: %w", err)
			return
		}
		execDir := filepath.Dir(execAbsPath)

		// 在执行文件同级目录创建 logs 目录
		logsDir := filepath.Join(execDir, "logs")
		if err := os.MkdirAll(logsDir, 0755); err != nil {
			initErr = fmt.Errorf("failed to create logs directory: %w", err)
			return
		}

		// 创建日志文件，使用时间戳命名
		timestamp := time.Now().Format("20060102-150405")
		filename := filepath.Join(logsDir, fmt.Sprintf("agent-%s.log", timestamp))

		file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			initErr = fmt.Errorf("failed to open log file: %w", err)
			return
		}

		logFile = file
		logWriter = file

		// 写入日志开始标记
		LogInfo("=== 日志文件初始化完成 ===")
		LogInfo("执行文件路径: " + execAbsPath)
		LogInfo("日志目录: " + logsDir)
	})

	return initErr
}

// CloseLogger 关闭日志文件
func CloseLogger() error {
	logMutex.Lock()
	defer logMutex.Unlock()

	if logWriter != nil {
		LogInfo("=== 日志文件关闭 ===")
		err := logWriter.Close()
		logWriter = nil
		logFile = nil
		return err
	}
	return nil
}

// LogInfo 写入普通日志
func LogInfo(msg string) {
	writeLog(fmt.Sprintf("[INFO] %s", msg))
}

// LogDebug 写入调试日志
func LogDebug(msg string) {
	writeLog(fmt.Sprintf("[DEBUG] %s", msg))
}

// LogSection 写入分区日志（带分隔线）
func LogSection(title string) {
	separator := "========================================"
	writeLog(fmt.Sprintf("%s %s %s", separator, title, separator))
}

// LogStart 开始一个日志区块
func LogStart(title string) {
	LogSection(title + " Start")
}

// LogEnd 结束一个日志区块
func LogEnd(title string) {
	LogSection(title + " End")
}

// LogContent 写入内容日志
func LogContent(content interface{}) {
	writeLog(fmt.Sprintf("%+v", content))
}

// writeLog 实际写入日志的函数（线程安全）
func writeLog(msg string) {
	logMutex.Lock()
	defer logMutex.Unlock()

	// 如果是控制台模式，不写入文件
	if consoleOnly {
		fmt.Printf("%s\n", msg)
		return
	}

	if logWriter == nil {
		// 如果日志文件未初始化，输出到控制台
		fmt.Printf("%s\n", msg)
		return
	}

	// 添加时间戳
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	logMsg := fmt.Sprintf("%s %s\n", timestamp, msg)

	// 写入文件
	if _, err := logWriter.WriteString(logMsg); err != nil {
		fmt.Printf("日志写入失败: %v\n", err)
	}

	// 同时输出到控制台
	fmt.Println(logMsg)
}

// SetConsoleOnly 设置为仅控制台模式（不写入文件）
func SetConsoleOnly(only bool) {
	logMutex.Lock()
	defer logMutex.Unlock()
	consoleOnly = only
}

// GetLogFile 获取日志文件路径
func GetLogFile() string {
	if logFile == nil {
		return ""
	}
	return logFile.Name()
}
