# MCP 客户端功能实现计划

## 目标

为 rocksdb-cli 工具添加 MCP 客户端功能，使其能够：
1. 作为 MCP 服务器（已实现）
2. 同时作为 MCP 客户端连接其他 MCP 服务器（新功能）

## 变更后的项目结构

```
rocksdb-cli/
├── cmd/
│   ├── root.go                    # (修改) 添加 MCP 相关配置加载
│   ├── ai.go                      # (修改) 集成 MCP 客户端工具
│   ├── mcp.go                     # (新增) MCP 管理命令
│   └── mcp-server/
│       └── main.go                # (现有) MCP 服务器入口
│
├── internal/
│   ├── config/
│   │   ├── config.go              # (新增) 统一配置管理
│   │   ├── mcp_client.go          # (新增) MCP 客户端配置
│   │   └── mcp_server.go          # (移动) 从 mcp/config.go
│   │
│   ├── mcp/
│   │   ├── client/                # (新增) MCP 客户端模块
│   │   │   ├── client.go          # MCP 客户端接口
│   │   │   ├── stdio_client.go    # STDIO 传输客户端
│   │   │   ├── tcp_client.go      # TCP 传输客户端
│   │   │   ├── manager.go         # 客户端管理器
│   │   │   ├── tool_proxy.go      # 远程工具代理
│   │   │   └── client_test.go     # 测试
│   │   │
│   │   ├── server/                # (重构) MCP 服务器模块
│   │   │   ├── server.go          # (移动) 从 mcp/*.go
│   │   │   ├── transport.go       # (移动) 传输层
│   │   │   ├── tools.go           # (移动) 工具定义
│   │   │   └── prompts.go         # (移动) 提示定义
│   │   │
│   │   ├── registry/              # (新增) 工具注册中心
│   │   │   ├── registry.go        # 统一工具注册表
│   │   │   ├── local_tools.go     # 本地工具注册
│   │   │   ├── remote_tools.go    # 远程工具注册
│   │   │   └── registry_test.go   # 测试
│   │   │
│   │   └── protocol/              # (新增) MCP 协议定义
│   │       ├── types.go           # 协议类型
│   │       ├── messages.go        # 消息格式
│   │       └── errors.go          # 错误定义
│   │
│   ├── graphchain/
│   │   ├── agent.go               # (修改) 集成工具注册中心
│   │   ├── tools.go               # (修改) 使用统一工具接口
│   │   └── tools_test.go          # (修改) 更新测试
│   │
│   └── service/
│       └── ...                    # (现有) 服务层
│
├── config/
│   ├── rocksdb-cli.example.yaml  # (新增) CLI 完整配置示例
│   ├── mcp-server.example.yaml   # (现有) MCP 服务器配置
│   └── mcp-clients.example.yaml  # (新增) MCP 客户端配置
│
├── docs/
│   ├── MCP_CLIENT_GUIDE.md        # (新增) 客户端使用指南
│   ├── MCP_SERVER_README.md       # (现有) 服务器文档
│   ├── MCP_CLIENT_IMPLEMENTATION_PLAN.md  # (本文件)
│   └── ARCHITECTURE.md            # (新增) 架构设计文档
│
└── examples/
    ├── mcp-client/                # (新增) 客户端示例
    │   ├── basic_usage.go
    │   └── multi_server.go
    └── integration/               # (新增) 集成示例
        └── ai_with_mcp.go
```

## 详细任务拆解

### Phase 1: 基础架构 (优先级: 高)

#### Task 1.1: 配置管理重构
- [ ] 创建 `internal/config/config.go` - 统一配置管理
- [ ] 创建 `internal/config/mcp_client.go` - MCP 客户端配置
- [ ] 移动 `internal/mcp/config.go` → `internal/config/mcp_server.go`
- [ ] 添加配置文件示例 `config/rocksdb-cli.example.yaml`
- [ ] 编写测试 `internal/config/config_test.go`

