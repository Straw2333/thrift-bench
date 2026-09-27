.PHONY: build run clean help

# 默认目标
all: build

# 构建项目
build:
	@echo "开始编译项目..."
	cd backend && go mod tidy
	cd backend && go build -o ../thrift-bench.exe ./cmd/main.go
	@echo "编译完成！可执行文件: thrift-bench.exe"

# 构建 Linux 版本
build-linux:
	@echo "开始编译 Linux 版本..."
	cd backend && GOOS=linux GOARCH=amd64 go build -o ../thrift-bench-linux ./cmd/main.go
	@echo "编译完成！可执行文件: thrift-bench-linux"

# 构建 macOS 版本
build-mac:
	@echo "开始编译 macOS 版本..."
	cd backend && GOOS=darwin GOARCH=amd64 go build -o ../thrift-bench-mac ./cmd/main.go
	@echo "编译完成！可执行文件: thrift-bench-mac"

# 构建所有平台
build-all: build build-linux build-mac
	@echo "所有平台编译完成！"

# 运行项目
run: build
	@echo "启动服务..."
	./thrift-bench.exe

# 清理编译文件
clean:
	@echo "清理编译文件..."
	rm -f thrift-bench.exe thrift-bench-linux thrift-bench-mac
	rm -rf uploads/
	@echo "清理完成！"

# 测试
test:
	@echo "运行测试..."
	cd backend && go test ./...

# 帮助信息
help:
	@echo "Thrift 性能测试工具 - Makefile 使用说明"
	@echo ""
	@echo "可用命令:"
	@echo "  make build       - 编译 Windows 版本（默认）"
	@echo "  make build-linux - 编译 Linux 版本"
	@echo "  make build-mac   - 编译 macOS 版本"
	@echo "  make build-all   - 编译所有平台版本"
	@echo "  make run         - 编译并运行"
	@echo "  make clean       - 清理编译文件"
	@echo "  make test        - 运行测试"
	@echo "  make help        - 显示此帮助信息"
