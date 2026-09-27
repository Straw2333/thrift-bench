#!/bin/bash

echo "=========================================="
echo "  Thrift 性能测试工具 - 完整测试流程"
echo "=========================================="
echo ""

# 检查服务端是否运行
echo "步骤 1: 检查 Thrift 服务端状态..."
if nc -z localhost 8888 2>/dev/null; then
    echo "✅ Thrift 服务端正在运行 (端口 8888)"
else
    echo "❌ Thrift 服务端未运行，请先启动: cd demo-server && ./demo-server.exe"
    exit 1
fi

echo ""

# 检查测试工具是否运行
echo "步骤 2: 检查测试工具状态..."
if nc -z localhost 8080 2>/dev/null; then
    echo "✅ 测试工具正在运行 (端口 8080)"
else
    echo "❌ 测试工具未运行，请先启动: ./thrift-bench.exe"
    exit 1
fi

echo ""

# 测试 1: 上传 Thrift 文件
echo "步骤 3: 测试上传 Thrift 文件..."
UPLOAD_RESULT=$(curl -s -X POST \
  -F "file=@demo-server/example.thrift" \
  http://localhost:8080/api/upload)

if echo "$UPLOAD_RESULT" | grep -q "services"; then
    echo "✅ 上传成功，已解析到服务"
    echo "$UPLOAD_RESULT" | head -c 200
    echo "..."
else
    echo "❌ 上传失败"
    echo "$UPLOAD_RESULT"
    exit 1
fi

echo ""
echo ""

# 测试 2: 执行性能测试
echo "步骤 4: 执行性能测试 (GetUser 方法)..."
TEST_RESULT=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "service": "UserService",
    "method": "getUser",
    "host": "localhost",
    "port": 8888,
    "protocol": "binary",
    "params": "{\"req\": {\"userId\": 123}}",
    "concurrency": 5,
    "duration": 3
  }' \
  http://localhost:8080/api/test)

if echo "$TEST_RESULT" | grep -q "total"; then
    echo "✅ 测试成功！"
    echo ""
    echo "测试报告:"
    echo "----------------------------------------"
    echo "$TEST_RESULT" | python3 -m json.tool 2>/dev/null || echo "$TEST_RESULT"
else
    echo "❌ 测试失败"
    echo "$TEST_RESULT"
    exit 1
fi

echo ""
echo ""

# 测试 3: 测试无参数方法
echo "步骤 5: 执行性能测试 (Ping 方法 - 无参数)..."
PING_RESULT=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "service": "UserService",
    "method": "ping",
    "host": "localhost",
    "port": 8888,
    "protocol": "binary",
    "params": "{}",
    "concurrency": 10,
    "duration": 3
  }' \
  http://localhost:8080/api/test)

if echo "$PING_RESULT" | grep -q "total"; then
    echo "✅ Ping 测试成功！"
    echo ""
    echo "测试报告:"
    echo "----------------------------------------"
    echo "$PING_RESULT" | python3 -m json.tool 2>/dev/null || echo "$PING_RESULT"
else
    echo "❌ Ping 测试失败"
    echo "$PING_RESULT"
    exit 1
fi

echo ""
echo ""
echo "=========================================="
echo "  ✅ 所有测试通过！"
echo "=========================================="
echo ""
echo "你现在可以："
echo "1. 打开浏览器访问: http://localhost:8080"
echo "2. 通过 Web 界面进行测试"
echo "3. 查看服务端日志了解请求详情"
echo ""
