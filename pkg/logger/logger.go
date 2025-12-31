package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Logger 结构体负责处理多路日志输出（文件和控制台）
type Logger struct {
	file    *os.File    // 日志文件句柄，用于后期关闭
	fileLog *log.Logger // 封装好的文件日志记录器
	stdLog  *log.Logger // 封装好的控制台日志记录器（仅在调试模式开启）
}

// New 初始化日志系统
// isDebug: 是否在控制台同步输出
// logName: 日志文件名（存储在程序同级目录下）
func New(isDebug bool, logName string) *Logger {
	l := &Logger{}

	// 1. 获取程序运行路径，确保日志文件生成在程序所在目录
	ex, err := os.Executable()
	if err != nil {
		// 如果无法获取路径，退而求其次使用当前目录
		ex = "."
	}
	logPath := filepath.Join(filepath.Dir(ex), logName)

	// 2. 打开日志文件：创建、追加、只写模式
	// 权限 0644：文件所有者可读写，其他用户只读
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		// 如果日志文件无法打开，为了不影响程序运行，只输出到控制台并警告
		fmt.Printf("[CRITICAL] 无法创建日志文件 %s: %v\n", logPath, err)
	} else {
		l.file = f
		// 使用标准库 log 封装文件输出，自动处理时间戳和并发安全
		l.fileLog = log.New(f, "", log.LstdFlags)
	}

	// 3. 如果开启调试模式，初始化控制台输出
	if isDebug {
		l.stdLog = log.New(os.Stdout, "", log.LstdFlags)
	}

	return l
}

// Log 核心记录函数，支持格式化输入
func (l *Logger) Log(level, format string, v ...interface{}) {
	msg := fmt.Sprintf("[%s] %s", level, fmt.Sprintf(format, v...))

	// 写入文件（自带互斥锁安全）
	if l.fileLog != nil {
		l.fileLog.Println(msg)
	}

	// 写入控制台
	if l.stdLog != nil {
		l.stdLog.Println(msg)
	}
}

// Info 记录普通信息
func (l *Logger) Info(f string, v ...interface{}) {
	l.Log("INFO", f, v...)
}

// Error 记录错误信息
func (l *Logger) Error(f string, v ...interface{}) {
	l.Log("ERROR", f, v...)
}

// Close 关闭日志文件，在 main 函数退出前通过 defer 调用
func (l *Logger) Close() {
	if l.file != nil {
		// 强制将缓冲区数据刷入磁盘并关闭文件
		_ = l.file.Sync()
		_ = l.file.Close()
	}
}
