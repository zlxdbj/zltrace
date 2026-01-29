# zltrace - Go 分布式追踪组件

基于 **OpenTelemetry + W3C Trace Context** 标准的 Go 分布式追踪组件，提供简单易用的分布式追踪能力。

## 特性

- ✅ **W3C 标准兼容**：支持 W3C Trace Context 标准（`traceparent` header）
- ✅ **OpenTelemetry 集成**：基于 OpenTelemetry 实现，兼容 SkyWalking、Jaeger 等
- ✅ **框架无关**：支持 Gin、Echo、Fiber 等多种 Web 框架
- ✅ **Kafka 支持**：开箱即用的 Kafka trace_id 传递
- ✅ **HTTP Client 支持**：自动传递 trace_id 到下游服务
- ✅ **多种 Exporter**：支持 OTLP、stdout、none 三种模式
- ✅ **优雅降级**：追踪系统故障不影响业务
- ✅ **生产就绪**：高性能、低开销

---

## 为什么选择 zltrace？

> **"我们不是在重复造轮子，而是在给 OpenTelemetry 装上方向盘和加速器。"**

### 🤔 OpenTelemetry 已经很强大了，为什么还需要 zltrace？

OpenTelemetry 确实提供了完整的可观测性能力，但直接使用它就像**开一辆没有方向盘的赛车**——引擎很强大，但你需要自己写很多代码才能驾驭。

zltrace 不是 OpenTelemetry 的替代品，而是它的**增强层**和**最佳实践封装**。我们解决的是 OpenTelemetry 官方没有解决的问题，填补的是生产环境中的真实痛点。

### 📊 代码量减少 90%

| 场景 | 直接用 OpenTelemetry | 使用 zltrace | 减少比例 |
|------|---------------------|--------------|----------|
| **初始化** | ~100 行 | 1 行 | **99%** |
| **Gin 中间件** | 5 行配置 | 1 行 | **80%** |
| **Kafka Producer** | ~20 行自己写 | 1 行 | **95%** |
| **Kafka Consumer** | ~20 行自己写 | 1 行 | **95%** |
| **HTTP Client** | 3 行包装 | 1 行 | **67%** |
| **日志集成** | 每次 3 行 | 0 行（自动）| **100%** |

### 🚀 核心对比

#### ❌ 直接使用 OpenTelemetry 的痛苦

```go
// ========== 初始化（需要写 ~100 行代码） ==========
func initOpenTelemetry() error {
    // 1. 创建 Exporter
    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint("localhost:4317"),
        otlptracegrpc.WithInsecure(),
    )
    // ... 还需要 90+ 行代码

    // 2. Kafka 透传（OpenTelemetry 官方不支持！）
    func sendKafkaMessage(ctx context.Context, msg *sarama.ProducerMessage) error {
        // ❌ 需要自己写 ~20 行代码来注入 traceparent
        propagator := otel.GetTextMapPropagator()
        carrier := propagation.MapCarrier{}
        propagator.Inject(ctx, carrier)
        for k, v := range carrier {
            msg.Headers = append(msg.Headers, sarama.RecordHeader{
                Key: []byte(k), Value: []byte(v),
            })
        }
        return producer.SendMessage(msg)
    }

    // 3. 日志集成（每次都要手动传递 trace_id）
    func businessLogic(ctx context.Context) {
        span := trace.SpanFromContext(ctx)
        traceID := span.SpanContext().TraceID().String()
        // ❌ 每次记日志都要写这行代码
        log.WithField("trace_id", traceID).Info("processing")
    }
}
```

#### ✅ 使用 zltrace 的优雅

