package zltrace

import (
	"context"
	"testing"
)

func TestRegisterTracer(t *testing.T) {
	mockTracer := &mockTracer{}
	RegisterTracer(mockTracer)

	if GetTracer() != mockTracer {
		t.Error("failed to register tracer")
	}
}

func TestGetSafeTracer(t *testing.T) {
	// 清空全局 tracer
	RegisterTracer(nil)

	tracer := GetSafeTracer()
	if tracer == nil {
		t.Error("GetSafeTracer should never return nil")
	}

	// 测试方法不会 panic
	ctx := context.Background()
	span, spanCtx := tracer.StartSpan(ctx, "test")
	if span == nil {
		t.Error("span should not be nil")
	}
	if spanCtx == nil {
		t.Error("spanCtx should not be nil")
	}

	span.Finish()
	tracer.Close()
}

func TestContextWithSpan(t *testing.T) {
	span := &mockSpan{}
	ctx := context.Background()
	ctx = ContextWithSpan(ctx, span)

	retrievedSpan := SpanFromContext(ctx)
	if retrievedSpan != span {
		t.Error("failed to retrieve span from context")
	}
}

// mockTracer 用于测试
type mockTracer struct{}

func (m *mockTracer) StartSpan(ctx context.Context, operationName string) (Span, context.Context) {
	span := &mockSpan{}
	return span, ContextWithSpan(ctx, span)
}

func (m *mockTracer) Inject(ctx context.Context, carrier Carrier) error {
	carrier.Set("traceparent", "00-test123-test456-01")
	return nil
}

func (m *mockTracer) Extract(ctx context.Context, carrier Carrier) (context.Context, error) {
	return ctx, nil
}

func (m *mockTracer) Close() error {
	return nil
}

// mockSpan 用于测试
type mockSpan struct{}

func (m *mockSpan) Context() context.Context {
	return context.Background()
}

func (m *mockSpan) SetTag(key string, value interface{}) {}

func (m *mockSpan) SetError(err error) {}

func (m *mockSpan) Finish() {}

func (m *mockSpan) TraceID() string {
	return "test123"
}
