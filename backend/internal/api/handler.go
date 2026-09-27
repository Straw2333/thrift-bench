package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"thrift-bench/internal/model"
	"thrift-bench/internal/service"

	"github.com/gin-gonic/gin"
)

// defaultUploadDir 返回基于可执行文件位置的 uploads 目录绝对路径
func defaultUploadDir() string {
	exePath, err := os.Executable()
	if err != nil {
		// 获取可执行文件路径失败时退回到当前工作目录
		return "uploads"
	}
	return filepath.Join(filepath.Dir(exePath), "uploads")
}

// validProtocols 支持的传输协议
var validProtocols = map[string]struct{}{
	"framed":   {},
	"binary":   {},
	"ttheader": {},
}

// Handler API 处理器
type Handler struct {
	mu     sync.Mutex // 保护 currentIDL / currentInfo
	parser *service.ThriftParser

	uploadDir   string
	currentIDL  string         // 当前上传的 IDL 文件路径（修复后）
	currentInfo *model.IDLInfo // 当前解析的 IDL 信息
}

// NewHandler 创建新的处理器
func NewHandler() *Handler {
	uploadDir := defaultUploadDir()
	os.MkdirAll(uploadDir, 0755)

	return &Handler{
		parser:    service.NewThriftParser(),
		uploadDir: uploadDir,
	}
}

// UploadIDL 处理 IDL 文件上传
func (h *Handler) UploadIDL(c *gin.Context) {
	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf(`❌ 文件上传失败

【可能原因】
1. 未选择文件
2. 文件大小超过限制
3. 网络传输中断

【原始错误】
%v`, err)})
		return
	}

	// 检查文件扩展名
	if filepath.Ext(file.Filename) != ".thrift" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf(`❌ 文件类型不支持

【上传文件】
%s

【支持的类型】
仅支持 .thrift 文件

【解决方法】
请选择扩展名为 .thrift 的 Thrift IDL 文件`, file.Filename)})
		return
	}

	// 保存文件（使用 Base 防止路径穿越）
	safeName := filepath.Base(file.Filename)
	savePath := filepath.Join(h.uploadDir, safeName)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf(`❌ 保存文件失败

【文件名】
%s

【可能原因】
1. 磁盘空间不足
2. 文件权限问题
3. 路径不存在

【原始错误】
%v`, file.Filename, err)})
		return
	}

	// 解析 IDL 文件
	idlInfo, err := h.parser.ParseIDL(savePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf(`❌ IDL 解析失败

【文件名】
%s

【可能原因】
1. Thrift 语法错误
2. 文件编码问题（请使用 UTF-8）
3. 缺少必要的 struct 或 service 定义

【原始错误】
%v`, file.Filename, err)})
		return
	}

	// 记录本次解析结果，并清理上一次上传留下的文件
	h.mu.Lock()
	oldIDL := h.currentIDL
	h.currentIDL = idlInfo.FixedIDLPath
	h.currentInfo = idlInfo
	h.mu.Unlock()
	removeUploadedIDL(oldIDL, idlInfo.FixedIDLPath)

	// 返回解析结果
	c.JSON(http.StatusOK, idlInfo)
}

// removeUploadedIDL 删除旧的 IDL 上传文件（原始文件与修复文件）。
// 忽略删除失败的情况——残留文件不影响功能。
func removeUploadedIDL(oldFixed, newFixed string) {
	if oldFixed == "" || oldFixed == newFixed {
		return
	}
	original := strings.TrimSuffix(oldFixed, ".fixed")
	os.Remove(oldFixed)
	os.Remove(original)
}

// RunTest 执行性能测试
func (h *Handler) RunTest(c *gin.Context) {
	engine, err := h.createTestEngine(c)
	if err != nil {
		return // 错误已在 createTestEngine 中写入响应
	}

	// 执行测试（请求上下文取消时测试随之终止）
	report := engine.Run(c.Request.Context())

	// 返回测试报告
	c.JSON(http.StatusOK, report)
}

// RunTestStream 执行性能测试并通过 SSE 流式返回实时数据
func (h *Handler) RunTestStream(c *gin.Context) {
	engine, err := h.createTestEngine(c)
	if err != nil {
		return // 错误已在 createTestEngine 中写入响应
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // 禁用 nginx 缓冲

	// SSE 写入必须串行化：日志回调来自多个 worker goroutine，
	// 进度快照来自独立的 ticker goroutine，并发写会破坏 HTTP 分帧
	var writeMu sync.Mutex
	writeEvent := func(event string, payload interface{}) {
		data, err := json.Marshal(payload)
		if err != nil {
			return
		}
		writeMu.Lock()
		defer writeMu.Unlock()
		if event != "" {
			// SSE 格式: event: log\ndata: {json}\n\n
			fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data)
		} else {
			// SSE 格式: data: {json}\n\n
			fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		}
		c.Writer.Flush()
	}

	// 进度回调：推送聚合统计数据(每秒)
	progressCallback := func(report *model.TestReport) {
		writeEvent("", report)
	}

	// 日志回调：推送单次调用的详细记录
	logCallback := func(log *model.CallLog) {
		writeEvent("log", log)
	}

	// 执行测试,定期推送进度和日志；客户端断开（如点击"停止测试"）时测试终止
	finalReport := engine.RunWithCallback(c.Request.Context(), progressCallback, logCallback)

	// 发送最终报告（使用 event: done 标识）
	writeEvent("done", finalReport)
}