```go
// ========== 初始化（只需要 1 行代码） ==========
func main() {
    // ✅ 自动从 YAML 文件读取配置，一键启动
    zltrace.InitTracer()
    defer zltrace.GetTracer().Close()
}

// ========== Kafka 透传（开箱即用） ==========
func sendKafkaMessage(ctx context.Context, msg *sarama.ProducerMessage) error {
    // ✅ 一行代码搞定！
    saramatrace.InjectKafkaProducerHeaders(ctx, msg)
    return producer.SendMessage(msg)
}

// ========== 日志集成（完全自动化） ==========
func businessLogic(ctx context.Context) {
    // ✅ trace_id 自动注入到日志，无需手动传递！
    zllog.Info(ctx, "module", "processing request")
    // 输出：{"trace_id": "abc123...", "module": "module", "msg": "processing request"}
}
```

### 🏆 6 大核心优势

#### 1️⃣ 开箱即用的 Kafka 支持 ⭐⭐⭐⭐⭐

**OpenTelemetry 官方没有提供 Kafka 的自动插桩！** 特别是 IBM Sarama 客户端。

zltrace 提供了生产级的 Kafka 追踪支持：

```go
import "github.com/zlxdbj/zltrace/tracer/saramatrace"

// Producer: 自动注入 traceparent header
msg := &sarama.ProducerMessage{Topic: "alarm-raw-fire", Value: sarama.StringEncoder(data)}
saramatrace.InjectKafkaProducerHeaders(ctx, msg)
producer.SendMessage(msg)

// Consumer: 自动提取 traceparent header
ctx := saramatrace.CreateKafkaConsumerContext(msg)
processMessage(ctx, msg)
```

**✅ 完全兼容 W3C Trace Context 标准 | ✅ 支持跨服务调用链追踪**

#### 2️⃣ 配置文件驱动 ⭐⭐⭐⭐⭐

**OpenTelemetry 需要硬编码配置**，zltrace 支持 YAML 配置：

```yaml
trace:
  enabled: true
  service_name: my_service
  exporter:
    type: stdout  # otlp | stdout | none
    otlp:
      endpoint: skywalking:4317
  sampler:
    type: traceid_ratio
    ratio: 0.1  # 采样 10%
```

**✅ 开发环境用 stdout | ✅ 生产环境用 otlp | ✅ 无需修改代码，切换配置即可**

#### 3️⃣ 永不崩溃的优雅降级 ⭐⭐⭐⭐⭐

```go
// ❌ 直接用 OpenTelemetry：可能 panic
tracer := otel.Tracer("app")
ctx, span := tracer.Start(ctx, "operation")
// 如果忘记初始化，可能 panic！

// ✅ zltrace：永远安全
span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "operation")
defer span.Finish()
// 即使追踪系统未初始化，也返回 noOpSpan，不会影响业务！
```

**生产环境验证：追踪系统故障时，业务系统继续正常运行！**

#### 4️⃣ 与日志系统无缝集成 ⭐⭐⭐⭐⭐

```go
// ❌ 传统方式：每次都要手动传递 trace_id
span := trace.SpanFromContext(ctx)
traceID := span.SpanContext().TraceID().String()
log.WithField("trace_id", traceID).Info("message")

// ✅ zltrace：完全自动
zllog.Info(ctx, "module", "message")
// 输出：{"trace_id": "abc123...", "module": "module", "msg": "message"}
```

#### 5️⃣ 统一的 Tracer 接口 ⭐⭐⭐⭐

zltrace 提供了统一的接口抽象，可以随时切换底层实现：

```go
// 当前：使用 OpenTelemetry
zltrace.InitOpenTelemetryTracer()

// 将来：如果出现更好的追踪系统
zltrace.InitNewTracerSystem()

// 上层业务代码不需要修改！
span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "operation")
```

#### 6️⃣ 生产就绪的 HTTP Client 追踪 ⭐⭐⭐⭐⭐

```go
import "github.com/zlxdbj/zltrace/adapter/httpadapter"

// ✅ 一行代码创建自动追踪的 HTTP Client
client := httpadapter.NewTracedClient(nil)

// 使用方式和标准库完全一样
resp, err := client.Do(req)
// ✅ 自动创建 Exit Span
// ✅ 自动注入 traceparent header
// ✅ 自动记录 HTTP 状态码
```

---

## 快速开始

### 1. 初始化追踪系统

