# HTTP 追踪

zltrace 提供了完整的 HTTP 追踪支持，包括服务端和客户端。

## 服务端追踪

### Gin 中间件

使用 Gin 框架时，只需添加中间件即可：

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/zlxdbj/zltrace/tracer/httptracer"
)

func main() {
    r := gin.Default()

    // 添加追踪中间件
    r.Use(httptracer.TraceMiddleware())

    r.GET("/api/users", func(c *gin.Context) {
        // 自动创建 Entry Span
        // trace_id 自动从上游提取或生成
        zllog.Info(c.Request.Context(), "api", "处理请求")
        c.JSON(200, gin.H{"users": []string{}})
    })

    r.Run(":8080")
}
```

### 中间件功能

`TraceMiddleware()` 会自动：

1. ✅ 从 HTTP 请求头提取 `traceparent` (W3C 标准)
2. ✅ 如果没有，自动生成新的 trace_id
3. ✅ 创建 Entry Span
4. ✅ 将 span 注入到 request context
5. ✅ 记录请求方法和路径
6. ✅ 完成 span 并记录耗时

### 其他框架集成

对于其他框架，可以使用通用接口：

```go
import "github.com/zlxdbj/zltrace"

// 实现 HTTPTraceHandler 接口
type MyFrameworkHandler struct {
    // 你的框架特定字段
}

func (h *MyFrameworkHandler) GetMethod() string {
    return "GET"
}

func (h *MyFrameworkHandler) GetURL() string {
    return "/api/users"
}

func (h *MyFrameworkHandler) GetHeader(key string) string {
    // 返回请求头
}

func (h *MyFrameworkHandler) SetSpanContext(ctx context.Context) {
    // 设置 context 到请求
}

func (h *MyFrameworkHandler) GetSpanContext() context.Context {
    // 获取请求的 context
}

// 在中间件中使用
func MyMiddleware(h *MyFrameworkHandler, next func()) {
    zltrace.TraceHTTPRequest(context.Background(), h, next)
}
```

## 客户端追踪

### 自动追踪的 HTTP Client

使用 `httpadapter.NewTracedClient()` 创建自动追踪的 HTTP Client：

```go
import "github.com/zlxdbj/zltrace/adapter/httpadapter"

// 创建带追踪的客户端
client := httpadapter.NewTracedClient(nil)

// 使用方式和标准库完全一样
resp, err := client.Do(req)
```

### 自动功能

`TracedClient` 会自动：

1. ✅ 创建 Exit Span（表示调用外部服务）
2. ✅ 自动注入 `traceparent` header 到请求
3. ✅ 记录 HTTP 方法、URL、主机
4. ✅ 记录响应状态码
5. ✅ 4xx/5xx 自动标记为错误

### 使用自定义配置

```go
// 使用现有的 http.Client
customClient := &http.Client{
    Timeout: 5 * time.Second,
}
client := httpadapter.NewTracedClient(customClient)

// 或手动配置 Transport
client := &http.Client{
    Transport: &httpadapter.TracingRoundTripper{
        Base: http.DefaultTransport,
    },
}
```

### 手动注入 Headers

如果不想使用 `TracedClient`，可以手动注入：

```go
import "github.com/zlxdbj/zltrace"

func callDownstream(ctx context.Context) error {
    req, _ := http.NewRequestWithContext(ctx, "GET", "http://downstream/api", nil)

    // 手动注入 traceparent header
    tracer := zltrace.GetTracer()
    if tracer != nil {
        carrier := &httpHeaderCarrier{headers: req.Header}
        tracer.Inject(ctx, carrier)
    }

    client := &http.Client{}
    resp, err := client.Do(req)
    // ...
}

type httpHeaderCarrier struct {
    headers http.Header
}

func (c *httpHeaderCarrier) Set(key, value string) {
    c.headers.Set(key, value)
}

func (c *httpHeaderCarrier) Get(key string) (string, bool) {
    values := c.headers.Values(key)
    if len(values) == 0 {
        return "", false
    }
    return values[0], true
}
```

## W3C Trace Context

zltrace 使用 W3C Trace Context 标准（`traceparent` header）：

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

## 完整示例

```go
package main

import (
    "context"
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/zlxdbj/zllog"
    "github.com/zlxdbj/zltrace"
    "github.com/zlxdbj/zltrace/adapter/httpadapter"
    "github.com/zlxdbj/zltrace/tracer/httptracer"
)

func main() {
    zltrace.InitTracer()
    defer zltrace.GetTracer().Close()

    r := gin.Default()
    r.Use(httptracer.TraceMiddleware())

    // 创建带追踪的 HTTP Client
    downstreamClient := httpadapter.NewTracedClient(nil)

    r.GET("/api/proxy", func(c *gin.Context) {
        ctx := c.Request.Context()

        // 调用下游服务（trace_id 自动传递）
        req, _ := http.NewRequestWithContext(ctx, "GET", "http://downstream/api", nil)
        resp, err := downstreamClient.Do(req)
        if err != nil {
            zllog.Error(ctx, "proxy", "调用下游服务失败", err)
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }
        defer resp.Body.Close()

        zllog.Info(ctx, "proxy", "调用成功",
            zllog.Int("status", resp.StatusCode))

        c.JSON(200, gin.H{"status": "ok"})
    })

    r.Run(":8080")
}
```

## 相关文档

- [快速开始](./getting-started.md)
- [配置说明](./configuration.md)
- [示例代码](../_examples/http/)