#### Task 1.2: MCP 协议定义
- [ ] 创建 `internal/mcp/protocol/types.go` - JSON-RPC 类型定义
- [ ] 创建 `internal/mcp/protocol/messages.go` - MCP 消息格式
- [ ] 创建 `internal/mcp/protocol/errors.go` - 错误码定义
- [ ] 编写测试 `internal/mcp/protocol/protocol_test.go`

#### Task 1.3: 项目结构重组
- [ ] 创建 `internal/mcp/server/` 目录
- [ ] 移动现有 MCP 服务器代码到 `server/` 子目录
- [ ] 更新所有导入路径
- [ ] 确保所有测试通过

### Phase 2: MCP 客户端核心 (优先级: 高)

#### Task 2.1: 客户端接口设计 (TDD)
接口定义：
```go
type MCPClient interface {
    // 连接管理
    Connect(ctx context.Context) error
    Disconnect() error
    IsConnected() bool

    // 工具调用
    ListTools(ctx context.Context) ([]Tool, error)
    CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error)

    // 提示和资源
    ListPrompts(ctx context.Context) ([]Prompt, error)
    ListResources(ctx context.Context) ([]Resource, error)
}
```
- [ ] 编写接口测试 `client_test.go` (TDD - RED)
- [ ] 实现 `client.go` 基础结构
- [ ] 实现 Mock 客户端用于测试

#### Task 2.2: STDIO 客户端实现 (TDD)
- [ ] 编写 STDIO 客户端测试 `stdio_client_test.go` (RED)
- [ ] 实现 `stdio_client.go` - 启动子进程、JSON-RPC 通信
- [ ] 测试通过 (GREEN)
- [ ] 重构优化 (REFACTOR)

#### Task 2.3: TCP 客户端实现 (TDD)
- [ ] 编写 TCP 客户端测试 `tcp_client_test.go` (RED)
- [ ] 实现 `tcp_client.go` - TCP 连接、消息处理
- [ ] 测试通过 (GREEN)
- [ ] 添加重连机制

#### Task 2.4: 客户端管理器 (TDD)
- [ ] 编写管理器测试 `manager_test.go` (RED)
- [ ] 实现 `manager.go` - 管理多个客户端连接
- [ ] 实现生命周期管理（启动、停止、健康检查）
- [ ] 测试通过 (GREEN)

### Phase 3: 工具注册中心 (优先级: 高)

#### Task 3.1: 注册中心接口 (TDD)
接口定义：
```go
type ToolRegistry interface {
    // 工具注册
    RegisterLocal(tool *LocalTool) error
    RegisterRemote(serverName string, tool *RemoteTool) error

    // 工具查询
    ListAll() []Tool
    ListByServer(serverName string) []Tool
    GetTool(name string) (Tool, error)

    // 工具调用
    CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error)
}
```
- [ ] 编写注册中心测试 `registry_test.go` (RED)
- [ ] 实现 `registry.go` 核心逻辑
- [ ] 测试通过 (GREEN)

#### Task 3.2: 本地工具适配器
- [ ] 编写测试 `local_tools_test.go` (RED)
- [ ] 实现 `local_tools.go` - 将现有工具包装为统一接口
- [ ] 测试通过 (GREEN)

#### Task 3.3: 远程工具代理 (TDD)
- [ ] 编写测试 `remote_tools_test.go` (RED)
- [ ] 实现 `tool_proxy.go` - 远程工具调用代理
- [ ] 添加错误处理和重试机制
- [ ] 测试通过 (GREEN)

### Phase 4: GraphChain 集成 (优先级: 中)

#### Task 4.1: GraphChain 工具集成
- [ ] 修改 `internal/graphchain/agent.go` - 使用工具注册中心
- [ ] 更新 `internal/graphchain/tools.go` - 统一工具接口
- [ ] 更新所有测试确保兼容性
- [ ] 添加集成测试

