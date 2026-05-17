package service

import (
	"context"
	"fmt"
	"strings"
	"thrift-perf-tool/internal/model"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/client/genericclient"
	"github.com/cloudwego/kitex/pkg/generic"
	"github.com/cloudwego/kitex/transport"
)

// ThriftClient Thrift 客户端管理器
type ThriftClient struct {
	client   genericclient.Client
	provider generic.DescriptorProvider
	config   *model.TestConfig
}

// NewThriftClient 创建新的 Thrift 客户端
func NewThriftClient(config *model.TestConfig) (*ThriftClient, error) {
	// 创建 Thrift 文件提供者
	provider, err := generic.NewThriftFileProvider(config.IDLPath)
	if err != nil {
		return nil, fmt.Errorf("加载 IDL 文件失败: %w", err)
	}

	// 创建 JSON 泛化调用对象
	g, err := generic.JSONThriftGeneric(provider)
	if err != nil {
		provider.Close() // 创建泛化对象失败时释放 provider
		return nil, fmt.Errorf("创建泛化调用对象失败: %w", err)
	}

	// 构建客户端选项
	opts := []client.Option{
		client.WithHostPorts(fmt.Sprintf("%s:%d", config.Host, config.Port)),
		getProtocolOption(config.Protocol),
	}

	// 创建泛化客户端
	cli, err := genericclient.NewClient(config.Service, g, opts...)
	if err != nil {
		provider.Close() // 创建客户端失败时释放 provider
		return nil, fmt.Errorf("创建泛化客户端失败: %w", err)
	}

	return &ThriftClient{
		client:   cli,
		provider: provider,
		config:   config,
	}, nil
}

// Close 释放客户端资源（连接池、IDL 文件句柄等）
func (c *ThriftClient) Close() {
	c.provider.Close()
}

// Call 调用 Thrift 方法
func (c *ThriftClient) Call(ctx context.Context) (interface{}, error) {
	resp, err := c.client.GenericCall(ctx, c.config.Method, c.config.Params)
	if err != nil {
		return nil, c.wrapError(err)
	}
	return resp, nil
}

// wrapError 包装错误，提供详细的中文说明和原始错误
func (c *ThriftClient) wrapError(err error) error {
	errMsg := strings.ToLower(err.Error())

	// 1. 网络连接错误
	if strings.Contains(errMsg, "connection refused") {
		return fmt.Errorf(`❌ 连接被拒绝

【连接信息】
目标地址：%s:%d

【可能原因】
1. Thrift 服务未启动（demo-server 默认端口 8888）
2. 端口配置错误
3. 防火墙阻止连接

【原始错误】
%v`, c.config.Host, c.config.Port, err)
	}

	if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline exceeded") {
		return fmt.Errorf(`❌ 连接超时

【连接信息】
目标地址：%s:%d

【可能原因】
1. 服务响应缓慢或未响应
2. 网络延迟过高
3. 服务端处理时间过长

【原始错误】
%v`, c.config.Host, c.config.Port, err)
	}

	if strings.Contains(errMsg, "no such host") || strings.Contains(errMsg, "unknown host") {
		return fmt.Errorf(`❌ 主机不存在

【连接信息】
目标地址：%s:%d

【可能原因】
1. 主机地址输入错误
2. DNS 解析失败
3. 网络配置问题

【原始错误】
%v`, c.config.Host, c.config.Port, err)
	}

	if strings.Contains(errMsg, "network") || strings.Contains(errMsg, "dial") || strings.Contains(errMsg, "connection") {
		return fmt.Errorf(`❌ 网络连接失败

【连接信息】
目标地址：%s:%d

【可能原因】
1. 网络不可达
2. 服务未启动
3. 端口配置错误

【原始错误】
%v`, c.config.Host, c.config.Port, err)
	}

	// 2. 方法不存在错误
	if strings.Contains(errMsg, "unknown method") || strings.Contains(errMsg, "method not found") {
		return fmt.Errorf(`❌ 方法不存在

【调用信息】
服务：%s
方法：%s

【可能原因】
1. 方法名拼写错误
2. IDL 文件与实际服务不匹配
3. 服务版本不一致

【原始错误】
%v`, c.config.Service, c.config.Method, err)
	}

	// 3. 参数格式错误
	if strings.Contains(errMsg, "json") || strings.Contains(errMsg, "unmarshal") || strings.Contains(errMsg, "parse") {
		return fmt.Errorf(`❌ 参数格式错误

【调用信息】
方法：%s
参数：%s

【可能原因】
1. JSON 格式不正确
2. 参数类型不匹配
3. 缺少必填字段

【原始错误】
%v`, c.config.Method, c.config.Params, err)
	}

	// 4. 协议错误
	if strings.Contains(errMsg, "protocol") || strings.Contains(errMsg, "thrift") {
		return fmt.Errorf(`❌ 协议错误

【配置信息】
协议：%s
服务：%s

【可能原因】
1. 客户端与服务端协议不匹配
2. IDL 文件损坏或不完整
3. Thrift 版本不兼容

【原始错误】
%v`, c.config.Protocol, c.config.Service, err)
	}

	// 5. 其他未分类错误，直接返回原始错误
	return fmt.Errorf(`❌ 调用失败

【原始错误】
%v`, err)
}

// getProtocolOption 根据协议类型获取客户端选项
// 注意：这里的"协议"指的是传输协议（Transport Protocol），而非序列化协议。
// Kitex 泛化调用使用 JSONThriftGeneric，序列化层固定为 Thrift Binary + JSON 映射，
// 但传输层需要与服务端匹配。
func getProtocolOption(protocolType string) client.Option {
	switch protocolType {
	case "binary":
		// PurePayload：无额外头部，发送原始序列化数据（兼容老版本 Thrift 服务）
		return client.WithTransportProtocol(transport.PurePayload)
	case "framed":
		// Framed：在数据前添加 4 字节长度前缀（Kitex v0.13+ 默认传输方式）
		return client.WithTransportProtocol(transport.Framed)
	case "ttheader":
		// TTHeader：带丰富元数据的传输协议（推荐，支持服务治理和链路追踪）
		return client.WithTransportProtocol(transport.TTHeader)
	default:
		// 默认使用 Framed（Kitex v0.13+ 默认）
		return client.WithTransportProtocol(transport.Framed)
	}
}
