# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

zltrace 是一个基于 OpenTelemetry + W3C Trace Context 标准的 Go 分布式追踪组件，提供简单易用的分布式追踪能力。核心特点是：
- 支持 Kafka（IBM Sarama 和 segmentio/kafka-go）和 HTTP 的自动 trace_id 传递
- 配置文件驱动（YAML），支持开发/生产环境无缝切换
- 优雅降级：追踪系统故障不影响业务运行
- 与 zllog 无缝集成，trace_id 自动注入到日志

## 开发命令

### 基础命令
```bash
# 编译
go build ./...

# 运行所有测试
go test ./...

# 运行单个包的测试
go test ./tracer/kafkagotracer

# 运行单个测试
go test -run TestCreateKafkaConsumerContext ./tracer/kafkagotracer

# 测试并显示覆盖率
go test -cover ./...

# 整理依赖
go mod tidy

# 代码格式化
gofmt -l .
gofmt -w .

# 静态分析
go vet ./...
```

### 示例代码
```bash
# HTTP 示例
cd _examples/http
go run simple.go

# Kafka (Sarama) 示例
cd _examples/kafka
go run producer.go
go run consumer.go

# Kafka (kafka-go) 示例
cd _examples/kafka-go
go run producer.go
go run consumer.go
```

## 核心架构

### 分层设计
```
业务代码 (HTTP Handler / Kafka Producer/Consumer)
    ↓
zltrace API (InitTracer, GetSafeTracer, httpadapter, *tracer)
    ↓
OpenTelemetry SDK (W3C Trace Context, OTLP Exporter, Span 管理)
    ↓
追踪系统 (SkyWalking / Jaeger)
```

### 核心接口

**Tracer 接口**（`tracer.go`）：定义追踪器核心操作
- `StartSpan(ctx, operationName)` - 创建 Span
- `Inject(ctx, carrier)` - 注入 trace 上下文（客户端）
- `Extract(ctx, carrier)` - 提取 trace 上下文（服务端）
- `Close()` - 关闭追踪器

**Span 接口**（`tracer.go`）：表示一个追踪片段
- `SetTag(key, value)` - 设置标签
- `SetError(err)` - 设置错误
- `Finish()` - 结束 Span
- `TraceID()` - 获取 trace_id

**Carrier 接口**（`tracer.go`）：trace 上下文载体，用于跨进程传递
- `Get(key) (string, bool)` - 获取值
- `Set(key, value)` - 设置值

### 目录结构

**根目录**（核心包）：
- `config.go` - 配置管理，支持从 YAML/Viper 加载
- `init.go` - 初始化入口，`InitTracer()` 是推荐使用的初始化函数
- `opentelemetry.go` - OpenTelemetry 实现（OTELTracer、OTELSpan）
- `tracer.go` - Tracer 和 Span 接口定义，全局 Tracer 管理

