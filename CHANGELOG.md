# 更新日志

本项目的所有重要变更都会记录在这个文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [v1.2.2] - 待发布

### 修复

- **关键修复**：修复 `OTELProvider.GetTraceID()` 使用错误的 `SpanFromContext()` 导致 trace_id 不一致的问题
  - 症状：每条日志的 trace_id 都是独立生成的，无法追踪同一请求的完整链路
  - 原因：使用了 `zltrace.SpanFromContext()` 而不是 OpenTelemetry SDK 的 `trace.SpanFromContext()`
  - 修复：改用 OpenTelemetry SDK 的 `trace.SpanFromContext(ctx)` 正确获取 span

### 移除

- 删除废弃的 `tracerProvider` 结构体和 `NewTraceIDProvider()` 函数
- 删除误导性的 `SpanFromContext()` 和 `ContextWithSpan()` 函数
- 删除相关的 `contextKey` 和 `_spanKey` 常量

---

## [v1.2.1] - 2025-03-14

### 修复

- 优化追踪性能，降低 CPU 占用
- 追踪系统初始化失败使用 `zllog.Error` 替代 `panic`，不影响业务运行

---

## [v1.2.0] - 2025-03-10

### 新增

- **配置加载器**：新增 `ConfigLoader` 支持多种配置来源
  - 支持 `trace.yaml` 独立配置文件（推荐）
  - 支持 `resource/application.yaml` 集成配置
  - 支持环境配置文件 `application_{ENV}.yaml`
  - 支持环境变量 `$ZLTRACE_CONFIG`
  - 自动检测运行环境和服务名

### 配置加载优先级

1. `trace.yaml` - 独立配置文件
2. `resource/application.yaml` - 项目配置文件
3. `resource/application_{ENV}.yaml` - 环境配置文件
4. `zltrace.yaml` - 向后兼容
5. `$ZLTRACE_CONFIG` - 环境变量
6. `/etc/zltrace/config.yaml` - 系统配置
7. 默认配置 - 兜底

---

## [v1.1.1] - 2025-02-17

### 修复

- 修正 `saramatracer` 包名错误

### 文档

- 添加关于 span 的解释说明

---

## [v1.1.0] - 2025-02-14

### 新增

- **kafka-go 追踪支持**：新增 `kafkagotracer` 包支持 segmentio/kafka-go 客户端
  - `InjectKafkaProducerHeaders()` - 注入 trace_id 到生产者消息
  - `CreateKafkaConsumerContext()` - 从消费者消息提取 trace_id

### 改进

- 统一命名规范：所有协议特定追踪器使用 `*tracer` 后缀
  - `httptracer` - HTTP 追踪
  - `saramatracer` - IBM Sarama Kafka 追踪
  - `kafkagotracer` - segmentio/kafka-go Kafka 追踪

### 构建

- 升级到 Go 1.24 并更新依赖版本
- 修复 go.mod 版本兼容性问题

### 文档

- 重构文档结构，精简 README
- 添加 CLAUDE.md 开发指南
- 添加代码提交和版本发布规约

---

## [v1.0.0] - 2025-01-20

### 新增

- **核心功能**
  - `Tracer` 接口：定义分布式追踪器核心操作
  - `Span` 接口：表示一个追踪片段
  - `Carrier` 接口：trace 上下文载体

- **OpenTelemetry 集成**
  - 支持 W3C Trace Context 标准
  - 支持 OTLP 导出器（SkyWalking、Jaeger）
  - 支持 stdout 导出器（降级模式）

- **HTTP 追踪**
  - Gin 中间件自动追踪
  - HTTP Client 自动注入 trace_id

- **Kafka 追踪 (IBM Sarama)**
  - 生产者自动注入 trace_id
  - 消费者自动提取 trace_id

- **与 zllog 集成**
  - 自动注册 `TraceIDProvider`
  - 日志自动注入 trace_id

- **配置驱动**
  - YAML 配置文件
  - 多种采样器支持
  - 优雅降级机制
