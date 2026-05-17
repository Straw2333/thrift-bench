package service

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ===================== Record =====================

func TestRecord_SuccessAndFailure(t *testing.T) {
	m := NewMetricsCollector()
	m.Record(1*time.Millisecond, true)
	m.Record(2*time.Millisecond, false)

	report := m.GenerateReport()
	assert.Equal(t, int64(1), report.Success)
	assert.Equal(t, int64(1), report.Failed)
	assert.Equal(t, int64(2), report.Total)
}

func TestRecord_MultipleSuccesses(t *testing.T) {
	m := NewMetricsCollector()
	for i := 0; i < 100; i++ {
		m.Record(time.Duration(i+1)*time.Millisecond, true)
	}

	report := m.GenerateReport()
	assert.Equal(t, int64(100), report.Success)
	assert.Equal(t, int64(0), report.Failed)
}

// ===================== GenerateReport =====================

func TestGenerateReport_Empty(t *testing.T) {
	m := NewMetricsCollector()
	report := m.GenerateReport()
	assert.Equal(t, int64(0), report.Total)
	assert.Equal(t, int64(0), report.Success)
	assert.Equal(t, int64(0), report.Failed)
}

func TestGenerateReport_AvgLatency(t *testing.T) {
	m := NewMetricsCollector()
	m.Record(10*time.Millisecond, true)
	m.Record(20*time.Millisecond, true)

	report := m.GenerateReport()
	// 平均延迟应在 10~20ms 之间（直方图近似值）
	assert.Greater(t, report.AvgLatency, 0.0)
	assert.LessOrEqual(t, report.AvgLatency, 20.0)
}

func TestGenerateReport_SuccessRate(t *testing.T) {
	m := NewMetricsCollector()
	m.Record(1*time.Millisecond, true)
	m.Record(1*time.Millisecond, true)
	m.Record(1*time.Millisecond, false)

	report := m.GenerateReport()
	assert.InDelta(t, 66.67, report.SuccessRate, 0.1)
}

func TestGenerateReport_QPS(t *testing.T) {
	m := NewMetricsCollector()
	m.Record(1*time.Millisecond, true)

	report := m.GenerateReport()
	assert.Greater(t, report.QPS, 0.0)
}

// ===================== GetSnapshot =====================

func TestGetSnapshot_MatchesReport(t *testing.T) {
	m := NewMetricsCollector()
	m.Record(5*time.Millisecond, true)
	m.Record(10*time.Millisecond, true)

	report := m.GenerateReport()
	snapshot := m.GetSnapshot()
	assert.Equal(t, report.Total, snapshot.Total)
	assert.Equal(t, report.Success, snapshot.Success)
	assert.Equal(t, report.Failed, snapshot.Failed)
	assert.Equal(t, report.AvgLatency, snapshot.AvgLatency)
}

// ===================== percentile =====================

func TestPercentile_SingleBucket(t *testing.T) {
	m := NewMetricsCollector()
	// 所有请求落入第一个桶 (0.1ms)
	bucketCounts := make([]int64, len(latencyBounds))
	bucketCounts[0] = 10
	result := m.percentile(bucketCounts, 0, 10, 0.5)
	assert.Equal(t, latencyBounds[0], result)
}

func TestPercentile_P50(t *testing.T) {
	m := NewMetricsCollector()
	// 10 个请求：5 个在 1ms 桶，5 个在 100ms 桶
	bucketCounts := make([]int64, len(latencyBounds))
	bucketCounts[3] = 5  // 1ms 桶
	bucketCounts[9] = 5  // 100ms 桶

	result := m.percentile(bucketCounts, 0, 10, 0.5)
	assert.Equal(t, latencyBounds[3], result) // P50 = 1ms
}

func TestPercentile_OverBucket(t *testing.T) {
	m := NewMetricsCollector()
	// 所有请求超过最大桶边界
	bucketCounts := make([]int64, len(latencyBounds))
	result := m.percentile(bucketCounts, 10, 10, 0.5)
	assert.Equal(t, latencyBounds[len(latencyBounds)-1], result)
}

func TestPercentile_ZeroTotal(t *testing.T) {
	m := NewMetricsCollector()
	bucketCounts := make([]int64, len(latencyBounds))
	result := m.percentile(bucketCounts, 0, 0, 0.5)
	assert.Equal(t, 0.0, result)
}

// ===================== 并发安全 =====================

func TestMetricsCollector_ConcurrentRecord(t *testing.T) {
	m := NewMetricsCollector()
	var wg sync.WaitGroup
	numGoroutines := 100
	recordsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < recordsPerGoroutine; j++ {
				m.Record(1*time.Millisecond, true)
			}
		}()
	}
	wg.Wait()

	report := m.GenerateReport()
	expected := int64(numGoroutines * recordsPerGoroutine)
	assert.Equal(t, expected, report.Success)
	assert.Equal(t, int64(0), report.Failed)
}
