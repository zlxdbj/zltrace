# 常见问题 (FAQ)

## 基础问题

### Q1: 是否必须创建 Span？

**不是必须的。** 创建 Span 取决于你的需求：

#### 何时需要创建 Span

- ✅ 需要追踪这个操作的**耗时**和**执行情况**
- ✅ 需要在 SkyWalking/Jaeger 中看到这个操作作为独立的调用节点
- ✅ 需要记录业务标签、错误信息等
- ✅ 想建立清晰的调用层次结构

```go
// 需要追踪"处理消息"这个操作
span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "HandleMessage")
defer span.Finish()

err := processBusiness(ctx)
if err != nil {
    span.SetError(err)  // 记录错误
}
span.SetTag("message_id", msg.ID)  // 记录业务标签
```

#### 何时不创建 Span

- ❌ 只是简单传递 trace_id 给下游调用
- ❌ 操作太简单，不值得单独追踪（如简单的数据转换）
- ❌ 只想保证日志中有 trace_id（zltrace 已经自动注入）

```go
// 不创建 Span，直接传递 ctx
func handleMessage(ctx context.Context, msg *Message) error {
    // 直接调用下游，下游会创建自己的 Span
    return callDownstreamService(ctx, msg)
}
```

#### 关键区别

| 方面 | 创建 Span | 不创建 Span |
|------|----------|-------------|
| trace_id 传递 | ✅ 自动传递 | ✅ 自动传递 |
| 日志 trace_id | ✅ 自动注入 | ✅ 自动注入 |
| 追踪系统可见 | ✅ 显示为节点 | ❌ 不可见 |
| 耗时统计 | ✅ 记录耗时 | ❌ 无记录 |
| 标签/错误 | ✅ 可添加 | ❌ 不可添加 |

**重要**：Context 本身就携带 trace_id，创建 Span 是为了在追踪系统中"记录"这个操作。如果不需要追踪这个操作本身，不创建 Span 完全没问题。

---

### Q2: 为什么需要传递 context.Context？

**常见疑问**：为什么每个函数都要传 `context.Context`？这样不是让代码变复杂了吗？

#### Go 语言的标准做法

在 Go 语言中，**context 是请求范围的元数据传递的标准方式**：

- `database/sql` 包：所有查询方法都接收 context
- `net/http` 包：Request 包含 context
- Go 官方推荐：所有接收请求的函数都应接收 context

#### trace_id 的传递

```go
// ❌ 错误：trace_id 链中断
func ProcessData(data string) {
    zltrace.GetSafeTracer().StartSpan(context.Background(), "ProcessData")
    // 每次都是新的 trace_id，无法追踪！
}

// ✅ 正确：trace_id 贯穿调用链
func ProcessData(ctx context.Context) error {
    span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "ProcessData")
    defer span.Finish()
    // trace_id 从上游传递过来，可以追踪完整流程
}
```

#### 生产环境影响

| 方案 | 代码简洁性 | 可追踪性 | 生产环境适用性 |
|------|------------|----------|--------------|
| 所有函数传递 context | 较复杂 | ⭐⭐⭐⭐⭐ | ✅ 推荐 |
| 使用 `context.Background()` | 简单 | ⭐ | ❌ 不推荐 |

**结论**：传递 context 是 **Go 语言的规约**，也是 **分布式系统的标准做法**。生产环境的可观测性比开发便利性更重要。

### Q3: Exporter 类型如何选择？

| 场景 | 推荐类型 | 说明 |
|------|----------|------|
| 开发环境 | `stdout` | 输出到日志，便于调试 |
| 生产环境 | `otlp` | 发送到 SkyWalking 等系统 |
| 测试环境 | `none` | 仅生成 trace_id，不发送 |
| 无追踪系统 | `stdout` 或 `none` | 降级模式 |

**降级策略**：当 SkyWalking 不可用时，可临时切换到 `stdout` 模式。

### Q4: 如何查看追踪数据？

**方式1：stdout 模式**

```bash
# 查看日志
tail -f logs/app.log | grep trace_id

# 输出示例
{"trace_id": "abc123...", "span_id": "def456...", "name": "ProcessOrder", ...}
```

