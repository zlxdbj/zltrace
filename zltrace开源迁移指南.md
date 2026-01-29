# zltrace 开源迁移指南

## 概述

将 zltrace 从 go_shield 项目中独立出来，作为一个开源项目发布。

## ⚠️ 关键注意事项

### 1. 依赖关系处理

zltrace 当前依赖 `github.com/zlxdbj/zllog`，有以下选择：

**方案A：保持依赖 zllog（推荐）**
```go
// 独立后仍然依赖 zllog
import "github.com/zlxdbj/zllog"

// 优点：功能完整，与 zllog 无缝集成
// 缺点：需要 zllog 也开源
```

**方案B：可选依赖 zllog**
```go
// 使用接口解耦，zllog 作为可选依赖
type Logger interface {
    Info(ctx context.Context, module, msg string, args ...Field)
    Error(ctx context.Context, module, msg string, err error, args ...Field)
}

// 优点：更灵活，可以独立使用
// 缺点：需要额外设计
```

**方案C：使用标准库 log**
```go
// 完全移除 zllog 依赖
import "log"

// 优点：完全独立，无外部依赖
// 缺点：功能受限，失去上下文传递能力
```

### 2. 配置文件加载路径

当前代码从 `resource/` 目录读取配置：
```go
// 当前（在 go_shield 项目中）
configPaths := []string{
    "resource/application.yaml",
    "resource/application_" + mode + ".yaml",
}

// 独立后需要改为：
configPaths := []string{
    "./zltrace.yaml",                    // 当前目录
    os.Getenv("ZLTRACE_CONFIG"),         // 环境变量
    "/etc/zltrace/config.yaml",          // 系统配置目录
}
```

### 3. 导入路径替换

需要全局替换导入路径：
```bash
# 当前
go_shield/zltrace

# 替换为（假设使用 GitHub）
github.com/zhonglinxinda/zltrace

# 或使用其他 Git 托管平台
github.com/zlxdbj/zltrace
gitlab.com/zhonglinxinda/zltrace
```

## 📋 迁移步骤

### 步骤1：创建独立仓库

```bash
# 1. 在 GitHub/GitLab 创建新仓库 zltrace

# 2. 克隆到本地
git clone git@github.com:zhonglinxinda/zltrace.git
cd zltrace

# 3. 创建基础目录结构
mkdir -p .github/workflows
mkdir -p _examples/{http,kafka}
mkdir -p tracer/{httptrace,saramatrace}
mkdir -p adapter/httpadapter
```

### 步骤2：拷贝代码

```bash
# 从 go_shield 项目拷贝代码
cp -r /path/to/go_shield/zltrace/* .

# 查看拷贝的文件
ls -la
# 应该看到：
# - README.md
# - config.go
# - init.go
# - opentelemetry.go
# - tracer.go
# - tracer/
# - adapter/
```

### 步骤3：创建 go.mod

```bash
# 初始化 go module
cat > go.mod <<'EOF'
module github.com/zhonglinxinda/zltrace

go 1.19

require (
    go.opentelemetry.io/otel v1.39.0
    go.opentelemetry.io/otel/trace v1.39.0
    go.opentelemetry.io/otel/sdk v1.39.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.39.0
    go.opentelemetry.io/otel/semconv/v1.24.0
    github.com/IBM/sarama v1.40.1
    github.com/zlxdbj/zllog v1.0.0  # 如果保持 zllog 依赖
)
EOF

# 整理依赖
go mod tidy
```

### 步骤4：全局替换导入路径

```bash
# 替换所有 Go 文件中的导入路径
find . -name "*.go" -type f -exec sed -i 's|go_shield/zltrace|github.com/zhonglinxinda/zltrace|g' {} +

# 验证替换结果
grep -r "go_shield/zltrace" .
# 应该没有输出（除了 README.md 中的示例）
```

### 步骤5：修改 README.md

```bash
# 替换 README.md 中的导入路径示例
sed -i 's|go_shield/zltrace|github.com/zhonglinxinda/zltrace|g' README.md
sed -i 's|go_shield/zllog|github.com/zlxdbj/zllog|g' README.md
```

### 步骤6：修改配置加载逻辑

编辑 `config.go`：
```go
// LoadConfig 加载配置
func LoadConfig() (*Config, error) {
    // 尝试多个配置文件路径
    configPaths := []string{
        "./zltrace.yaml",                    // 当前目录
        os.Getenv("ZLTRACE_CONFIG"),         // 环境变量
        "/etc/zltrace/config.yaml",          // 系统配置目录
    }

    for _, path := range configPaths {
        if path == "" {
            continue
        }

        if _, err := os.Stat(path); err == nil {
            return loadConfigFromFile(path)
        }
    }

    // 没找到配置文件，返回默认配置
    return DefaultConfig(), nil
}
```

### 步骤7：创建配置文件示例

