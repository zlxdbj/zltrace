# 配置加载指南

zltrace 支持多种灵活的配置加载方式，参考 zllog 的设计，让集成更加简单。

## 配置加载优先级

zltrace 按以下优先级查找配置文件（高优先级 → 低优先级）：

1. **trace.yaml** - 独立配置文件（推荐用于微服务）
2. **resource/application.yaml** - 项目配置文件中的 `trace` 配置项
3. **resource/application_{ENV}.yaml** - 环境配置文件（如 `application_dev.yaml`）
4. **zltrace.yaml** - 向后兼容旧版本
5. **$ZLTRACE_CONFIG** - 环境变量指定路径
6. **/etc/zltrace/config.yaml** - 系统配置目录
7. **默认配置** - 兜底方案

### 配置查找目录

默认从以下目录查找配置文件：
- `./` - 当前目录
- `./resource/` - 资源目录（推荐）

## 方式 1：独立配置文件（推荐）

### 使用 trace.yaml

创建 `trace.yaml` 或 `resource/trace.yaml`：

```yaml
# 独立配置文件格式（无需 trace 前缀）
enabled: true
service_name: my_service

sampler:
  type: always_on
  ratio: 1.0

exporter:
  type: stdout
  otlp:
    endpoint: localhost:4317
    timeout: 10
    insecure: true

batch:
  batch_size: 512
  timeout: 5
  max_queue_size: 2048
```

### 使用方式

```go
import (
    "context"
    "github.com/zlxdbj/zllog"
    "github.com/zlxdbj/zltrace"
)

func main() {
    // 自动从 trace.yaml 加载配置
    if err := zltrace.InitTracer(); err != nil {
        zllog.Error(context.Background(), "init", "追踪系统初始化失败", err)
        // 追踪系统初始化失败不影响业务运行，程序可以继续
    }

    // 使用 zltrace...
}
```

## 方式 2：集成到项目配置文件

### 集成到 application.yaml

在 `resource/application.yaml` 中添加 `trace` 配置项：

```yaml
app:
  name: my_service  # zltrace 会自动读取

# 追踪配置（需要 trace 前缀）
trace:
  enabled: true
  service_name: ${app.name}  # 可选，不配置则使用 app.name

  sampler:
    type: always_on
    ratio: 1.0

  exporter:
    type: otlp
    otlp:
      endpoint: skywalking:4317
      timeout: 10
      insecure: false

  batch:
    batch_size: 512
    timeout: 5
    max_queue_size: 2048
```

### 集成到环境配置文件

`resource/application_dev.yaml`（开发环境）：

```yaml
app:
  name: my_service
  env: dev

trace:
  enabled: true
  exporter:
    type: stdout  # 开发环境输出到日志
```

`resource/application_prod.yaml`（生产环境）：

```yaml
app:
  name: my_service
  env: prod

trace:
  enabled: true
  exporter:
    type: otlp  # 生产环境发送到 SkyWalking
    otlp:
      endpoint: skywalking-prod:4317
      insecure: false
```

## 方式 3：使用 ConfigLoader（高级）

如果需要自定义配置加载行为：

```go
import (
    "context"
    "github.com/zlxdbj/zllog"
    "github.com/zlxdbj/zltrace"
)

func main() {
    // 创建配置加载器
    loader := zltrace.NewConfigLoader()

    // 自定义配置目录
    loader.SetConfigDirs("./config", "/opt/myapp/conf")

    // 设置环境（默认自动检测）
    loader.SetEnv("production")

    // 加载配置
    config, err := loader.Load()
    if err != nil {
        zllog.Error(context.Background(), "config", "加载追踪配置失败", err)
    }

    // 使用配置初始化
    if err := zltrace.InitTracer(); err != nil {
        zllog.Error(context.Background(), "init", "追踪系统初始化失败", err)
    }
}
```

## 环境自动检测

zltrace 自动从以下环境变量检测运行环境：

- `ENV`
- `APP_ENV`
- `GO_ENV`
- `MODE`

如果未设置，默认为 `dev`。

根据环境自动加载对应的配置文件：
- `application.yaml` - 基础配置
- `application_dev.yaml` - 开发环境配置（覆盖基础配置）
- `application_test.yaml` - 测试环境配置
- `application_prod.yaml` - 生产环境配置

## 服务名称自动检测

服务名称按以下优先级确定：

1. **配置文件** - `service_name` 或 `trace.service_name`
2. **环境变量** - `SERVICE_NAME` 或 `APP_NAME`
3. **可执行文件名** - 去掉 `.exe` 后缀
4. **当前目录名** - 项目目录名
5. **默认值** - `zltrace`

