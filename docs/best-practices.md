# 最佳实践

本文档总结了使用 zltrace 的最佳实践和推荐模式。

## 1. Context 传递规范

### ✅ 推荐：所有函数都接收 context

```go
// HTTP Handler
func Handler(c *gin.Context) {
    ctx := c.Request.Context()
    ProcessOrder(ctx, orderID)
}

// 业务函数
func ProcessOrder(ctx context.Context, orderID string) error {
    span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "ProcessOrder")
    defer span.Finish()
    // ...
}
```

### ❌ 避免：不接收 context

```go
func ProcessOrder(orderID string) error {
    // trace_id 链中断！
}
```

## 2. 何时创建 Span

创建 Span 是**可选的**，取决于你的具体需求。

### 需要创建 Span 的场景

- ✅ 追踪**重要业务操作**的耗时（如订单处理、支付流程）
- ✅ 需要在追踪系统中看到调用层次结构
- ✅ 需要记录业务标签（订单ID、用户ID等）
- ✅ 需要记录错误信息和异常情况
- ✅ 性能分析和优化

```go
// 示例：追踪重要的业务操作
func ProcessOrder(ctx context.Context, orderID string) error {
    // 创建 Span 记录这个操作
    span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "ProcessOrder")
    defer span.Finish()

    // 记录业务标签
    span.SetTag("order_id", orderID)
    span.SetTag("user_id", getUserID(ctx))

    // 业务逻辑
    if err := validateOrder(ctx, orderID); err != nil {
        span.SetError(err)
        return err
    }

    return nil
}
```

### 不需要创建 Span 的场景

- ❌ 只是简单传递 trace_id 给下游
- ❌ 操作太简单（如数据转换、字段映射）
- ❌ 只想保证日志中有 trace_id（已自动实现）
- ❌ 透传操作（中间层、代理层）

```go
// 示例：简单透传，不需要创建 Span
func handleMessage(ctx context.Context, msg *Message) error {
    // 不创建 Span，直接传递 ctx 给下游
    // 下游会创建自己的 Span（如 HTTP 调用、数据库查询）
    return callDownstreamService(ctx, msg)
}
```

### 实际应用示例

**场景1：Kafka 消费者 - 需要创建 Span**
```go
func consumeMessage(msg *sarama.ConsumerMessage) error {
    // 提取 trace_id
    ctx := saramatracer.CreateKafkaConsumerContext(msg)

    // 创建 Span - 因为消费消息是重要操作，需要追踪
    span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "ConsumeMessage")
    defer span.Finish()

    span.SetTag("kafka.topic", msg.Topic)
    span.SetTag("kafka.partition", msg.Partition)

    return processMessage(ctx, msg)
}
```

**场景2：HTTP 透传 - 不需要创建 Span**
```go
func proxyRequest(ctx context.Context, req *http.Request) error {
    // 不创建 Span - 只是透传请求
    // HTTP Client 会自动创建 Span
    client := httpadapter.NewTracedClient(nil)
    resp, err := client.Do(req.WithContext(ctx))
    return err
}
```

**场景3：复杂业务 - 关键操作创建 Span**
```go
func ProcessOrder(ctx context.Context, orderID string) error {
    // 创建 Span - 这是关键业务操作
    span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "ProcessOrder")
    defer span.Finish()

    // 简单操作，不创建 Span
    order := parseOrder(orderID)

    // 关键操作，创建 Span
    if err := validateOrder(ctx, order); err != nil {
        span.SetError(err)
        return err
    }

    // 关键操作，创建 Span
    if err := saveToDatabase(ctx, order); err != nil {
        span.SetError(err)
        return err
    }

    return nil
}
```

### 判断标准

| 问题 | 是 → 创建 Span | 否 → 不创建 |
|------|--------------|-----------|
| 是否是关键业务操作？ | ✅ | ❌ |
| 需要了解这个操作的耗时吗？ | ✅ | ❌ |
| 需要记录业务标签吗？ | ✅ | ❌ |
| 需要在追踪系统看到这个节点吗？ | ✅ | ❌ |
| 只是传递 trace_id？ | ❌ | ✅ |