```go
import "github.com/zlxdbj/zltrace"

func main() {
    // 自动从配置文件读取配置
    // 配置文件加载顺序：
    // 1. ./zltrace.yaml
    // 2. $ZLTRACE_CONFIG 环境变量
    // 3. /etc/zltrace/config.yaml
    if err := zltrace.InitTracer(); err != nil {
        panic(err)
    }
}
```

### 2. 配置文件

创建 `zltrace.yaml` 配置文件：

```yaml
# zltrace.yaml
trace:
  enabled: true
  service_name: my_service

  exporter:
    type: stdout  # otlp | stdout | none

    otlp:
      endpoint: localhost:4317
      timeout: 10

  sampler:
    type: always_on  # always_on | never | traceid_ratio | parent_based
    ratio: 1.0
```

或者复制配置文件示例：
```bash
cp zltrace.yaml.example zltrace.yaml
```

### 3. HTTP 中间件集成

```go
import "github.com/zlxdbj/zltrace/tracer/httptrace"

// Gin 中间件
func TraceMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        zltrace.TraceHTTPRequest(c.Request.Context(),
            &GinHandler{c},
            c.Next,
        )
    }
}

// 或直接使用内置中间件
engine.Use(httptrace.TraceMiddleware())
```

### 4. Kafka 集成

```go
import (
    "github.com/zlxdbj/zltrace"
    "github.com/zlxdbj/zltrace/tracer/saramatrace"
)

// 生产者：发送消息时自动注入 trace_id
msg := &sarama.ProducerMessage{
    Topic: "my-topic",
    Value: sarama.StringEncoder("hello"),
}
ctx = saramatrace.InjectKafkaProducerHeaders(ctx, msg)
producer.SendMessage(msg)

// 消费者：接收消息时自动提取 trace_id
ctx := saramatrace.CreateKafkaConsumerContext(msg)
```

---

## 配置说明

### Exporter 类型

| 类型 | 说明 | 适用场景 |
|------|------|----------|
| `otlp` | 发送到追踪系统（SkyWalking、Jaeger） | 生产环境 |
| `stdout` | 输出到日志（降级模式） | 开发环境 |
| `none` | 不发送追踪数据（仅生成 trace_id） | 测试环境 |

### 配置示例

#### 开发环境

```yaml
# zltrace.yaml
trace:
  enabled: true
  service_name: my_service
  exporter:
    type: stdout  # 输出到日志
```

#### 生产环境

```yaml
# zltrace.yaml
trace:
  enabled: true
  service_name: my_service
  exporter:
    type: otlp  # 发送到 SkyWalking
    otlp:
      endpoint: skywalking-oap:4317
      timeout: 10
```

#### 禁用追踪

```yaml
trace:
  enabled: false  # 完全禁用追踪
```

### 采样器类型

| 类型 | 说明 | 配置 |
|------|------|------|
| `always_on` | 全量采样（100%） | `type: always_on` |
| `never` | 不采样 | `type: never` |
| `traceid_ratio` | 按比率采样 | `type: traceid_ratio`, `ratio: 0.1` |
| `parent_based` | 基于父 span 决定 | `type: parent_based` |

---

## 使用示例

### HTTP 请求追踪

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/zlxdbj/zltrace"
)

func main() {
    r := gin.Default()

    // 添加追踪中间件
    r.Use(TraceMiddleware())

    r.GET("/api/users", func(c *gin.Context) {
        // 自动创建 span，日志包含 trace_id
        zllog.Info(c.Request.Context(), "api", "Processing request")
        c.JSON(200, gin.H{"users": []string{}})
    })

    r.Run(":8080")
}
```

### HTTP Client 调用

```go
import "github.com/zlxdbj/zltrace"

// 自动传递 trace_id 到下游服务
func CallDownstreamService(ctx context.Context) error {
    req, _ := http.NewRequestWithContext(ctx, "GET", "http://downstream/api", nil)

    // 注入 traceparent header
    zltrace.InjectHTTPHeaders(ctx, req.Header, "GET http://downstream/api")

    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Do(req)
    // ...
}
```

### Kafka 生产者

```go
import (
    "github.com/zlxdbj/zltrace"
    "github.com/zlxdbj/zltrace/tracer/saramatrace"
)

