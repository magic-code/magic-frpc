# Magic FRPc Makefile
# 构建脚本

.PHONY: all dev build clean install-deps help

# 默认目标
all: build

# 开发模式
dev:
	wails3 dev

# 构建
build:
	wails3 build

# Windows 构建（隐藏控制台窗口）
build-windows:
	wails3 build -platform windows/amd64 -ldflags "-H windowsgui"

# macOS 构建
build-darwin:
	wails3 build -platform darwin/amd64
	wails3 build -platform darwin/arm64

# Linux 构建
build-linux:
	wails3 build -platform linux/amd64

# 清理构建产物
clean:
	rm -rf build/bin
	rm -rf frontend/dist
	rm -rf frontend/.svelte-kit
	rm -rf frontend/node_modules

# 安装依赖
install-deps:
	cd frontend && npm install
	go mod download

# 生成 Wails 绑定
generate:
	wails3 generate bindings

# 运行测试
test:
	go test -v ./...

# 格式化代码
fmt:
	go fmt ./...
	cd frontend && npm run check

# 帮助
help:
	@echo "Magic FRPc 构建命令:"
	@echo "  make dev           - 启动开发服务器"
	@echo "  make build         - 构建应用"
	@echo "  make build-windows - 构建 Windows 版本"
	@echo "  make build-darwin  - 构建 macOS 版本"
	@echo "  make build-linux   - 构建 Linux 版本"
	@echo "  make clean         - 清理构建产物"
	@echo "  make install-deps  - 安装依赖"
	@echo "  make generate      - 生成 Wails 绑定"
	@echo "  make test          - 运行测试"
	@echo "  make fmt           - 格式化代码"
