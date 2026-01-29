# 架构设计

本文档介绍 zltrace 的技术架构和设计理念。

## 分层架构

```
┌─────────────────────────────────────────────────────┐
│                    你的业务代码                       │
│  (HTTP Handler / Kafka Producer/Consumer)           │
└─────────────────────────────────────────────────────┘
                         │
                         ↓ 使用 zltrace API
┌─────────────────────────────────────────────────────┐
│                   zltrace API                        │
│  - InitTracer()                                      │
│  - GetSafeTracer().StartSpan()                       │
│  - httpadapter.NewTracedClient()                     │
│  - saramatracer.InjectKafkaProducerHeaders()         │
└─────────────────────────────────────────────────────┘
                         │
                         ↓ 底层实现（可替换）
┌─────────────────────────────────────────────────────┐
│              OpenTelemetry SDK                       │
│  - W3C Trace Context                                 │
│  - OTLP Exporter                                     │
│  - Span 管理                                         │
└─────────────────────────────────────────────────────┘
                         │
                         ↓ 数据发送
┌─────────────────────────────────────────────────────┐
│         追踪系统（SkyWalking / Jaeger）              │
└─────────────────────────────────────────────────────┘
```

## 核心设计理念

### 1. 依赖倒置

**业务代码依赖抽象接口，不依赖具体实现**：

```go
// 业务代码只依赖 Tracer 接口
type Tracer interface {
    StartSpan(ctx context.Context, operationName string) (Span, context.Context)
    Inject(ctx context.Context, carrier Carrier) error
    Extract(ctx context.Context, carrier Carrier) (context.Context, error)
    Close() error
}

// 具体实现可以替换
type OTELTracer struct { ... }
type MockTracer struct { ... }
```

**优势**：
- ✅ 底层实现可以随时替换
- ✅ 不锁定特定供应商
- ✅ 便于单元测试

### 2. 优雅降级

**追踪系统故障不影响业务**：

```go
// GetSafeTracer 永远返回可用的 Tracer
func GetSafeTracer() Tracer {
    tracer := GetTracer()
    if tracer == nil {
        return &noOpTracer{}  // 空操作 Tracer
    }
    return tracer
}

// noOpTracer 的所有操作都是空操作
type noOpTracer struct{}

func (t *noOpTracer) StartSpan(ctx context.Context, operationName string) (Span, context.Context) {
    return &noOpSpan{}, ctx
}
```

**生产环境验证**：即使追踪系统未初始化或故障，业务系统继续正常运行。

### 3. 配置驱动

**通过配置文件控制行为，无需修改代码**：

```yaml
# 开发环境：输出到日志
exporter:
  type: stdout

# 生产环境：发送到 SkyWalking
exporter:
  type: otlp
```

**优势**：
- ✅ 不同环境使用不同配置
- ✅ 无需修改代码即可切换
- ✅ 支持热更新（部分实现）

### 4. 标准兼容

**基于 W3C Trace Context 标准**：

```
traceparent: 00-trace_id-span_id-flags
```

**优势**：
- ✅ 跨语言兼容
- ✅ 跨系统互操作
- ✅ 符合行业规范

## 核心组件

### 1. Tracer 接口

追踪器接口，定义了核心操作：

```go
type Tracer interface {
    // 创建 Span
    StartSpan(ctx context.Context, operationName string) (Span, context.Context)

    // 注入 trace 上下文（用于客户端）
    Inject(ctx context.Context, carrier Carrier) error

    // 提取 trace 上下文（用于服务端）
    Extract(ctx context.Context, carrier Carrier) (context.Context, error)

    // 关闭追踪器
    Close() error
}
```

### 2. Span 接口

Span 接口，表示一个追踪片段：

```go
type Span interface {
    // 获取 context
    Context() context.Context

    // 设置标签
    SetTag(key string, value interface{})

    // 设置错误
    SetError(err error)

    // 结束 Span
    Finish()

    // 获取 trace_id
    TraceID() string
}
```

### 3. Carrier 接口

Trace 上下文载体，用于跨进程传递：

```go
type Carrier interface {
    Get(key string) (string, bool)
    Set(key, value string)
}
```

**实现**：
- HTTP Headers
- Kafka Headers
- 自定义协议

### 4. OpenTelemetry 集成

**OTELTracer** 实现了 Tracer 接口：

```go
type OTELTracer struct {
    tracer     trace.Tracer
    propagator propagation.TextMapPropagator
}

func (t *OTELTracer) StartSpan(ctx context.Context, operationName string) (Span, context.Context) {
    ctx, span := t.tracer.Start(ctx, operationName)
    return &OTELSpan{span: span}, ctx
}

func (t *OTELTracer) Inject(ctx context.Context, carrier Carrier) error {
    otelCarrier := &carrierAdapter{carrier: carrier}
    t.propagator.Inject(ctx, otelCarrier)
    return nil
}
```

