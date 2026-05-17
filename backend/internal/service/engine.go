package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"thrift-perf-tool/internal/model"
	"time"
)

// TestEngine 测试引擎
type TestEngine struct {
	client  *ThriftClient
	config  *model.TestConfig
	metrics *MetricsCollector
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
func (e *TestEngine) Run() *model.TestReport {
	defer e.client.Close()

	var ctx context.Context
	var cancel context.CancelFunc

	if e.config.TestMode == "count" {
		ctx, cancel = context.WithCancel(context.Background())
	} else {
		ctx, cancel = context.WithTimeout(
			context.Background(),
			time.Duration(e.config.Duration)*time.Second,
		)
	}
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
	return e.metrics.GenerateReport()
}

// RunWithCallback 执行性能测试,并定期通过回调函数推送进度（用于实时流式传输）
func (e *TestEngine) RunWithCallback(
	progressCallback func(*model.TestReport),
	logCallback func(*model.CallLog),
) *model.TestReport {
	defer e.client.Close()
	var ctx context.Context
	var cancel context.CancelFunc

	// 根据测试模式设置停止条件
	if e.config.TestMode == "count" {
		// 按次数模式：不设置超时
		ctx, cancel = context.WithCancel(context.Background())
	} else {
		// 按时间模式（默认）：设置超时
		ctx, cancel = context.WithTimeout(
			context.Background(),
			time.Duration(e.config.Duration)*time.Second,
		)
	}
	defer cancel()

	var wg sync.WaitGroup
	var callCounter int64
	var totalCalls int64 // 新增：总调用次数计数

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

	// 生成最终测试报告
	return e.metrics.GenerateReport()
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

			if newCount == int64(e.config.TotalCount) {
				cancel()
			}
		}
	}
}

// workerWithLog 工作协程，持续发送请求直到超时，并推送每次调用的日志
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

			// 推送调用日志
			if logCallback != nil {
				callID := atomic.AddInt64(counter, 1)
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

			// 推送调用日志
			if logCallback != nil {
				callID := atomic.AddInt64(counter, 1)
				logCallback(&model.CallLog{
					ID:        callID,
					Timestamp: start.Format("15:04:05.000"),
					Success:   success,
					Latency:   float64(latency.Milliseconds()),
					Result:    formatResult(resp, err),
				})
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