#### Task 4.2: AI 命令增强
- [ ] 修改 `cmd/ai.go` - 初始化 MCP 客户端
- [ ] 在 AI 会话启动时连接所有配置的 MCP 服务器
- [ ] 展示可用工具列表
- [ ] 测试端到端流程

### Phase 5: CLI 命令 (优先级: 中)

#### Task 5.1: MCP 管理命令
- [ ] 创建 `cmd/mcp.go` - 主命令
- [ ] 实现 `mcp list` - 列出所有 MCP 服务器
- [ ] 实现 `mcp connect <name>` - 测试连接
- [ ] 实现 `mcp tools <name>` - 列出工具
- [ ] 实现 `mcp call <server> <tool> [args...]` - 调用工具

#### Task 5.2: 配置命令
- [ ] 实现 `mcp config show` - 显示当前配置
- [ ] 实现 `mcp config add <name>` - 添加 MCP 服务器
- [ ] 实现 `mcp config remove <name>` - 移除服务器
- [ ] 实现 `mcp config enable/disable <name>` - 启用/禁用

### Phase 6: 文档和示例 (优先级: 低)

#### Task 6.1: 文档编写
- [ ] 创建 `docs/MCP_CLIENT_GUIDE.md` - 客户端使用指南
- [ ] 创建 `docs/ARCHITECTURE.md` - 架构设计文档
- [ ] 更新 `README.md` - 添加 MCP 客户端功能说明
- [ ] 添加配置示例到 `docs_site/`

#### Task 6.2: 示例代码
- [ ] 创建基础使用示例 `examples/mcp-client/basic_usage.go`
- [ ] 创建多服务器示例 `examples/mcp-client/multi_server.go`
- [ ] 创建集成示例 `examples/integration/ai_with_mcp.go`

## 采用的设计模式

### 1. 适配器模式 (Adapter Pattern)
**目的**: 统一本地工具和远程 MCP 工具的接口

```
┌─────────────────┐
│  Tool Interface │  ← 统一接口
└─────────────────┘
         ↑
    ┌────┴────┐
    │         │
┌───┴───┐ ┌──┴────┐
│ Local │ │Remote │
│ Tool  │ │ Tool  │
│Adapter│ │ Proxy │
└───────┘ └───────┘
```

**实现**:
```go
// 统一工具接口
type Tool interface {
    Name() string
    Description() string
    Schema() map[string]interface{}
    Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
}

// 本地工具适配器
type LocalToolAdapter struct {
    tool *graphchain.Tool
}

// 远程工具代理
type RemoteToolProxy struct {
    client     MCPClient
    serverName string
    toolInfo   ToolInfo
}
```

### 2. 注册中心模式 (Registry Pattern)
**目的**: 集中管理所有可用工具

```
┌──────────────────┐
│  Tool Registry   │
│                  │
│  ┌────────────┐  │
│  │Local Tools │  │
│  ├────────────┤  │
│  │Remote:fs   │  │
│  │Remote:gh   │  │
│  │Remote:pg   │  │
│  └────────────┘  │
└──────────────────┘
```

### 3. 代理模式 (Proxy Pattern)
**目的**: 透明地调用远程 MCP 工具

```
Client → ToolProxy → MCPClient → Remote Server
                ↓
          Cache & Retry
```

### 4. 工厂模式 (Factory Pattern)
**目的**: 根据配置创建不同类型的 MCP 客户端

```
┌──────────────┐
│ClientFactory │
└──────────────┘
       │
       ├── CreateSTDIOClient()
       ├── CreateTCPClient()
       └── CreateWebSocketClient()
```

### 5. 观察者模式 (Observer Pattern)
**目的**: 监控 MCP 客户端连接状态

```
ClientManager (Subject)
    ↓
    ├─→ Observer 1: Logger
    ├─→ Observer 2: Metrics
    └─→ Observer 3: Alerting
```

## 关键技术决策

