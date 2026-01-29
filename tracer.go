package zltrace

import (
	"context"
	"sync"
)

// ============================================================================
// 核心接口定义
// ============================================================================

// Tracer 定义分布式追踪器接口
type Tracer interface {
	// StartSpan 启动一个新的 span
	// 返回 span 和带有 trace 上下文的 context
	StartSpan(ctx context.Context, operationName string) (Span, context.Context)

	// Inject 将 trace 上下文注入到 carrier
	// 用于 HTTP 客户端、Kafka 生产者等
	Inject(ctx context.Context, carrier Carrier) error

	// Extract 从 carrier 中提取 trace 上下文
	// 用于 HTTP 服务端、Kafka 消费者等
	Extract(ctx context.Context, carrier Carrier) (context.Context, error)

	// Close 关闭追踪器
	Close() error
}

// Span 定义分布式追踪 span 接口
type Span interface {
	// Context 返回 span 的 context
	Context() context.Context

	// SetTag 设置标签
	SetTag(key string, value interface{})

	// SetError 设置错误信息
	SetError(err error)

	// Finish 结束 span
	Finish()

	// TraceID 返回 trace_id
	TraceID() string
}

// Carrier 定义 trace 上下文载体接口
// 用于在跨进程调用时传递 trace 上下文
type Carrier interface {
	// Get 根据 key 获取值
	Get(key string) (string, bool)

	// Set 设置 key-value 对
	Set(key, value string)
}

// ============================================================================
// 全局 Tracer 管理
// ============================================================================

var (
	globalTracer   Tracer
	globalTracerMu sync.RWMutex
)

// RegisterTracer 注册全局追踪器（线程安全）
func RegisterTracer(tracer Tracer) {
	globalTracerMu.Lock()
	defer globalTracerMu.Unlock()
	globalTracer = tracer
}

// GetTracer 获取全局追踪器（线程安全）
func GetTracer() Tracer {
	globalTracerMu.RLock()
	defer globalTracerMu.RUnlock()
	return globalTracer
}

// ============================================================================
// TraceIDProvider 实现（用于注册到 zllog）
// ============================================================================

// tracerProvider 实现 zllog.TraceIDProvider 接口
type tracerProvider struct {
	tracer Tracer
	name   string
}

// NewTraceIDProvider 创建 TraceIDProvider（用于注册到 zllog）
// 这个 provider 会从当前 tracer 中提取 trace_id
func NewTraceIDProvider(tracer Tracer, name string) *tracerProvider {
	return &tracerProvider{
		tracer: tracer,
		name:   name,
	}
}

// GetTraceID 从 context 中提取 trace_id
func (p *tracerProvider) GetTraceID(ctx context.Context) string {
	if p.tracer == nil {
		return ""
	}

	// 从 span 中提取 trace_id
	span := SpanFromContext(ctx)
	if span == nil {
		return ""
	}

	// 调用 span 的 TraceID() 方法
	return span.TraceID()
}

// Name 返回追踪系统名称
func (p *tracerProvider) Name() string {
	return p.name
}

// ============================================================================
// Context 相关辅助函数
// ============================================================================

// contextKey 是 context 的 key 类型
type contextKey int

const (
_spanKey contextKey = iota
)

// SpanFromContext 从 context 中获取 span
func SpanFromContext(ctx context.Context) Span {
	if span, ok := ctx.Value(_spanKey).(Span); ok {
		return span
	}
	return nil
}

// ContextWithSpan 将 span 添加到 context
func ContextWithSpan(ctx context.Context, span Span) context.Context {
	return context.WithValue(ctx, _spanKey, span)
}
