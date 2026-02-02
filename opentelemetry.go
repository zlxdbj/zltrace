package zltrace

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"github.com/zlxdbj/zllog"
)

// ============================================================================
// OTELTracer - OpenTelemetry 追踪器实现
// ============================================================================

// OTELTracer 使用 OpenTelemetry 实现 Tracer 接口
// 支持 W3C Trace Context 标准（traceparent header）
type OTELTracer struct {
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
}

// StartSpan 启动一个新的 span（实现 Tracer 接口）
//
// 自动生成 trace_id（如果 context 中没有），
// 并创建符合 W3C Trace Context 标准的 span。
func (t *OTELTracer) StartSpan(ctx context.Context, operationName string) (Span, context.Context) {
	ctx, span := t.tracer.Start(ctx, operationName)
	return &OTELSpan{span: span}, ctx
}

// Inject 将 trace 上下文注入到 carrier（实现 Tracer 接口）
//
// 使用 W3C Trace Context 标准格式（traceparent header）。
func (t *OTELTracer) Inject(ctx context.Context, carrier Carrier) error {
	// 将我们的 Carrier 接口适配为 OTEL 的 TextMapCarrier 接口
	otelCarrier := &carrierAdapter{carrier: carrier}
	t.propagator.Inject(ctx, otelCarrier)
	return nil
}

// Extract 从 carrier 中提取 trace 上下文（实现 Tracer 接口）
//
// 从 W3C Trace Context 格式（traceparent header）中提取 trace 信息。
func (t *OTELTracer) Extract(ctx context.Context, carrier Carrier) (context.Context, error) {
	// 将我们的 Carrier 接口适配为 OTEL 的 TextMapCarrier 接口
	otelCarrier := &carrierAdapter{carrier: carrier}
	ctx = t.propagator.Extract(ctx, otelCarrier)
	return ctx, nil
}

// Close 关闭追踪器（实现 Tracer 接口）
func (t *OTELTracer) Close() error {
	return nil
}

// ============================================================================
// OTELSpan - OpenTelemetry Span 实现
// ============================================================================

// OTELSpan 使用 OpenTelemetry 实现 Span 接口
type OTELSpan struct {
	span trace.Span
}

// Context 返回 span 的 context（实现 Span 接口）
func (s *OTELSpan) Context() context.Context {
	return context.Background()
}

// SetTag 设置标签（实现 Span 接口）
func (s *OTELSpan) SetTag(key string, value interface{}) {
	s.span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", value)))
}

// SetError 设置错误信息（实现 Span 接口）
func (s *OTELSpan) SetError(err error) {
	if err != nil {
		s.span.RecordError(err)
		s.span.SetAttributes(attribute.String("error", err.Error()))
	}
}

// Finish 结束 span（实现 Span 接口）
func (s *OTELSpan) Finish() {
	s.span.End()
}

// TraceID 返回 trace_id（实现 Span 接口）
//
// 返回 W3C Trace Context 格式的 trace_id（32位十六进制字符串）
func (s *OTELSpan) TraceID() string {
	spanCtx := s.span.SpanContext()
	return spanCtx.TraceID().String()
}

// ============================================================================
// OpenTelemetry 初始化函数
// ============================================================================

