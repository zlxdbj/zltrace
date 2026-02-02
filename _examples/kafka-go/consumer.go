package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/zlxdbj/zllog"
	"github.com/zlxdbj/zltrace"
	"github.com/zlxdbj/zltrace/tracer/kafkagotracer"
)

func main() {
	// 1. 初始化日志系统
	if err := zllog.InitLogger(); err != nil {
		zllog.Error(context.Background(), "init", "日志系统初始化失败", err)
	}

	// 2. 初始化追踪系统
	if err := zltrace.InitTracer(); err != nil {
		zllog.Error(context.Background(), "init", "追踪系统初始化失败", err)
		// 追踪系统初始化失败不影响业务运行，程序可以继续
	}
	defer func() {
		if tracer := zltrace.GetTracer(); tracer != nil {
			tracer.Close()
		}
	}()

	// 3. 创建 Kafka Reader
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{"localhost:9092"},
		GroupID:  "consumer-group-1",
		Topic:    "test-topic",
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	defer r.Close()

	// 4. 处理信号
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Consumer started, waiting for messages...")

	// 5. 消费消息
	for {
		select {
		case <-sigterm:
			fmt.Println("\nReceived shutdown signal, exiting...")
			return

		default:
			// 读取消息（带超时）
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			msg, err := r.ReadMessage(ctx)
			cancel()

			if err != nil {
				// 超时或其他错误，继续循环
				continue
			}

			// 处理消息（从消息中提取 trace_id）
			if err := handleMessage(&msg); err != nil {
				zllog.Error(context.Background(), "consumer", "Failed to handle message", err)
			}
		}
	}
}

// handleMessage 处理 Kafka 消息（演示 trace_id 提取）
func handleMessage(msg *kafka.Message) error {
	// 从消息中提取 trace_id（关键步骤！）
	ctx := kafkagotracer.CreateKafkaConsumerContext(msg)

	// 【可选】创建 Span 用于追踪消息处理操作
	// 如果不需要追踪这个操作本身，可以不创建 Span
	// trace_id 已经在 ctx 中，会自动传递给下游调用
	// 详见：https://github.com/zlxdbj/zltrace/blob/main/docs/faq.md#q1-是否必须创建-span
	span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "HandleMessage")
	defer span.Finish()

	// 设置 span 标签
	span.SetTag("kafka.topic", msg.Topic)
	span.SetTag("kafka.partition", msg.Partition)
	span.SetTag("kafka.offset", msg.Offset)

	// 业务逻辑处理
	zllog.Info(ctx, "consumer", "Received message from Kafka",
		zllog.String("topic", msg.Topic),
		zllog.Int("partition", msg.Partition),
		zllog.Int64("offset", msg.Offset),
		zllog.Int("bytes", len(msg.Value)),
		zllog.String("trace_id", getTraceID(ctx)))

	// 处理消息内容...
	return processMessage(ctx, msg)
}

// processMessage 处理消息内容
func processMessage(ctx context.Context, msg *kafka.Message) error {
	// 【可选】创建子 Span
	// 如果不需要追踪这个操作，可以不创建 Span，直接传递 ctx 即可
	span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "ProcessMessage")
	defer span.Finish()

	// 这里添加你的业务逻辑
	// 例如：解析 JSON、写入数据库等

	zllog.Info(ctx, "consumer", "Message processed successfully",
		zllog.String("value", string(msg.Value)))

	return nil
}

// getTraceID 从 context 获取 trace_id（用于演示）
func getTraceID(ctx context.Context) string {
	span := zltrace.SpanFromContext(ctx)
	if span != nil {
		return span.TraceID()
	}
	return ""
}