### 记住

**Context 本身就携带 trace_id，创建 Span 是为了"记录"操作。** 如果只是传递 trace_id，不需要创建 Span。

```go
// HTTP Handler
func Handler(c *gin.Context) {
    ctx := c.Request.Context()
    ProcessOrder(ctx, orderID)
}

// 业务函数
func ProcessOrder(ctx context.Context, orderID string) error {
    span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "ProcessOrder")
    defer span.Finish()
    // ...
}
```

### ❌ 避免：不接收 context

```go
func ProcessOrder(orderID string) error {
    // trace_id 链中断！
}
```

## 3. Span 命名规范

### ✅ 推荐：清晰的命名

```go
zltrace.GetSafeTracer().StartSpan(ctx, "ProcessOrder")
zltrace.GetSafeTracer().StartSpan(ctx, "QueryDatabase")
zltrace.GetSafeTracer().StartSpan(ctx, "Kafka/Produce/alarm-topic")
```

### ❌ 避免：模糊的命名

```go
zltrace.GetSafeTracer().StartSpan(ctx, "doSomething")
zltrace.GetSafeTracer().StartSpan(ctx, "handle")
```

### 命名建议

- 使用动词+名词：`ProcessOrder`、`QueryUser`
- HTTP 请求：`HTTP GET /api/users`
- Kafka 操作：`Kafka/Produce/{topic}`、`Kafka/Consume/{topic}`
- 数据库操作：`DB/Query/{table}`、`DB/Insert/{table}`

## 4. 标签使用规范

### ✅ 推荐：结构化标签

```go
span.SetTag("order_id", orderID)
span.SetTag("user_id", userID)
span.SetTag("status", "success")
span.SetTag("error.code", "DB_ERROR")
```

### ❌ 避免：字符串拼接

```go
span.SetTag("info", fmt.Sprintf("order=%s user=%s", orderID, userID))
```

### 常用标签

```go
// 业务标签
span.SetTag("order_id", orderID)
span.SetTag("user_id", userID)
span.SetTag("product_id", productID)

// 技术标签
span.SetTag("db.query", query)
span.SetTag("cache.hit", true)
span.SetTag("http.status_code", 200)
```

## 5. 错误处理规范

### ✅ 推荐：记录错误到 span

```go
if err := doSomething(); err != nil {
    span.SetError(err)
    span.SetTag("error.code", "DB_ERROR")
    span.SetTag("error.type", "connection")
    return err
}
```

### ❌ 避免：仅返回错误

```go
if err := doSomething(); err != nil {
    return err  // trace 信息丢失
}
```

## 6. 采样策略

### 开发环境

```yaml
trace:
  sampler:
    type: always_on  # 全量采样
```

### 生产环境

```yaml
trace:
  sampler:
    type: traceid_ratio
    ratio: 0.1  # 采样 10%，降低开销
```

### 高流量场景

```yaml
trace:
  sampler:
    type: parent_based  # 基于父 span 决定
```

## 7. Exporter 选择

### 开发环境

```yaml
trace:
  exporter:
    type: stdout  # 输出到日志，便于调试
```

### 生产环境

```yaml
trace:
  exporter:
    type: otlp  # 发送到 SkyWalking
    otlp:
      endpoint: skywalking-oap:4317
```

### 测试环境

```yaml
trace:
  exporter:
    type: none  # 不发送数据，仅生成 trace_id
```

## 8. 性能优化

### 使用采样器

```yaml
trace:
  sampler:
    type: traceid_ratio
    ratio: 0.1  # 只采样 10%
```

### 调整批量参数

```yaml
trace:
  batch:
    batch_size: 1024     # 增大批量大小
    timeout: 10          # 增加超时时间
    max_queue_size: 4096 # 增加队列大小
```

### 高并发场景