// InitOpenTelemetryTracer 初始化 OpenTelemetry Tracer
//
// 从配置文件读取配置，创建并注册 OpenTelemetry Tracer。
// 根据 exporter.type 决定追踪数据发送到哪里：
//   - otlp: 发送到追踪系统（SkyWalking、Jaeger 等）
//   - stdout: 输出到日志（降级模式）
//   - none: 不发送追踪数据
func InitOpenTelemetryTracer() error {
	// 1. 读取配置
	config, err := LoadConfig()
	if err != nil {
		zllog.Error(context.Background(), "trace.init", "读取追踪配置失败", err)
		return fmt.Errorf("读取追踪配置失败: %w", err)
	}

	if !config.Enabled {
		zllog.Info(context.Background(), "trace", "追踪系统未启用")
		return nil
	}

	// 2. 创建 Resource
	res, err := createResource(config.ServiceName)
	if err != nil {
		zllog.Error(context.Background(), "trace.init", "创建 OpenTelemetry Resource 失败", err)
		return fmt.Errorf("创建 OpenTelemetry Resource 失败: %w", err)
	}

	// 3. 创建 Exporter（根据 type 决定）
	exporter, err := createExporterByType(config)
	if err != nil {
		zllog.Error(context.Background(), "trace.init", "创建 OpenTelemetry Exporter 失败", err)
		return fmt.Errorf("创建 OpenTelemetry Exporter 失败: %w", err)
	}

	// 4. 创建采样器
	sampler := createSampler(config.Sampler)

	// 5. 创建 TracerProvider
	var tpOpts []sdktrace.TracerProviderOption
	if exporter != nil {
		// 只有当 exporter 不是 none 时才添加 Batcher
		tpOpts = append(tpOpts,
			sdktrace.WithBatcher(exporter,
				sdktrace.WithBatchTimeout(time.Duration(config.Batch.Timeout)*time.Second),
				sdktrace.WithMaxQueueSize(config.Batch.MaxQueueSize),
			),
		)
	}
	tpOpts = append(tpOpts,
		sdktrace.WithSampler(sampler),
		sdktrace.WithResource(res),
	)

	tp := sdktrace.NewTracerProvider(tpOpts...)

	// 6. 设置全局 TracerProvider
	otel.SetTracerProvider(tp)

	// 7. 创建包装器并注册
	tracer := tp.Tracer(config.ServiceName)
	otelTracer := &OTELTracer{
		tracer:     tracer,
		propagator: propagation.TraceContext{}, // W3C Trace Context
	}

	RegisterTracer(otelTracer)

	// 8. 注册到 zllog
	zllog.RegisterTraceIDProvider(&OTELProvider{tracer: otelTracer, name: "opentelemetry"})

	// 9. 设置全局传播器
	SetGlobalPropagators()

	zllog.Info(context.Background(), "trace", "OpenTelemetry Tracer 初始化成功",
		zllog.String("service_name", config.ServiceName),
		zllog.String("exporter_type", config.Exporter.Type),
		zllog.String("endpoint", config.Exporter.OTLP.Endpoint))

	return nil
}

// SetGlobalPropagators 设置全局传播器
//
// 配置 W3C Trace Context 作为默认的上下文传播器。
func SetGlobalPropagators() {
	// 使用 W3C Trace Context 作为默认传播器
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // W3C Trace Context (traceparent)
		propagation.Baggage{},       // W3C Baggage
	)

	otel.SetTextMapPropagator(propagator)

	zllog.Info(context.Background(), "trace", "全局传播器已设置为 W3C Trace Context")
}

// ============================================================================
// OTELProvider - TraceIDProvider 实现（用于注册到 zllog）
// ============================================================================

// OTELProvider 实现 zllog.TraceIDProvider 接口
// 用于从 OpenTelemetry Span 中提取 trace_id
type OTELProvider struct {
	tracer *OTELTracer
	name   string
}

// GetTraceID 从 context 中提取 trace_id（实现 zllog.TraceIDProvider 接口）
func (p *OTELProvider) GetTraceID(ctx context.Context) string {
	span := SpanFromContext(ctx)
	if span == nil {
		return ""
	}

	if otelSpan, ok := span.(*OTELSpan); ok {
		return otelSpan.TraceID()
	}

	return ""
}

// Name 返回追踪系统名称（实现 zllog.TraceIDProvider 接口）
func (p *OTELProvider) Name() string {
	return p.name
}

// ============================================================================
// Carrier 适配器
// ============================================================================

// carrierAdapter 将我们的 Carrier 接口适配为 OTEL 的 TextMapCarrier 接口
type carrierAdapter struct {
	carrier Carrier
}

