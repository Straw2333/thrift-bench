package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"thrift-perf-tool/internal/model"
	"thrift-perf-tool/internal/service"

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

// Handler API 处理器
type Handler struct {
	parser      *service.ThriftParser
	uploadDir   string
	currentIDL  string          // 当前上传的 IDL 文件路径（修复后）
	currentInfo *model.IDLInfo  // 当前解析的 IDL 信息
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

	// 保存解析结果和修复后的 IDL 路径
	h.currentIDL = idlInfo.FixedIDLPath
	h.currentInfo = idlInfo

	// 返回解析结果
	c.JSON(http.StatusOK, idlInfo)
}

// RunTest 执行性能测试
func (h *Handler) RunTest(c *gin.Context) {
	engine, err := h.createTestEngine(c)
	if err != nil {
		return // 错误已在 createTestEngine 中写入响应
	}

	// 执行测试
	report := engine.Run()

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

	// 创建进度回调函数,用于推送聚合统计数据(每秒)
	progressCallback := func(report *model.TestReport) {
		data, err := json.Marshal(report)
		if err != nil {
			return
		}
		// SSE 格式: data: {json}\n\n
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		c.Writer.Flush()
	}

	// 创建日志回调函数,用于推送每次调用的详细记录
	logCallback := func(log *model.CallLog) {
		data, err := json.Marshal(log)
		if err != nil {
			return
		}
		// SSE 格式: event: log\ndata: {json}\n\n
		fmt.Fprintf(c.Writer, "event: log\ndata: %s\n\n", data)
		c.Writer.Flush()
	}

	// 执行测试,定期推送进度和日志
	finalReport := engine.RunWithCallback(progressCallback, logCallback)

	// 发送最终报告（使用 event: done 标识）
	finalData, err := json.Marshal(finalReport)
	if err != nil {
		fmt.Fprintf(c.Writer, "event: done\ndata: {\"error\":\"序列化报告失败\"}\n\n")
	} else {
		fmt.Fprintf(c.Writer, "event: done\ndata: %s\n\n", finalData)
	}
	c.Writer.Flush()
}

// createTestEngine 解析请求、验证配置并创建测试引擎
// 如果出错，会直接向响应写入错误信息并返回 nil
func (h *Handler) createTestEngine(c *gin.Context) (*service.TestEngine, error) {
	var config model.TestConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return nil, err
	}

	config.IDLPath = h.currentIDL
	if config.IDLPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先上传 IDL 文件"})
		return nil, fmt.Errorf("IDL not uploaded")
	}

	if err := h.validateConfig(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, err
	}

	engine, err := service.NewTestEngine(&config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建测试引擎失败: " + err.Error()})
		return nil, err
	}

	return engine, nil
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

	// 根据测试模式验证不同参数
	if config.TestMode == "duration" {
		if config.Duration <= 0 {
			return fmt.Errorf(`❌ 配置错误：持续时间无效

【当前值】
%d 秒

【要求】
持续时间必须大于 0`, config.Duration)
		}
	} else if config.TestMode == "count" {
		if config.TotalCount <= 0 {
			return fmt.Errorf(`❌ 配置错误：总请求次数无效

【当前值】
%d

【要求】
总请求次数必须大于 0`, config.TotalCount)
		}
	}

	return nil
}