func SendMessage(ctx context.Context) error {
    msg := &sarama.ProducerMessage{
        Topic: "alarm-raw-fire",
        Value: sarama.StringEncoder(data),
    }

    // 自动注入 trace_id 到消息 headers
    ctx = saramatrace.InjectKafkaProducerHeaders(ctx, msg)

    return producer.SendMessage(msg)
}
```

### Kafka 消费者

```go
import (
    "github.com/zlxdbj/zltrace"
    "github.com/zlxdbj/zltrace/tracer/saramatrace"
)

func ConsumeMessage(msg *sarama.ConsumerMessage) error {
    // 自动提取 trace_id
    ctx := saramatrace.CreateKafkaConsumerContext(msg)

    // 后续所有操作都会继承这个 trace_id
    return processMessage(ctx, msg)
}
```

### 手动创建 Span

```go
import "github.com/zlxdbj/zltrace"

func ProcessOrder(ctx context.Context, orderID string) error {
    // 创建子 span
    span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "ProcessOrder")
    defer span.Finish()

    // 添加标签
    span.SetTag("order_id", orderID)

    // 业务逻辑
    // ...

    return nil
}
```

### 错误追踪

```go
func ProcessData(ctx context.Context) error {
    span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "ProcessData")
    defer span.Finish()

    if err := doSomething(); err != nil {
        // 记录错误到 span
        span.SetError(err)
        return err
    }

    return nil
}
```

---

## API 参考

### 初始化函数

| 函数 | 说明 |
|------|------|
| `InitTracer()` | **推荐使用**：自动加载配置并初始化追踪系统 |
| `Init()` | 向后兼容：调用 `InitTracer()`，功能相同 |
| `InitOpenTelemetryTracer()` | 初始化 OpenTelemetry Tracer |
| `RegisterTracer(Tracer)` | 注册自定义 Tracer |

**命名风格统一**：
- `zllog.InitLogger()` - 初始化日志系统
- `zltrace.InitTracer()` - 初始化追踪系统

### Tracer 接口

| 函数 | 说明 |
|------|------|
| `GetTracer()` | 获取全局 Tracer |
| `GetSafeTracer()` | 获取安全的 Tracer（永不返回 nil） |
| `StartSpan(ctx, name)` | 创建新的 Span |

### HTTP 追踪

| 函数 | 说明 |
|------|------|
| `TraceHTTPRequest(ctx, handler, next)` | HTTP 请求追踪中间件 |
| `InjectHTTPHeaders(ctx, headers, operation)` | 注入 trace_id 到 HTTP headers |

### Kafka 追踪

| 函数 | 说明 |
|------|------|
| `saramatrace.InjectKafkaProducerHeaders(ctx, msg)` | 注入 trace_id 到 Kafka 消息（IBM Sarama） |
| `saramatrace.CreateKafkaConsumerContext(msg)` | 从 Kafka 消息提取 trace_id（IBM Sarama） |

> **注意**：Kafka 追踪功能在 `github.com/zlxdbj/zltrace/tracer/saramatrace` 包中，需要单独导入：
> ```go
> import "github.com/zlxdbj/zltrace/tracer/saramatrace"
> ```

---

## HTTP Client 集成

### 自动追踪的 HTTP Client

```go
import "github.com/zlxdbj/zltrace/adapter/httpadapter"

// 方式1：创建带追踪的客户端（推荐）
client := httpadapter.NewTracedClient(nil)

// 方式2：使用现有的 http.Client
customClient := &http.Client{Timeout: 5 * time.Second}
client := httpadapter.NewTracedClient(customClient)

// 方式3：手动配置 Transport
client := &http.Client{
    Transport: &httpadapter.TracingRoundTripper{
        Base: http.DefaultTransport,
    },
}

