@echo off
chcp 65001 >nul
echo ==========================================
echo   单文件版本功能测试
echo ==========================================
echo.

echo [1/4] 检查可执行文件是否存在...
if exist thrift-perf-tool.exe (
    echo     ✓ thrift-perf-tool.exe 存在
) else (
    echo     ✗ thrift-perf-tool.exe 不存在
    echo     请先编译: cd backend ^&^& go build -o ..\thrift-perf-tool.exe .\cmd\main.go
    pause
    exit /b 1
)

echo.
echo [2/4] 检查文件大小...
for %%A in (thrift-perf-tool.exe) do (
    set size=%%~zA
)
echo     文件大小: %size% 字节
if %size% GTR 10000000 (
    echo     ✓ 文件大小正常 (包含嵌入的前端资源)
) else (
    echo     ⚠ 文件偏小，可能未正确嵌入前端资源
)

echo.
echo [3/4] 启动测试工具...
echo     正在后台启动服务...
start /B thrift-perf-tool.exe

echo     等待服务启动 (3秒)...
timeout /t 3 /nobreak >nul

echo.
echo [4/4] 测试服务是否响应...
curl -s -o nul -w "    HTTP状态码: %%{http_code}\n" http://localhost:11064/

echo.
echo ==========================================
echo   测试完成！
echo ==========================================
echo.
echo 如果看到 "HTTP状态码: 200"，说明服务运行正常。
echo 浏览器应该已经自动打开 http://localhost:11064
echo.
echo 按任意键停止服务并退出...
pause >nul

taskkill /F /IM thrift-perf-tool.exe >nul 2>&1
echo 服务已停止。
