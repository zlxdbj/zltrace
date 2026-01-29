package main

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/zlxdbj/zllog"
	"github.com/zlxdbj/zltrace"
	"github.com/zlxdbj/zltrace/tracer/kafkagotracer"
)

func main() {
	// 1. 初始化日志系统
	if err := zllog.InitLogger(); err != nil {
		panic("初始化日志系统失败: " + err.Error())
	}

	// 2. 初始化追踪系统
	if err := zltrace.InitTracer(); err != nil {
		panic("初始化追踪系统失败: " + err.Error())
	}
	defer func() {
		if tracer := zltrace.GetTracer(); tracer != nil {
			tracer.Close()
		}
	}()

	// 3. 创建 Kafka Writer
	writer := &kafka.Writer{
		Addr:     kafka.TCP("localhost:9092"),
		Topic:    "test-topic",
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	// 4. 发送消息
	ctx := context.Background()
	if err := sendMessage(ctx, writer); err != nil {
		zllog.Error(ctx, "example", "Failed to send message", err)
	}
}

// sendMessage 发送消息到 Kafka（演示 trace_id 注入）
func sendMessage(ctx context.Context, writer *kafka.Writer) error {
	// 创建子 span
	span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "SendMessage")
	defer span.Finish()

	// 准备消息
	msg := kafka.Message{
		Key:   []byte("key1"),
		Value: []byte(`{"message": "Hello, Kafka-Go!"}`),
	}

	// 注入 trace_id 到消息 headers（关键步骤！）
	ctx = kafkagotracer.InjectKafkaProducerHeaders(ctx, &msg)

	// 发送消息
	if err := writer.WriteMessages(ctx, msg); err != nil {
		span.SetError(err)
		return fmt.Errorf("发送消息失败: %w", err)
	}

	// 记录成功
	zllog.Info(ctx, "example", "Message sent to Kafka",
		zllog.String("topic", msg.Topic),
		zllog.Int("bytes", len(msg.Value)))

	return nil
}