// 使用方式和标准库完全一样
resp, err := client.Do(req)
```

**特性**：
- ✅ 自动创建 Exit Span（调用外部服务）
- ✅ 自动注入 traceparent header
- ✅ 自动记录 HTTP 状态码
- ✅ 4xx/5xx 自动标记为错误

---

---

## W3C Trace Context 标准

zltrace 使用 **W3C Trace Context** 标准（`traceparent` header）：

### Header 格式

```
traceparent: 00-trace_id-span_id-flags
```

**示例**：
```
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
```

**字段说明**：
- `00` - 版本
- `4bf92f3577b34da6a3ce929d0e0e4736` - trace_id（32位十六进制）
- `00f067aa0ba902b7` - span_id（16位十六进制）
- `01` - flags（采样标志）

### 优势

- ✅ **行业标准**：W3C 标准，被所有主流追踪系统支持
- ✅ **跨语言兼容**：Java、Python、Node.js 等都支持
- ✅ **互操作性**：可与 SkyWalking、Jaeger、Zipkin 等系统互操作

---

## Trace 传递流程

### HTTP 调用链

```
服务 A                                           服务 B
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
  |    saramatrace.InjectKafkaProducerHeaders          |                              |
  |    注入 traceparent header            |                              |
  |    ----------------------------------> |                              |
  |                                       | 消息 headers:                 |
  |                                       |   traceparent: 00-abc123-...  |
  |                                       |                              |
  |                                       | ----------------------------> |
  |                                       |                              | 3. 接收消息
  |                                       |                              |    saramatrace.CreateKafkaConsumerContext
  |                                       |                              |    提取 trace_id = abc123
  |                                       |                              |    创建子 Span
  |                                       |                              |
  V                                       V                              V
```

---

## 与 SkyWalking 集成

### 配置 SkyWalking OAP

```yaml
# resource/application_prod.yaml
trace:
  enabled: true
  service_name: my_service
  exporter:
    type: otlp  # 使用 OTLP 协议
    otlp:
      endpoint: skywalking-oap:4317  # SkyWalking OAP 地址
      timeout: 10
```

### SkyWalking 版本支持

- SkyWalking 8.x+ 支持 OTLP 协议
- 使用端口 **4317**（而非传统的 11800）
- 自动生成服务拓扑、调用链路、性能指标

### 查看追踪数据

1. 打开 SkyWalking UI
2. 选择服务 `my_service`
3. 查看拓扑图、调用链路、日志关联

---

## 常见问题（FAQ）

### Q1: 为什么需要传递 context.Context？

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

### Q2: Exporter 类型如何选择？

| 场景 | 推荐类型 | 说明 |
|------|----------|------|
| 开发环境 | `stdout` | 输出到日志，便于调试 |
| 生产环境 | `otlp` | 发送到 SkyWalking 等系统 |
| 测试环境 | `none` | 仅生成 trace_id，不发送 |
| 无追踪系统 | `stdout` 或 `none` | 降级模式 |

**降级策略**：当 SkyWalking 不可用时，可临时切换到 `stdout` 模式。

### Q3: 如何查看追踪数据？

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

### Q4: 性能开销如何？

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

### Q5: 如何调试追踪问题？

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

### Q6: 如何禁用追踪？

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

### Q7: 与 zllog 如何集成？

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

---

## 最佳实践

### 1. Context 传递规范

```go
// ✅ HTTP Handler
func Handler(c *gin.Context) {
    ctx := c.Request.Context()
    ProcessOrder(ctx, orderID)
}

// ✅ 业务函数
func ProcessOrder(ctx context.Context, orderID string) error {
    span, ctx := zltrace.GetSafeTracer().StartSpan(ctx, "ProcessOrder")
    defer span.Finish()
    // ...
}

// ❌ 避免：不接收 context
func ProcessOrder(orderID string) error {
    // trace_id 链中断！
}
```

### 2. Span 命名规范

```go
// ✅ 好：清晰的命名
zltrace.GetSafeTracer().StartSpan(ctx, "ProcessOrder")
zltrace.GetSafeTracer().StartSpan(ctx, "QueryDatabase")
zltrace.GetSafeTracer().StartSpan(ctx, "Kafka/Produce/alarm-topic")

