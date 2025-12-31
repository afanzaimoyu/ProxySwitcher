# ProxySwitcher

> Environment-aware Windows proxy auto switcher

---

## What is this?

**ProxySwitcher** is a lightweight Windows background tool that automatically enables or disables the system proxy based on the **actual network environment**.

Designed for developers who frequently switch between:

- Corporate intranet (10.x.x.x)
- Mobile hotspot or home Wi-Fi
- Mixed wired + wireless connections

---

## Why does it exist?

Windows often keeps the wrong proxy state when multiple network interfaces are active.

This leads to:

- GitHub not reachable
- Internal services failing
- Manual proxy toggling

**ProxySwitcher fixes this automatically.**

---

## How it works (in short)

- Detects network adapters at the hardware level
- Classifies environment (intranet / external)
- Uses a debounced state machine to avoid flapping
- Updates system proxy settings
- Broadcasts WinInet refresh (no browser restart)

```

Network change网络变化  
↓  
Environment detection环境检测  
↓  
Debounced decision去抖动决策  
↓  
Proxy on / off代理开启/关闭  
↓  
Browser updates instantly浏览器立即更新

````
## 架构概览

![ProxySwitcher Architecture](docs/architecture.png)

---

## Usage

### Build

```powershell
make debug
make build
````

### Run

| Command命令    | Description描述                            |
|--------------|------------------------------------------|
| `-debug`     | Run in foreground with logs在前台运行并记录日志    |
| `-install`   | Install as Windows service安装为 Windows 服务 |
| `-uninstall` | Uninstall and clean up卸载并清理              |

> ⚠ Administrator privileges required⚠ 

* * *

Configuration
---------------

Proxy address is currently defined in source code:

```go
const DefaultProxy = ""
```

* * *

What it does NOT do
------------------------

*   Does not modify routing tables
*   Does not interfere with VPNs
*   Does not depend on adapter names

* * *

License
---------
<a href="./LICENSE">MIT</a>