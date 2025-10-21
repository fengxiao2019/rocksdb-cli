# Transform Feature Development Plan

## 📋 功能概述

为 rocksdb-cli 添加 `transform` 命令，允许用户使用 Python 脚本对 key-value 数据进行转换、过滤和批量处理。

## 🎯 核心需求

### 用户故事
- 作为用户，我想使用 Python 表达式批量修改 value
- 作为用户，我想使用 Python 脚本文件进行复杂的数据转换
- 作为用户，我想在应用更改前预览结果（dry-run 模式）
- 作为用户，我想过滤满足条件的 key-value 对
- 作为用户，我想安全地处理大量数据而不损坏数据库

## 📐 功能设计

### 命令语法

#### 1. 内联表达式模式
```bash
# 基础转换
transform <cf> --expr="value.upper()" [--dry-run]

# JSON 数据处理
transform users --expr="json.loads(value)['age'] = 30; json.dumps(value)"

# 条件过滤
transform users --filter="json.loads(value).get('age', 0) > 25" --expr="value"

# Key 转换
transform logs --key-expr="key.replace(':', '_')" --value-expr="value"
```

#### 2. 脚本文件模式
```bash
# 使用外部 Python 脚本
transform <cf> --script=transform.py [--dry-run]

# 脚本文件格式：
# def transform_key(key: str) -> str:
#     return key
#
# def transform_value(key: str, value: str) -> str:
#     return value
#
# def should_process(key: str, value: str) -> bool:
#     return True
```

#### 3. 交互式 REPL 命令
```
transform users --expr="value.upper()" --limit=10 --dry-run
transform users --script=transform.py
```

### 功能参数

| 参数 | 类型 | 说明 |
|-----|------|------|
| `--expr` | string | Python 表达式（value 转换） |
| `--key-expr` | string | Key 转换表达式 |
| `--value-expr` | string | Value 转换表达式（显式） |
| `--filter` | string | 过滤条件（返回 bool） |
| `--script` | string | Python 脚本文件路径 |
| `--dry-run` | bool | 预览模式，不写入数据库 |
| `--limit` | int | 限制处理的条目数 |
| `--batch-size` | int | 批处理大小（默认 1000） |
| `--backup` | string | 备份文件路径 |
| `--verbose` | bool | 显示详细处理信息 |

## 🏗️ 技术架构

### 模块结构
```
internal/
├── transform/
│   ├── transform.go          # 核心转换逻辑
│   ├── transform_test.go     # 单元测试
│   ├── python_executor.go    # Python 执行器
│   ├── python_executor_test.go
│   ├── batch_processor.go    # 批处理器
│   └── batch_processor_test.go
├── command/
│   └── command.go            # 添加 transform 命令
└── db/
    └── db.go                 # 可能需要添加批量写入支持
```

### 核心组件

#### 1. PythonExecutor
```go
type PythonExecutor interface {
    ExecuteExpression(expr string, context map[string]interface{}) (interface{}, error)
    ExecuteScript(scriptPath string, key string, value string) (string, string, error)
    ValidateExpression(expr string) error
}
```

#### 2. TransformProcessor
```go
type TransformOptions struct {
    Expression      string
    KeyExpression   string
    ValueExpression string
    FilterExpression string
    ScriptPath      string
    DryRun          bool
    Limit           int
    BatchSize       int
    Verbose         bool
}

type TransformResult struct {
    Processed  int
    Modified   int
    Skipped    int
    Errors     []TransformError
    DryRunData []DryRunEntry
}

type TransformProcessor interface {
    Process(cf string, opts TransformOptions) (*TransformResult, error)
}
```

## 🧪 测试驱动开发计划

### Phase 1: 基础测试用例（第1-2天）

#### Test Suite 1: Python Executor 测试
```go
// internal/transform/python_executor_test.go

func TestPythonExecutor_SimpleExpression(t *testing.T) {
    // 测试简单表达式执行
    // Input: "value.upper()", value="hello"
    // Expected: "HELLO"
}

func TestPythonExecutor_JSONParsing(t *testing.T) {
    // 测试 JSON 解析和修改
    // Input: JSON value, 修改某个字段
    // Expected: 修改后的 JSON
}

func TestPythonExecutor_FilterExpression(t *testing.T) {
    // 测试过滤表达式
    // Input: 过滤条件
    // Expected: true/false
}

func TestPythonExecutor_ErrorHandling(t *testing.T) {
    // 测试错误处理
    // Input: 无效的 Python 代码
    // Expected: 明确的错误信息
}

func TestPythonExecutor_ScriptFile(t *testing.T) {
    // 测试脚本文件执行
    // Input: 脚本文件路径
    // Expected: 正确的转换结果
}
```

#### Test Suite 2: Transform Processor 测试
```go
// internal/transform/transform_test.go

func TestTransformProcessor_DryRun(t *testing.T) {
    // 测试 dry-run 模式
    // Expected: 不修改数据库，返回预览结果
}

func TestTransformProcessor_BasicTransform(t *testing.T) {
    // 测试基本转换
    // Input: 简单的转换表达式
    // Expected: 数据被正确修改
}

func TestTransformProcessor_WithFilter(t *testing.T) {
    // 测试带过滤条件的转换
    // Expected: 只处理符合条件的数据
}

func TestTransformProcessor_BatchProcessing(t *testing.T) {
    // 测试批处理
    // Input: 大量数据
    // Expected: 正确的批量处理和进度报告
}

func TestTransformProcessor_ErrorRecovery(t *testing.T) {
    // 测试错误恢复
    // Input: 部分数据处理失败
    // Expected: 继续处理其他数据，记录错误
}

func TestTransformProcessor_Limit(t *testing.T) {
    // 测试 limit 参数
    // Expected: 只处理指定数量的数据
}
```

