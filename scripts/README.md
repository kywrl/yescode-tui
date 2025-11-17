# Git Hooks

这个目录包含了项目的 Git hooks 脚本，用于在提交前自动运行代码质量检查。

## 快速安装

```bash
./scripts/install-hooks.sh
```

## 包含的 Hooks

### pre-commit

在每次 `git commit` 前自动运行以下检查：

1. **代码格式化检查** (`go fmt`)
   - 确保所有代码符合 Go 标准格式
   - 如果发现未格式化的文件，提交将被阻止

2. **静态代码分析** (`go vet`)
   - 检测常见的编程错误
   - 发现问题时提交将被阻止

3. **单元测试** (`go test`)
   - 运行所有测试（带竞态检测）
   - 使用 `-short` 标志跳过耗时测试
   - 测试失败时提交将被阻止

## 工作流程示例

```bash
# 修改代码
vim internal/api/client.go

# 尝试提交
git commit -m "Update client"

# Hook 自动运行检查
🔍 Running pre-commit checks...
📝 Checking code formatting...
✅ Code formatting check passed
🔬 Running go vet...
✅ go vet passed
🧪 Running tests...
✅ All tests passed

✨ All pre-commit checks passed! Ready to commit.

# 提交成功 ✓
```

## 如果检查失败

### 格式化问题

```bash
❌ The following files are not formatted:
internal/api/client.go

💡 Run 'go fmt ./...' to fix formatting issues
```

**修复：**
```bash
go fmt ./...
git add .
git commit -m "Your message"
```

### 静态分析问题

```bash
❌ go vet found issues
internal/api/client.go:123: undefined: someVariable
```

**修复：**
```bash
# 修复代码中的问题
vim internal/api/client.go
git add .
git commit -m "Your message"
```

### 测试失败

```bash
❌ Tests failed
--- FAIL: TestSomething (0.00s)
```

**修复：**
```bash
# 修复测试或代码
vim internal/api/client.go
git add .
git commit -m "Your message"
```

## 绕过 Hooks（紧急情况）

如果需要临时跳过检查（**不推荐**）：

```bash
git commit --no-verify -m "Emergency fix"
```

⚠️ **警告**：这会跳过所有检查，可能导致不符合标准的代码被提交。

## 卸载 Hooks

```bash
rm .git/hooks/pre-commit
```

## 故障排查

### Hook 没有运行

检查权限：
```bash
ls -l .git/hooks/pre-commit
# 应该显示 -rwxr-xr-x（可执行）
```

如果没有执行权限：
```bash
chmod +x .git/hooks/pre-commit
```

### Hook 运行太慢

如果测试耗时太长，可以修改 `.git/hooks/pre-commit`：

```bash
# 将
go test -race -short ./...

# 改为（跳过测试）
# go test -race -short ./...
```

或者只运行快速检查：
```bash
go fmt ./...
go vet ./...
# 不运行测试，留给 CI
```

## 推荐实践

1. **首次克隆后安装**：团队成员首次克隆仓库后应该运行 `./scripts/install-hooks.sh`
2. **保持更新**：hook 脚本更新时重新运行安装脚本
3. **CI 作为最终保障**：即使本地 hook 失败，CI 仍会运行完整检查
4. **快速反馈**：hook 在本地提供即时反馈，节省 CI 时间

## 与 CI 的关系

```
本地开发者 → pre-commit hook → git commit → git push → GitHub Actions CI
              ├─ go fmt         ✓         ✓            ├─ go fmt
              ├─ go vet         ✓         ✓            ├─ go vet
              └─ go test        ✓         ✓            ├─ go test
                                                       └─ go build
```

- **Hook**：第一道防线（快速本地检查）
- **CI**：最终保障（完整自动化检查）
