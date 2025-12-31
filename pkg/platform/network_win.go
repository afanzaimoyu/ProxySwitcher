package platform

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	// IfTypeIEEE80211  IEEE 802.11 无线网络接口的标准类型代码
	IfTypeIEEE80211 = 71
	// MaxGetAdaptersRetries 限制因缓冲区不足导致的 API 重试次数
	MaxGetAdaptersRetries = 3
)

// GetAdaptersAddresses 调用 Windows GetAdaptersAddresses API 获取系统适配器信息。
// 该函数被导出（首字母大写），以便 manager 包获取快照。
func GetAdaptersAddresses() ([]*windows.IpAdapterAddresses, error) {
	var size uint32 = 15000 // 初始分配 15KB
	var err error
	var b []byte

	for i := 0; i < MaxGetAdaptersRetries; i++ {
		b = make([]byte, size)
		ptr := (*windows.IpAdapterAddresses)(unsafe.Pointer(&b[0]))

		// 执行系统调用
		err = windows.GetAdaptersAddresses(windows.AF_UNSPEC, windows.GAA_FLAG_INCLUDE_PREFIX, 0, ptr, &size)

		if err == nil {
			return parseAdapterChain(ptr), nil
		}

		if !errors.Is(err, syscall.ERROR_BUFFER_OVERFLOW) {
			return nil, fmt.Errorf("GetAdaptersAddresses 系统调用失败: %w", err)
		}
	}

	return nil, fmt.Errorf("获取适配器列表失败：超过最大重试次数")
}

// UTF16PtrToString 一个工具函数，方便外部包转换 Windows UTF-16 指针。
func UTF16PtrToString(p *uint16) string {
	return windows.UTF16PtrToString(p)
}

// parseAdapterChain 将 Windows 链表结构转换为 Go 切片。
func parseAdapterChain(ptr *windows.IpAdapterAddresses) []*windows.IpAdapterAddresses {
	var adapters []*windows.IpAdapterAddresses
	for a := ptr; a != nil; a = a.Next {
		adapters = append(adapters, a)
	}
	return adapters
}

// ILogger 定义日志接口，解耦具体实现。
type ILogger interface {
	Info(string, ...interface{})
	Error(string, ...interface{})
}

// IProxyManager 定义业务管理器接口。
type IProxyManager interface {
	CheckAndApply()
}

// WatchWithInterrupt 监听中断信号并定时执行监测任务。
func WatchWithInterrupt(mgr IProxyManager, l ILogger) {
	l.Info("=== 开启代理自动监测 (间隔: 5秒) ===")
	l.Info("=== 按下 Ctrl+C 退出并重置系统代理 ===")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	mgr.CheckAndApply()

	for {
		select {
		case <-ticker.C:
			mgr.CheckAndApply()
		case sig := <-sigChan:
			l.Info("接收到系统信号 [%v]，准备清理现场...", sig)
			if err := CleanProxyRegistry(); err != nil {
				l.Error("清理注册表失败: %v", err)
			} else {
				l.Info("系统代理已重置为初始状态。")
			}
			l.Info("程序安全退出。")
			return
		}
	}
}
