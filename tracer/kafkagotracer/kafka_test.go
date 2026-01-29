package kafkagotracer

import (
	"context"
	"fmt"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/zlxdbj/zltrace"
)

func TestCreateKafkaConsumerContext(t *testing.T) {
	// 注册 mock tracer
	mockTracer := &mockTracer{}
	zltrace.RegisterTracer(mockTracer)
	defer zltrace.RegisterTracer(nil)

	// 测试没有 traceparent header 的情况
	msg := &kafka.Message{
		Topic: "test-topic",
		Value: []byte("test"),
	}

	ctx := CreateKafkaConsumerContext(msg)
	if ctx == nil {
		t.Error("context should not be nil")
	}

	// 验证创建了新的 span
	span := zltrace.SpanFromContext(ctx)
	if span == nil {
		t.Error("span should not be nil")
	}
}

func TestCreateKafkaConsumerContextWithTraceParent(t *testing.T) {
	// 注册 mock tracer
	mockTracer := &mockTracer{}
	zltrace.RegisterTracer(mockTracer)
	defer zltrace.RegisterTracer(nil)

	// 测试有 traceparent header 的情况
	msg := &kafka.Message{
		Topic: "test-topic",
		Value: []byte("test"),
		Headers: []kafka.Header{
			{Key: "traceparent", Value: []byte("00-test123-test456-01")},
		},
	}

	ctx := CreateKafkaConsumerContext(msg)
	if ctx == nil {
		t.Error("context should not be nil")
	}

	// 验证从消息中提取了 trace 信息
	span := zltrace.SpanFromContext(ctx)
	if span == nil {
		t.Error("span should not be nil")
	}
}

func TestInjectKafkaProducerHeaders(t *testing.T) {
	// 注册 mock tracer
	mockTracer := &mockTracer{}
	zltrace.RegisterTracer(mockTracer)
	defer zltrace.RegisterTracer(nil)

	ctx := context.Background()
	msg := &kafka.Message{
		Topic: "test-topic",
		Value: []byte("test"),
	}

	// 注入 trace_id
	ctx = InjectKafkaProducerHeaders(ctx, msg)

	// 验证消息 headers 中包含 traceparent
	var found bool
	for _, header := range msg.Headers {
		if header.Key == "traceparent" {
			found = true
			if len(header.Value) == 0 {
				t.Error("traceparent value should not be empty")
			}
			break
		}
	}

	if !found {
		t.Error("traceparent header should be injected")
	}
}

func TestKafkaProducerHeaderCarrier(t *testing.T) {
	headers := []kafka.Header{}
	carrier := &kafkaProducerHeaderCarrier{headers: &headers}

	// 测试 Set
	carrier.Set("traceparent", "00-test123-test456-01")
	if len(headers) != 1 {
		t.Error("should have 1 header")
	}

	// 测试 Get
	value, ok := carrier.Get("traceparent")
	if !ok {
		t.Error("should find traceparent header")
	}
	if value != "00-test123-test456-01" {
		t.Errorf("expected 00-test123-test456-01, got %s", value)
	}

	// 测试更新已存在的 header
	carrier.Set("traceparent", "00-newvalue-newspan-01")
	value, ok = carrier.Get("traceparent")
	if !ok {
		t.Error("should still find traceparent header")
	}
	if value != "00-newvalue-newspan-01" {
		t.Errorf("expected 00-newvalue-newspan-01, got %s", value)
	}
	if len(headers) != 1 {
		t.Error("should still have only 1 header")
	}
}

func TestKafkaConsumerHeaderCarrier(t *testing.T) {
	headers := []kafka.Header{
		{Key: "traceparent", Value: []byte("00-test123-test456-01")},
		{Key: "other", Value: []byte("value")},
	}
	carrier := &kafkaConsumerHeaderCarrier{headers: headers}

	// 测试 Get
	value, ok := carrier.Get("traceparent")
	if !ok {
		t.Error("should find traceparent header")
	}
	if value != "00-test123-test456-01" {
		t.Errorf("expected 00-test123-test456-01, got %s", value)
	}

	// 测试 Get 不存在的 key
	_, ok = carrier.Get("notexist")
	if ok {
		t.Error("should not find notexist header")
	}

	// 测试 Get 多个相同 key（第一个）
	headersWithDup := []kafka.Header{
		{Key: "traceparent", Value: []byte("00-first-first-01")},
		{Key: "traceparent", Value: []byte("00-second-second-01")},
	}
	carrier2 := &kafkaConsumerHeaderCarrier{headers: headersWithDup}
	value, ok = carrier2.Get("traceparent")
	if !ok {
		t.Error("should find traceparent header")
	}
	if value != "00-first-first-01" {
		t.Errorf("expected first value, got %s", value)
	}
}

// mockTracer 用于测试
type mockTracer struct{}

func (m *mockTracer) StartSpan(ctx context.Context, operationName string) (zltrace.Span, context.Context) {
	span := &mockSpan{}
	return span, zltrace.ContextWithSpan(ctx, span)
}

func (m *mockTracer) Inject(ctx context.Context, carrier zltrace.Carrier) error {
	carrier.Set("traceparent", "00-test123-test456-01")
	return nil
}

func (m *mockTracer) Extract(ctx context.Context, carrier zltrace.Carrier) (context.Context, error) {
	if _, ok := carrier.Get("traceparent"); ok {
		return ctx, nil
	}
	return nil, fmt.Errorf("no trace context")
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
