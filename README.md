# Thrift 性能测试工具

一个简单易用的 Thrift 接口性能测试工具，类似 JMeter，但更加轻量和简单。

## 功能特性

✅ **动态解析** - 上传 `.thrift` 文件自动解析接口和参数
✅ **多传输协议** - 支持 Framed、Buffered、TTHeader 三种 Thrift 传输协议
✅ **高并发测试** - 基于 Go 协程的高性能并发引擎
✅ **实时报告** - 自动生成测试报告，包含 QPS、延迟分布、成功率等指标
✅ **极简界面** - 单页应用，无需复杂配置，4 步完成测试
✅ **本地部署** - 单个可执行文件，无需依赖，开箱即用

## 技术栈

- **后端**: Go 1.21+ + Gin + Kitex（字节跳动开源）
- **前端**: Vue 3 CDN + Pico.css（单个 HTML 文件，无需构建）
- **部署**: 本地部署，单个可执行文件

## 快速开始

### 1. 下载并运行

#### Windows

```bash
# 双击运行
thrift-perf-tool.exe

# 或命令行运行
.\thrift-perf-tool.exe
```

#### Linux / macOS

```bash
# 首次运行需要添加执行权限
chmod +x thrift-perf-tool

# 运行
./thrift-perf-tool
```

### 2. 访问 Web 界面

程序启动后，在浏览器中打开：

```
http://localhost:11064
```

### 3. 开始测试

按照界面提示的 4 个步骤进行操作：

**步骤 1**: 上传 Thrift 文件
- 选择本地的 `.thrift` 文件
- 点击"上传并解析"

**步骤 2**: 选择接口和配置
- 选择要测试的服务和方法
- 填写 Thrift 服务的主机地址和端口
- 选择协议类型（Binary / Compact / JSON）

**步骤 3**: 填写参数和测试配置
- 以 JSON 格式填写请求参数
- 设置并发数（同时发送请求的线程数）
- 设置持续时间（测试运行的总时长，单位：秒）
- 点击"开始测试"

**步骤 4**: 查看测试报告
- 查看测试结果（总请求数、成功率、延迟分布、QPS 等）
- 可导出报告为文本文件

## 测试报告指标说明

| 指标 | 说明 |
|------|------|
| 总请求数 | 测试期间发送的总请求数量 |
| 成功数 | 成功响应的请求数量 |
| 失败数 | 失败的请求数量 |
| 成功率 | 成功请求占总请求的百分比 |
| 平均延迟 | 所有请求的平均响应时间（毫秒） |
| P50 延迟 | 50% 的请求响应时间在此值以下（毫秒） |
| P95 延迟 | 95% 的请求响应时间在此值以下（毫秒） |
| P99 延迟 | 99% 的请求响应时间在此值以下（毫秒） |
| QPS | 每秒处理的请求数量（Queries Per Second） |

## 示例

### 示例 Thrift 文件

创建一个名为 `example.thrift` 的文件：

```thrift
namespace go example

struct User {
    1: i64 id,
    2: string name,
    3: i32 age
}

struct GetUserRequest {
    1: i64 userId
}

struct GetUserResponse {
    1: User user
}

service UserService {
    GetUserResponse getUser(1: GetUserRequest req)
}
```

### 示例请求参数

在步骤 3 中，填写以下 JSON 参数：

```json
{
    "userId": 123
}
```

> 注意：Kitex 泛化调用要求**扁平**的 JSON 格式——单参数方法直接写请求 struct 的字段，
> 不要按参数名包装（`{"req": {...}}` 的旧写法会导致字段值被静默丢弃）。

### 示例测试配置

```
服务: UserService
方法: getUser
主机: localhost
端口: 9090
协议: Framed
并发数: 10
持续时间: 10
```

## 项目结构

```
thrift-perf-tool/
├── backend/                    # Go 后端
│   ├── cmd/
│   │   └── main.go            # 程序入口
│   ├── internal/
│   │   ├── api/               # HTTP 接口层
│   │   │   ├── router.go      # 路由配置
│   │   │   └── handler.go     # 请求处理
│   │   ├── service/           # 业务逻辑层
│   │   │   ├── parser.go      # Thrift IDL 解析
│   │   │   ├── client.go      # Thrift 客户端
│   │   │   ├── engine.go      # 测试引擎
│   │   │   └── metrics.go     # 指标收集
│   │   └── model/             # 数据模型
│   │       └── types.go       # 类型定义
│   ├── go.mod
│   └── go.sum
├── web/
│   └── index.html             # 前端界面（单文件）
├── thrift-perf-tool.exe       # 编译后的可执行文件
└── README.md                  # 本文档
```

## 从源码构建

如果需要从源码构建项目：

### 前置要求

- Go 1.21 或更高版本
- Git

### 构建步骤

```bash
# 1. 克隆项目（如果是从 Git 仓库）
# git clone <repository-url>
# cd thrift-perf-tool

# 2. 进入后端目录
cd backend

# 3. 下载依赖
go mod tidy

# 4. 编译项目
# Windows
go build -o ../thrift-perf-tool.exe ./cmd/main.go

# Linux / macOS
go build -o ../thrift-perf-tool ./cmd/main.go

# 5. 返回项目根目录
cd ..

# 6. 运行程序
# Windows
.\thrift-perf-tool.exe

# Linux / macOS
./thrift-perf-tool
```

## 支持的 Thrift 协议

| 协议 | 说明 | 适用场景 |
|------|------|----------|
| **Binary** | Thrift 默认的二进制协议 | 高性能场景，最常用 |
| **Compact** | 紧凑型二进制协议，体积更小 | 网络带宽受限场景 |
| **JSON** | 基于 JSON 的文本协议 | 调试和兼容性场景 |

## 常见问题

### 1. 无法连接到 Thrift 服务？

- 检查主机地址和端口是否正确
- 确认 Thrift 服务正在运行
- 检查防火墙设置
- 确认协议类型是否与服务端一致

### 2. 上传 Thrift 文件失败？

- 确保文件扩展名为 `.thrift`
- 检查 Thrift 文件语法是否正确
- 确保文件包含至少一个 `service` 定义

### 3. 测试失败或成功率低？

- 检查请求参数 JSON 格式是否正确
- 确认参数与 Thrift 接口定义匹配
- 降低并发数重试
- 检查 Thrift 服务端日志

### 4. 如何修改监听端口？

目前默认端口为 `11064`，如需修改：

1. 启动时加参数：`./thrift-perf-tool -port 9000`（或设置环境变量 `PORT`）
2. 或编辑 `backend/cmd/main.go` 中的默认值后重新编译
3. 加 `-no-browser` 参数可禁止自动打开浏览器

## 性能建议

- **并发数**: 根据目标服务的处理能力设置，建议从小开始逐步增加
- **持续时间**: 建议至少 10 秒，以获得更准确的统计数据
- **网络环境**: 建议在内网环境测试，减少网络延迟影响
- **资源监控**: 测试时同时监控服务端的 CPU、内存使用情况

## 开发路线图

- [x] 基础功能实现
- [x] 多协议支持
- [x] 并发测试引擎
- [ ] 实时进度显示（WebSocket）
- [ ] 测试用例保存和复用
- [ ] 历史报告对比
- [ ] 分布式测试支持
- [ ] Docker 部署支持

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

## 联系方式

如有问题或建议，请提交 Issue。
