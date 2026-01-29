package main

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/zlxdbj/zllog"
	"github.com/zlxdbj/zltrace"
	"github.com/zlxdbj/zltrace/tracer/saramatrace"
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

	// 3. 创建 Kafka 生产者配置
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 3
	config.Producer.Return.Successes = true

	// 4. 创建生产者
	producer, err := sarama.NewSyncProducer([]string{"localhost:9092"}, config)
	if err != nil {
		panic("创建 Kafka 生产者失败: " + err.Error())
	}
	defer producer.Close()

	// 5. 发送消息
	ctx := context.Background()
	if err := sendMessage(ctx, producer); err != nil {
		zllog.Error(ctx, "example", "Failed to send message", err)
	}
}

// sendMessage 发送消息到 Kafka（演示 trace_id 注入）
func sendMessage(ctx context.Context, producer sarama.SyncProducer) error {
	// 创建子 span
	span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "SendMessage")
	defer span.Finish()

	// 准备消息
	msg := &sarama.ProducerMessage{
		Topic: "test-topic",
		Key:   sarama.StringEncoder("key1"),
		Value: sarama.StringEncoder(`{"message": "Hello, Kafka!"}`),
	}

	// 注入 trace_id 到消息 headers（关键步骤！）
	ctx = saramatrace.InjectKafkaProducerHeaders(ctx, msg)

	// 发送消息
	partition, offset, err := producer.SendMessage(msg)
	if err != nil {
		span.SetError(err)
		return fmt.Errorf("发送消息失败: %w", err)
	}

	// 记录成功
	zllog.Info(ctx, "example", "Message sent to Kafka",
		zllog.String("topic", msg.Topic),
		zllog.Int32("partition", partition),
		zllog.Int64("offset", offset))

	return nil
}
