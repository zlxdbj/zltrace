# 配置加载器更新说明

## 版本：v1.1.0

## 更新内容

本次更新参考 `zllog` 的设计，为 zltrace 添加了灵活的配置加载器，支持多种配置来源。

### 主要改进

#### 1. 新增 ConfigLoader

新增 `ConfigLoader` 配置加载器，支持按优先级从多个位置加载配置：

```go
loader := zltrace.NewConfigLoader()
config, err := loader.Load()
```

#### 2. 配置加载优先级

zltrace 现在按以下优先级查找配置文件（高 → 低）：

1. **trace.yaml** - 独立配置文件（推荐）
2. **resource/application.yaml** - 项目配置文件
3. **resource/application_{ENV}.yaml** - 环境配置文件
4. **zltrace.yaml** - 向后兼容
5. **$ZLTRACE_CONFIG** - 环境变量
6. **/etc/zltrace/config.yaml** - 系统配置
7. **默认配置** - 兜底

#### 3. 支持环境配置

自动根据环境变量加载对应配置：
- `application_dev.yaml` - 开发环境
- `application_test.yaml` - 测试环境
- `application_prod.yaml` - 生产环境

#### 4. 两种配置格式

**独立配置（trace.yaml）**：
```yaml
# 无需 trace 前缀
enabled: true
service_name: my_service
exporter:
  type: stdout
```

**集成配置（application.yaml）**：
```yaml
app:
  name: my_service

# 需要 trace 前缀
trace:
  enabled: true
  service_name: ${app.name}
  exporter:
    type: stdout
```

#### 5. 改进的服务名检测

服务名自动检测优先级：
1. 配置文件 `service_name`
2. 环境变量 `SERVICE_NAME` / `APP_NAME`
3. 可执行文件名
4. 当前目录名
5. 默认值 `zltrace`

#### 6. 环境自动检测

自动从以下环境变量检测运行环境：
- `ENV`
- `APP_ENV`
- `GO_ENV`
- `MODE`

### 向后兼容

✅ 完全兼容旧的 `zltrace.yaml` 配置格式
✅ 支持环境变量 `$ZLTRACE_CONFIG`
✅ 兼容全局 Viper 配置

### 使用示例

#### 方式 1：独立配置文件

```bash
# 创建 trace.yaml
cp trace.yaml.example trace.yaml

# 修改配置
vim trace.yaml

# 运行程序（自动加载）
go run main.go
```

#### 方式 2：集成到项目配置

```bash
# 创建 resource 目录
mkdir -p resource

# 复制示例配置
cp application.yaml.example resource/application.yaml

# 创建环境配置
cp application.yaml.example resource/application_dev.yaml
cp application.yaml.example resource/application_prod.yaml

# 运行程序（自动加载）
go run main.go
```

#### 方式 3：自定义配置加载

```go
loader := zltrace.NewConfigLoader()
loader.SetConfigDirs("./config", "/opt/myapp/conf")
loader.SetEnv("production")

config, err := loader.Load()
if err != nil {
    panic(err)
}

// 使用配置
if err := zltrace.InitTracer(); err != nil {
    panic(err)
}
```

### 配置文件位置

默认查找目录：
- `./` - 当前目录
- `./resource/` - 资源目录（推荐）

### 环境配置示例

**开发环境** (`resource/application_dev.yaml`)：
```yaml
trace:
  enabled: true
  exporter:
    type: stdout  # 输出到日志
```

**生产环境** (`resource/application_prod.yaml`)：
```yaml
trace:
  enabled: true
  exporter:
    type: otlp
    otlp:
      endpoint: skywalking-prod:4317
      insecure: false
```

### 文件变更

**新增文件**：
- `trace.yaml.example` - 独立配置文件示例
- `application.yaml.example` - 集成配置文件示例
- `CHANGELOG_CONFIG_LOADER.md` - 本更新说明

**修改文件**：
- `config.go` - 新增 ConfigLoader，改进配置加载逻辑
- `config_test.go` - 更新测试用例
- `docs/configuration.md` - 更新配置文档

### 升级指南

#### 从旧版本升级

**无破坏性变更**！旧的 `zltrace.yaml` 配置文件仍然有效。

#### 推荐升级方式

1. **保持现有配置**：无需任何修改，继续使用 `zltrace.yaml`
2. **迁移到新格式**：
   ```bash
   # 备份旧配置
   cp zltrace.yaml zltrace.yaml.bak

   # 使用新的独立配置
   cp trace.yaml.example trace.yaml

   # 或集成到项目配置
   cp application.yaml.example resource/application.yaml
   ```

### 相关文档

- [配置加载指南](./docs/configuration.md)
- [快速开始](./docs/getting-started.md)
- [API 文档](./docs/api-reference.md)

### 测试

所有测试通过：
```bash
go test ./...
# ok      github.com/zlxdbj/zltrace                    0.744s
# ok      github.com/zlxdbj/zltrace/tracer/kafkagotracer     0.087s
```

### 下一步计划

- [ ] 添加配置热加载支持
- [ ] 支持更多配置格式（TOML、JSON）
- [ ] 添加配置验证命令行工具
- [ ] 支持配置文件加密

### 贡献

欢迎提交 Issue 和 Pull Request！
