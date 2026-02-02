package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zlxdbj/zllog"
	"github.com/zlxdbj/zltrace"
	"github.com/zlxdbj/zltrace/adapter/httpadapter"
	"github.com/zlxdbj/zltrace/tracer/httptracer"
)

func main() {
	// 1. 初始化日志系统
	if err := zllog.InitLogger(); err != nil {
		zllog.Error(context.Background(), "init", "日志系统初始化失败", err)
	}

	// 2. 初始化追踪系统
	if err := zltrace.InitTracer(); err != nil {
		zllog.Error(context.Background(), "init", "追踪系统初始化失败", err)
		// 追踪系统初始化失败不影响业务运行，程序可以继续
	}
	defer func() {
		if tracer := zltrace.GetTracer(); tracer != nil {
			tracer.Close()
		}
	}()

	// 3. 创建 HTTP 服务
	r := gin.Default()

	// 4. 添加追踪中间件
	r.Use(httptracer.TraceMiddleware())

	// 5. 注册路由
	r.GET("/api/hello", handleHello)
	r.GET("/api/users/:id", handleGetUser)
	r.POST("/api/users", handleCreateUser)

	// 6. 启动服务
	fmt.Println("Server started on :8080")
	r.Run(":8080")
}

// handleHello 简单的 Hello 处理器
func handleHello(c *gin.Context) {
	zllog.Info(c.Request.Context(), "example", "Hello, World!",
		zllog.String("path", c.Request.URL.Path))
	c.JSON(http.StatusOK, gin.H{
		"message": "Hello, World!",
	})
}

// handleGetUser 获取用户信息（演示 HTTP Client 调用）
func handleGetUser(c *gin.Context) {
	userID := c.Param("id")

	// 创建子 span
	span, ctx := zltrace.GetSafeTracer().StartSpan(c.Request.Context(), "GetUser")
	defer span.Finish()

	span.SetTag("user_id", userID)

	// 调用下游服务（演示 trace_id 传递）
	userData, err := fetchUserFromService(ctx, userID)
	if err != nil {
		span.SetError(err)
		zllog.Error(ctx, "example", "Failed to fetch user", err,
			zllog.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	span.SetTag("status", "success")
	c.JSON(http.StatusOK, userData)
}

// handleCreateUser 创建用户
func handleCreateUser(c *gin.Context) {
	var user map[string]interface{}
	if err := c.ShouldBindJSON(&user); err != nil {
		zllog.Error(c.Request.Context(), "example", "Invalid request", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 创建子 span
	span, ctx := zltrace.GetSafeTracer().StartSpan(c.Request.Context(), "CreateUser")
	defer span.Finish()

	// 业务逻辑...
	zllog.Info(ctx, "example", "User created",
		zllog.String("user_id", user["id"].(string)))

	c.JSON(http.StatusCreated, user)
}

// fetchUserFromService 调用下游服务（演示 HTTP Client 的 trace_id 传递）
func fetchUserFromService(ctx context.Context, userID string) (map[string]interface{}, error) {
	// 创建带追踪的 HTTP Client
	client := httpadapter.NewTracedClient(nil)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8081/api/users/"+userID, nil)
	if err != nil {
		return nil, err
	}

	// 发送请求（trace_id 会自动注入到请求头）
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 解析响应...
	return map[string]interface{}{
		"id":       userID,
		"name":     "Test User",
		"email":    "test@example.com",
		"trace_id": getTraceID(ctx),
	}, nil
}

// getTraceID 从 context 获取 trace_id（用于演示）
func getTraceID(ctx context.Context) string {
	span := zltrace.SpanFromContext(ctx)
	if span != nil {
		return span.TraceID()
	}
	return ""
}