#### Test Suite 3: Command Integration 测试
```go
// internal/command/command_test.go

func TestCommand_Transform_Syntax(t *testing.T) {
    // 测试命令语法解析
}

func TestCommand_Transform_Execution(t *testing.T) {
    // 测试完整的命令执行流程
}

func TestCommand_Transform_Help(t *testing.T) {
    // 测试帮助信息显示
}
```

### Phase 2: 核心实现（第3-5天）

#### 步骤 1: 实现 Python Executor
- 选择 Python 集成方案（推荐：`github.com/go-python/gopy` 或执行外部 Python 进程）
- 实现表达式执行
- 实现脚本文件支持
- 添加错误处理和安全限制

#### 步骤 2: 实现 Transform Processor
- 实现批处理逻辑
- 添加 dry-run 支持
- 实现进度报告
- 添加错误处理和回滚机制

#### 步骤 3: 集成到 Command Handler
- 在 `command.go` 中添加 `transform` 命令
- 解析命令参数
- 调用 TransformProcessor
- 格式化输出结果

### Phase 3: 高级功能（第6-7天）

#### Test Suite 4: 高级特性测试
```go
func TestTransform_Backup(t *testing.T) {
    // 测试备份功能
}

func TestTransform_LargeDataset(t *testing.T) {
    // 测试大数据集性能
}

func TestTransform_ConcurrentSafety(t *testing.T) {
    // 测试并发安全性
}

func TestTransform_MemoryUsage(t *testing.T) {
    // 测试内存使用
}
```

#### 实现高级功能
- 备份功能
- 性能优化
- 内存管理
- 并发控制

### Phase 4: 文档和示例（第8天）

- 更新 README.md
- 创建使用示例
- 编写最佳实践指南
- 添加故障排除文档

## 🔧 技术选型

### Python 集成方案对比

#### 方案 1: 外部 Python 进程（推荐）
**优点**:
- ✅ 隔离性好，安全
- ✅ 容易实现
- ✅ 支持任何 Python 版本
- ✅ 不增加二进制大小

**缺点**:
- ❌ 需要系统安装 Python
- ❌ 进程间通信开销
- ❌ 启动较慢

**实现方式**:
```go
cmd := exec.Command("python3", "-c", expression)
cmd.Stdin = strings.NewReader(input)
output, err := cmd.Output()
```

#### 方案 2: go-python/gopy
**优点**:
- ✅ 嵌入式，无需外部 Python
- ✅ 性能较好

**缺点**:
- ❌ 编译复杂
- ❌ 跨平台问题
- ❌ 增加二进制大小

#### 方案 3: Starlark (Python 子集)
**优点**:
- ✅ 纯 Go 实现
- ✅ 安全沙箱
- ✅ 语法类似 Python

**缺点**:
- ❌ 不是完整的 Python
- ❌ 生态有限

**推荐**: 方案 1（外部进程）用于 MVP，方案 3（Starlark）用于生产环境

## 📊 成功指标

### 功能性指标
- ✅ 所有测试用例通过
- ✅ 支持常见的转换场景
- ✅ Dry-run 模式正确预览
- ✅ 错误处理完善

### 性能指标
- ✅ 每秒处理 > 1000 条记录（小数据）
- ✅ 每秒处理 > 100 条记录（JSON 解析）
- ✅ 内存使用 < 100MB（批处理 10k 记录）

### 用户体验指标
- ✅ 命令语法简洁直观
- ✅ 错误信息清晰易懂
- ✅ 进度显示实时准确
- ✅ 文档完整易用

## 🚀 开发里程碑

### Milestone 1: MVP (3天)
- [ ] Python executor 基础实现
- [ ] 简单表达式转换
- [ ] Dry-run 模式
- [ ] 基础测试覆盖

### Milestone 2: 完整功能 (5天)
- [ ] 脚本文件支持
- [ ] 过滤功能
- [ ] 批处理优化
- [ ] 完整测试覆盖

### Milestone 3: 生产就绪 (7天)
- [ ] 备份功能
- [ ] 性能优化
- [ ] 完整文档
- [ ] 集成测试

## 🔒 安全考虑

### 沙箱限制
- 限制文件系统访问
- 限制网络访问
- 限制执行时间（超时机制）
- 限制内存使用

### 数据安全
- Dry-run 默认开启
- 自动备份选项
- 事务支持（如可能）
- 详细的审计日志

## 📝 文档结构

```
docs/
├── TRANSFORM_FEATURE_PLAN.md   # 本文档
├── TRANSFORM_USER_GUIDE.md     # 用户指南
├── TRANSFORM_API.md             # API 文档
└── TRANSFORM_EXAMPLES.md        # 示例集合
```

## 🎯 下一步行动

1. **Review 计划** - 确认需求和技术选型
2. **创建测试文件** - 编写失败的测试用例
3. **实现 MVP** - 让测试通过
4. **迭代优化** - 添加更多功能和测试
5. **文档和发布** - 完善文档，准备发布
