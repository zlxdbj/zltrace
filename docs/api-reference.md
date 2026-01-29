# API 参考

## 核心接口

### Tracer

追踪器接口，用于创建和管理 Span。

```go
type Tracer interface {
    // StartSpan 启动一个新的 span
    StartSpan(ctx context.Context, operationName string) (Span, context.Context)

    // Inject 将 trace 上下文注入到 carrier
    Inject(ctx context.Context, carrier Carrier) error

    // Extract 从 carrier 中提取 trace 上下文
    Extract(ctx context.Context, carrier Carrier) (context.Context, error)

    // Close 关闭追踪器
    Close() error
}
```

### Span

Span 接口，表示一个追踪片段。

```go
type Span interface {
    // Context 返回 span 的 context
    Context() context.Context

    // SetTag 设置标签
    SetTag(key string, value interface{})

    // SetError 设置错误信息
    SetError(err error)

    // Finish 结束 span
    Finish()

    // TraceID 返回 trace_id
    TraceID() string
}
```

### Carrier

Trace 上下文载体接口。

```go
type Carrier interface {
    // Get 根据 key 获取值
    Get(key string) (string, bool)

    // Set 设置 key-value 对
    Set(key, value string)
}
```

## 初始化函数

### InitTracer()

初始化追踪系统（推荐使用）。

```go
func InitTracer() error
```

**示例**：
```go
if err := zltrace.InitTracer(); err != nil {
    panic(err)
}
```

### Init()

`InitTracer()` 的别名，功能相同。

```go
func Init() error
```

### InitOpenTelemetryTracer()

初始化 OpenTelemetry Tracer。

```go
func InitOpenTelemetryTracer() error
```

**说明**：通常不需要直接调用，`InitTracer()` 会自动调用此方法。

## Tracer 管理

### RegisterTracer()

注册全局追踪器。

```go
func RegisterTracer(tracer Tracer)
```

**示例**：
```go
mockTracer := &mockTracer{}
zltrace.RegisterTracer(mockTracer)
```

### GetTracer()

获取全局追踪器。

```go
func GetTracer() Tracer
```

**返回**：可能返回 `nil`

### GetSafeTracer()

获取安全的追踪器（永不返回 `nil`）。

```go
func GetSafeTracer() Tracer
```

**返回**：如果全局 tracer 为 `nil`，返回 `noOpTracer`

**建议**：优先使用此方法，避免 nil pointer 错误

## Context 辅助函数

### SpanFromContext()

从 context 中获取 span。

```go
func SpanFromContext(ctx context.Context) Span
```

**示例**：
```go
span := zltrace.SpanFromContext(ctx)
if span != nil {
    traceID := span.TraceID()
}
```

### ContextWithSpan()

将 span 添加到 context。

```go
func ContextWithSpan(ctx context.Context, span Span) context.Context
```

## HTTP 追踪

### TraceHTTPRequest()

通用 HTTP 请求追踪函数。

```go
func TraceHTTPRequest(ctx context.Context, handler HTTPTraceHandler, next func())
```

**参数**：
- `ctx` - context
- `handler` - HTTP 处理器（实现 `HTTPTraceHandler` 接口）
- `next` - 下一个处理函数

**示例**：
```go
func MyMiddleware(c *gin.Context) {
    handler := &MyHandler{c}
    zltrace.TraceHTTPRequest(c.Request.Context(), handler, c.Next)
}
```

### HTTPTraceHandler

HTTP 追踪处理器接口。

```go
type HTTPTraceHandler interface {
    GetMethod() string
    GetURL() string
    GetHeader(key string) string
    SetSpanContext(ctx context.Context)
    GetSpanContext() context.Context
}
```

## HTTPAdapter

### NewTracedClient()

创建自动追踪的 HTTP Client。

```go
func NewTracedClient(client *http.Client) *http.Client
```

**参数**：
- `client` - 可选的基础客户端（为 `nil` 时创建新的）

**返回**：配置好追踪的 HTTP 客户端

**示例**：
```go
client := httpadapter.NewTracedClient(nil)
resp, err := client.Do(req)
```

### TracingRoundTripper

自动注入 trace_id 的 HTTP Transport。

```go
type TracingRoundTripper struct {
    Base http.RoundTripper
}
```

**示例**：
```go
client := &http.Client{
    Transport: &httpadapter.TracingRoundTripper{
        Base: http.DefaultTransport,
    },
}
```

## Kafka 追踪

### IBM Sarama

#### InjectKafkaProducerHeaders()

注入 trace_id 到 Kafka 消息。

```go
func InjectKafkaProducerHeaders(ctx context.Context, msg *sarama.ProducerMessage) context.Context
```

**示例**：
```go
msg := &sarama.ProducerMessage{Topic: "test", Value: sarama.StringEncoder("hello")}
ctx = saramatracer.InjectKafkaProducerHeaders(ctx, msg)
producer.SendMessage(msg)
```

#### CreateKafkaConsumerContext()

从 Kafka 消息提取 trace_id。

```go
func CreateKafkaConsumerContext(message *sarama.ConsumerMessage) context.Context
```

**示例**：
```go
ctx := saramatracer.CreateKafkaConsumerContext(msg)
processMessage(ctx, msg)
```

### segmentio/kafka-go

#### InjectKafkaProducerHeaders()

注入 trace_id 到 Kafka 消息。

```go
func InjectKafkaProducerHeaders(ctx context.Context, msg *kafka.Message) context.Context
```

**示例**：
```go
msg := kafka.Message{Topic: "test", Value: []byte("hello")}
ctx = kafkagotracer.InjectKafkaProducerHeaders(ctx, &msg)
writer.WriteMessages(ctx, msg)
```

#### CreateKafkaConsumerContext()

从 Kafka 消息提取 trace_id。

```go
func CreateKafkaConsumerContext(msg *kafka.Message) context.Context
```

**示例**：
```go
ctx := kafkagotracer.CreateKafkaConsumerContext(&msg)
processMessage(ctx, &msg)
```

## 配置管理

### LoadConfig()

从配置文件加载配置。

```go
func LoadConfig() (*TraceConfig, error)
```

**返回**：
- `*TraceConfig` - 配置对象
- `error` - 错误信息

### TraceConfig

追踪配置结构。

```go
type TraceConfig struct {
    Enabled     bool
    ServiceName string
    Sampler     SamplerConfig
    Exporter    ExporterConfig
    Batch       BatchConfig
}
```

### GetExampleConfig()

获取配置示例。

```go
func GetExampleConfig() string
```

**返回**：YAML 格式的配置示例

## 相关文档

- [快速开始](./getting-started.md)
- [配置说明](./configuration.md)
- [HTTP 追踪](./http-tracing.md)
- [Kafka 追踪](./kafka-tracing.md)
