package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"thrift-perf-tool/internal/model"
)

// ===================== getProtocolOption =====================

func TestGetProtocolOption_Binary(t *testing.T) {
	opt := getProtocolOption("binary")
	assert.NotNil(t, opt)
}

func TestGetProtocolOption_Framed(t *testing.T) {
	opt := getProtocolOption("framed")
	assert.NotNil(t, opt)
}

func TestGetProtocolOption_TTHeader(t *testing.T) {
	opt := getProtocolOption("ttheader")
	assert.NotNil(t, opt)
}

func TestGetProtocolOption_Default(t *testing.T) {
	opt := getProtocolOption("unknown")
	assert.NotNil(t, opt)
	// 默认应返回 Framed
}

// ===================== wrapError =====================

func TestWrapError_ConnectionRefused(t *testing.T) {
	c := &ThriftClient{config: &model.TestConfig{Host: "localhost", Port: 8888}}
	wrapped := c.wrapError(errors.New("connection refused"))
	assert.True(t, strings.Contains(wrapped.Error(), "连接被拒绝"))
}

func TestWrapError_Timeout(t *testing.T) {
	c := &ThriftClient{config: &model.TestConfig{Host: "localhost", Port: 8888}}
	wrapped := c.wrapError(errors.New("context deadline exceeded"))
	assert.True(t, strings.Contains(wrapped.Error(), "连接超时"))
}

func TestWrapError_NoSuchHost(t *testing.T) {
	c := &ThriftClient{config: &model.TestConfig{Host: "badhost", Port: 8888}}
	wrapped := c.wrapError(errors.New("no such host"))
	assert.True(t, strings.Contains(wrapped.Error(), "主机不存在"))
}

func TestWrapError_NetworkError(t *testing.T) {
	c := &ThriftClient{config: &model.TestConfig{Host: "localhost", Port: 8888}}
	wrapped := c.wrapError(errors.New("network is unreachable"))
	assert.True(t, strings.Contains(wrapped.Error(), "网络连接失败"))
}

func TestWrapError_UnknownMethod(t *testing.T) {
	c := &ThriftClient{config: &model.TestConfig{Service: "UserService", Method: "badMethod"}}
	wrapped := c.wrapError(errors.New("unknown method badMethod"))
	assert.True(t, strings.Contains(wrapped.Error(), "方法不存在"))
}

func TestWrapError_JSONParse(t *testing.T) {
	c := &ThriftClient{config: &model.TestConfig{Method: "getUser", Params: `{"id": 1}`}}
	wrapped := c.wrapError(errors.New("json unmarshal failed"))
	assert.True(t, strings.Contains(wrapped.Error(), "参数格式错误"))
}

func TestWrapError_Protocol(t *testing.T) {
	c := &ThriftClient{config: &model.TestConfig{Protocol: "framed", Service: "UserService"}}
	wrapped := c.wrapError(errors.New("thrift protocol error"))
	assert.True(t, strings.Contains(wrapped.Error(), "协议错误"))
}

func TestWrapError_Unknown(t *testing.T) {
	c := &ThriftClient{config: &model.TestConfig{Host: "localhost", Port: 8888}}
	wrapped := c.wrapError(errors.New("something completely unexpected"))
	assert.True(t, strings.Contains(wrapped.Error(), "调用失败"))
}
