# Windows 构建指南

本文档说明如何使用 GitHub Actions 自动构建 Windows 版本。

## 🚀 快速开始

### 自动触发构建

GitHub Actions 会在以下情况自动触发 Windows 构建：

1. **推送到 main/develop 分支**
   ```bash
   git push origin main
   ```

2. **创建 Pull Request**
   ```bash
   # PR 合并前会自动构建测试
   ```

3. **推送版本标签**（会创建 Release）
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

4. **手动触发**
   - 进入 GitHub 仓库
   - 点击 **Actions** 标签
   - 选择 **Build Windows** workflow
   - 点击 **Run workflow** 按钮

## 📥 下载构建产物

### 方法 1: 从 Actions 下载

1. 进入 GitHub 仓库
2. 点击 **Actions** 标签
3. 选择最新的 **Build Windows** workflow run
4. 在页面底部 **Artifacts** 区域找到：
   - `rocksdb-cli-windows-amd64` - 单个可执行文件
   - `rocksdb-cli-windows-package` - 完整包（含文档和校验和）

### 方法 2: 从 Releases 下载（推荐）

如果是通过 tag 触发的构建：

1. 进入 GitHub 仓库
2. 点击 **Releases** 标签
3. 下载对应版本的 `rocksdb-cli-windows-amd64.zip`

## ⚙️ 构建配置说明

### 触发条件

```yaml
on:
  push:
    branches: [ main, develop ]  # 推送到这些分支
    tags: [ 'v*' ]               # 推送 v* 标签（如 v1.0.0）
  pull_request:
    branches: [ main ]           # 针对 main 的 PR
  workflow_dispatch:             # 手动触发
```

### 构建环境

- **Runner**: `windows-latest` (Windows Server 2022)
- **Go 版本**: 1.24
- **架构**: amd64 (64位)
- **依赖管理**: vcpkg

### 缓存策略

为了加速构建，配置了以下缓存：

1. **Go 模块缓存** - 自动（setup-go action）
2. **vcpkg 包缓存** - 手动配置
   - 首次构建: ~8-12 分钟
   - 有缓存: ~3-5 分钟

## 🔧 本地测试

如果想在本地测试 GitHub Actions workflow：

### 使用 act (本地运行 GitHub Actions)

```powershell
# 安装 act (需要 Docker)
choco install act-cli

# 运行 Windows workflow
act -W .github/workflows/build-windows.yml
```

### 手动模拟构建步骤

```powershell
# 1. 安装 vcpkg
git clone https://github.com/Microsoft/vcpkg.git C:\vcpkg
C:\vcpkg\bootstrap-vcpkg.bat

# 2. 安装依赖
C:\vcpkg\vcpkg install rocksdb:x64-windows

# 3. 设置环境变量
$env:CGO_ENABLED = "1"
$env:CGO_CFLAGS = "-IC:/vcpkg/installed/x64-windows/include"
$env:CGO_LDFLAGS = "-LC:/vcpkg/installed/x64-windows/lib -lrocksdb"

# 4. 构建
go build -o rocksdb-cli.exe ./cmd
```

## 📊 构建时间

| 场景 | 时间 |
|------|------|
| 首次构建（无缓存） | 8-12 分钟 |
| 有缓存（vcpkg） | 3-5 分钟 |
| 仅代码变更 | 2-3 分钟 |

## 🐛 故障排查

### 构建失败

1. **检查 Actions 日志**
   - Actions > 失败的 workflow > 查看详细日志

2. **常见问题**
   - vcpkg 安装超时 → 重新运行 workflow
   - CGO 链接错误 → 检查依赖是否正确安装
   - Go 版本不兼容 → 更新 `go-version` 配置

### 缓存问题

如果缓存导致问题：

1. 进入 Actions 页面
2. 点击 **Caches**
3. 删除问题缓存
4. 重新运行 workflow

## 📝 版本管理

### 开发版本

推送到 main/develop 分支：
- 版本号: `dev-{git-short-hash}`
- 例如: `dev-a1b2c3d`

### 正式版本

推送版本标签：
```bash
git tag v1.0.0
git push origin v1.0.0
```

- 版本号: 标签名（如 `v1.0.0`）
- 自动创建 GitHub Release
- 上传构建产物到 Release

## 🔐 安全性

### 校验和验证

每次构建都会生成 SHA256 校验和：

```powershell
# 验证下载的文件
Get-FileHash rocksdb-cli-windows-amd64.zip -Algorithm SHA256

# 对比 checksums.txt 中的值
```

### 构建环境

- GitHub-hosted runner（隔离环境）
- 每次构建都是全新环境
- 无访问仓库 secrets 的权限（除非配置）

## 🎯 优化建议

### 进一步加速构建

1. **使用自托管 runner**
   - 保留 vcpkg 安装
   - 可节省 5-8 分钟

2. **使用预构建的 Docker 镜像**
   - 包含所有依赖
   - 更快速和可重复

3. **并行构建多个目标**
   ```yaml
   strategy:
     matrix:
       goarch: [amd64, arm64]
   ```

## 📞 支持

如有问题，请：
1. 查看 Actions 日志
2. 查看 [BUILD.md](../../BUILD.md)
3. 提交 Issue