## 数据流

### HTTP 调用链

```
服务A                                           服务B
  |                                                |
  | 1. 接收 HTTP 请求                               |
  |    Header: traceparent: 00-abc123-...          |
  |                                                |
  | 2. TraceHTTPRequest 提取 trace_id              |
  |    创建 Entry Span                             |
  |                                                |
  | 3. 调用下游服务                                |
  |    InjectHTTPHeaders 注入 traceparent          |
  |    ------------------------------------------> |
  |                                                | 4. 接收请求
  |                                                |    Header: traceparent: 00-abc123-...
  |                                                | 5. TraceHTTPRequest 提取 trace_id
  |                                                |    创建子 Span（同一个 trace_id）
  |                                                |
  | 6. 发送追踪数据到 SkyWalking                    | 7. 发送追踪数据到 SkyWalking
  |                                                |
  V                                                V
```

### Kafka 消息流

```
生产者服务                              Kafka                        消费者服务
  |                                       |                              |
  | 1. 创建 Span                           |                              |
  |    trace_id = abc123                   |                              |
  |                                       |                              |
  | 2. 发送消息                            |                              |
  |    InjectKafkaProducerHeaders          |                              |
  |    注入 traceparent header            |                              |
  |    ----------------------------------> |                              |
  |                                       | 消息 headers:                 |
  |                                       |   traceparent: 00-abc123-...  |
  |                                       |                              |
  |                                       | ----------------------------> |
  |                                       |                              | 3. 接收消息
  |                                       |                              |    CreateKafkaConsumerContext
  |                                       |                              |    提取 trace_id = abc123
  |                                       |                              |    创建子 Span
  |                                       |                              |
  V                                       V                              V
```

## 扩展点

### 1. 自定义 Tracer

实现 Tracer 接口：

```go
type MyTracer struct{}

func (t *MyTracer) StartSpan(ctx context.Context, operationName string) (Span, context.Context) {
    // 自定义实现
}

// 注册自定义 Tracer
zltrace.RegisterTracer(&MyTracer{})
```

### 2. 自定义 Carrier

实现 Carrier 接口：

```go
type MyCarrier struct{}

func (c *MyCarrier) Get(key string) (string, bool) {
    // 自定义实现
}

func (c *MyCarrier) Set(key, value string) {
    // 自定义实现
}
```

### 3. 框架适配器

实现 HTTPTraceHandler 接口：

```go
type MyFrameworkHandler struct{}

func (h *MyFrameworkHandler) GetMethod() string { ... }
func (h *MyFrameworkHandler) GetURL() string { ... }
func (h *MyFrameworkHandler) GetHeader(key string) string { ... }
func (h *MyFrameworkHandler) SetSpanContext(ctx context.Context) { ... }
func (h *MyFrameworkHandler) GetSpanContext() context.Context { ... }
```

## 性能考虑

### 内存管理

- **Span 大小**：约 1KB/span
- **队列大小**：可配置（默认 2048）
- **批量发送**：减少网络开销

### CPU 开销

- **ID 生成**：使用高效算法
- **序列化**：protobuf 格式
- **采样**：减少 span 数量

### 网络优化

- **批量发送**：合并多个 span
- **压缩**：OTLP 支持 gzip
- **连接复用**：gRPC 长连接

## 安全考虑

### TLS 支持

```yaml
trace:
  exporter:
    type: otlp
    otlp:
      endpoint: skywalking-oap:4317
      insecure: false  # 使用 TLS
```

### 数据脱敏

```go
// ❌ 不要记录敏感信息
span.SetTag("password", password)

// ✅ 只记录必要信息
span.SetTag("user.id", userID)
```

### 访问控制

- 追踪系统应配置访问控制
- 生产环境使用 TLS 加密
- 定期审计访问日志

## 监控指标

### 关键指标

- Span 生成速率
- 采样率
- Exporter 成功率
- 队列深度
- 错误率

### 告警规则

```yaml
# Exporter 失败率过高
- alert: HighExporterFailureRate
  expr: exporter_failure_rate > 0.05

# 队列积压
- alert: SpanQueueBacklog
  expr: span_queue_size > 10000
```

## 未来演进

### 短期计划

- [ ] 支持 RabbitMQ
- [ ] 支持 RocketMQ
- [ ] 增强采样策略

### 中期计划

- [ ] OpenTelemetry Auto-Instrumentation 集成
- [ ] 性能优化
- [ ] 更多框架适配器

### 长期计划

- [ ] 支持分布式事务追踪
- [ ] AI 辅助的异常检测
- [ ] 实时流式分析

## 相关文档

- [快速开始](./getting-started.md)
- [配置说明](./configuration.md)
- [API 参考](./api-reference.md)
