package service

import (
	"sync/atomic"
	"thrift-bench/internal/model"
	"time"
)

// 预定义延迟桶边界（毫秒），覆盖 0~10s 范围
// 每个桶记录延迟 <= 上界的请求数，桶越密精度越高
var latencyBounds = []float64{
	0.1, 0.2, 0.5,
	1, 2, 5,
	10, 20, 50,
	100, 200, 500,
	1000, 2000, 5000,
	10000,
}

// MetricsCollector 线程安全的指标收集器
// 使用固定大小的延迟直方图替代切片，内存恒定 O(1)
type MetricsCollector struct {
	buckets   []atomic.Int64 // 延迟直方图，buckets[i] 记录延迟 <= latencyBounds[i] 的请求数
	totalOver atomic.Int64   // 延迟超过最大桶边界的请求数
	success   atomic.Int64
	failed    atomic.Int64
	totalLat  atomic.Int64 // 总延迟（纳秒），用于计算平均值
	startTime time.Time
}

// NewMetricsCollector 创建新的指标收集器
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		buckets:   make([]atomic.Int64, len(latencyBounds)),
		startTime: time.Now(),
	}
}

// Record 记录一次请求的结果
func (m *MetricsCollector) Record(latency time.Duration, success bool) {
	latencyMs := float64(latency.Nanoseconds()) / 1e6

	// 记录到对应的延迟桶
	recorded := false
	for i, bound := range latencyBounds {
		if latencyMs <= bound {
			m.buckets[i].Add(1)
			recorded = true
			break
		}
	}
	if !recorded {
		m.totalOver.Add(1)
	}

	// 累加延迟用于计算平均值
	m.totalLat.Add(latency.Nanoseconds())

	if success {
		m.success.Add(1)
	} else {
		m.failed.Add(1)
	}
}

// GenerateReport 生成测试报告
func (m *MetricsCollector) GenerateReport() *model.TestReport {
	return m.generateReport()
}

// GetSnapshot 获取当前测试指标快照（实时数据,用于流式传输）
func (m *MetricsCollector) GetSnapshot() *model.TestReport {
	return m.generateReport()
}

// generateReport 生成报告（无需加锁，所有字段使用原子操作）
func (m *MetricsCollector) generateReport() *model.TestReport {
	success := m.success.Load()
	failed := m.failed.Load()
	total := success + failed

	if total == 0 {
		return &model.TestReport{}
	}

	// 读取桶计数快照
	bucketCounts := make([]int64, len(latencyBounds))
	for i := range latencyBounds {
		bucketCounts[i] = m.buckets[i].Load()
	}

	// 计算成功率
	successRate := float64(success) / float64(total) * 100

	// 计算平均延迟
	totalLatNs := m.totalLat.Load()
	avgLatency := float64(totalLatNs) / float64(total) / 1e6 // 转毫秒

	// 计算百分位延迟
	p50 := m.percentile(bucketCounts, total, 0.50)
	p95 := m.percentile(bucketCounts, total, 0.95)
	p99 := m.percentile(bucketCounts, total, 0.99)

	// 计算 QPS
	duration := time.Since(m.startTime).Seconds()
	qps := float64(total) / duration

	return &model.TestReport{
		Total:       total,
		Success:     success,
		Failed:      failed,
		SuccessRate: successRate,
		AvgLatency:  avgLatency,
		P50:         p50,
		P95:         p95,
		P99:         p99,
		QPS:         qps,
	}
}

// percentile 从直方图桶计算百分位延迟（毫秒）
// 返回满足 >= p 比例请求的延迟桶上界
func (m *MetricsCollector) percentile(bucketCounts []int64, total int64, p float64) float64 {
	if total == 0 {
		return 0
	}

	// 找到第 p 百分位对应的请求序号
	target := int64(float64(total) * p)
	if target == 0 {
		target = 1
	}

	// 累加桶计数，找到目标请求所在的桶
	cumulative := int64(0)
	for i, count := range bucketCounts {
		cumulative += count
		if cumulative >= target {
			return latencyBounds[i]
		}
	}

	// 超过最大桶边界，返回最大桶边界作为近似值
	return latencyBounds[len(latencyBounds)-1]
}
