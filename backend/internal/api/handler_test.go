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

func TestValidateMethodParams_MultiParamRejected(t *testing.T) {
	h := &Handler{}
	h.currentInfo = &model.IDLInfo{
		Services: []model.ServiceInfo{
			{
				Name: "UserService",
				Methods: []model.MethodInfo{
					{Name: "getUser", Params: []model.ParamInfo{{Name: "req"}}},
					{Name: "getLicenseInfo", Params: []model.ParamInfo{{Name: "token"}, {Name: "device"}}},
				},
			},
		},
	}

	err := h.validateMethodParams("getLicenseInfo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "2 个参数")
	assert.Contains(t, err.Error(), "token, device")

	assert.NoError(t, h.validateMethodParams("getUser"))
	// 未在解析结果中的方法不拦截（由 Kitex 描述符检查兜底）
	assert.NoError(t, h.validateMethodParams("notParsed"))
}

func TestValidateMethodParams_NoInfo(t *testing.T) {
	h := &Handler{}
	assert.NoError(t, h.validateMethodParams("anything"))
}

func TestValidateConfig_InvalidTestMode(t *testing.T) {
	h := &Handler{}
	config := &model.TestConfig{
		Service: "UserService", Method: "getUser", Host: "localhost",
		Port: 8888, Concurrency: 1, TestMode: "", Duration: 10,
	}
	err := h.validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "测试模式无效")
}

func TestValidateConfig_InvalidProtocol(t *testing.T) {
	h := &Handler{}
	config := &model.TestConfig{
		Service: "UserService", Method: "getUser", Host: "localhost",
		Port: 8888, Concurrency: 1, Protocol: "compact",
		TestMode: "duration", Duration: 10,
	}
	err := h.validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "传输协议无效")
}

func TestValidateConfig_TimeoutDefaulted(t *testing.T) {
	h := &Handler{}
	config := &model.TestConfig{
		Service: "UserService", Method: "getUser", Host: "localhost",
		Port: 8888, Concurrency: 1, Protocol: "framed",
		TestMode: "duration", Duration: 10, TimeoutMs: 0,
	}
	err := h.validateConfig(config)
	assert.NoError(t, err)
	assert.Equal(t, 5000, config.TimeoutMs)
}

func TestValidateConfig_InvalidTimeout(t *testing.T) {
	h := &Handler{}
	config := &model.TestConfig{
		Service: "UserService", Method: "getUser", Host: "localhost",
		Port: 8888, Concurrency: 1, Protocol: "framed",
		TestMode: "duration", Duration: 10, TimeoutMs: -5,
	}
	err := h.validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "单次请求超时无效")
}