```bash
cat > zltrace.yaml.example <<'EOF'
# zltrace 配置文件示例
# 复制此文件为 zltrace.yaml 并根据需要修改

trace:
  enabled: true
  service_name: my_service

  exporter:
    type: stdout  # otlp | stdout | none

    otlp:
      endpoint: localhost:4317
      timeout: 10

  sampler:
    type: always_on  # always_on | never | traceid_ratio | parent_based
    ratio: 1.0
EOF
```

### 步骤8：添加测试文件

```bash
# 创建基础测试
cat > tracer_test.go <<'EOF'
package zltrace

import (
    "context"
    "testing"
)

func TestRegisterTracer(t *testing.T) {
    mockTracer := &mockTracer{}
    RegisterTracer(mockTracer)

    if GetTracer() != mockTracer {
        t.Error("failed to register tracer")
    }
}

func TestGetSafeTracer(t *testing.T) {
    // 清空全局 tracer
    RegisterTracer(nil)

    tracer := GetSafeTracer()
    if tracer == nil {
        t.Error("GetSafeTracer should never return nil")
    }
}

// mock 实现省略...
EOF

# 为每个主要包添加测试
go test ./...
```

### 步骤9：创建 LICENSE

```bash
cat > LICENSE <<'EOF'
MIT License

Copyright (c) 2025 中林信达

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
EOF
```

### 步骤10：创建贡献指南

```bash
cat > CONTRIBUTING.md <<'EOF'
# 贡献指南

感谢你对 zltrace 的关注！

## 开发流程

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'feat: add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 提交 Pull Request

## 代码规范

- 遵循 Go 官方代码风格
- 添加必要的注释和文档
- 确保所有测试通过 (`go test ./...`)
- 更新相关文档

## 提交信息规范

使用 Conventional Commits 规范：
- `feat:` 新功能
- `fix:` 修复bug
- `docs:` 文档更新
- `test:` 测试相关
- `refactor:` 重构代码
EOF
```

### 步骤11：添加 CI/CD

```bash
cat > .github/workflows/ci.yml <<'EOF'
name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest

    strategy:
      matrix:
        go-version: ['1.19', '1.20', '1.21']

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: ${{ matrix.go-version }}

      - name: Install dependencies
        run: go mod download

      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: coverage.out

  lint:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v3
        with:
          version: latest
EOF
```

### 步骤12：创建示例代码

```bash
# HTTP 示例
cat > _examples/http/simple.go <<'EOF'
package main

import (
    "context"
    "net/http"

    "github.com/zhonglinxinda/zltrace"
    "github.com/zhonglinxinda/zltrace/tracer/httptrace"
)

func main() {
    // 初始化追踪系统
    if err := zltrace.InitTracer(); err != nil {
        panic(err)
    }
    defer zltrace.GetTracer().Close()

    // 创建 HTTP 服务
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        zllog.Info(r.Context(), "example", "Hello, World!")
        w.Write([]byte("Hello, World!"))
    })

    // 添加追踪中间件
    http.ListenAndServe(":8080", httptrace.TraceMiddleware(mux))
}
EOF

# Kafka 示例（略）
```

## ✅ 验证清单

迁移完成后，检查以下项目：

- [ ] 所有导入路径已替换
- [ ] go.mod 文件正确
- [ ] 可以执行 `go mod tidy`
- [ ] 可以执行 `go build ./...`
- [ ] 可以执行 `go test ./...`
- [ ] README.md 中的示例代码可以运行
- [ ] LICENSE 文件存在
- [ ] CONTRIBUTING.md 文件存在
- [ ] CI/CD 配置存在
- [ ] 示例代码可以运行

## 📦 发布流程

```bash
# 1. 提交代码
git add .
git commit -m "feat: 初始化 zltrace 开源项目"
git push origin main

# 2. 打标签（语义化版本）
git tag -a v1.0.0 -m "第一个稳定版本"
git push origin v1.0.0

# 3. 在 GitHub 创建 Release
# 上传编译好的二进制文件（可选）

# 4. 推送到 Go Package Registry
# 使用 go get 命令测试安装
go get github.com/zhonglinxinda/zltrace@v1.0.0
```

## 🔧 后续维护

1. **版本管理**：遵循语义化版本规范
2. **发布周期**：建议每月发布一个小版本
3. **问题追踪**：及时处理 GitHub Issues
4. **文档更新**：每次功能更新都要更新文档

## 📝 注意事项

1. **配置兼容性**：考虑提供配置迁移工具，帮助用户从 go_shield 内置版本迁移到独立版本
2. **API 稳定性**：发布 v1.0 后，保持 API 向后兼容
3. **依赖管理**：定期更新依赖，修复安全漏洞
4. **性能优化**：持续监控性能，优化开销

## 🎯 预期收益

独立开源后，zltrace 将获得：
- ✅ 更广泛的用户群体
- ✅ 社区贡献和反馈
- ✅ 更高的代码质量
- ✅ 更好的可维护性
- ✅ 独立的发展路线

---

**祝迁移顺利！🎉**
