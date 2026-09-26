package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"thrift-perf-tool/internal/model"
	"time"
)

// 日志采样策略：调用日志通过 SSE 推送给前端，高 QPS 下逐条推送会打满
// 通道并拖垮浏览器，因此前 logSampleFull 条全量推送，之后每 logSampleEvery
// 条采样一条。指标统计不受影响（所有请求均计入报告）。
const (
	logSampleFull  = 100
	logSampleEvery = 50
)

// maxErrorSampleLen 报告中首个错误信息的最大长度
const maxErrorSampleLen = 300

// TestEngine 测试引擎
type TestEngine struct {
	client  *ThriftClient
	config  *model.TestConfig
	metrics *MetricsCollector

	firstError atomic.Pointer[string] // 首个调用错误（仅记录一次）
}

// NewTestEngine 创建新的测试引擎
func NewTestEngine(config *model.TestConfig) (*TestEngine, error) {
	client, err := NewThriftClient(config)
	if err != nil {
		return nil, err
	}

	return &TestEngine{
		client:  client,
		config:  config,
		metrics: NewMetricsCollector(),
	}, nil
}

// Run 执行性能测试
// parent 用于控制测试生命周期（如 HTTP 请求上下文：客户端断开时测试随之终止）
func (e *TestEngine) Run(parent context.Context) *model.TestReport {
	defer e.client.Close()

	ctx, cancel := e.testContext(parent)
	defer cancel()

	var wg sync.WaitGroup
	var totalCalls int64

	for i := 0; i < e.config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e.config.TestMode == "count" {
				e.workerWithCount(ctx, &totalCalls, cancel)
			} else {
				e.worker(ctx)
			}
		}()
	}

	wg.Wait()
	return e.report()
}

// RunWithCallback 执行性能测试,并定期通过回调函数推送进度（用于实时流式传输）
func (e *TestEngine) RunWithCallback(
	parent context.Context,
	progressCallback func(*model.TestReport),
	logCallback func(*model.CallLog),
) *model.TestReport {
	defer e.client.Close()

	ctx, cancel := e.testContext(parent)

	var wg sync.WaitGroup
	var callCounter int64
	var totalCalls int64

	// 启动多个 goroutine 并发调用
	for i := 0; i < e.config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e.config.TestMode == "count" {
				e.workerWithCountAndLog(ctx, &callCounter, &totalCalls, logCallback, cancel)
			} else {
				e.workerWithLog(ctx, &callCounter, logCallback)
			}
		}()
	}

	// 启动定时推送 goroutine
	stopChan := make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// 每秒生成快照并推送
				snapshot := e.metrics.GetSnapshot()
				progressCallback(snapshot)
			case <-stopChan:
				return
			}
		}
	}()

	// 等待所有 worker 完成
	wg.Wait()
	close(stopChan)
	cancel()

	// 生成最终测试报告
	return e.report()
}

// testContext 根据测试模式构建控制上下文：
// count 模式仅随 parent 取消（无总时长限制）；duration 模式附加超时。
func (e *TestEngine) testContext(parent context.Context) (context.Context, context.CancelFunc) {
	if e.config.TestMode == "count" {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, time.Duration(e.config.Duration)*time.Second)
}

// report 生成测试报告并附加首个错误信息
func (e *TestEngine) report() *model.TestReport {
	report := e.metrics.GenerateReport()
	if p := e.firstError.Load(); p != nil {
		report.ErrorSample = *p
	}
	return report
}

// recordFirstError 记录首个调用错误（只保留第一次）
func (e *TestEngine) recordFirstError(err error) {
	if err == nil || e.firstError.Load() != nil {
		return
	}
	s := flattenError(err)
	e.firstError.CompareAndSwap(nil, &s)
}

// flattenError 将错误信息压成单行并截断，便于在报告中展示
func flattenError(err error) string {
	s := strings.Join(strings.Fields(err.Error()), " ")
	if len(s) > maxErrorSampleLen {
		s = s[:maxErrorSampleLen] + "..."
	}
	return s
}