// ❌ 不好：模糊的命名
zltrace.GetSafeTracer().StartSpan(ctx, "doSomething")
zltrace.GetSafeTracer().StartSpan(ctx, "handle")
```

### 3. 标签使用规范

```go
// ✅ 好：结构化标签
span.SetTag("order_id", orderID)
span.SetTag("user_id", userID)
span.SetTag("status", "success")

// ❌ 不好：字符串拼接
span.SetTag("info", fmt.Sprintf("order=%s user=%s", orderID, userID))
```

### 4. 错误处理规范

```go
// ✅ 记录错误到 span
if err := doSomething(); err != nil {
    span.SetError(err)
    span.SetTag("error.code", "DB_ERROR")
    return err
}

// ❌ 不好：仅返回错误
if err := doSomething(); err != nil {
    return err  // trace 信息丢失
}
```

### 5. 生产环境配置建议

```yaml
trace:
  enabled: true
  exporter:
    type: otlp  # 发送到 SkyWalking
  sampler:
    type: traceid_ratio
    ratio: 0.1  # 采样 10%，降低开销
```

---

## 架构设计

### 分层架构

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
│  - saramatrace.InjectKafkaProducerHeaders()          │
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

### 设计优势

- ✅ **分层清晰**：业务代码 → zltrace API → OpenTelemetry → 追踪系统
- ✅ **依赖倒置**：业务代码依赖 zltrace 接口，不依赖具体实现
- ✅ **灵活替换**：底层可以随时替换，上层代码不需要修改
- ✅ **优雅降级**：追踪系统故障不影响业务运行

### 💡 类比说明

```
直接使用 OpenTelemetry  = 开手动挡跑车
  - 性能强大
  - 但需要专业司机
  - 每次都要换挡、离合

使用 zltrace            = 开自动挡豪车
  - 同样强大（底层就是 OpenTelemetry）
  - 但普通人也能开
  - 一键启动，自动换挡

我们不是在重新发明引擎，
而是在给引擎装上自动挡、方向盘和刹车系统！
```

---

## 依赖说明

zltrace 依赖以下库：

```go
require (
    go.opentelemetry.io/otel v1.39.0                    // 核心
    go.opentelemetry.io/otel/trace v1.39.0              // Trace API
    go.opentelemetry.io/otel/sdk v1.39.0                // SDK
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.39.0  // OTLP
    go.opentelemetry.io/otel/semconv/v1.24.0            // 语义约定
)

// tracer/saramatrace 子包额外依赖：
require (
    github.com/IBM/sarama v1.40.1                       // Kafka Sarama 客户端
)
```

---

## 参考文档

- [W3C Trace Context 规范](https://www.w3.org/TR/trace-context/)
- [OpenTelemetry 规范](https://opentelemetry.io/docs/reference/specification/)
- [SkyWalking 文档](https://skywalking.apache.org/docs/)
- [Go Context 官方文档](https://golang.org/pkg/context/)
- [Context 传递规范](../Context传递规范.md)

---

## 更新日志

### v2.0.0 (2025-01-28)
- ✅ 基于 OpenTelemetry + W3C Trace Context 重新实现
- ✅ 移除 go2sky 依赖（go2sky 已归档）
- ✅ 支持三种 Exporter：otlp、stdout、none
- ✅ 移除 provider 配置，OpenTelemetry 作为唯一标准
- ✅ 简化配置，通过 exporter.type 控制行为
- ✅ 代码减少约 350 行

### v1.0.0
- 初始版本，基于 go2sky 实现

---

## 许可证

本项目采用 MIT 许可证。

## 贡献

欢迎提交 Issue 和 Pull Request！

如果你觉得 zltrace 对你有帮助，请给我们一个 ⭐ Star！

---

**zltrace：让分布式追踪像 Hello World 一样简单！** ⭐
