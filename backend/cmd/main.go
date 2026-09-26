package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"thrift-perf-tool/internal/api"
	"time"
)

//go:embed web
var webFS embed.FS

func main() {
	// 命令行参数
	port := flag.Int("port", 0, "服务监听端口 (默认 11064，也可通过环境变量 PORT 设置)")
	noBrowser := flag.Bool("no-browser", false, "启动后不自动打开浏览器")
	flag.Parse()

	// 优先级：命令行参数 > 环境变量 > 默认值
	listenPort := 11064
	if envPort := os.Getenv("PORT"); envPort != "" {
		if p, err := parsePort(envPort); err == nil {
			listenPort = p
		} else {
			log.Printf("忽略无效的环境变量 PORT=%s: %v", envPort, err)
		}
	}
	if *port > 0 {
		listenPort = *port
	}

	// 打印启动信息
	fmt.Println("========================================")
	fmt.Println("  Thrift 性能测试工具")
	fmt.Println("========================================")
	fmt.Println()

	// 创建处理器
	handler := api.NewHandler()

	// 设置路由(传入嵌入的文件系统)
	router := api.SetupRouter(handler, webFS)

	// 启动服务
	addr := fmt.Sprintf(":%d", listenPort)
	url := fmt.Sprintf("http://localhost%s", addr)
	fmt.Printf("服务已启动，访问地址: %s\n", url)
	fmt.Println()

	// 延迟一小段时间后自动打开浏览器
	if !*noBrowser {
		go func() {
			time.Sleep(500 * time.Millisecond)
			if err := openBrowser(url); err != nil {
				log.Printf("自动打开浏览器失败: %v (请手动访问 %s)\n", err, url)
			} else {
				fmt.Println("已自动打开浏览器")
			}
		}()
	}

	if err := router.Run(addr); err != nil {
		log.Fatalf("启动服务失败: %v", err)
	}
}

// parsePort 解析端口号字符串
func parsePort(s string) (int, error) {
	p, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || p < 1 || p > 65535 {
		return 0, fmt.Errorf("端口号必须在 1-65535 范围内")
	}
	return p, nil
}

// openBrowser 跨平台打开浏览器
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // linux, freebsd, openbsd, netbsd
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
}