```yaml
trace:
  sampler:
    type: traceid_ratio
    ratio: 0.01  # 仅采样 1%

  batch:
    batch_size: 2048
    timeout: 5
    max_queue_size: 8192
```

## 9. 安全建议

### 生产环境配置

```yaml
trace:
  exporter:
    type: otlp
    otlp:
      endpoint: skywalking-oap:4317
      insecure: false  # 使用 TLS
```

### 不要记录敏感信息

```go
// ❌ 错误
span.SetTag("password", password)
span.SetTag("token", token)

// ✅ 正确
span.SetTag("user.id", userID)
span.SetTag("auth.method", "jwt")
```

## 10. 日志集成

### ✅ 推荐：使用 zllog

```go
zllog.Info(ctx, "module", "message",
    zllog.String("key", "value"))

// trace_id 自动注入到日志
// {"trace_id": "abc123...", "module": "module", "key": "value"}
```

### ❌ 避免：手动传递 trace_id

```go
span := zltrace.SpanFromContext(ctx)
traceID := span.TraceID()
log.WithField("trace_id", traceID).Info("message")
```

## 11. 优雅降级

### ✅ 推荐：使用 GetSafeTracer

```go
span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "operation")
defer span.Finish()
// 即使追踪系统未初始化，也不会 panic
```

### ❌ 避免：直接使用 GetTracer

```go
tracer := zltrace.GetTracer()
if tracer != nil {  // 需要手动检查 nil
    tracer.StartSpan(ctx, "operation")
}
```

## 12. Kafka 消息处理

### ✅ 推荐：立即提取 trace_id

```go
func ConsumeMessage(msg *sarama.ConsumerMessage) error {
    // 立即提取 trace_id
    ctx := saramatracer.CreateKafkaConsumerContext(msg)

    // 后续所有操作都使用这个 context
    return processMessage(ctx, msg)
}
```

### ❌ 避免：延迟提取 trace_id

```go
func ConsumeMessage(msg *sarama.ConsumerMessage) error {
    // 先处理消息，后面再提取 trace_id
    data := parseMessage(msg)

    ctx := context.Background()  // trace 链中断！
    return processMessage(ctx, data)
}
```

## 13. HTTP Client 调用

### ✅ 推荐：使用 TracedClient

```go
client := httpadapter.NewTracedClient(nil)
resp, err := client.Do(req)
// trace_id 自动传递
```

### ❌ 避免：手动注入

```go
// 每次都要手动注入，容易遗漏
tracer.Inject(ctx, carrier)
req.Header.Set("traceparent", traceparent)
client.Do(req)
```

## 14. 测试策略

### 单元测试

```go
func TestProcessOrder(t *testing.T) {
    mockTracer := &mockTracer{}
    zltrace.RegisterTracer(mockTracer)

    ctx := context.Background()
    err := ProcessOrder(ctx, "123")

    // 验证 span 是否创建
    assert.True(t, mockTracer.spanCreated)
    assert.NoError(t, err)
}
```

### 集成测试

```yaml
# 测试环境配置
trace:
  enabled: true
  exporter:
    type: none  # 不发送实际数据
```

## 15. 监控和告警

### 关键指标

- Span 生成速率
- 采样率
- Exporter 成功率
- 队列深度

### 告警规则

```yaml
# Exporter 失败率过高
- alert: HighExporterFailureRate
  expr: exporter_failure_rate > 0.05

# 队列积压
- alert: SpanQueueBacklog
  expr: span_queue_size > 10000
```

## 16. 版本管理

### API 稳定性

zltrace 遵循语义化版本：
- Major 版本变更：不兼容的 API 变更
- Minor 版本变更：向后兼容的功能新增
- Patch 版本变更：向后兼容的问题修正

### 升级建议

```bash
# 固定 Major 版本
go get github.com/zlxdbj/zltrace@v1

# 允许自动升级 Patch 版本
go get github.com/zlxdbj/zltrace@v1.0.x
```

## 相关文档

- [快速开始](./getting-started.md)
- [配置说明](./configuration.md)
- [常见问题](./faq.md)