## 向后兼容

### 旧版配置文件（zltrace.yaml）

仍然支持旧的 `zltrace.yaml` 配置格式：

```yaml
trace:
  enabled: true
  service_name: my_service
  # ...
```

### 环境变量指定路径

```bash
export ZLTRACE_CONFIG=/path/to/custom-trace.yaml
```

### 全局 Viper 兼容

如果项目已经使用 Viper 加载了配置，zltrace 也能从中读取：

```go
import (
    "context"
    "github.com/zlxdbj/zllog"
    "github.com/zlxdbj/zltrace"
)

func main() {
    // 项目先加载自己的配置
    viper.SetConfigFile("application.yaml")
    viper.ReadInConfig()

    // zltrace 会尝试从全局 Viper 中读取 trace 配置
    if err := zltrace.InitTracer(); err != nil {
        zllog.Error(context.Background(), "init", "追踪系统初始化失败", err)
    }
}
```

## 配置示例对比

### 独立配置（trace.yaml）

```yaml
# 无需 trace 前缀
enabled: true
service_name: my_service
exporter:
  type: stdout
```

### 集成配置（application.yaml）

```yaml
app:
  name: my_service

# 需要 trace 前缀
trace:
  enabled: true
  exporter:
    type: stdout
```

## 配置项说明

### 全局配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `enabled` | bool | `true` | 是否启用追踪（总开关） |
| `service_name` | string | 自动检测 | 服务名称 |

### 采样配置 (sampler)

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `type` | string | `always_on` | 采样类型：`always_on`, `never`, `traceid_ratio`, `parent_based` |
| `ratio` | float64 | `1.0` | 采样比率（0.0-1.0），仅当 `type=traceid_ratio` 时生效 |

### 导出器配置 (exporter)

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `type` | string | `stdout` | 导出类型：`otlp`, `stdout`, `none` |
| `otlp.endpoint` | string | `localhost:4317` | OTLP 服务器地址 |
| `otlp.timeout` | int | `10` | 连接超时时间（秒） |
| `otlp.insecure` | bool | `true` | 是否使用 insecure 连接 |
| `max_queue_size` | int | `2048` | 最大队列大小 |

### 批量处理配置 (batch)

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `batch_size` | int | `512` | 批量发送的最大 span 数量 |
| `timeout` | int | `5` | 批量发送的超时时间（秒） |
| `max_queue_size` | int | `2048` | 最大队列大小 |

## 最佳实践

### 开发环境

使用 `trace.yaml` 独立配置：

```yaml
enabled: true
exporter:
  type: stdout  # 输出到日志，方便调试
```

### 测试环境

使用 `application_test.yaml` 集成配置：

```yaml
trace:
  enabled: true
  exporter:
    type: none  # 测试环境不发送追踪数据
```

### 生产环境

使用 `application_prod.yaml` 集成配置：

```yaml
trace:
  enabled: true
  exporter:
    type: otlp
    otlp:
      endpoint: skywalking-prod:4317
      insecure: false
```

## 故障排查

### 配置文件未生效

1. 检查文件路径是否正确（`./trace.yaml` 或 `./resource/trace.yaml`）
2. 检查文件格式是否正确（YAML 语法）
3. 查看日志确认配置加载情况

### 环境配置未生效

1. 确认环境变量已设置：`echo $ENV`
2. 确认环境配置文件存在：`resource/application_{ENV}.yaml`
3. 检查文件名拼写（`application_dev.yaml` 而非 `application-dev.yaml`）

### 服务名称不正确

1. 优先使用配置文件中的 `service_name`
2. 或设置环境变量：`export SERVICE_NAME=my_service`
3. 检查可执行文件名和目录名是否合理

## 环境变量

### SERVICE_NAME

服务名称，覆盖配置文件中的 `service_name` 或 `trace.service_name`。

```bash
export SERVICE_NAME=my_service
```

### APP_NAME

应用名称（备用，优先级低于 `SERVICE_NAME`）。

```bash
export APP_NAME=my_app
```

### ENV / APP_ENV / GO_ENV / MODE

环境名称，用于加载环境特定配置文件。

```bash
export ENV=production  # 加载 application_prod.yaml
```

### ZLTRACE_CONFIG

配置文件路径。

```bash
export ZLTRACE_CONFIG=/path/to/custom-trace.yaml
```

## 相关文档

- [快速开始](./getting-started.md)
- [API 文档](./api-reference.md)
- [配置最佳实践](./best-practices.md)
- [常见问题](./faq.md)