**方式2：SkyWalking UI**

1. 打开 SkyWalking UI（通常在 `http://skywalking:8080`）
2. 选择服务
3. 查看：
   - 拓扑图（服务依赖关系）
   - 调用链路（Trace）
   - 性能指标（响应时间、吞吐量）

### Q5: 性能开销如何？

zltrace 的性能开销非常小：

| 项目 | 开销 | 说明 |
|------|------|------|
| 内存分配 | ~1KB/span | 仅存储 trace_id、span_id、时间戳 |
| CPU 开销 | <1% | 仅生成 ID 和时间戳 |
| 网络开销 | 取决于 exporter | `stdout` 无网络开销，`otlp` 有网络开销 |
| 日志注入 | 0 | zllog 自动获取 trace_id，无额外调用 |

**优化建议**：
- 使用采样器减少 span 数量（`traceid_ratio`）
- 生产环境使用 `otlp` 批量发送
- 高并发场景合理设置采样率

### Q6: 如何调试追踪问题？

**启用 stdout 模式**：
```yaml
trace:
  exporter:
    type: stdout  # 查看追踪数据
```

**查看日志输出**：
```json
{
  "trace_id": "abc123...",
  "span_id": "def456...",
  "name": "ProcessOrder",
  "duration": 123  // 毫秒
}
```

**检查 trace_id 传递**：
```go
// 在关键位置打印 trace_id
zllog.Info(ctx, "debug", "trace_id check",
    zllog.String("trace_id", getTraceID(ctx)),
)
```

### Q7: 如何禁用追踪？

**方式1：配置文件**
```yaml
trace:
  enabled: false  # 完全禁用
```

**方式2：环境变量**
```bash
export TRACE_ENABLED=false
```

**方式3：使用 none exporter**
```yaml
trace:
  exporter:
    type: none  # 不发送追踪数据
```

### Q8: 与 zllog 如何集成？

zltrace 通过 `TraceIDProvider` 接口与 zllog 解耦：

```go
// 1. zltrace 实现 TraceIDProvider
type OTELProvider struct {
    tracer *OTELTracer
}

func (p *OTELProvider) GetTraceID(ctx context.Context) string {
    span := SpanFromContext(ctx)
    if span == nil {
        return ""
    }
    return span.TraceID()
}

// 2. 注册到 zllog（zltrace.InitTracer() 自动完成）
zllog.RegisterTraceIDProvider(&OTELProvider{...})

// 3. zllog 自动获取 trace_id
zllog.Info(ctx, "module", "message")
// 输出：{"trace_id": "abc123...", "module": "module", ...}
```

**优势**：
- ✅ 完全解耦：zllog 不依赖 zltrace
- ✅ 自动集成：无需手动传递 trace_id
- ✅ 灵活切换：可以使用不同的追踪系统

## 配置问题

### Q9: 配置文件找不到怎么办？

**查找顺序**：
1. `./zltrace.yaml`
2. `$ZLTRACE_CONFIG` 环境变量
3. `/etc/zltrace/config.yaml`
4. 使用默认配置

**解决方案**：
```bash
# 方式1：创建默认配置文件
cp zltrace.yaml.example zltrace.yaml

# 方式2：使用环境变量指定
export ZLTRACE_CONFIG=/path/to/zltrace.yaml

# 方式3：使用系统配置目录
sudo cp zltrace.yaml /etc/zltrace/config.yaml
```

### Q10: 如何覆盖服务名称？

**优先级**：环境变量 > 配置文件 > 默认值

```bash
# 方式1：SERVICE_NAME 环境变量（推荐）
export SERVICE_NAME=my_service

# 方式2：APP_NAME 环境变量（备用）
export APP_NAME=my_app

# 方式3：配置文件
# trace:
#   service_name: my_service
```

### Q11: 采样率如何设置？

**开发环境**：
```yaml
sampler:
  type: always_on  # 100%
```

**生产环境**：
```yaml
sampler:
  type: traceid_ratio
  ratio: 0.1  # 10%，降低开销
```

**高流量场景**：
```yaml
sampler:
  type: traceid_ratio
  ratio: 0.01  # 1%
```

