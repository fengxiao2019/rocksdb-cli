# Task 03: 命令处理架构重构

**优先级**: 🔴 高优先级  
**状态**: ⏳ 待开始  
**预估工作量**: 5-6小时  
**负责人**: 待分配  

## 问题描述

当前`Handler.Execute`方法过长（597行），包含所有命令逻辑，导致：

### 代码结构问题
1. **单一方法过长**:
   ```go
   func (h *Handler) Execute(input string) bool {
       // 巨大的switch语句，597行代码
       switch cmd {
       case "usecf":   // 20行代码
       case "scan":    // 130行代码  
       case "get":     // 50行代码
       // ... 更多命令
       }
   }
   ```

2. **职责不清晰**:
   - 命令解析与执行混合
   - 参数验证分散在各个case中
   - 错误处理重复代码多

3. **扩展性差**:
   - 添加新命令需要修改核心Execute方法
   - 命令间耦合度高
   - 难以进行单元测试

## 影响分析

- **可维护性**: 单个文件过大，难以维护
- **可测试性**: 难以对单个命令进行独立测试
- **可扩展性**: 添加新命令需要修改核心逻辑
- **代码复用**: 命令间通用逻辑重复

## 解决方案

### 1. 命令模式架构
```
internal/command/
├── command.go          # 命令接口定义
├── handler.go          # 命令处理器和路由
├── registry.go         # 命令注册器
├── context.go          # 命令上下文
├── commands/           # 具体命令实现
│   ├── get.go         
│   ├── put.go
│   ├── scan.go
│   ├── usecf.go
│   ├── export.go
│   └── ...
└── utils/              # 命令工具类
    ├── parser.go       # 参数解析器
    ├── validator.go    # 参数验证器
    └── formatter.go    # 输出格式化器
```

### 2. 命令接口设计
```go
type Command interface {
    Name() string
    Description() string
    Usage() string
    Execute(ctx *Context) error
    Validate(args []string) error
}
```

### 3. 命令上下文
```go
type Context struct {
    DB      db.KeyValueDB
    State   *ReplState
    Args    []string
    Flags   map[string]string
    Output  io.Writer
    Logger  logger.Logger
}
```

## 实施步骤

### Phase 1: 架构设计 (2小时)
1. 定义Command接口
2. 创建命令上下文Context
3. 实现命令注册器Registry
4. 设计参数解析和验证框架

### Phase 2: 核心命令迁移 (2.5小时)
1. 迁移get、put、scan命令
2. 迁移CF相关命令（usecf、createcf、dropcf）
3. 迁移辅助命令（help、listcf）
4. 更新命令处理器

### Phase 3: 高级命令迁移 (1.5小时)
1. 迁移export、last、jsonquery命令
2. 迁移prefix命令
3. 更新测试用例
4. 清理旧代码

## 验收标准

### 功能要求
- [ ] 所有现有命令功能保持不变
- [ ] 新的命令注册机制工作正常
- [ ] 参数解析和验证统一
- [ ] 错误处理统一
- [ ] 帮助系统完整

### 质量要求
- [ ] 每个命令文件 < 200行
- [ ] 命令模块单元测试覆盖率 > 90%
- [ ] 集成测试通过率 100%
- [ ] 代码重复率 < 5%

### 性能要求
- [ ] 命令执行性能无回退
- [ ] 内存使用无明显增加
- [ ] 启动时间 < 100ms

## 测试计划

1. **单元测试**
   - 每个命令的独立测试
   - 参数解析器测试
   - 命令注册器测试
   - 错误处理测试

2. **集成测试**
   - 命令执行流程测试
   - 命令间交互测试
   - REPL集成测试

3. **重构验证测试**
   - 所有现有测试用例通过
   - 功能回归测试
   - 性能基准测试

## 实现示例

### 命令接口实现
```go
type GetCommand struct{}

func (c *GetCommand) Name() string {
    return "get"
}

func (c *GetCommand) Description() string {
    return "Query by key with optional pretty JSON formatting"
}

func (c *GetCommand) Usage() string {
    return "get [<cf>] <key> [--pretty]"
}

func (c *GetCommand) Validate(args []string) error {
    if len(args) < 1 {
        return fmt.Errorf("key is required")
    }
    return nil
}

func (c *GetCommand) Execute(ctx *Context) error {
    cf, key, pretty := parseGetArgs(ctx.Args, ctx.Flags, ctx.State.CurrentCF)
    
    value, err := ctx.DB.GetCF(cf, key)
    if err != nil {
        return fmt.Errorf("failed to get key %q from CF %q: %w", key, cf, err)
    }
    
    output := formatValue(value, pretty)
    fmt.Fprintln(ctx.Output, output)
    return nil
}
```

### 命令注册
```go
func RegisterCommands(registry *Registry) {
    registry.Register(&GetCommand{})
    registry.Register(&PutCommand{})
    registry.Register(&ScanCommand{})
    registry.Register(&UseCFCommand{})
    // ... 更多命令
}
```

### 新的处理器
```go
func (h *Handler) Execute(input string) bool {
    args := strings.Fields(strings.TrimSpace(input))
    if len(args) == 0 {
        return true
    }

    cmdName := strings.ToLower(args[0])
    if cmdName == "exit" || cmdName == "quit" {
        return false
    }

    cmd := h.registry.Get(cmdName)
    if cmd == nil {
        fmt.Println("Unknown command. Type 'help' for available commands.")
        return true
    }

    ctx := &Context{
        DB:     h.DB,
        State:  h.State.(*ReplState),
        Args:   args[1:],
        Flags:  parseFlags(args[1:]),
        Output: os.Stdout,
        Logger: h.logger,
    }

    if err := cmd.Validate(ctx.Args); err != nil {
        fmt.Printf("Error: %v\nUsage: %s\n", err, cmd.Usage())
        return true
    }

    if err := cmd.Execute(ctx); err != nil {
        h.handleError(err, cmd.Name())
    }

    return true
}
```

## 迁移策略

### 阶段1: 并行开发
- 创建新的命令架构
- 保持旧代码不变
- 逐个实现新命令

### 阶段2: 渐进替换
- 逐个切换到新命令
- 运行双重测试
- 确保功能一致性

### 阶段3: 清理重构
- 移除旧的Execute方法
- 清理重复代码
- 优化性能

## 风险评估

**中等风险**
- 重构涉及核心逻辑
- 可能引入回归问题
- 需要大量测试验证

**缓解措施**
- 充分的单元测试和集成测试
- 渐进式迁移策略
- 保留回滚能力

## 后续任务

- Task 05: 错误处理增强（利用新的命令架构）
- Task 06: 测试覆盖完善（为新架构添加测试）
- Task 08: 硬编码清理（在新架构中消除硬编码）

## 参考资料

- [命令模式设计](https://refactoring.guru/design-patterns/command)
- [Go项目架构最佳实践](https://github.com/golang-standards/project-layout)
- [CLI应用架构设计](https://blog.golang.org/command-line-interfaces)
- [重构：改善既有代码的设计](https://refactoring.com/) 