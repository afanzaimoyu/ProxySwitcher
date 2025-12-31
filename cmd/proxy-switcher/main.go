package main

import (
	"ProxySwitcher/pkg/logger"
	"ProxySwitcher/pkg/manager"
	"ProxySwitcher/pkg/platform"
	"flag"
	"fmt"
	"os"
)

// 全局配置常量
const (
	ServiceName  = "GoProxySwitcher"
	DefaultProxy = ""
	LogFileName  = "proxy_service.log"
)

func main() {
	// 1. 命令行参数定义
	isDebug := flag.Bool("debug", false, "开启调试模式 (显示黑窗口并输出日志)")
	isInstall := flag.Bool("install", false, "将程序安装为 Windows 系统服务")
	isUninstall := flag.Bool("uninstall", false, "卸载服务并清理所有残留 (设置与日志)")
	flag.Parse()

	// 2. 初始化核心组件 (日志与业务管理器)
	l := logger.New(*isDebug, LogFileName)
	defer l.Close()

	mgr := manager.NewProxyManager(DefaultProxy, l)

	// 3. 判断运行模式
	// 优先判断是否作为 Windows 服务运行 (系统自动调用)
	if inSvc, _ := platform.IsService(); inSvc {
		platform.RunAsService(ServiceName, mgr, l)
		return
	}

	// 4. 根据命令行参数分发任务 (使用 Switch 消除嵌套)
	switch {
	case *isInstall:
		runInstall(l)
	case *isUninstall:
		runUninstall(l)
	case *isDebug:
		runDebug(mgr, l)
	default:
		showUsage()
	}
}

// --- 任务处理器 (Handlers) ---

// runInstall 处理安装任务
func runInstall(l *logger.Logger) {
	l.Info("正在安装服务 [%s]...", ServiceName)
	if err := platform.InstallService(); err != nil {
		l.Error("安装过程中出现错误: %v", err)
		os.Exit(1)
	}
	l.Info("服务安装成功并已尝试启动。")
}

// runUninstall 处理卸载与清理任务
func runUninstall(l *logger.Logger) {
	l.Info("正在启动深度卸载流程...")
	// 这里的 UninstallService 已经包含了对注册表和日志文件的清理
	if err := platform.UninstallService(LogFileName); err != nil {
		l.Error("深度卸载未完全成功: %v", err)
		os.Exit(1)
	}
	l.Info("卸载完成：服务已删除，注册表已重置，日志文件已清理。")
}

// runDebug 处理本地调试任务
func runDebug(mgr *manager.ProxyManager, l *logger.Logger) {
	// 该函数封装了信号监听（Ctrl+C）和计时器循环
	platform.WatchWithInterrupt(mgr, l)
}

// showUsage 显示工具使用说明
func showUsage() {
	fmt.Printf("\nProxySwitcher - 智能代理自动切换工具\n")
	fmt.Printf("\nTODO：界面\n")
	fmt.Printf("\nTODO：\n")
	fmt.Println("------------------------------------------")
	fmt.Println("用法示例:")
	fmt.Println("  管理模式 (需管理员权限):")
	fmt.Println("    .\\ProxySwitcher.exe -install     安装并启动后台服务")
	fmt.Println("    .\\ProxySwitcher.exe -uninstall   彻底卸载服务并清理残留")
	fmt.Println("\n  开发模式:")
	fmt.Println("    .\\ProxySwitcher.exe -debug       在当前窗口运行并查看实时日志")
	fmt.Println("------------------------------------------")
}
