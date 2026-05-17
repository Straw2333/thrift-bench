package model

// IDLInfo 表示解析后的 Thrift IDL 信息
type IDLInfo struct {
	Services     []ServiceInfo          `json:"services"`
	Structs      map[string]StructInfo  `json:"structs"` // 新增: struct 定义映射表
	FixedIDLPath string                 `json:"-"`        // 修复后的 IDL 文件路径（不返回给前端）
}

// ServiceInfo 表示一个 Thrift 服务
type ServiceInfo struct {
	Name    string       `json:"name"`
	Methods []MethodInfo `json:"methods"`
}

// MethodInfo 表示一个服务方法
type MethodInfo struct {
	Name         string      `json:"name"`         // 方法名
	Params       []ParamInfo `json:"params"`       // 参数列表
	ReturnType   string      `json:"returnType"`   // 返回值类型
	ReturnFields []FieldInfo `json:"returnFields,omitempty"` // 如果返回类型是 struct，包含字段详情
}

// ParamInfo 表示方法参数
type ParamInfo struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	Required     bool        `json:"required"`               // 是否必填（required/optional）
	StructFields []FieldInfo `json:"structFields,omitempty"` // 如果类型是 struct，包含字段详情
}

// StructInfo 表示一个 struct 定义
type StructInfo struct {
	Name   string      `json:"name"`
	Fields []FieldInfo `json:"fields"`
}

// FieldInfo 表示 struct 的字段
type FieldInfo struct {
	Index    int    `json:"index"`    // 字段索引（Thrift 中的序号）
	Name     string `json:"name"`     // 字段名
	Type     string `json:"type"`     // 字段类型
	Required bool   `json:"required"` // 是否必填（required/optional）
}

// TestConfig 表示测试配置
type TestConfig struct {
	IDLPath     string `json:"idlPath"`
	Service     string `json:"service"`
	Method      string `json:"method"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`    // binary, compact, json
	Params      string `json:"params"`      // JSON 格式的参数
	Concurrency int    `json:"concurrency"` // 并发数
	Duration    int    `json:"duration"`    // 持续时间（秒）
	TestMode    string `json:"testMode"`    // 测试模式："duration" 或 "count"
	TotalCount  int    `json:"totalCount"`  // 总请求次数
}

// TestReport 表示测试报告
type TestReport struct {
	Total       int64   `json:"total"`       // 总请求数
	Success     int64   `json:"success"`     // 成功数
	Failed      int64   `json:"failed"`      // 失败数
	SuccessRate float64 `json:"successRate"` // 成功率
	AvgLatency  float64 `json:"avgLatency"`  // 平均延迟（毫秒）
	P50         float64 `json:"p50"`         // P50 延迟（毫秒）
	P95         float64 `json:"p95"`         // P95 延迟（毫秒）
	P99         float64 `json:"p99"`         // P99 延迟（毫秒）
	QPS         float64 `json:"qps"`         // 每秒请求数
}

// CallLog 单次调用记录
type CallLog struct {
	ID        int64   `json:"id"`        // 调用序号
	Timestamp string  `json:"timestamp"` // 时间戳 HH:mm:ss.SSS
	Success   bool    `json:"success"`   // 是否成功
	Latency   float64 `json:"latency"`   // 延迟(ms)
	Result    string  `json:"result"`    // 响应结果或错误信息
}
