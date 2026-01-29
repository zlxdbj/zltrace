package kafkagotracer

import (
	"context"

	"github.com/segmentio/kafka-go"
	"github.com/zlxdbj/zltrace"
)

// CreateKafkaConsumerContext 从 Kafka 消息中创建 context
//
// 从消息 headers 中提取 trace 上下文，创建包含 trace_id 的 context。
// 如果消息 headers 中没有 trace 信息，则创建新的 trace_id（保险逻辑）。
//
// **支持的协议**：
//   - **W3C Trace Context** (OpenTelemetry): traceparent header（推荐）
//     - 格式：00-trace_id-span_id-flags
//     - 示例：00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
//
// **提取优先级**：
//   1. 优先提取 traceparent（W3C 标准）
//   2. 都没有则创建新的 trace_id
//
// 参数：
//   - message: Kafka 消费者消息
//
// 返回：
//   - context.Context: 包含 trace_id 的 context
func CreateKafkaConsumerContext(message *kafka.Message) context.Context {
	tracer := zltrace.GetTracer()
	if tracer == nil {
		// 没有注册 tracer，返回 background context
		return context.Background()
	}

	// 使用 tracer.Extract 从消息 headers 中提取 trace 上下文
	carrier := &kafkaConsumerHeaderCarrier{headers: message.Headers}
	ctx, err := tracer.Extract(context.Background(), carrier)

	if err != nil {
		// 提取失败（没有 trace 信息），创建新的 trace_id
		// ✅ 保险逻辑：确保总是有 trace_id
		span, newCtx := tracer.StartSpan(context.Background(), "Kafka/Consume")
		span.Finish()
		return newCtx
	}

	// 提取成功，基于提取的 context 创建 span
	span, spanCtx := tracer.StartSpan(ctx, "Kafka/Consume")
	span.Finish()

	return spanCtx
}

// kafkaConsumerHeaderCarrier 实现 Carrier 接口，用于 Kafka Consumer Headers
type kafkaConsumerHeaderCarrier struct {
	headers []kafka.Header
}

// Set 设置 header（消费者通常不需要设置）
func (c *kafkaConsumerHeaderCarrier) Set(key, value string) {
	// 消费者场景下，通常不需要设置 header
	// 这里是空实现
}

// Get 获取 header
func (c *kafkaConsumerHeaderCarrier) Get(key string) (string, bool) {
	for _, header := range c.headers {
		if header.Key == key {
			return string(header.Value), true
		}
	}
	return "", false
}

// ============================================================================
// Kafka Producer 相关（用于发送消息时注入 trace_id）
// ============================================================================

// InjectKafkaProducerHeaders 将 trace_id 注入到 Kafka 生产者消息的 headers 中
//
// **核心功能**：
//   - ✅ 自动创建 Exit Span（表示发送消息）
//   - ✅ 自动注入 trace_id 到消息 headers
//   - ✅ 如果 context 中没有 trace_id，自动生成新的
//
// **注入的消息头**（W3C Trace Context 标准）：
//   - traceparent: W3C 标准追踪头（格式：00-trace_id-span_id-flags）
//   - 示例：{Key: "traceparent", Value: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"}
//
// **执行流程**：
//   1. 检查是否注册了 tracer（优雅降级）
//   2. 创建 Exit Span（"Kafka/Produce/{topic}"）
//   3. 注入 trace_id 到消息 headers（使用 tracer.Inject）
//
// **使用示例**：
//
//	msg := kafka.Message{
//	    Topic: "my-topic",
//	    Value: []byte("hello"),
//	}
//	ctx := InjectKafkaProducerHeaders(ctx, &msg)
//	writer.WriteMessages(ctx, msg)
//
// 参数：
//   - ctx: 上下文（可能包含或不包含 trace_id）
//   - msg: Kafka 生产者消息指针（Headers 会被修改）
//
// 返回：
//   - context.Context: 包含 span 的 context（建议用于后续操作）
func InjectKafkaProducerHeaders(ctx context.Context, msg *kafka.Message) context.Context {
	tracer := zltrace.GetTracer()
	if tracer == nil {
		// 没有注册 tracer，返回原 context（优雅降级）
		return ctx
	}

	// 1. 创建 Exit Span（表示发送消息到 Kafka）
	// span 的 operationName 格式：Kafka/Produce/{topic}
	operationName := "Kafka/Produce/" + msg.Topic
	span, spanCtx := tracer.StartSpan(ctx, operationName)
	defer span.Finish()

	// 2. 注入 trace_id 到消息 headers
	carrier := &kafkaProducerHeaderCarrier{headers: &msg.Headers}
	if err := tracer.Inject(spanCtx, carrier); err != nil {
		// 注入失败不应该阻止发送消息，记录日志后继续
		// 这里不使用 zllog 避免循环依赖
	}

	// 3. 设置 span 标签
	span.SetTag("kafka.topic", msg.Topic)
	if len(msg.Key) > 0 {
		span.SetTag("kafka.key", string(msg.Key))
	}

	return spanCtx
}

// kafkaProducerHeaderCarrier 实现 Carrier 接口，用于 Kafka Producer Headers
type kafkaProducerHeaderCarrier struct {
	headers *[]kafka.Header
}

// Set 设置 header
func (c *kafkaProducerHeaderCarrier) Set(key, value string) {
	// 查找是否已存在该 key
	for i, header := range *c.headers {
		if header.Key == key {
			// 已存在，更新值
			(*c.headers)[i].Value = []byte(value)
			return
		}
	}
	// 不存在，追加新的 header
	*c.headers = append(*c.headers, kafka.Header{
		Key:   key,
		Value: []byte(value),
	})
}

// Get 获取 header
func (c *kafkaProducerHeaderCarrier) Get(key string) (string, bool) {
	for _, header := range *c.headers {
		if header.Key == key {
			return string(header.Value), true
		}
	}
	return "", false
}