| 决策点 | 选择 | 理由 |
|--------|------|------|
| **配置格式** | YAML | 已有配置使用 YAML，保持一致性 |
| **客户端通信** | JSON-RPC 2.0 | MCP 标准协议 |
| **并发模型** | Goroutine per client | Go 原生支持，简单高效 |
| **工具调用** | 同步调用 + Context | 便于超时控制和取消 |
| **错误处理** | 包装错误 + 重试 | 提高可靠性 |
| **测试策略** | TDD + 集成测试 | 保证质量 |

## 配置文件示例

```yaml
# config/rocksdb-cli.yaml
name: "RocksDB CLI"
version: "1.0.0"

# RocksDB 数据库配置
database:
  path: "./data/rocksdb"
  read_only: false

# MCP 服务器配置（当前工具作为 MCP 服务器）
mcp_server:
  enabled: true
  name: "RocksDB MCP Server"
  transport:
    type: "stdio"
    timeout: 30s

# MCP 客户端配置（连接其他 MCP 服务器）
mcp_clients:
  filesystem:
    enabled: true
    transport: "stdio"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/Users/username/Documents"]
    timeout: 30s
    retry:
      max_attempts: 3
      backoff: "exponential"

  github:
    enabled: true
    transport: "stdio"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_PERSONAL_ACCESS_TOKEN: "${GITHUB_TOKEN}"
    timeout: 60s

  postgres:
    enabled: false
    transport: "tcp"
    host: "localhost"
    port: 5432
    timeout: 30s

# GraphChain AI 配置
graphchain:
  model: "gpt-4"
  enable_mcp_tools: true  # 启用 MCP 工具集成
  auto_connect: true      # 启动时自动连接所有 MCP 客户端
```

## 开发时间表

### Week 1: 基础架构
- Day 1-2: Task 1.1-1.2 (配置管理 + 协议定义)
- Day 3-5: Task 1.3 + Task 2.1 (结构重组 + 客户端接口)

### Week 2: 客户端核心
- Day 1-3: Task 2.2 (STDIO 客户端)
- Day 4-5: Task 2.3 (TCP 客户端)

### Week 3: 工具注册
- Day 1-2: Task 2.4 (客户端管理器)
- Day 3-5: Task 3.1-3.3 (工具注册中心)

### Week 4: 集成和完善
- Day 1-3: Task 4.1-4.2 (GraphChain 集成)
- Day 4-5: Task 5.1 (CLI 命令)

### Week 5: 文档和测试
- Day 1-3: 全面测试和 Bug 修复
- Day 4-5: Task 6.1-6.2 (文档和示例)

## 里程碑

- ✅ **Milestone 1**: 基础架构完成 (Week 1)
- ✅ **Milestone 2**: MCP 客户端可用 (Week 2)
- ✅ **Milestone 3**: 工具注册中心可用 (Week 3)
- ✅ **Milestone 4**: GraphChain 集成完成 (Week 4)
- ✅ **Milestone 5**: 完整功能发布 (Week 5)

## 成功标准

1. **功能完整性**
   - [ ] 可以连接至少 3 种不同的 MCP 服务器（filesystem, github, postgres）
   - [ ] AI 助手可以无缝使用本地和远程工具
   - [ ] CLI 命令可以管理 MCP 连接

2. **质量标准**
   - [ ] 测试覆盖率 > 80%
   - [ ] 所有 TDD 测试通过
   - [ ] 集成测试通过

3. **性能标准**
   - [ ] 工具调用延迟 < 100ms (本地)
   - [ ] 支持并发调用 > 10
   - [ ] 内存使用合理

4. **文档完整性**
   - [ ] API 文档完整
   - [ ] 使用示例齐全
   - [ ] 故障排查指南

## 风险和缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| MCP 协议变更 | 高 | 使用稳定版本，添加版本检测 |
| 客户端连接不稳定 | 中 | 实现重试和健康检查机制 |
| 工具冲突 | 中 | 工具命名空间隔离 |
| 性能问题 | 低 | 添加缓存和连接池 |

## 下一步行动

立即开始 **Phase 1: Task 1.1** - 创建统一配置管理系统。
