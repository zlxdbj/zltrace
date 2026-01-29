package httptrace

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/zlxdbj/zltrace"
)

// ============================================================================
// Gin 框架适配器
// ============================================================================

// ginHTTPHandler 实现zltrace.HTTPTraceHandler接口（Gin框架）
type ginHTTPHandler struct {
	c *gin.Context
}

func (h *ginHTTPHandler) GetMethod() string {
	return h.c.Request.Method
}

func (h *ginHTTPHandler) GetURL() string {
	return h.c.Request.URL.Path
}

func (h *ginHTTPHandler) GetHeader(key string) string {
	return h.c.GetHeader(key)
}

func (h *ginHTTPHandler) SetSpanContext(ctx context.Context) {
	h.c.Request = h.c.Request.WithContext(ctx)
}

func (h *ginHTTPHandler) GetSpanContext() context.Context {
	return h.c.Request.Context()
}

// TraceMiddleware 自动创建HTTP请求span的Gin中间件
// 使用示例：
//
//	engine := gin.Default()
//	engine.Use(middleware.TraceMiddleware())
func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		handler := &ginHTTPHandler{c: c}
		zltrace.TraceHTTPRequest(c.Request.Context(), handler, c.Next)
	}
}
