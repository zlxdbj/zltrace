package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/IBM/sarama"
	"github.com/zlxdbj/zllog"
	"github.com/zlxdbj/zltrace"
	"github.com/zlxdbj/zltrace/tracer/saramatracer"
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

	// 3. 创建 Kafka 消费者配置
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true

	// 4. 创建消费者
	consumer, err := sarama.NewConsumer([]string{"localhost:9092"}, config)
	if err != nil {
		zllog.Error(context.Background(), "kafka", "创建 Kafka 消费者失败", err)
		return
	}
	defer consumer.Close()

	// 5. 订阅主题
	partitionConsumer, err := consumer.ConsumePartition("test-topic", 0, sarama.OffsetNewest)
	if err != nil {
		zllog.Error(context.Background(), "kafka", "订阅主题失败", err)
		return
	}
	defer partitionConsumer.Close()

	// 6. 处理信号
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Consumer started, waiting for messages...")

	// 7. 消费消息
	for {
		select {
		case msg := <-partitionConsumer.Messages():
			// 处理消息（从消息中提取 trace_id）
			if err := handleMessage(msg); err != nil {
				zllog.Error(context.Background(), "consumer", "Failed to handle message", err)
			}

		case err := <-partitionConsumer.Errors():
			zllog.Error(context.Background(), "consumer", "Kafka consumer error", err)

		case <-sigterm:
			fmt.Println("\nReceived shutdown signal, exiting...")
			return
		}
	}
}

// handleMessage 处理 Kafka 消息（演示 trace_id 提取）
func handleMessage(msg *sarama.ConsumerMessage) error {
	// 从消息中提取 trace_id（关键步骤！）
	ctx := saramatracer.CreateKafkaConsumerContext(msg)

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
		zllog.Int64("partition", int64(msg.Partition)),
		zllog.Int64("offset", msg.Offset),
		zllog.String("value", string(msg.Value)),
		zllog.String("trace_id", getTraceID(ctx)))

	// 处理消息内容...
	return processMessage(ctx, msg)
}

// processMessage 处理消息内容
func processMessage(ctx context.Context, msg *sarama.ConsumerMessage) error {
	// 【可选】创建子 Span
	// 如果不需要追踪这个操作，可以不创建 Span，直接传递 ctx 即可
	span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "ProcessMessage")
	defer span.Finish()

	// 这里添加你的业务逻辑
	// 例如：解析 JSON、写入数据库等

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
