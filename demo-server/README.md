# Thrift 示例服务端

这是一个简单的 Thrift 服务端示例，用于配合性能测试工具进行测试。

## 快速启动

### 方式 1：直接运行可执行文件（推荐）

```bash
# Windows
.\demo-server.exe

# Linux / macOS
./demo-server
```

### 方式 2：从源码编译运行

#### 步骤 1：生成 Thrift 代码

```bash
# 安装 Kitex 代码生成工具
go install github.com/cloudwego/kitex/tool/cmd/kitex@latest

# 生成代码
kitex -module demo-server -service userservice example.thrift
```

#### 步骤 2：编译并运行

```bash
# 下载依赖
go mod tidy

# 编译
go build -o demo-server.exe

# 运行
.\demo-server.exe
```

## 服务信息

- **监听端口**: 9090
- **协议**: Thrift Binary Protocol
- **服务名**: UserService

## 支持的接口

### 1. GetUser - 获取用户信息

**请求参数**:
```json
{
    "req": {
        "userId": 123
    }
}
```

**响应**:
```json
{
    "user": {
        "id": 123,
        "name": "张三",
        "age": 25,
        "email": "zhangsan@example.com"
    },
    "code": 200,
    "message": "success"
}
```

### 2. Ping - 测试连接

**请求参数**:
```json
{}
```

**响应**:
```
"pong"
```

## 使用性能测试工具测试

### 步骤 1：启动服务端

```bash
.\demo-server.exe
```

### 步骤 2：启动测试工具

```bash
cd ..
.\thrift-perf-tool.exe
```

### 步骤 3：在浏览器中测试

1. 访问 http://localhost:8080
2. 上传 `example.thrift` 文件
3. 配置:
   - 服务: UserService
   - 方法: getUser
   - 主机: localhost
   - 端口: 9090
   - 协议: Binary
4. 请求参数:
```json
{
    "req": {
        "userId": 123
    }
}
```
5. 并发数: 10
6. 持续时间: 10 秒
7. 点击"开始测试"

## 修改服务实现

编辑 `main.go` 中的 `GetUser` 方法来返回不同的数据：

```go
func (s *UserServiceImpl) GetUser(ctx context.Context, req *example.GetUserRequest) (*example.GetUserResponse, error) {
    // 在这里修改返回的用户数据
    user := &example.User{
        Id:    req.UserId,
        Name:  "你的名字",  // 修改这里
        Age:   30,          // 修改这里
        Email: "your@email.com",  // 修改这里
    }

    return &example.GetUserResponse{
        User:    user,
        Code:    200,
        Message: "success",
    }, nil
}
```

重新编译并运行即可。

## 停止服务

在命令行窗口按 `Ctrl + C` 停止服务。
