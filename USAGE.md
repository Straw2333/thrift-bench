# 🚀 完整的使用流程说明

## 架构说明

```
┌──────────────────┐         ┌──────────────────┐
│  测试工具（客户端） │         │  Thrift 服务端     │
│  thrift-perf-tool │ ──────→│  demo-server      │
│  localhost:11064   │   请求  │  localhost:8888   │
└──────────────────┘         └──────────────────┘
```

这个工具是**客户端测试工具**，需要连接到一个运行中的 **Thrift 服务端**。

## 🎯 快速开始（3步）

### 第 1 步：启动示例 Thrift 服务端

```bash
cd demo-server
.\demo-server.exe
```

看到以下输出表示服务端启动成功：
```
[Info] KITEX: server listen at addr=[::]:8888
```

**服务端信息**：
- 监听端口：8888
- 协议：Thrift Binary
- 服务名：UserService
- 支持的方法：
  - `GetUser(userId)` - 获取用户信息
  - `Ping()` - 测试连接（无参数）

### 第 2 步：启动性能测试工具

**打开新的命令行窗口**，运行：

```bash
.\thrift-perf-tool.exe
```

看到以下输出表示测试工具启动成功：
```
========================================
  Thrift 性能测试工具
========================================
服务已启动，访问地址: http://localhost:11064
```

### 第 3 步：在浏览器中测试

1. 打开浏览器访问：http://localhost:11064

2. **步骤 1：上传 Thrift 文件**
   - 点击"选择文件"
   - 选择 `demo-server\example.thrift`
   - 点击"上传并解析"

3. **步骤 2：配置接口**
   - 服务：`UserService`
   - 方法：`getUser`
   - 主机：`localhost`
   - 端口：`8888`  ⚠️ **注意：是 8888 而不是 9090**
   - 协议：`Binary`
   - 点击"下一步"

4. **步骤 3：填写参数和测试配置**

   界面会自动显示参数结构：
   ```
   📋 参数结构说明
   ┌────────────┬──────┐
   │ 参数名     │ 类型 │
   ├────────────┼──────┤
   │ userId     │ i64  │
   └────────────┴──────┘
   ```

   请求参数（已自动填充，修改值即可）：
   ```json
   {
     "userId": 123
   }
   ```

   测试配置：
   - 并发数：`10`
   - 持续时间：`10` 秒

   点击"开始测试"

5. **步骤 4：查看测试报告**

   测试完成后会显示：
   ```
   总请求数:    1000
   成功数:      1000
   失败数:      0
   成功率:      100.00%
   平均延迟:    5.2 ms
   P50 延迟:    4 ms
   P95 延迟:    10 ms
   P99 延迟:    15 ms
   QPS:         100
   ```

## 📊 测试 Ping 方法（无参数）

1. 在步骤 2 中选择方法：`ping`
2. 在步骤 3 中会看到提示："ℹ️ 该方法无需参数，使用默认空对象 `{}`"
3. 参数保持 `{}` 即可
4. 点击"开始测试"

## 🛠️ 修改服务端响应数据

编辑 `demo-server\handler.go` 文件：

```go
func (s *UserServiceImpl) GetUser(ctx context.Context, req *example.GetUserRequest) (resp *example.GetUserResponse, err error) {
    log.Printf("收到请求: GetUser, userId=%d", req.UserId)

    // 修改返回的用户数据
    user := &example.User{
        Id:    req.UserId,
        Name:  "李四",        // ← 修改这里
        Age:   30,            // ← 修改这里
        Email: "lisi@example.com",  // ← 修改这里
    }

    resp = &example.GetUserResponse{
        User:    user,
        Code:    200,
        Message: "success",
    }

    return resp, nil
}
```

保存后重新编译：

```bash
cd demo-server
go build -o demo-server.exe
```

停止旧服务（按 Ctrl+C），重新运行：

```bash
.\demo-server.exe
```

## ⚠️ 常见问题

### 问题 1：连接失败

**错误**：测试失败，连接超时

**解决**：
1. 确认服务端已启动（看到 "server listen at addr=[::]:8888"）
2. 确认端口填写正确（是 **8888** 而不是 9090）
3. 检查防火墙设置

### 问题 2：无法找到服务

**错误**：未找到服务

**解决**：
1. 确认上传的 thrift 文件正确
2. 确认服务名拼写正确（区分大小写）

### 问题 3：参数错误

**错误**：参数格式错误

**解决**：
1. 查看界面上的"参数结构说明"表格
2. 严格按照 JSON 格式填写
3. 参数名和类型必须匹配

## 🔄 停止服务

### 停止服务端

在服务端窗口按 `Ctrl + C`

### 停止测试工具

在测试工具窗口按 `Ctrl + C`

## 📝 日志查看

### 服务端日志

服务端窗口会显示接收到的请求：

```
收到请求: GetUser, userId=123
收到请求: GetUser, userId=456
收到请求: Ping
```

### 测试工具日志

测试工具窗口会显示 HTTP 请求：

```
[GIN] 2025/10/23 - 10:20:30 | 200 | 5.234ms | POST "/api/test"
```

## 🎓 进阶使用

### 1. 创建自己的 Thrift 服务

1. 编写 `.thrift` 文件定义接口
2. 使用 Kitex 生成代码：
   ```bash
   kitex -module your-project -service yourservice your.thrift
   ```
3. 实现 `handler.go` 中的方法
4. 编译并运行

### 2. 测试生产环境的服务

1. 确保可以访问生产服务的网络
2. 在步骤 2 中填写生产服务的 IP 和端口
3. 选择正确的协议（Binary/Compact/JSON）
4. 小心控制并发数，避免压垮生产服务

### 3. 批量测试多个接口

对每个接口重复步骤 2-4，记录测试报告并对比性能。

---

**祝你使用愉快！** 🚀

如有问题请查看：
- 测试工具文档：`README.md`
- 快速入门：`QUICKSTART.md`
- 服务端文档：`demo-server/README.md`
