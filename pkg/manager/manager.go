package manager

import (
	"ProxySwitcher/pkg/platform"
	"net"
	"strings"
)

// 定义代理状态常量
const (
	StateUnknown = -1
	StateOff     = 0
	StateOn      = 1

	// DebounceThreshold 防抖阈值：连续 2 次检测一致才触发切换，减少瞬时网络波动干扰
	DebounceThreshold = 2
)

// ILogger 定义了 Manager 需要的日志能力
type ILogger interface {
	Info(string, ...interface{})
	Error(string, ...interface{})
}

// ProxyManager 负责环境感知与代理状态决策
type ProxyManager struct {
	ProxyAddress string  // 目标代理地址
	Logger       ILogger // 日志接口

	lastState      int // 最终确认的状态
	pendingState   int // 正在观察中的候选状态
	consecutiveCnt int // 候选状态连续出现的次数（用于防抖）
}

// NewProxyManager 创建并初始化管理器
func NewProxyManager(addr string, l ILogger) *ProxyManager {
	return &ProxyManager{
		ProxyAddress:   addr,
		Logger:         l,
		lastState:      StateUnknown,
		pendingState:   StateUnknown,
		consecutiveCnt: 0,
	}
}

// CheckAndApply 执行环境扫描并根据加权逻辑决策代理开关
func (m *ProxyManager) CheckAndApply() {
	// 1. 扫描当前网络环境并获取最优目标状态
	target := m.evaluateEnvironment()

	// 2. 状态机防抖逻辑
	if target == m.lastState {
		// 状态稳定，重置观察计数
		m.consecutiveCnt = 0
		m.pendingState = target
		return
	}

	// 如果目标状态发生了变化，进入观察期
	if target != m.pendingState {
		m.pendingState = target
		m.consecutiveCnt = 1
	} else {
		m.consecutiveCnt++
	}

	// 达到防抖阈值，执行实质性切换
	if m.consecutiveCnt >= DebounceThreshold {
		m.applyState(target)
	}
}

// evaluateEnvironment 通过网卡加权计算当前最合理的代理状态
func (m *ProxyManager) evaluateEnvironment() int {
	var (
		hasCompanyWired = false // 权重2: 公司有线内网 (10.x)
		hasExternalWifi = false // 权重3: 外部热点 (非 10.x Wi-Fi)
	)
	//性能优化 在循环外只获取一次所有网卡的快照
	adapters, err := platform.GetAdaptersAddresses()
	if err != nil {
		m.Logger.Error("获取系统适配器快照失败: %v", err)
		return m.lastState
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		m.Logger.Error("扫描网卡失败: %v", err)
		return m.lastState // 报错时保持现状，不轻易改变代理
	}

	for _, iface := range interfaces {
		// 过滤无效网卡：未启动、回环接口、无地址接口
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// 逻辑优化 直接从快照里找当前这块网卡的硬件类型，不再重复扫描系统
		isWifi := false
		for _, a := range adapters {
			if platform.UTF16PtrToString(a.FriendlyName) == iface.Name {
				isWifi = a.IfType == platform.IfTypeIEEE80211
				break
			}
		}

		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ipv4 := ipnet.IP.To4()
			if ipv4 == nil {
				continue
			}

			ipStr := ipv4.String()
			isInternal := strings.HasPrefix(ipStr, "10.")

			// 加权判断决策逻辑：
			if isWifi {
				// 场景：连接了手机热点或外部公共 Wi-Fi
				if !isInternal {
					hasExternalWifi = true
				}
			} else {
				// 场景：插了公司网线，分到了 10.x 段 IP
				if isInternal {
					hasCompanyWired = true
				}
			}
		}
	}

	// --- 多接口加权冲突决策 ---
	// 策略：外部 Wi-Fi 优先级最高。一旦连接热点，无论是否插着公司网线，都关闭代理。
	if hasExternalWifi {
		return StateOff
	}
	if hasCompanyWired {
		return StateOn
	}
	return StateOff
}

// applyState 调用底层驱动修改注册表并通知系统
func (m *ProxyManager) applyState(state int) {
	m.Logger.Info("检测到环境变化，正在准备切换代理状态为: %v", stateDesc(state))

	enable := (state == StateOn)
	if err := platform.UpdateProxyRegistry(enable, m.ProxyAddress); err != nil {
		m.Logger.Error("核心切换动作失败: %v", err)
		return
	}

	m.lastState = state
	m.consecutiveCnt = 0
	m.Logger.Info(">>> 代理切换成功！当前状态: [%s], 目标地址: [%s]", stateDesc(state), m.ProxyAddress)
}

// stateDesc 辅助函数：将状态码转换为可读文字
func stateDesc(state int) string {
	switch state {
	case StateOn:
		return "开启"
	case StateOff:
		return "关闭"
	default:
		return "未知"
	}
}