## 集成问题

### Q12: 如何与现有代码集成？

**渐进式集成**：
```go
// 阶段1：仅在入口处集成
func Handler(c *gin.Context) {
    span, ctx := zltrace.GetSafeTracer().StartSpan(c.Request.Context(), "Handler")
    defer span.Finish()

    // 原有代码不变
    processRequest(c)
}

// 阶段2：逐步添加到关键函数
func processRequest(c *gin.Context) {
    // ...
}

// 阶段3：全面覆盖
func everyFunction(ctx context.Context) {
    span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "function")
    defer span.Finish()
}
```

### Q13: 支持哪些 Go 版本？

- **最低版本**：Go 1.19
- **推荐版本**：Go 1.21+
- **测试覆盖**：Go 1.19, 1.20, 1.21

### Q14: 支持哪些框架？

**HTTP 框架**：
- ✅ Gin（开箱即用）
- ✅ Echo（需要适配器）
- ✅ Fiber（需要适配器）
- ✅ 标准库 `net/http`（需要适配器）

**消息队列**：
- ✅ Kafka（IBM Sarama）
- ✅ Kafka（segmentio/kafka-go）
- 🚧 RabbitMQ（计划中）
- 🚧 RocketMQ（计划中）

### Q15: 如何适配其他框架？

参考 Gin 的实现：

```go
// 1. 实现 HTTPTraceHandler 接口
type MyFrameworkHandler struct {
    // 你的框架特定字段
}

func (h *MyFrameworkHandler) GetMethod() string {
    return h.Request.Method
}

func (h *MyFrameworkHandler) GetURL() string {
    return h.Request.URL.Path
}

func (h *MyFrameworkHandler) GetHeader(key string) string {
    return h.Request.Header.Get(key)
}

func (h *MyFrameworkHandler) SetSpanContext(ctx context.Context) {
    h.Request = h.Request.WithContext(ctx)
}

func (h *MyFrameworkHandler) GetSpanContext() context.Context {
    return h.Request.Context()
}

// 2. 在中间件中使用
func MyMiddleware(h *MyFrameworkHandler, next func()) {
    zltrace.TraceHTTPRequest(context.Background(), h, next)
}
```

## 故障排查

### Q16: 追踪系统故障会影响业务吗？

**不会**。zltrace 采用优雅降级设计：

```go
// 即使追踪系统未初始化，也返回 noOpTracer
span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "operation")
defer span.Finish()
// 业务代码继续正常运行
```

### Q17: trace_id 丢失怎么办？

**检查清单**：
1. ✅ 确认中间件已添加
2. ✅ 确认所有函数都传递 `ctx`
3. ✅ 确认使用 `GetSafeTracer()` 而非 `GetTracer()`
4. ✅ 确认 HTTP Client 使用 `TracedClient`
5. ✅ 确认 Kafka 使用 `InjectKafkaProducerHeaders`

**调试方法**：
```go
// 在关键位置打印 trace_id
zllog.Info(ctx, "debug", "trace_id",
    zllog.String("trace_id", getTraceID(ctx)))
```

### Q18: 内存泄漏怎么办？

**检查项**：
1. ✅ 确认调用了 `span.Finish()`
2. ✅ 确认使用了 `defer span.Finish()`
3. ✅ 确认队列大小合理（`max_queue_size`）
4. ✅ 确认采样率不会过低

**推荐做法**：
```go
func processOrder(ctx context.Context) error {
    span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "ProcessOrder")
    defer span.Finish()  // 确保总是调用 Finish

    // 业务逻辑
}
```

## 其他问题

### Q19: 开源协议是什么？

采用 **MIT License**，允许商业使用。

### Q20: 如何贡献代码？

请参考[贡献指南](../CONTRIBUTING.md)。

### Q21: 如何获取帮助？

- 📖 查看[文档](./index.md)
- 💡 查看[示例代码](../_examples/)
- 🐛 提交 [GitHub Issue](https://github.com/zlxdbj/zltrace/issues)
- 📧 发送邮件到维护者

## 相关文档

- [快速开始](./getting-started.md)
- [配置说明](./configuration.md)
- [最佳实践](./best-practices.md)
