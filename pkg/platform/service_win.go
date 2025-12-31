package platform

import (
	"ProxySwitcher/pkg/logger"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// ServiceRunner 实现了 Windows 服务的核心接口
type ServiceRunner struct {
	Handler func()         // 实际要循环执行的业务逻辑 (CheckAndApply)
	Logger  *logger.Logger // 日志句柄
}

// Execute 是服务的核心循环，由 Windows 服务管理器调用
func (m *ServiceRunner) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	// 定义服务支持的控制命令：停止、关机
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	s <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	m.Logger.Info("服务逻辑循环已启动")

loop:
	for {
		select {
		case <-ticker.C:
			// 执行心跳检查
			m.Handler()
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				// 系统询问当前状态
				s <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				m.Logger.Info("接收到停止信号，正在清理环境...")
				// 退出：停止前先重置代理，防止残留
				if err := CleanProxyRegistry(); err != nil {
					m.Logger.Error("服务停止时重置注册表失败: %v", err)
				}
				break loop
			default:
				m.Logger.Error("接收到未预料的控制请求: #%d", c.Cmd)
			}
		}
	}
	// 报告服务已停止
	s <- svc.Status{State: svc.Stopped}
	return false, 0
}

// IsService 动态判断当前进程是否作为 Windows 服务运行
func IsService() (bool, error) {
	return svc.IsWindowsService()
}

// RunAsService 将业务逻辑注册到 Windows 服务控制器
func RunAsService(name string, mgrObj interface{ CheckAndApply() }, l *logger.Logger) {
	err := svc.Run(name, &ServiceRunner{
		Handler: mgrObj.CheckAndApply,
		Logger:  l,
	})
	if err != nil {
		l.Error("服务运行失败: %v", err)
	}
}

// InstallService 负责将本程序注册为系统自启动服务
func InstallService() error {
	const name = "GoProxySwitcher"
	exepath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取程序路径失败: %w", err)
	}

	// 连接 Windows 服务管理器 (SCM)
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("无法连接服务管理器 (需管理员权限): %w", err)
	}
	defer m.Disconnect()

	// 检查服务是否已存在
	s, err := m.OpenService(name)
	if err == nil {
		s.Close()
		return fmt.Errorf("服务 [%s] 已经安装，请勿重复操作", name)
	}

	// 创建服务配置
	s, err = m.CreateService(name, exepath, mgr.Config{
		DisplayName: "Go 代理自动切换服务",
		Description: "根据有线/无线网卡环境自动感知并配置系统代理",
		StartType:   mgr.StartAutomatic, // 开机自启
	})
	if err != nil {
		return fmt.Errorf("注册服务失败: %w", err)
	}
	defer s.Close()

	// 4. 立即启动服务
	if err := s.Start(); err != nil {
		return fmt.Errorf("服务创建成功但启动失败: %w", err)
	}

	return nil
}

// UninstallService 执行“彻底清理”逻辑：重置注册表 -> 删服务 -> 删日志
func UninstallService(logPath string) error {
	const name = "GoProxySwitcher"

	// --- 重置注册表 ---
	// 无论后面成不成功，先把代理关了，保证正常上网
	if err := CleanProxyRegistry(); err != nil {
		fmt.Printf("[警告] 注册表清理失败: %v\n", err)
	}

	// --- 移除 Windows 服务 ---
	if err := stopAndRemoveService(name); err != nil {
		fmt.Printf("[警告] 服务移除失败 (可能服务未运行): %v\n", err)
	}

	// --- 移除日志文件 ---
	if err := safeRemoveFile(logPath); err != nil {
		return fmt.Errorf("服务已卸载，但日志清理出错: %w", err)
	}

	return nil
}

// stopAndRemoveService 封装了停止并删除服务的细节
func stopAndRemoveService(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return err // 服务不存在直接返回
	}
	defer s.Close()

	// 尝试停止服务 (忽略报错，因为服务可能已经停止)
	_, _ = s.Control(svc.Stop)

	// 给一点缓冲时间让进程释放资源
	time.Sleep(500 * time.Millisecond)

	return s.Delete()
}

// safeRemoveFile 防御性删除文件
func safeRemoveFile(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil // 文件本来就不存在，直接成功
	}
	if err != nil {
		return fmt.Errorf("无法读取文件状态: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("目标路径是文件夹而非日志文件")
	}

	// 执行删除
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("文件删除被拒绝 (可能正在被其他程序占用): %w", err)
	}
	return nil
}
