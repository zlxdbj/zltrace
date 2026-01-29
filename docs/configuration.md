# 配置说明

zltrace 使用 YAML 配置文件，支持灵活的配置选项。

## 配置文件位置

配置文件查找顺序（按优先级）：

1. `./zltrace.yaml` - 当前目录
2. `$ZLTRACE_CONFIG` - 环境变量指定的路径
3. `/etc/zltrace/config.yaml` - 系统配置目录
4. 默认配置

## 完整配置示例

```yaml
# ========== 追踪系统配置 ==========
trace:
  # 是否启用追踪（总开关）
  enabled: true

  # 服务名称（用于追踪系统中标识本服务）
  service_name: my_service

  # 采样配置
  sampler:
    # 采样类型
    type: always_on  # always_on | never | traceid_ratio | parent_based
    # 采样比率（0.0 - 1.0），仅当 type=traceid_ratio 时生效
    ratio: 1.0

  # Exporter 配置（决定追踪数据发送到哪里）
  exporter:
    # 导出类型
    type: stdout  # otlp | stdout | none

    # OTLP gRPC 配置（type=otlp 时生效）
    otlp:
      # 追踪系统服务地址
      endpoint: localhost:4317
      # 连接超时时间（秒）
      timeout: 10
      # 是否使用 insecure 连接
      insecure: true

    # 最大队列大小
    max_queue_size: 2048

  # 批量处理配置
  batch:
    # 批量发送的最大 span 数量
    batch_size: 512
    # 批量发送的超时时间（秒）
    timeout: 5
    # 最大队列大小
    max_queue_size: 2048
```

## 配置项说明

### trace.enabled

是否启用追踪系统（总开关）。

- **类型**: `bool`
- **默认值**: `true`
- **说明**: 设置为 `false` 时，完全不启用追踪，不生成 trace_id

### trace.service_name

服务名称，用于在追踪系统中标识本服务。

- **类型**: `string`
- **默认值**: 从环境变量读取（`SERVICE_NAME` > `APP_NAME` > `"zltrace"`）
- **优先级**: 环境变量 > 配置文件 > 默认值

### trace.sampler.type

采样器类型，决定如何采样 trace。

- **类型**: `string`
- **可选值**:
  - `always_on` - 全量采样（100%）
  - `never` - 不采样
  - `traceid_ratio` - 按比率采样
  - `parent_based` - 基于父 span 决定
- **默认值**: `"always_on"`

### trace.sampler.ratio

采样比率。

- **类型**: `float64`
- **范围**: 0.0 - 1.0
- **默认值**: 1.0
- **说明**: 仅当 `sampler.type = traceid_ratio` 时生效

### trace.exporter.type

Exporter 类型，决定追踪数据发送到哪里。

- **类型**: `string`
- **可选值**:
  - `otlp` - 发送到追踪系统（SkyWalking、Jaeger 等）
  - `stdout` - 输出到日志（降级模式）
  - `none` - 不发送追踪数据（仅生成 trace_id）
- **默认值**: `"stdout"`

### trace.exporter.otlp.endpoint

OTLP gRPC 服务地址。

- **类型**: `string`
- **默认值**: `"localhost:4317"`
- **说明**: 仅当 `exporter.type = otlp` 时生效

### trace.exporter.otlp.timeout

连接超时时间（秒）。

- **类型**: `int`
- **默认值**: 10
- **单位**: 秒

### trace.exporter.otlp.insecure

是否使用 insecure 连接。

- **类型**: `bool`
- **默认值**: `true`
- **说明**: 开发环境使用 `true`，生产环境建议使用 `false`

### trace.batch.batch_size

批量发送的最大 span 数量。

- **类型**: `int`
- **默认值**: 512

### trace.batch.timeout

批量发送的超时时间（秒）。

- **类型**: `int`
- **默认值**: 5
- **单位**: 秒

## 环境变量

### SERVICE_NAME

服务名称，覆盖配置文件中的 `trace.service_name`。

```bash
export SERVICE_NAME=my_service
```

### APP_NAME

应用名称（备用，优先级低于 `SERVICE_NAME`）。

```bash
export APP_NAME=my_app
```

### ZLTRACE_CONFIG

配置文件路径。

```bash
export ZLTRACE_CONFIG=/path/to/zltrace.yaml
```

## 不同环境的配置

### 开发环境

```yaml
trace:
  enabled: true
  service_name: my_service_dev
  exporter:
    type: stdout  # 输出到日志，便于调试
  sampler:
    type: always_on
```

### 测试环境

```yaml
trace:
  enabled: true
  service_name: my_service_test
  exporter:
    type: none  # 不发送数据，仅生成 trace_id
  sampler:
    type: always_on
```

### 生产环境

```yaml
trace:
  enabled: true
  service_name: my_service_prod
  exporter:
    type: otlp  # 发送到 SkyWalking
    otlp:
      endpoint: skywalking-oap:4317
      timeout: 10
      insecure: false
  sampler:
    type: traceid_ratio
    ratio: 0.1  # 采样 10%，降低开销
  batch:
    batch_size: 1024
    timeout: 5
```

## 配置验证

如果配置有误，`InitTracer()` 会返回错误：

```go
if err := zltrace.InitTracer(); err != nil {
    log.Fatalf("初始化追踪系统失败: %v", err)
}
```

常见错误：
- `invalid exporter type` - exporter 类型不正确
- `trace.exporter.otlp.endpoint is required` - 使用 otlp 时缺少 endpoint 配置

## 相关文档

- [快速开始](./getting-started.md)
- [常见问题](./faq.md)
