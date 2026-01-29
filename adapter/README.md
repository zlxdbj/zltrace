# HTTP Client Trace Adapter

## 概述

`TracingRoundTripper` 是一个 HTTP Transport 包装器，**只负责一件事**：

> **自动将 trace_id 注入到 HTTP 请求头**

## 核心功能

- ✅ 自动创建 Exit Span（表示调用外部服务）
- ✅ 自动注入 `trace-id` 到 HTTP 请求头
- ✅ 自动记录 HTTP 状态码到 Span
- ✅ 使用标准 `http.RoundTripper` 接口

## 设计理念

### 职责单一

只负责 trace_id 注入，不改变 HTTP Client 的任何行为：
- ❌ 不负责日志记录（日志由使用者自由决定）
- ❌ 不负责重试逻辑
- ❌ 不负责超时控制
- ✅ 只负责 trace_id 传递

### 使用标准接口

实现了 Go 标准库的 `http.RoundTripper` 接口：
```go
type RoundTripper interface {
    RoundTrip(*Request) (*Response, error)
}
```

## 使用方式

### 方式1：创建带追踪的客户端

```go
import "go_shield/zltrace/adapter"

// 创建自动追踪的客户端
client := adapter.NewTracedClient(nil)

// 使用方式和标准库完全一样
resp, err := client.Do(req)
```

### 方式2：包装现有的 Client

```go
// 创建自定义客户端
customClient := &http.Client{
    Timeout: 10 * time.Second,
}

// 包装为带追踪的客户端
client := adapter.NewTracedClient(customClient)

resp, err := client.Do(req)
```

### 方式3：手动配置 Transport

```go
client := &http.Client{
    Transport: &adapter.TracingRoundTripper{
        Base: http.DefaultTransport,
    },
}

resp, err := client.Do(req)
```

### 方式4：在现有代码中替换

**替换前**：
```go
var client = http.Client{
    Timeout: 5 * time.Second,
}
```

**替换后**：
```go
var client = http.Client{
    Timeout: 5 * time.Second,
    Transport: &adapter.TracingRoundTripper{
        Base: http.DefaultTransport,
    },
}
```

## 完整示例

### 示例1：简单 GET 请求

```go
package main

import (
    "context"
    "net/http"
    "go_shield/zltrace/adapter"
)

func main() {
    // 初始化追踪系统
    zltrace.Init()

    // 创建客户端
    client := adapter.NewTracedClient(nil)

    // 发送请求
    req, _ := http.NewRequestWithContext(
        context.Background(),
        "GET",
        "http://api.example.com/users",
        nil,
    )

    // ✅ trace_id 自动注入到请求头
    resp, err := client.Do(req)
}
```

### 示例2：在 restful 包中使用

```go
package restful

import (
    "net/http"
    "go_shield/zltrace/adapter"
)

var client = http.Client{
    Timeout: 5 * time.Second,
    Transport: &adapter.TracingRoundTripper{  // ✅ 自动传递 trace_id
        Base: http.DefaultTransport,
    },
}
```

### 示例3：结合其他 Transport

```go
// 可以和其他 Transport 组合使用
client := &http.Client{
    Transport: &adapter.TracingRoundTripper{
        Base: &SomeOtherTransport{},  // 可以包装其他 Transport
    },
}
```

## 对比 Spring Cloud

### Spring Cloud + OpenFeign

```java
// 完全自动
@FeignClient(name = "user-service")
interface UserClient {
    @GetMapping("/users/{id}")
    User getUser(@PathVariable Long id);
}

// 使用
User user = userClient.getUser(123L);
// ✅ trace_id 自动注入到 header
// ✅ Exit Span 自动创建
```

### Go + zltrace adapter

```go
// 一样自动化！
client := adapter.NewTracedClient(nil)
req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
resp, err := client.Do(req)
// ✅ trace_id 自动注入到 header
// ✅ Exit Span 自动创建
```

## 自动注入的请求头

请求会自动包含以下 header：

```
trace-id: abc123-def456-...
```

下游服务可以从请求头中提取 trace-id，继续传递。

## 优雅降级

如果没有注册 Tracer（`zltrace.RegisterTracer(nil)`），或者 Tracer 为 nil：
- ✅ HTTP 请求正常执行
- ✅ 不会影响业务逻辑
- ❌ 只是不会有 trace_id（无法追踪）

## Span 标签

自动设置的 Span 标签：

- `http.url`: 请求URL
- `http.method`: 请求方法（GET/POST/PUT/DELETE）
- `http.host`: 目标主机
- `http.status_code`: 响应状态码
- `error`: 错误信息（如果返回 4xx/5xx）

## 重要提示

### 1. 必须使用带 Context 的 Request

```go
// ✅ 正确：使用 http.NewRequestWithContext
req, err := http.NewRequestWithContext(ctx, "GET", url, nil)

// ❌ 错误：使用 http.NewRequest（没有 context）
req, err := http.NewRequest("GET", url, nil)
```

### 2. Context 必须包含 trace_id

```go
// ✅ 在 HTTP handler 中使用（有 trace_id）
func Handler(c *gin.Context) {
    ctx := c.Request.Context()  // 包含 trace_id
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    client.Do(req)  // ✅ trace_id 会传递
}

// ⚠️ 使用 context.Background()（没有 trace_id）
func someFunction() {
    ctx := context.Background()  // 没有 trace_id
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    client.Do(req)  // ⚠️ 无法追踪
}
```

### 3. 日志记录由使用者决定

这个 adapter **不负责日志记录**，因为：
- 调用谁、是否记录日志，是使用者的自由
- 不同的 HTTP 调用可能需要不同的日志策略
- 保持职责单一，只负责 trace_id 传递

如果需要日志，可以在调用处手动记录：

```go
resp, err := client.Do(req)
if err != nil {
    zllog.Error(ctx, "module", "HTTP请求失败", err,
        zllog.String("url", url))
}
```

## 实现原理

### http.RoundTripper 接口

```go
type RoundTripper interface {
    RoundTrip(*Request) (*Response, error)
}
```

### TracingRoundTripper 包装器

```go
type TracingRoundTripper struct {
    Base http.RoundTripper  // 底层 Transport
}

func (t *TracingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
    // 1. 创建 Exit Span
    // 2. 注入 trace_id 到请求头
    // 3. 调用 Base.RoundTrip(req)  // 执行实际的 HTTP 请求
    // 4. 记录状态码到 Span
}
```

## 总结

- ✅ **职责单一**：只负责 trace_id 注入
- ✅ **使用简单**：与标准库用法完全一致
- ✅ **自动追踪**：无需手动注入 trace_id
- ✅ **灵活组合**：可以和其他 Transport 组合
- ✅ **优雅降级**：没有 Tracer 时正常工作
