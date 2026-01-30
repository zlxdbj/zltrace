# Kafka 追踪

zltrace 支持两个主流的 Kafka Go 客户端：
- **IBM Sarama** - 功能完整，成熟稳定
- **segmentio/kafka-go** - API 简洁，易用性好

## IBM Sarama

### 安装依赖

```bash
go get github.com/IBM/sarama
```

### 生产者

发送消息时自动注入 trace_id：

```go
import (
    "github.com/IBM/sarama"
    "github.com/zlxdbj/zltrace/tracer/saramatracer"
)

func sendMessage(ctx context.Context) error {
    msg := &sarama.ProducerMessage{
        Topic: "my-topic",
        Value: sarama.StringEncoder("hello"),
    }

    // 注入 trace_id 到消息 headers
    ctx = saramatracer.InjectKafkaProducerHeaders(ctx, msg)

    return producer.SendMessage(msg)
}
```

### 消费者

接收消息时自动提取 trace_id：

```go
import (
    "github.com/IBM/sarama"
    "github.com/zlxdbj/zltrace/tracer/saramatracer"
)

func consumeMessage(msg *sarama.ConsumerMessage) error {
    // 从消息中提取 trace_id
    ctx := saramatracer.CreateKafkaConsumerContext(msg)

    // 【可选】创建 Span 用于追踪消息处理操作
    // 如果不需要追踪这个操作本身，可以不创建 Span
    // span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "ConsumeMessage")
    // defer span.Finish()

    // 后续所有操作都继承这个 trace_id
    return processMessage(ctx, msg)
}
```

### 完整示例

生产者示例：

```go
package main

import (
    "context"
    "log"

    "github.com/IBM/sarama"
    "github.com/zlxdbj/zllog"
    "github.com/zlxdbj/zltrace"
    "github.com/zlxdbj/zltrace/tracer/saramatracer"
)

func main() {
    zllog.InitLogger()
    zltrace.InitTracer()
    defer zltrace.GetTracer().Close()

    // 创建生产者
    config := sarama.NewConfig()
    config.Producer.RequiredAcks = sarama.WaitForAll
    config.Producer.Return.Successes = true

    producer, err := sarama.NewSyncProducer([]string{"localhost:9092"}, config)
    if err != nil {
        log.Fatal(err)
    }
    defer producer.Close()

    // 发送消息
    ctx := context.Background()
    msg := &sarama.ProducerMessage{
        Topic: "test-topic",
        Value: sarama.StringEncoder("Hello, Kafka!"),
    }

    // 注入 trace_id
    ctx = saramatracer.InjectKafkaProducerHeaders(ctx, msg)

    partition, offset, err := producer.SendMessage(msg)
    if err != nil {
        zllog.Error(ctx, "producer", "发送失败", err)
        return
    }

    zllog.Info(ctx, "producer", "发送成功",
        zllog.Int32("partition", partition),
        zllog.Int64("offset", offset))
}
```

消费者示例：

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/IBM/sarama"
    "github.com/zlxdbj/zllog"
    "github.com/zlxdbj/zltrace"
    "github.com/zlxdbj/zltrace/tracer/saramatracer"
)

func main() {
    zllog.InitLogger()
    zltrace.InitTracer()
    defer zltrace.GetTracer().Close()

    // 创建消费者
    config := sarama.NewConfig()
    config.Consumer.Return.Errors = true

    consumer, err := sarama.NewConsumer([]string{"localhost:9092"}, config)
    if err != nil {
        log.Fatal(err)
    }
    defer consumer.Close()

    // 订阅主题
    partitionConsumer, err := consumer.ConsumePartition("test-topic", 0, sarama.OffsetNewest)
    if err != nil {
        log.Fatal(err)
    }
    defer partitionConsumer.Close()

    // 处理信号
    sigterm := make(chan os.Signal, 1)
    signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

    for {
        select {
        case msg := <-partitionConsumer.Messages():
            // 提取 trace_id（必须）
            ctx := saramatracer.CreateKafkaConsumerContext(msg)

            // 【可选】创建 Span - 如果不需要追踪这个操作，可以不创建
            // span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "ConsumeMessage")
            // defer span.Finish()

            // 处理消息
            zllog.Info(ctx, "consumer", "收到消息",
                zllog.Int("partition", msg.Partition),
                zllog.Int64("offset", msg.Offset))

        case err := <-partitionConsumer.Errors():
            zllog.Error(context.Background(), "consumer", "Kafka 错误", err)

        case <-sigterm:
            return
        }
    }
}
```

## segmentio/kafka-go

### 安装依赖

```bash
go get github.com/segmentio/kafka-go
```

### 生产者

```go
import (
    "github.com/segmentio/kafka-go"
    "github.com/zlxdbj/zltrace/tracer/kafkagotracer"
)

