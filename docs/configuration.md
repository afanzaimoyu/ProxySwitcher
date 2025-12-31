# 配置设计说明（规划）

当前版本 ProxySwitcher 使用源码内硬编码配置。

本文档用于说明未来配置文件的设计方向。

---

## 一、当前状态

当前代理地址定义于源码中，例如：

```go
const DefaultProxy = "x.x.x.x:x"
```
修改配置需要重新编译。

---

## 二、未来目标
计划支持外部配置文件，例如：
```yaml
proxy:
  address: "x.x.x.x:x"

rules:
  - name: corporate_lan
    match:
      ip_cidr: "10.0.0.0/8"
    action: enable_proxy

  - name: external_wifi
    match:
      interface_type: wifi
      exclude_ip_cidr: "10.0.0.0/8"
    action: disable_proxy

```

---

## 三、设计原则

配置优先级明确

行为可预测

默认配置足够安全

错误配置不影响系统稳定

---

## 四、暂未实现说明

本文件为设计预留，当前版本不解析配置文件