// worker 工作协程，持续发送请求直到上下文取消
func (e *TestEngine) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			start := time.Now()
			_, err := e.client.Call(context.Background())
			latency := time.Since(start)

			success := err == nil
			e.metrics.Record(latency, success)
			e.recordFirstError(err)
		}
	}
}

// workerWithCount 按次数的工作协程，发送指定次数的请求后停止
func (e *TestEngine) workerWithCount(
	ctx context.Context,
	totalCalls *int64,
	cancel context.CancelFunc,
) {
	for {
		current := atomic.LoadInt64(totalCalls)
		if current >= int64(e.config.TotalCount) {
			return
		}

		newCount := atomic.AddInt64(totalCalls, 1)
		if newCount > int64(e.config.TotalCount) {
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
			start := time.Now()
			_, err := e.client.Call(context.Background())
			latency := time.Since(start)

			success := err == nil
			e.metrics.Record(latency, success)
			e.recordFirstError(err)

			if newCount == int64(e.config.TotalCount) {
				cancel()
			}
		}
	}
}

// shouldPushLog 判断第 idx 次调用是否需要推送日志（前 logSampleFull 条全量，之后按间隔采样）
func shouldPushLog(idx int64) bool {
	return idx <= logSampleFull || idx%logSampleEvery == 0
}

// workerWithLog 工作协程，持续发送请求直到超时，并按采样策略推送调用日志
func (e *TestEngine) workerWithLog(ctx context.Context, counter *int64, logCallback func(*model.CallLog)) {
	for {
		select {
		case <-ctx.Done():
			// 超时，退出
			return
		default:
			// 执行一次请求
			start := time.Now()
			resp, err := e.client.Call(context.Background())
			latency := time.Since(start)

			// 记录结果
			success := err == nil
			e.metrics.Record(latency, success)
			e.recordFirstError(err)

			// 按采样策略推送调用日志
			if logCallback != nil {
				callID := atomic.AddInt64(counter, 1)
				if shouldPushLog(callID) {
					logCallback(&model.CallLog{
						ID:        callID,
						Timestamp: start.Format("15:04:05.000"),
						Success:   success,
						Latency:   float64(latency.Milliseconds()),
						Result:    formatResult(resp, err),
					})
				}
			}
		}
	}
}

// workerWithCountAndLog 按次数的工作协程（带日志回调），发送指定次数的请求后停止
func (e *TestEngine) workerWithCountAndLog(
	ctx context.Context,
	counter *int64,
	totalCalls *int64,
	logCallback func(*model.CallLog),
	cancel context.CancelFunc,
) {
	for {
		// 检查是否达到总次数
		current := atomic.LoadInt64(totalCalls)
		if current >= int64(e.config.TotalCount) {
			return
		}

		// 原子递增并检查
		newCount := atomic.AddInt64(totalCalls, 1)
		if newCount > int64(e.config.TotalCount) {
			// 超过了总次数，不执行请求
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
			// 执行一次请求
			start := time.Now()
			resp, err := e.client.Call(context.Background())
			latency := time.Since(start)

			// 记录结果
			success := err == nil
			e.metrics.Record(latency, success)
			e.recordFirstError(err)

			// 按采样策略推送调用日志
			if logCallback != nil {
				callID := atomic.AddInt64(counter, 1)
				if shouldPushLog(callID) {
					logCallback(&model.CallLog{
						ID:        callID,
						Timestamp: start.Format("15:04:05.000"),
						Success:   success,
						Latency:   float64(latency.Milliseconds()),
						Result:    formatResult(resp, err),
					})
				}
			}

			// 如果这是最后一个请求，触发取消
			if newCount == int64(e.config.TotalCount) {
				cancel()
			}
		}
	}
}

// formatResult 格式化响应结果或错误信息
func formatResult(resp interface{}, err error) string {
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	// 返回完整的响应结果
	return fmt.Sprintf("%v", resp)
}