// createTestEngine 解析请求、验证配置并创建测试引擎
// 如果出错，会直接向响应写入错误信息并返回 nil
func (h *Handler) createTestEngine(c *gin.Context) (*service.TestEngine, error) {
	var config model.TestConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return nil, err
	}

	h.mu.Lock()
	config.IDLPath = h.currentIDL
	h.mu.Unlock()
	if config.IDLPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先上传 IDL 文件"})
		return nil, fmt.Errorf("IDL not uploaded")
	}

	if err := h.validateConfig(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, err
	}

	if err := h.validateMethodParams(config.Method); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, err
	}

	engine, err := service.NewTestEngine(&config)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "创建测试引擎失败: " + err.Error()})
		return nil, err
	}

	return engine, nil
}

// validateMethodParams 拒绝多参数方法。
// Kitex 泛化调用仅支持单参数方法：多参数方法的参数值无法被正确传递
// （首个参数会被写成整段 JSON 文本，其余参数为默认值），必须提前拦截。
func (h *Handler) validateMethodParams(method string) error {
	h.mu.Lock()
	info := h.currentInfo
	h.mu.Unlock()
	if info == nil {
		return nil
	}
	for _, svc := range info.Services {
		for _, m := range svc.Methods {
			if m.Name != method {
				continue
			}
			if len(m.Params) <= 1 {
				return nil
			}
			names := make([]string, 0, len(m.Params))
			for _, p := range m.Params {
				names = append(names, p.Name)
			}
			return fmt.Errorf(`❌ 无法测试方法 %q：它声明了 %d 个参数（%s）。

【原因】
Kitex 泛化调用仅支持单参数方法（通常是"一个请求 struct"的写法），
多参数方法的参数值无法被正确传递到服务端。

【解决方法】
1. 选择其他单参数方法进行测试
2. 或调整 IDL：将多个参数封装为一个请求 struct`, method, len(m.Params), strings.Join(names, ", "))
		}
	}
	return nil
}

// validateConfig 验证测试配置
func (h *Handler) validateConfig(config *model.TestConfig) error {
	if config.Service == "" {
		return fmt.Errorf(`❌ 配置错误：服务名不能为空

【解决方法】
请在界面填写 Thrift 服务名称（例如：UserService）`)
	}
	if config.Method == "" {
		return fmt.Errorf(`❌ 配置错误：方法名不能为空

【解决方法】
请从下拉列表中选择要测试的方法`)
	}
	if config.Host == "" {
		return fmt.Errorf(`❌ 配置错误：主机地址不能为空

【解决方法】
请填写 Thrift 服务的主机地址（例如：localhost 或 192.168.1.100）`)
	}
	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf(`❌ 配置错误：端口号无效

【当前值】
%d

【有效范围】
1-65535

【提示】
demo-server 默认端口为 8888`, config.Port)
	}
	if config.Concurrency <= 0 {
		return fmt.Errorf(`❌ 配置错误：并发数无效

【当前值】
%d

【要求】
并发数必须大于 0`, config.Concurrency)
	}
	if _, ok := validProtocols[config.Protocol]; config.Protocol != "" && !ok {
		return fmt.Errorf(`❌ 配置错误：传输协议无效

【当前值】
%s

【有效值】
framed, binary, ttheader`, config.Protocol)
	}
	if config.TimeoutMs == 0 {
		// 未配置时使用默认超时
		config.TimeoutMs = 5000
	}
	if config.TimeoutMs <= 0 || config.TimeoutMs > 600000 {
		return fmt.Errorf(`❌ 配置错误：单次请求超时无效

【当前值】
%d 毫秒

【有效范围】
1-600000 毫秒`, config.TimeoutMs)
	}

	// 根据测试模式验证不同参数
	switch config.TestMode {
	case "duration":
		if config.Duration <= 0 {
			return fmt.Errorf(`❌ 配置错误：持续时间无效

【当前值】
%d 秒

【要求】
持续时间必须大于 0`, config.Duration)
		}
	case "count":
		if config.TotalCount <= 0 {
			return fmt.Errorf(`❌ 配置错误：总请求次数无效

【当前值】
%d

【要求】
总请求次数必须大于 0`, config.TotalCount)
		}
	default:
		return fmt.Errorf(`❌ 配置错误：测试模式无效

【当前值】
%s

【有效值】
duration（基于持续时间）或 count（基于请求次数）`, config.TestMode)
	}

	return nil
}
