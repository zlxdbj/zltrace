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

## 2. Span 命名规范

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

## 3. 标签使用规范

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

## 4. 错误处理规范

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

## 5. 采样策略

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

## 6. Exporter 选择

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

## 7. 性能优化

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

## 8. 安全建议

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

## 9. 日志集成

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

## 10. 优雅降级

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

## 11. Kafka 消息处理

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

## 12. HTTP Client 调用

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

## 13. 测试策略

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

## 14. 监控和告警

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

## 15. 版本管理

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
