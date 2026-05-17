package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"thrift-perf-tool/internal/model"
)

func TestValidateConfig_MissingService(t *testing.T) {
	h := &Handler{}
	config := &model.TestConfig{
		Method:      "getUser",
		Host:        "localhost",
		Port:        8888,
		Concurrency: 1,
		Duration:    10,
		TestMode:    "duration",
	}
	err := h.validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "服务名不能为空")
}

func TestValidateConfig_MissingMethod(t *testing.T) {
	h := &Handler{}
	config := &model.TestConfig{
		Service:     "UserService",
		Host:        "localhost",
		Port:        8888,
		Concurrency: 1,
		Duration:    10,
		TestMode:    "duration",
	}
	err := h.validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "方法名不能为空")
}

func TestValidateConfig_InvalidPort(t *testing.T) {
	h := &Handler{}
	config := &model.TestConfig{
		Service:     "UserService",
		Method:      "getUser",
		Host:        "localhost",
		Port:        0,
		Concurrency: 1,
		Duration:    10,
		TestMode:    "duration",
	}
	err := h.validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "端口号无效")
}

func TestValidateConfig_InvalidConcurrency(t *testing.T) {
	h := &Handler{}
	config := &model.TestConfig{
		Service:     "UserService",
		Method:      "getUser",
		Host:        "localhost",
		Port:        8888,
		Concurrency: 0,
		Duration:    10,
		TestMode:    "duration",
	}
	err := h.validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "并发数无效")
}

func TestValidateConfig_InvalidDuration(t *testing.T) {
	h := &Handler{}
	config := &model.TestConfig{
		Service:     "UserService",
		Method:      "getUser",
		Host:        "localhost",
		Port:        8888,
		Concurrency: 1,
		Duration:    0,
		TestMode:    "duration",
	}
	err := h.validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "持续时间无效")
}

func TestValidateConfig_InvalidTotalCount(t *testing.T) {
	h := &Handler{}
	config := &model.TestConfig{
		Service:     "UserService",
		Method:      "getUser",
		Host:        "localhost",
		Port:        8888,
		Concurrency: 1,
		TestMode:    "count",
		TotalCount:  0,
	}
	err := h.validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "总请求次数无效")
}

func TestValidateConfig_MissingHost(t *testing.T) {
	h := &Handler{}
	config := &model.TestConfig{
		Service:     "UserService",
		Method:      "getUser",
		Port:        8888,
		Concurrency: 1,
		Duration:    10,
		TestMode:    "duration",
	}
	err := h.validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "主机地址不能为空")
}

func TestValidateConfig_ValidDurationMode(t *testing.T) {
	h := &Handler{}
	config := &model.TestConfig{
		Service:     "UserService",
		Method:      "getUser",
		Host:        "localhost",
		Port:        8888,
		Concurrency: 5,
		Duration:    10,
		TestMode:    "duration",
	}
	err := h.validateConfig(config)
	assert.NoError(t, err)
}

func TestValidateConfig_ValidCountMode(t *testing.T) {
	h := &Handler{}
	config := &model.TestConfig{
		Service:     "UserService",
		Method:      "getUser",
		Host:        "localhost",
		Port:        8888,
		Concurrency: 5,
		TestMode:    "count",
		TotalCount:  1000,
	}
	err := h.validateConfig(config)
	assert.NoError(t, err)
}
