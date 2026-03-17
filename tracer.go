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

