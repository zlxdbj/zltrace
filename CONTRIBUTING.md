# 贡献指南

感谢你对 zltrace 的关注！我们欢迎任何形式的贡献。

## 开发流程

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'feat: add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 提交 Pull Request

## 代码规范

- 遵循 Go 官方代码风格
- 使用 `gofmt` 格式化代码
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
- `chore:` 构建/工具链相关

### 示例

```bash
git commit -m "feat: 添加对 Echo 框架的支持"
git commit -m "fix: 修复 Kafka 消费者 trace_id 提取问题"
git commit -m "docs: 更新 README 中的配置说明"
```

## 测试

在提交 PR 前，请确保：

```bash
# 运行所有测试
go test ./...

# 运行测试并显示覆盖率
go test -cover ./...

# 代码格式化检查
gofmt -l .

# 静态分析（可选）
go vet ./...
```

## 问题反馈

如果你发现了 bug 或有功能建议，请：

1. 搜索现有的 Issues
2. 如果没有相关问题，创建新的 Issue
3. 提供详细的复现步骤和环境信息

## 行为准则

- 尊重所有贡献者
- 保持友好和专业的沟通
- 关注问题本身，而不是个人

---

再次感谢你的贡献！
