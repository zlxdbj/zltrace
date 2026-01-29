# zltrace - Go 分布式追踪组件

> 基于 OpenTelemetry + W3C Trace Context 标准的 Go 分布式追踪组件，提供简单易用的分布式追踪能力。

[![Go Version](https://img.shields.io/badge/Go-1.19%2B-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](./LICENSE)
[![GitHub release](https://img.shields.io/github/v/release/zlxdbj/zltrace)](https://github.com/zlxdbj/zltrace/releases)

## ✨ 特性

- ✅ **W3C 标准兼容** - 支持 W3C Trace Context 标准（`traceparent` header）
- ✅ **OpenTelemetry 集成** - 基于 OpenTelemetry 实现，兼容 SkyWalking、Jaeger 等
- ✅ **框架无关** - 支持 Gin、Echo、Fiber 等多种 Web 框架
- ✅ **Kafka 支持** - 开箱即用的 Kafka trace_id 传递（支持 IBM Sarama 和 kafka-go）
- ✅ **HTTP Client 支持** - 自动传递 trace_id 到下游服务
- ✅ **配置文件驱动** - 支持 YAML 配置，开发/生产环境无缝切换
- ✅ **优雅降级** - 追踪系统故障不影响业务
- ✅ **生产就绪** - 高性能、低开销

## 🚀 快速开始

### 安装

```bash
go get github.com/zlxdbj/zltrace@latest
```

### 1. 创建配置文件

```bash
cp zltrace.yaml.example zltrace.yaml
```

### 2. 初始化

```go
package main

import (
    "github.com/zlxdbj/zllog"
    "github.com/zlxdbj/zltrace"
)

func main() {
    // 初始化日志系统
    zllog.InitLogger()

    // 初始化追踪系统
    zltrace.InitTracer()
    defer zltrace.GetTracer().Close()

    // 你的业务代码...
}
```

### 3. HTTP 服务

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/zlxdbj/zltrace/tracer/httptracer"
)

r := gin.Default()
r.Use(httptracer.TraceMiddleware())

r.GET("/api/users", func(c *gin.Context) {
    // trace_id 自动注入到日志
    zllog.Info(c.Request.Context(), "api", "获取用户列表")
    c.JSON(200, gin.H{"users": []string{}})
})
```

### 4. Kafka 消息

```go
import "github.com/zlxdbj/zltrace/tracer/saramatracer"

// 生产者
msg := &sarama.ProducerMessage{Topic: "test", Value: sarama.StringEncoder("hello")}
ctx = saramatracer.InjectKafkaProducerHeaders(ctx, msg)
producer.SendMessage(msg)

// 消费者
ctx := saramatracer.CreateKafkaConsumerContext(msg)
processMessage(ctx, msg)
```

## 📖 文档

- 📘 [完整文档](./docs/) - 详细的文档和指南
- 🚀 [快速开始](./docs/getting-started.md) - 5分钟上手
- ⚙️ [配置说明](./docs/configuration.md) - 配置选项详解
- 🌐 [HTTP 追踪](./docs/http-tracing.md) - HTTP 服务和客户端追踪
- 📨 [Kafka 追踪](./docs/kafka-tracing.md) - Kafka 消息队列追踪
- 📚 [API 参考](./docs/api-reference.md) - 完整的 API 文档
- 💡 [最佳实践](./docs/best-practices.md) - 生产环境使用建议
- ❓ [常见问题](./docs/faq.md) - FAQ 和问题排查
- 🏗️ [架构设计](./docs/architecture.md) - 技术架构说明

## 💡 为什么选择 zltrace？

### 代码量减少 90%

| 场景 | 直接用 OpenTelemetry | 使用 zltrace | 减少 |
|------|---------------------|--------------|------|
| 初始化 | ~100 行 | 1 行 | **99%** |
| Kafka 透传 | ~20 行自己写 | 1 行 | **95%** |
| 日志集成 | 每次 3 行 | 0 行（自动）| **100%** |

### 核心优势

- ✅ **开箱即用的 Kafka 支持** - OpenTelemetry 官方没有提供 Kafka 的自动插桩
- ✅ **配置文件驱动** - 开发用 stdout，生产用 otlp，无需修改代码
- ✅ **优雅降级** - 追踪系统故障时，业务系统继续正常运行
- ✅ **与 zllog 无缝集成** - trace_id 自动注入到日志
- ✅ **统一的 Tracer 接口** - 底层实现可以随时替换

详细对比：[为什么选择 zltrace](./README_ADVANTAGES.md)

## 🎯 支持的框架和库

### HTTP 框架
- ✅ Gin
- ✅ Echo（需适配）
- ✅ Fiber（需适配）
- ✅ 标准库 `net/http`（需适配）

### 消息队列
- ✅ Kafka (IBM Sarama)
- ✅ Kafka (segmentio/kafka-go)
- 🚧 RabbitMQ（计划中）
- 🚧 RocketMQ（计划中）

### 追踪系统
- ✅ SkyWalking
- ✅ Jaeger
- ✅ Zipkin
- ✅ 任何支持 OTLP 的系统

## 📦 示例代码

完整示例请查看 [_examples](./_examples/) 目录：

- [HTTP 服务示例](./_examples/http/simple.go)
- [Kafka (Sarama) 示例](./_examples/kafka/)
- [Kafka (kafka-go) 示例](./_examples/kafka-go/)

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！详情请查看[贡献指南](./CONTRIBUTING.md)。

如果你觉得 zltrace 对你有帮助，请给我们一个 ⭐ Star！

## 📄 许可证

本项目采用 [MIT License](./LICENSE) 许可证。

## 🔗 相关项目

- [zllog](https://github.com/zlxdbj/zllog) - 结构化日志组件
- [OpenTelemetry Go](https://github.com/open-telemetry/opentelemetry-go)
- [SkyWalking](https://skywalking.apache.org/)

---

**zltrace：让分布式追踪像 Hello World 一样简单！** ⭐
