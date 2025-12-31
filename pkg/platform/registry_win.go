package platform

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

const (
	// RegSettings 定义了 Windows 系统代理设置在注册表中的核心路径。
	RegSettings = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

	// INTERNET_OPTION_SETTINGS_CHANGED 通知系统设置已更改，强制刷新。
	INTERNET_OPTION_SETTINGS_CHANGED = 39
	// INTERNET_OPTION_REFRESH 强制从注册表重新读取设置。
	INTERNET_OPTION_REFRESH = 37
)

var (
	wininet               = syscall.NewLazyDLL("wininet.dll")
	procInternetSetOption = wininet.NewProc("InternetSetOptionW")
)

// UpdateProxyRegistry 执行系统代理状态的更新，并通知系统立即生效。
func UpdateProxyRegistry(enable bool, address string) error {
	// 1. 以全权模式打开注册表键
	key, err := registry.OpenKey(registry.CURRENT_USER, RegSettings, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("无法访问代理注册表键 (请尝试以管理员权限运行): %w", err)
	}
	defer key.Close()

	// 2. 更新注册表数值
	if enable {
		if err := key.SetDWordValue("ProxyEnable", 1); err != nil {
			return fmt.Errorf("启用 ProxyEnable 失败: %w", err)
		}
		if err := key.SetStringValue("ProxyServer", address); err != nil {
			return fmt.Errorf("写入 ProxyServer 地址 [%s] 失败: %w", err)
		}
	} else {
		if err := key.SetDWordValue("ProxyEnable", 0); err != nil {
			return fmt.Errorf("关闭 ProxyEnable 失败: %w", err)
		}
	}

	// 3. 核心提升：强制通知 Windows 刷新设置
	// 修改注册表后，如果不调用此 API，某些浏览器进程会有缓存延迟。
	refreshSystemProxy()

	return nil
}

// CleanProxyRegistry 执行深层清理逻辑，将代理开关关闭并将地址项置为空。
func CleanProxyRegistry() error {
	if err := UpdateProxyRegistry(false, ""); err != nil {
		return fmt.Errorf("清理代理状态失败: %w", err)
	}

	// 显式抹除地址项以保持注册表整洁
	key, err := registry.OpenKey(registry.CURRENT_USER, RegSettings, registry.SET_VALUE)
	if err == nil {
		defer key.Close()
		_ = key.SetStringValue("ProxyServer", "")
	}

	// 再次触发刷新，确保清理动作立即生效
	refreshSystemProxy()
	return nil
}

// refreshSystemProxy 调用 Windows WinInet API 广播设置更改通知。
func refreshSystemProxy() {
	// 分两次通知：一次是设置已更改，一次是刷新缓存。
	// 这两行代码能让修改后的注册表在 Chrome/Edge 等浏览器中瞬间生效。
	_, _, _ = procInternetSetOption.Call(0, INTERNET_OPTION_SETTINGS_CHANGED, 0, 0)
	_, _, _ = procInternetSetOption.Call(0, INTERNET_OPTION_REFRESH, 0, 0)
}