func sendMessage(ctx context.Context, writer *kafka.Writer) error {
    msg := kafka.Message{
        Topic: "my-topic",
        Value: []byte("hello"),
    }

    // 注入 trace_id
    ctx = kafkagotracer.InjectKafkaProducerHeaders(ctx, &msg)

    return writer.WriteMessages(ctx, msg)
}
```

### 消费者

```go
import (
    "github.com/segmentio/kafka-go"
    "github.com/zlxdbj/zltrace/tracer/kafkagotracer"
)

func consumeMessage(msg *kafka.Message) error {
    // 提取 trace_id
    ctx := kafkagotracer.CreateKafkaConsumerContext(msg)

    // 【可选】创建 Span 用于追踪消息处理操作
    // 如果不需要追踪这个操作本身，可以不创建 Span
    // span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "ConsumeMessage")
    // defer span.Finish()

    // 处理消息
    return processMessage(ctx, msg)
}
```

### 完整示例

生产者：

```go
package main

import (
    "context"

    "github.com/segmentio/kafka-go"
    "github.com/zlxdbj/zllog"
    "github.com/zlxdbj/zltrace"
    "github.com/zlxdbj/zltrace/tracer/kafkagotracer"
)

func main() {
    zllog.InitLogger()
    zltrace.InitTracer()
    defer zltrace.GetTracer().Close()

    // 创建 Writer
    writer := &kafka.Writer{
        Addr:     kafka.TCP("localhost:9092"),
        Topic:    "test-topic",
        Balancer: &kafka.LeastBytes{},
    }
    defer writer.Close()

    // 发送消息
    ctx := context.Background()
    msg := kafka.Message{
        Key:   []byte("key"),
        Value: []byte("Hello, Kafka-Go!"),
    }

    // 注入 trace_id
    ctx = kafkagotracer.InjectKafkaProducerHeaders(ctx, &msg)

    if err := writer.WriteMessages(ctx, msg); err != nil {
        zllog.Error(ctx, "producer", "发送失败", err)
        return
    }

    zllog.Info(ctx, "producer", "发送成功")
}
```

消费者：

```go
package main

import (
    "context"
    "log"
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
    zllog.InitLogger()
    zltrace.InitTracer()
    defer zltrace.GetTracer().Close()

    // 创建 Reader
    r := kafka.NewReader(kafka.ReaderConfig{
        Brokers:  []string{"localhost:9092"},
        GroupID:  "consumer-group-1",
        Topic:    "test-topic",
        MinBytes: 10e3,
        MaxBytes: 10e6,
    })
    defer r.Close()

    // 处理信号
    sigterm := make(chan os.Signal, 1)
    signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

    for {
        select {
        case <-sigterm:
            return

        default:
            // 读取消息
            ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
            msg, err := r.ReadMessage(ctx)
            cancel()

            if err != nil {
                continue
            }

            // 提取 trace_id（必须）
            ctx = kafkagotracer.CreateKafkaConsumerContext(&msg)

            // 【可选】创建 Span - 如果不需要追踪这个操作，可以不创建
            // span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "ConsumeMessage")
            // defer span.Finish()

            // 处理消息
            zllog.Info(ctx, "consumer", "收到消息",
                zllog.Int("partition", msg.Partition),
                zllog.Int64("offset", msg.Offset))
        }
    }
}
```

## Trace Context 格式

zltrace 使用 W3C Trace Context 标准（`traceparent` header）：

### Header 格式

```
traceparent: 00-trace_id-span_id-flags
```

**示例**：
```
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
```

### 跨服务追踪

当消息在不同服务间传递时，trace_id 会自动传递：

```
服务A ─[Kafka]─> 服务B ─[Kafka]─> 服务C
  trace_id: abc123      abc123      abc123
```

## API 对比

两个客户端的 API 完全一致：

| 功能 | IBM Sarama | segmentio/kafka-go |
|------|-----------|-------------------|
| 包路径 | `github.com/IBM/sarama` | `github.com/segmentio/kafka-go` |
| tracer 包 | `tracer/saramatracer` | `tracer/kafkagotracer` |
| 注入方法 | `InjectKafkaProducerHeaders()` | `InjectKafkaProducerHeaders()` |
| 提取方法 | `CreateKafkaConsumerContext()` | `CreateKafkaConsumerContext()` |

## 相关文档

- [快速开始](./getting-started.md)
- [配置说明](./configuration.md)
- [示例代码](../_examples/kafka/) (Sarama)
- [示例代码](../_examples/kafka-go/) (kafka-go)
