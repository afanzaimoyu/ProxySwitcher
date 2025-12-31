# 项目元数据
BINARY_NAME=ProxySwitcher.exe
MAIN_PATH=./cmd/proxy-switcher/main.go
LOG_FILE=proxy_service.log

# 编译参数
# -s -w: 去除符号表和调试信息，大大减小二进制体积
# -H windowsgui: 针对 build 模式隐藏 CMD 黑窗口，适合后台服务运行
LDFLAGS_DEBUG=-ldflags "-s -w"
LDFLAGS_BUILD=-ldflags "-s -w -H windowsgui"

.PHONY: all debug build clean install uninstall help

all: build

## debug: 编译并保留控制台窗口，方便开发时查看实时日志
debug:
	@echo "正在编译调试版..."
	go build $(LDFLAGS_DEBUG) -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "编译完成！输入 ./$(BINARY_NAME) -debug 运行"

## build: 编译正式版，运行时不显示黑窗口
build:
	@echo "正在编译正式版 (隐藏窗口模式)..."
	go build $(LDFLAGS_BUILD) -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "编译完成！"

## clean: 强力清理编译产物和残留日志
clean:
	@echo "正在清理残留文件..."
	@if exist $(BINARY_NAME) del /f /q $(BINARY_NAME)
	@if exist $(LOG_FILE) del /f /q $(LOG_FILE)
	@echo "清理完毕"

## install: 以管理员身份安装服务 (仅限 Windows)
install: build
	@echo "正在安装服务 (需要管理员权限)..."
	./$(BINARY_NAME) -install

## uninstall: 调用程序内置的卸载逻辑，确保清理注册表、服务和日志
uninstall:
	@echo "正在启动深度卸载流程..."
	@if exist $(BINARY_NAME) ./$(BINARY_NAME) -uninstall

## help: 显示所有可用的管理命令
help:
	@echo "ProxySwitcher 编译管理工具:"
	@echo "  make debug     - 编译带控制台的调试版"
	@echo "  make build     - 编译后台运行的正式版"
	@echo "  make install   - 编译并安装为 Windows 服务"
	@echo "  make uninstall - 彻底卸载服务、重置代理并删除日志"
	@echo "  make clean     - 物理删除本地生成的 exe 和日志"