**adapter/** - 框架适配器：
- `httpadapter/http_client.go` - HTTP Client 自动追踪（`TracingRoundTripper`）

**tracer/** - 协议特定追踪器：
- `httptracer/` - HTTP 追踪（Gin 中间件）
- `saramatracer/` - Kafka IBM Sarama 客户端追踪
- `kafkagotracer/` - Kafka segmentio/kafka-go 客户端追踪

**_examples/** - 示例代码

**docs/** - 完整文档（快速开始、配置、API、最佳实践等）

## 关键设计模式

### 1. 依赖倒置
业务代码依赖抽象的 Tracer 接口，不依赖具体实现。底层可以随时替换（如 OTELTracer → MockTracer）。

### 2. 优雅降级
`GetSafeTracer()` 永远返回可用的 Tracer。如果全局 tracer 为 nil，返回 `noOpTracer`（空操作），避免 panic。

### 3. 配置驱动
通过 YAML 配置控制行为，无需修改代码。配置查找顺序：
1. `./zltrace.yaml`
2. `$ZLTRACE_CONFIG` 环境变量
3. `/etc/zltrace/config.yaml`
4. 默认配置

### 4. Carrier 模式
使用 Carrier 接口抽象不同的 trace 上下文传递方式：
- HTTP Headers → `HTTPHeaderCarrier`
- Kafka Headers → `kafkaProducerHeaderCarrier` / `kafkaConsumerHeaderCarrier`

## 追踪器命名规范

所有协议特定追踪器使用 `*tracer` 后缀：
- `httptracer` - HTTP 追踪
- `saramatracer` - IBM Sarama Kafka 追踪
- `kafkagotracer` - segmentio/kafka-go Kafka 追踪

## Kafka 追踪实现

Kafka 追踪的两个包 API 完全一致：

**生产者**（注入 trace_id）：
```go
// saramatracer
ctx = saramatracer.InjectKafkaProducerHeaders(ctx, msg)

// kafkagotracer
ctx = kafkagotracer.InjectKafkaProducerHeaders(ctx, &msg)
```

**消费者**（提取 trace_id）：
```go
// saramatracer
ctx := saramatracer.CreateKafkaConsumerContext(msg)

// kafkagotracer
ctx := kafkagotracer.CreateKafkaConsumerContext(&msg)
```

关键实现：使用 Carrier 接口适配 Kafka 的 Headers 结构，注入/提取 W3C Trace Context (`traceparent` header)。

## 与 zllog 集成

zltrace 通过 `TraceIDProvider` 接口与 zllog 解耦：
1. zltrace 实现 `GetTraceID(ctx)` 从 context 提取 trace_id
2. `InitTracer()` 自动注册到 zllog
3. zllog 记录日志时自动获取并注入 trace_id

## 配置文件

配置示例：`zltrace.yaml.example`

核心配置项：
- `trace.enabled` - 是否启用追踪（总开关）
- `trace.service_name` - 服务名称
- `trace.exporter.type` - 导出类型：`otlp` | `stdout` | `none`
- `trace.exporter.otlp.endpoint` - SkyWalking/Jaeger 地址
- `trace.sampler.type` - 采样类型：`always_on` | `never` | `traceid_ratio` | `parent_based`

## 提交规范

使用 Conventional Commits：
- `feat:` - 新功能
- `fix:` - 修复 bug
- `docs:` - 文档更新
- `test:` - 测试相关
- `refactor:` - 重构代码
- `chore:` - 构建/工具链相关

示例：
```bash
git commit -m "feat: 添加对 segmentio/kafka-go 的支持"
git commit -m "fix: 修复 Kafka 消费者 trace_id 提取问题"
git commit -m "docs: 更新 HTTP 追踪文档"
```

## 文档优先级

1. **README.md** - 精简版，只保留核心内容、快速开始、文档链接
2. **docs/** - 详细文档（快速开始、配置、API、最佳实践、FAQ、架构）
3. **CONTRIBUTING.md** - 贡献指南和代码规范
4. **_examples/** - 可运行的示例代码

## 测试策略

- 核心包（`tracer.go`, `config.go`）有单元测试
- 协议追踪器（`kafkagotracer`）有完整的单元测试
- 使用 mock Tracer 进行测试，避免依赖实际追踪系统
- 测试文件命名：`*_test.go`

## 依赖管理

核心依赖：
- `go.opentelemetry.io/otel` - OpenTelemetry 核心
- `github.com/zlxdbj/zllog` - 日志组件（v1.1.0）
- `github.com/IBM/sarama` - Kafka Sarama 客户端
- `github.com/segmentio/kafka-go` - Kafka kafka-go 客户端
- `github.com/spf13/viper` - 配置管理

## 注意事项

1. **GetSafeTracer() vs GetTracer()**
   - 优先使用 `GetSafeTracer()`，避免 nil pointer 错误
   - `GetTracer()` 可能返回 nil，需要手动检查

2. **Context 传递**
   - 所有需要追踪的函数都应该接收 `context.Context` 作为第一个参数
   - 不要在函数内部使用 `context.Background()`，这会中断 trace 链

3. **Span 命名**
   - 使用清晰的动词+名词：`ProcessOrder`、`QueryDatabase`
   - HTTP：`HTTP GET /api/users`
   - Kafka：`Kafka/Produce/{topic}`、`Kafka/Consume/{topic}`

4. **配置文件位置**
   - 开发环境：`exporter.type: stdout`（输出到日志）
   - 生产环境：`exporter.type: otlp`（发送到 SkyWalking）
   - 测试环境：`exporter.type: none`（不发送数据）

## 代码提交和版本发布规约

### ✅ 自动提交代码
Claude Code 可以自动执行以下操作：
```bash
# 添加更改
git add -A

# 创建提交（遵循 Conventional Commits 规范）
git commit -m "feat: 描述"
```

### ❌ 禁止自动发版
Claude Code **禁止**自动执行以下操作：
- ❌ 创建 Git 标签（`git tag`）
- ❌ 推送标签到远程（`git push origin <tag>`）
- ❌ 创建 GitHub Release

### 🎯 发版流程（需要用户明确指令）
发布新版本时，必须**等待用户明确确认**后再执行：

```bash
# 步骤1：用户确认版本号
# 例如：用户说 "发布 v1.1.0"

# 步骤2：确认后再执行
git tag -a v1.1.0 -m "v1.1.0: 版本说明"
git push origin v1.1.0

# 步骤3：用户手动在 GitHub 创建 Release
# 或等待用户明确指令后使用 gh 命令创建
```

**重要**：即使看到代码中有版本号更新或其他暗示发版的信息，也**不能自动执行发版操作**。必须等待用户明确说"发布 v1.x.x"或"打标签"等指令。