// Get 实现 OTEL 的 TextMapCarrier.Get 方法
// 如果 key 不存在，返回空字符串（而不是 panic）
func (a *carrierAdapter) Get(key string) string {
	if value, ok := a.carrier.Get(key); ok {
		return value
	}
	return ""
}

// Set 实现 OTEL 的 TextMapCarrier.Set 方法
func (a *carrierAdapter) Set(key, value string) {
	a.carrier.Set(key, value)
}

// Keys 实现 OTEL 的 TextMapCarrier.Keys 方法（可选）
func (a *carrierAdapter) Keys() []string {
	// 简化实现，返回空切片
	return []string{}
}

// ============================================================================
// 配置加载辅助函数
// ============================================================================

// createResource 创建 OpenTelemetry Resource
func createResource(serviceName string) (*resource.Resource, error) {
	// 使用空 SchemaURL，避免版本冲突
	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			"", // 不指定 SchemaURL，避免与 resource.Default() 冲突
			semconv.ServiceName(serviceName),
		),
	)
}

// createExporterByType 根据 exporter.type 创建对应的 Exporter
func createExporterByType(config *TraceConfig) (sdktrace.SpanExporter, error) {
	switch config.Exporter.Type {
	case "otlp":
		return createOTLPExporter(config)
	case "stdout":
		return createStdoutExporter()
	case "none":
		return nil, nil
	default:
		return nil, fmt.Errorf("不支持的 exporter 类型: %s", config.Exporter.Type)
	}
}

// createOTLPExporter 创建 OTLP gRPC Exporter
func createOTLPExporter(config *TraceConfig) (sdktrace.SpanExporter, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Exporter.OTLP.Timeout)*time.Second)
	defer cancel()

	var opts []otlptracegrpc.Option
	opts = append(opts, otlptracegrpc.WithEndpoint(config.Exporter.OTLP.Endpoint))

	if config.Exporter.OTLP.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	} else {
		// TODO: 支持 TLS
	}

	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("创建 OTLP gRPC Exporter 失败: %w", err)
	}

	return exporter, nil
}

// createSampler 创建采样器
func createSampler(config SamplerConfig) sdktrace.Sampler {
	switch config.Type {
	case "always_on":
		return sdktrace.AlwaysSample()
	case "never":
		return sdktrace.NeverSample()
	case "traceid_ratio":
		return sdktrace.TraceIDRatioBased(config.Ratio)
	case "parent_based":
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	default:
		return sdktrace.AlwaysSample()
	}
}

// createStdoutExporter 创建 Stdout Exporter（降级模式）
// 将追踪数据输出到日志，而不是发送到追踪系统
func createStdoutExporter() (sdktrace.SpanExporter, error) {
	// 使用 LoggingExporter 将 span 数据输出到日志
	return &LoggingExporter{}, nil
}

// ============================================================================
// LoggingExporter - 输出到日志的 Exporter
// ============================================================================

// LoggingExporter 将 span 数据输出到日志
// 用于降级模式或调试场景
type LoggingExporter struct{}

// NewLoggingExporter 创建日志 Exporter
func NewLoggingExporter() *LoggingExporter {
	return &LoggingExporter{}
}

// ExportSpans 导出 span 到日志
func (e *LoggingExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		spanCtx := span.SpanContext()
		traceID := spanCtx.TraceID().String()
		spanID := spanCtx.SpanID().String()

		zllog.Debug(ctx, "otel_exporter", "OpenTelemetry Span",
			zllog.String("trace_id", traceID),
			zllog.String("span_id", spanID),
			zllog.String("name", span.Name()),
			zllog.Int("duration", int(span.EndTime().Sub(span.StartTime()).Milliseconds())))
	}
	return nil
}

// Shutdown 关闭 Exporter
func (e *LoggingExporter) Shutdown(ctx context.Context) error {
	return nil
}

// ForceFlush 强制刷新
func (e *LoggingExporter) ForceFlush(ctx context.Context) error {
	return nil
}
