package zltrace

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestContextWithTraceID(t *testing.T) {
	// 初始化追踪器（确保 TracerProvider 已注册）
	_ = InitOpenTelemetryTracer()

	t.Run("有效的 trace_id", func(t *testing.T) {
		traceID := "4bf92f3577b34da6a3ce929d0e0e4736"
		ctx := ContextWithTraceID(context.Background(), traceID)

		// 验证 context 中包含了正确的 trace_id
		spanCtx := trace.SpanFromContext(ctx)
		sc := spanCtx.SpanContext()

		if !sc.IsValid() {
			t.Fatal("span context 应该有效")
		}
		if sc.TraceID().String() != traceID {
			t.Fatalf("trace_id 不匹配: 期望 %s, 实际 %s", traceID, sc.TraceID().String())
		}
		if !sc.IsSampled() {
			t.Fatal("span 应该被标记为 sampled")
		}
		if !sc.IsRemote() {
			t.Fatal("span 应该被标记为 remote")
		}
	})

	t.Run("无效的 trace_id 返回原始 context", func(t *testing.T) {
		original := context.Background()
		ctx := ContextWithTraceID(original, "invalid-trace-id")

		// 应该返回原始 context（不会 panic）
		if ctx != original {
			t.Fatal("无效 trace_id 应该返回原始 context")
		}
	})

	t.Run("空字符串返回原始 context", func(t *testing.T) {
		original := context.Background()
		ctx := ContextWithTraceID(original, "")

		if ctx != original {
			t.Fatal("空 trace_id 应该返回原始 context")
		}
	})

	t.Run("NewContextWithTraceID 简化版", func(t *testing.T) {
		traceID := "4bf92f3577b34da6a3ce929d0e0e4736"
		ctx := NewContextWithTraceID(traceID)

		sc := trace.SpanFromContext(ctx).SpanContext()
		if sc.TraceID().String() != traceID {
			t.Fatalf("trace_id 不匹配: 期望 %s, 实际 %s", traceID, sc.TraceID().String())
		}
	})

	t.Run("配合 StartSpan 使用", func(t *testing.T) {
		traceID := "4bf92f3577b34da6a3ce929d0e0e4736"
		ctx := ContextWithTraceID(context.Background(), traceID)

		// 基于手动构建的 context 创建 span
		tracer := GetSafeTracer()
		span, _ := tracer.StartSpan(ctx, "TestOperation")

		// span 的 trace_id 应该和手动注入的一致
		if span.TraceID() != traceID {
			t.Fatalf("span trace_id 不匹配: 期望 %s, 实际 %s", traceID, span.TraceID())
		}

		span.Finish()
	})

	t.Run("配合 zllog 使用（通过 OTELProvider 验证）", func(t *testing.T) {
		traceID := "4bf92f3577b34da6a3ce929d0e0e4736"
		ctx := ContextWithTraceID(context.Background(), traceID)

		// OTELProvider.GetTraceID 应该能从 context 中提取到 trace_id
		provider := &OTELProvider{name: "test"}
		extracted := provider.GetTraceID(ctx)

		if extracted != traceID {
			t.Fatalf("OTELProvider 提取的 trace_id 不匹配: 期望 %s, 实际 %s", traceID, extracted)
		}
	})
}
