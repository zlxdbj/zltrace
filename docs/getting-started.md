# 快速开始

本指南将帮助你在 5 分钟内上手 zltrace。

## 前置要求

- Go 1.19 或更高版本
- Kafka 环境（可选，仅在使用 Kafka 追踪时需要）
- 追踪系统（可选，如 SkyWalking、Jaeger）

## 安装

```bash
go get github.com/zlxdbj/zltrace@latest
```

## 基本使用

### 1. 创建配置文件

创建 `zltrace.yaml` 配置文件：

```yaml
trace:
  enabled: true
  service_name: my_service

  exporter:
    type: stdout  # 开发环境使用 stdout

    otlp:
      endpoint: localhost:4317
      timeout: 10

  sampler:
    type: always_on
    ratio: 1.0
```

### 2. 初始化追踪系统

```go
package main

import (
    "github.com/zlxdbj/zllog"
    "github.com/zlxdbj/zltrace"
)

func main() {
    // 1. 初始化日志系统
    if err := zllog.InitLogger(); err != nil {
        panic(err)
    }

    // 2. 初始化追踪系统
    if err := zltrace.InitTracer(); err != nil {
        panic(err)
    }
    defer zltrace.GetTracer().Close()

    // 3. 你的业务代码...
}
```

### 3. HTTP 服务集成

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/zlxdbj/zltrace/tracer/httptracer"
)

func main() {
    r := gin.Default()

    // 添加追踪中间件
    r.Use(httptracer.TraceMiddleware())

    r.GET("/api/users", func(c *gin.Context) {
        // trace_id 自动注入到日志
        zllog.Info(c.Request.Context(), "api", "获取用户列表")
        c.JSON(200, gin.H{"users": []string{}})
    })

    r.Run(":8080")
}
```

### 4. Kafka 生产者

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

    // 注入 trace_id
    ctx = saramatracer.InjectKafkaProducerHeaders(ctx, msg)
    return producer.SendMessage(msg)
}
```

### 5. Kafka 消费者

```go
import (
    "github.com/IBM/sarama"
    "github.com/zlxdbj/zltrace/tracer/saramatracer"
)

func consumeMessage(msg *sarama.ConsumerMessage) error {
    // 提取 trace_id
    ctx := saramatracer.CreateKafkaConsumerContext(msg)

    // 后续操作都继承这个 trace_id
    return processMessage(ctx, msg)
}
```

## 配置文件位置

zltrace 会按以下顺序查找配置文件：

1. `./zltrace.yaml` (当前目录)
2. `$ZLTRACE_CONFIG` 环境变量指定的路径
3. `/etc/zltrace/config.yaml` (系统配置目录)
4. 使用默认配置

## 环境变量

- `SERVICE_NAME` - 服务名称（覆盖配置文件）
- `APP_NAME` - 应用名称（备用）
- `ZLTRACE_CONFIG` - 配置文件路径

## 下一步

- [配置说明](./configuration.md) - 详细的配置选项
- [HTTP 追踪](./http-tracing.md) - HTTP 追踪详细说明
- [Kafka 追踪](./kafka-tracing.md) - Kafka 追踪详细说明
- [示例代码](../_examples/) - 完整的示例代码
