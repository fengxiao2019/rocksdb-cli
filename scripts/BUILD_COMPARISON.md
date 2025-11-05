# 构建方式对比

## 快速参考

| 特性 | 完整版 | 精简版 |
|------|-------|-------|
| **构建命令** | `make build` | `make build-minimal` |
| **Build Tags** | 无 | `-tags=minimal` |
| **Web UI** | ✅ 包含 | ❌ 不包含 |
| **文件大小** | ~57 MB | ~57 MB |
| **编译时间 (Windows)** | 2-5 分钟 | 1-3 分钟 |
| **编译时间 (Linux)** | 30-60 秒 | 20-40 秒 |
| **依赖** | Go + RocksDB + Node.js | Go + RocksDB |

## 使用场景

### 完整版
- ✅ 需要 Web UI 管理界面
- ✅ 团队协作，多人通过浏览器访问
- ✅ 生产环境可视化管理
- ✅ 演示和教学

### 精简版
- ✅ 纯命令行使用
- ✅ CI/CD 环境
- ✅ 快速开发迭代
- ✅ 服务器端脚本自动化
- ✅ MCP Server 模式

## 命令示例

### 完整版
\`\`\`bash
# 构建
make build

# 启动 Web UI
./build/rocksdb-cli web --db mydb --port 8080
\`\`\`

### 精简版
\`\`\`bash
# 构建
make build-minimal

# 使用命令行
./build/rocksdb-cli-minimal repl --db mydb
./build/rocksdb-cli-minimal get --db mydb --cf default key1
\`\`\`
