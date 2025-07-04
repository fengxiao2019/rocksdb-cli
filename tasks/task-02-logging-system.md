# Task 02: 结构化日志系统

**优先级**: 🔴 高优先级  
**状态**: ⏳ 待开始  
**预估工作量**: 3-4小时  
**负责人**: 待分配  

## 问题描述

当前项目使用`fmt.Printf`进行输出，缺乏专业的日志系统：

### 发现的问题
1. **非结构化输出**:
   ```go
   fmt.Printf("Failed to open database: %v\n", err)
   fmt.Printf("[%s] New: %s = %s\n", time.Now().Format("15:04:05"), key, value)
   ```

2. **缺乏日志级别**:
   - 无法区分DEBUG、INFO、WARN、ERROR
   - 生产环境无法控制日志详细程度

3. **缺乏上下文信息**:
   - 没有请求ID、用户信息等上下文
   - 难以追踪问题来源

4. **输出格式不统一**:
   - 时间格式不一致
   - 错误信息格式各异

## 影响分析

- **可观测性**: 生产环境问题难以诊断
- **性能**: 无法控制日志输出级别
- **集成性**: 无法与日志收集系统集成
- **维护性**: 调试信息与用户输出混合

## 解决方案

### 1. 创建日志管理模块
```
internal/logger/
├── logger.go          # 日志接口和实现
├── config.go          # 日志配置
├── formatter.go       # 日志格式化器
└── writer.go          # 日志输出器
```

### 2. 日志层级设计
```go
type Level int

const (
    DebugLevel Level = iota
    InfoLevel
    WarnLevel
    ErrorLevel
    FatalLevel
)
```

### 3. 结构化日志格式
```json
{
  "timestamp": "2024-01-15T10:30:45Z",
  "level": "INFO",
  "component": "db",
  "operation": "get_cf",
  "message": "Retrieved key from column family",
  "fields": {
    "cf": "users",
    "key": "user:1001",
    "duration_ms": 5.2
  }
}
```

## 实施步骤

### Phase 1: 日志框架 (1.5小时)
1. 选择日志库（推荐`slog`或`logrus`）
2. 创建日志接口和包装器
3. 实现基础的结构化日志
4. 添加日志级别控制

### Phase 2: 日志配置 (1小时)
1. 集成配置管理系统
2. 支持多种输出格式（JSON、Text）
3. 支持多种输出目标（Console、File）
4. 实现日志轮转配置

### Phase 3: 应用集成 (1.5小时)
1. 替换所有`fmt.Printf`为结构化日志
2. 为不同模块添加专用logger
3. 添加操作上下文和追踪ID
4. 分离用户输出和日志输出

## 验收标准

### 功能要求
- [ ] 支持5个日志级别（DEBUG到FATAL）
- [ ] 支持JSON和文本格式输出
- [ ] 支持文件和控制台输出
- [ ] 可配置的日志级别过滤
- [ ] 用户交互输出与日志分离

### 质量要求
- [ ] 日志模块单元测试覆盖率 > 85%
- [ ] 性能基准测试
- [ ] 日志格式一致性检查
- [ ] 并发安全性验证

### 性能要求
- [ ] 日志操作延迟 < 1ms
- [ ] 日志缓冲区配置
- [ ] 异步日志输出选项

## 测试计划

1. **单元测试**
   - 日志级别过滤
   - 格式化器功能
   - 输出器配置
   - 并发写入安全性

2. **集成测试**
   - 配置文件加载
   - 多输出目标同时写入
   - 日志轮转功能

3. **性能测试**
   - 高频日志写入压力测试
   - 内存使用量测试
   - 异步vs同步性能对比

## 使用示例

### 基础日志记录
```go
logger := logger.New("db")
logger.Info("Database opened successfully",
    "path", dbPath,
    "read_only", readOnly,
    "duration_ms", openDuration.Milliseconds())
```

### 错误日志记录
```go
logger.Error("Failed to get key from column family",
    "error", err,
    "cf", cfName,
    "key", key,
    "operation", "get_cf")
```

### 调试日志记录
```go
logger.Debug("Scanning column family",
    "cf", cf,
    "start_key", string(start),
    "end_key", string(end),
    "limit", opts.Limit)
```

## 配置示例

```yaml
logging:
  level: "info"                    # debug, info, warn, error, fatal
  format: "json"                   # json, text
  outputs:
    - type: "console"
      level: "info"
    - type: "file" 
      level: "debug"
      path: "/var/log/rocksdb-cli.log"
      max_size: "100MB"
      max_backups: 5
      max_age: 30
  fields:
    service: "rocksdb-cli"
    version: "1.0.0"
```

## 迁移策略

### 阶段1: 并行运行
- 保留现有`fmt.Printf`
- 添加新的日志调用
- 通过配置控制输出

### 阶段2: 逐步替换
- 模块级别逐步迁移
- 用户输出与日志分离
- 保持向后兼容

### 阶段3: 完全切换
- 移除所有`fmt.Printf`
- 统一日志输出
- 清理旧代码

## 风险评估

**低风险**
- 日志系统相对独立
- 可以渐进式迁移
- 不影响核心功能

**潜在风险**
- 性能影响（通过配置缓解）
- 用户输出格式变化（需要文档说明）

## 后续任务

- Task 09: 监控度量系统（使用日志进行指标记录）
- Task 05: 错误处理增强（结合结构化日志）
- Task 07: 并发安全性（日志系统的并发安全）

## 参考资料

- [Go标准库slog](https://pkg.go.dev/log/slog)
- [Logrus结构化日志库](https://github.com/sirupsen/logrus)
- [结构化日志最佳实践](https://blog.golang.org/slog)
- [云原生日志指南](https://12factor.net/logs) 