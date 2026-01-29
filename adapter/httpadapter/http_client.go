package httpadapter

import (
	"fmt"
	"net/http"

	"github.com/zlxdbj/zltrace"
)

// TracingRoundTripper 自动注入 trace_id 的 HTTP Transport
//
// **核心功能**：
//   - ✅ 自动创建 Exit Span（表示调用外部服务）
//   - ✅ 自动注入 trace_id 到 HTTP 请求头
//   - ✅ 自动记录 HTTP 状态码到 Span
//   - ✅ 使用标准 http.RoundTripper 接口
//
// **设计理念**：
//   这是一个 Transport 包装器，只负责 trace_id 注入，
//   不改变 HTTP Client 的任何行为。
//
// **使用方式**：
//
//	import "go_shield/zltrace/adapter"
//
//	// 方式1：创建带追踪的客户端
//	client := adapter.NewTracedClient(nil)
//
//	// 方式2：使用现有的 http.Client
//	client := &http.Client{
//	    Transport: &adapter.TracingRoundTripper{
//	        Base: http.DefaultTransport,
//	    },
//	}
//
//	// 使用方式和标准库完全一样
//	resp, err := client.Do(req)
//
// **对比 Spring Cloud**：
//
//	// Spring Cloud + OpenFeign
//	User user = feignClient.getUser(userId)  // 自动传递 trace_id
//
//	// Go + zltrace adapter
//	resp, err := client.Do(req)  // 自动传递 trace_id
type TracingRoundTripper struct {
	// Base 底层的 Transport，实际执行 HTTP 请求
	// 如果为 nil，使用 http.DefaultTransport
	Base http.RoundTripper
}

// RoundTrip 实现 http.RoundTripper 接口
//
// **执行流程**：
//   1. 从 context 提取 trace 信息
//   2. 创建 Exit Span（表示调用外部服务）
//   3. 自动注入 trace_id 到请求头
//   4. 调用 Base.RoundTrip 执行实际的 HTTP 请求
//   5. 记录 HTTP 状态码到 Span
//   6. 完成 Span
//
// **注入的请求头**（W3C Trace Context 标准）：
//   - traceparent: W3C 标准追踪头（格式：00-trace_id-span_id-flags）
//   - 示例：traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
//
// **支持的追踪系统**：
//   - OpenTelemetry: 注入 traceparent header（W3C 标准）
//   - SkyWalking go2sky: 注入 sw8 header（向后兼容）
//
// 参数：
//   - req: HTTP 请求
//
// 返回：
//   - *http.Response: HTTP 响应
//   - error: 错误信息
func (t *TracingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	tracer := zltrace.GetTracer()

	// 没有注册 tracer，直接执行请求（优雅降级）
	if tracer == nil {
		return t.base().RoundTrip(req)
	}

	// 1. 创建 Exit Span（调用外部服务）
	span, spanCtx := tracer.StartSpan(ctx, "HTTP/"+req.Method)
	defer span.Finish()

	// 2. 自动注入 trace_id 到请求头
	carrier := &httpHeaderCarrier{headers: req.Header}
	if err := tracer.Inject(spanCtx, carrier); err != nil {
		// 注入失败不应该阻止请求，继续执行
	}

	// 3. 设置 span 标签（HTTP 基本信息）
	span.SetTag("http.url", req.URL.String())
	span.SetTag("http.method", req.Method)
	span.SetTag("http.host", req.URL.Host)

	// 4. 执行实际的 HTTP 请求（使用带有 span 的 context）
	resp, err := t.base().RoundTrip(req.WithContext(spanCtx))

	if err != nil {
		// 请求失败，记录错误到 span
		span.SetError(err)
		return nil, err
	}

	// 5. 记录响应状态码到 span
	span.SetTag("http.status_code", resp.StatusCode)

	// 如果是 4xx 或 5xx，记录为错误
	if resp.StatusCode >= 400 {
		span.SetTag("error", fmt.Sprintf("HTTP %d", resp.StatusCode))
	}

	return resp, nil
}

// base 获取底层的 Transport
func (t *TracingRoundTripper) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

// NewTracedClient 创建自动追踪的 HTTP Client
//
// 这是一个便捷函数，创建一个已经配置好 TracingRoundTripper 的 http.Client。
//
// 参数：
//   - client: 可选的基础客户端（如果为 nil，创建新的客户端）
//
// 返回：
//   - *http.Client: 配置好追踪的 HTTP 客户端
//
// **使用示例**：
//
//	import "go_shield/zltrace/adapter"
//
//	// 使用默认配置
//	client := adapter.NewTracedClient(nil)
//
//	// 使用自定义配置
//	customClient := &http.Client{Timeout: 5 * time.Second}
//	client := adapter.NewTracedClient(customClient)
//
//	// 使用方式和标准库完全一样
//	resp, err := client.Do(req)
func NewTracedClient(client *http.Client) *http.Client {
	if client == nil {
		client = &http.Client{}
	}

	// 如果已经有 Transport，包装它
	// 否则包装 DefaultTransport
	baseTransport := client.Transport
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}

	client.Transport = &TracingRoundTripper{
		Base: baseTransport,
	}

	return client
}

// httpHeaderCarrier 实现 zltrace.Carrier 接口，使用 HTTP header
type httpHeaderCarrier struct {
	headers http.Header
}

// Set 设置 header
func (c *httpHeaderCarrier) Set(key, value string) {
	c.headers.Set(key, value)
}

// Get 获取 header
func (c *httpHeaderCarrier) Get(key string) (string, bool) {
	values := c.headers.Values(key)
	if len(values) == 0 {
		return "", false
	}
	return values[0], true
}
