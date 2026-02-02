package zltrace

import (
	"context"
	"fmt"

	"github.com/zlxdbj/zllog"
)

// ============================================================================
// 初始化函数
// ============================================================================

// Init 统一初始化函数
// 自动完成以下任务：
// 1. 从配置文件加载配置
// 2. 创建 OpenTelemetry Tracer（根据 exporter.type 决定行为）
// 3. 注册到 zllog 系统（自动提取 trace_id）
//
// 使用示例：
//
//	if err := zltrace.Init(); err != nil {
//	    zllog.Error(context.Background(), "init", "追踪系统初始化失败",
//	        zllog.Error(err))
//	    // 追踪系统初始化失败不影响业务运行，程序可以继续
//	}
//
// **配置决定行为**：
//   - exporter.type=otlp: 发送到追踪系统（SkyWalking、Jaeger）
//   - exporter.type=stdout: 输出到日志（降级模式）
//   - exporter.type=none: 不发送追踪数据
//   - trace.enabled=false: 完全禁用追踪
//
// **推荐使用 InitTracer()** - 与 zllog.InitLogger() 命名风格统一
func Init() error {
	return InitTracer()
}

// InitTracer 初始化追踪系统（推荐使用）
//
// 与 zllog.InitLogger() 命名风格统一，更清晰地表达初始化追踪器的语义。
// 这是 Init() 的别名，功能完全相同。
//
// 使用示例：
//
//	if err := zltrace.InitTracer(); err != nil {
//	    zllog.Error(context.Background(), "init", "追踪系统初始化失败",
//	        zllog.Error(err))
//	    // 追踪系统初始化失败不影响业务运行，程序可以继续
//	}
//
// **配置决定行为**：
//   - exporter.type=otlp: 发送到追踪系统（SkyWalking、Jaeger）
//   - exporter.type=stdout: 输出到日志（降级模式）
//   - exporter.type=none: 不发送追踪数据
//   - trace.enabled=false: 完全禁用追踪
func InitTracer() error {
	// 1. 加载配置
	config, err := LoadConfig()
	if err != nil {
		zllog.Error(context.Background(), "zltrace.init", "加载追踪配置失败", err)
		return fmt.Errorf("failed to load trace config: %w", err)
	}

	// 2. 如果未启用追踪，直接返回
	if !config.Enabled {
		zllog.Info(context.Background(), "zltrace.init", "追踪系统未启用")
		return nil
	}

	// 3. 初始化 OpenTelemetry Tracer
	if err := InitOpenTelemetryTracer(); err != nil {
		zllog.Error(context.Background(), "zltrace.init", "初始化 OpenTelemetry 追踪器失败", err)
		return fmt.Errorf("failed to init OpenTelemetry tracer: %w", err)
	}

	// 4. 记录初始化完成
	zllog.Info(context.Background(), "zltrace.init", "OpenTelemetry 追踪系统初始化完成",
		zllog.String("service_name", config.ServiceName),
		zllog.String("exporter_type", config.Exporter.Type),
		zllog.Bool("enabled", config.Enabled))

	return nil
}

// ============================================================================
// 辅助函数
// ============================================================================

// GetSafeTracer 获取安全的 Tracer
// 如果全局 tracer 为 nil，返回 noOpTracer 而不是 nil
// 这样可以避免调用方出现 nil pointer dereference
func GetSafeTracer() Tracer {
	tracer := GetTracer()
	if tracer == nil {
		return &noOpTracer{}
	}
	return tracer
}

// ============================================================================
// NoOp Tracer（空操作 Tracer）
// ============================================================================

// noOpTracer 空操作 Tracer
// 当全局 tracer 为 nil 时，返回 noOpTracer 避免 panic
type noOpTracer struct{}

// StartSpan 启动 span（空操作）
func (t *noOpTracer) StartSpan(ctx context.Context, operationName string) (Span, context.Context) {
	return &noOpSpan{}, ctx
}

// Inject 注入 trace 上下文（空操作）
func (t *noOpTracer) Inject(ctx context.Context, carrier Carrier) error {
	return nil
}

// Extract 提取 trace 上下文（空操作）
func (t *noOpTracer) Extract(ctx context.Context, carrier Carrier) (context.Context, error) {
	return ctx, nil
}

// Close 关闭追踪器（空操作）
func (t *noOpTracer) Close() error {
	return nil
}

// noOpSpan 空操作 Span
type noOpSpan struct{}

// Context 返回 context（空操作）
func (s *noOpSpan) Context() context.Context {
	return context.Background()
}

// SetTag 设置标签（空操作）
func (s *noOpSpan) SetTag(key string, value interface{}) {}

// SetError 设置错误（空操作）
func (s *noOpSpan) SetError(err error) {}

// Finish 结束 span（空操作）
func (s *noOpSpan) Finish() {}

// TraceID 返回 trace_id（空字符串）
func (s *noOpSpan) TraceID() string {
	return ""
}

// ============================================================================
// HTTP 中间件获取函数
// ============================================================================

// HTTPTraceHandler HTTP追踪处理器接口
// 用于不同框架的适配
type HTTPTraceHandler interface {
	// GetMethod 获取HTTP方法
	GetMethod() string
	// GetURL 获取请求URL
	GetURL() string
	// GetHeader 获取请求头
	GetHeader(key string) string
	// SetSpanContext 设置span到context
	SetSpanContext(ctx context.Context)
	// GetSpanContext 获取span从context
	GetSpanContext() context.Context
}

// TraceHTTPRequest 通用HTTP请求追踪函数
// 框架无关，可以在任何HTTP框架的中间件中调用
// 使用示例：
//
//	func MyMiddleware(ctx context.Context, handler HTTPTraceHandler, next func()) {
//		zltrace.TraceHTTPRequest(ctx, handler, next)
//	}
func TraceHTTPRequest(ctx context.Context, handler HTTPTraceHandler, next func()) {
	// 如果没有全局 tracer，直接调用next
	tracer := GetTracer()
	if tracer == nil {
		next()
		return
	}

	// 从请求头提取trace信息（支持 W3C traceparent header）
	carrier := &HTTPHeaderCarrier{handler}
	extractedCtx, _ := tracer.Extract(ctx, carrier)

	// 创建Entry Span（如果有上游trace则继承，否则生成新的）
	operationName := handler.GetMethod() + " " + handler.GetURL()
	span, spanCtx := tracer.StartSpan(extractedCtx, operationName)

	// 将span注入到context
	handler.SetSpanContext(spanCtx)

	// 调用下一个处理器
	next()

	// 结束span
	span.Finish()
}

// HTTPHeaderCarrier HTTP请求头载体（实现Carrier接口）
type HTTPHeaderCarrier struct {
	handler HTTPTraceHandler
}

// Get 根据key获取值
func (c *HTTPHeaderCarrier) Get(key string) (string, bool) {
	value := c.handler.GetHeader(key)
	return value, value != ""
}

// Set 设置key-value对（HTTP请求头不需要Set）
func (c *HTTPHeaderCarrier) Set(key, value string) {
	// HTTP请求头不需要设置（注入由其他地方处理）
}
