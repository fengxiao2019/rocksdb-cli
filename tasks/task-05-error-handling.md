# Task 05: 错误处理增强

**优先级**: 🟡 中优先级  
**状态**: ⏳ 待开始  
**预估工作量**: 3-4小时  
**负责人**: 待分配  

## 问题描述

虽然项目已经有自定义错误类型，但错误处理仍需改进：

### 发现的问题
1. **错误上下文信息不足**:
   ```go
   return "", ErrKeyNotFound  // 缺少key和CF的上下文信息
   return ErrColumnFamilyNotFound  // 缺少具体的CF名称
   ```

2. **错误链追踪缺失**:
   - 无法追踪错误的完整调用链
   - 丢失原始错误的详细信息
   - 难以定位问题根源

3. **错误分类不够细致**:
   - 缺乏错误严重性分级
   - 没有错误码系统
   - 无法程序化处理特定错误

4. **用户错误提示不够友好**:
   - 技术性错误信息直接暴露给用户
   - 缺乏错误解决建议
   - 国际化支持不足

## 影响分析

- **可诊断性**: 难以快速定位和解决问题
- **用户体验**: 错误信息不够友好
- **可维护性**: 错误处理逻辑分散且重复
- **可集成性**: 无法与监控系统良好集成

## 解决方案

### 1. 错误处理架构
```
internal/errors/
├── errors.go           # 错误类型定义
├── codes.go            # 错误码系统
├── context.go          # 错误上下文
├── formatter.go        # 错误格式化器
└── handler.go          # 错误处理器
```

### 2. 增强的错误类型
```go
type Error struct {
    Code        ErrorCode
    Message     string
    Details     map[string]interface{}
    Cause       error
    Stack       []uintptr
    Timestamp   time.Time
    Component   string
    Operation   string
}

type ErrorCode int

const (
    ErrCodeKeyNotFound ErrorCode = iota + 1000
    ErrCodeColumnFamilyNotFound
    ErrCodeInvalidArgument
    ErrCodeDatabaseError
    ErrCodeSystemError
)
```

### 3. 错误上下文和包装
```go
func Wrap(err error, code ErrorCode, msg string) *Error
func Wrapf(err error, code ErrorCode, format string, args ...interface{}) *Error
func WithContext(err error, ctx map[string]interface{}) *Error
func WithOperation(err error, component, operation string) *Error
```

## 实施步骤

### Phase 1: 错误系统设计 (1.5小时)
1. 定义错误码系统
2. 创建增强的Error结构体
3. 实现错误包装和上下文
4. 添加错误格式化器

### Phase 2: 数据库层集成 (1小时)
1. 更新db.go中的错误返回
2. 添加详细的错误上下文
3. 实现错误链追踪
4. 优化错误消息

### Phase 3: 命令层集成 (1.5小时)
1. 更新命令处理器错误处理
2. 实现用户友好的错误提示
3. 添加错误解决建议
4. 集成到新的命令架构

## 验收标准

### 功能要求
- [ ] 错误码系统覆盖所有错误类型
- [ ] 错误上下文信息完整（key、CF、操作等）
- [ ] 错误链完整追踪
- [ ] 用户友好的错误提示
- [ ] 错误解决建议

### 质量要求
- [ ] 错误处理模块单元测试覆盖率 > 90%
- [ ] 错误信息一致性检查
- [ ] 性能影响 < 5%
- [ ] 内存开销 < 1KB per error

### 用户体验要求
- [ ] 错误信息简洁明了
- [ ] 提供操作建议
- [ ] 支持详细模式和简洁模式
- [ ] 技术错误与用户错误分离

## 测试计划

1. **单元测试**
   - 错误创建和包装
   - 错误上下文管理
   - 错误格式化
   - 错误码映射

2. **集成测试**
   - 端到端错误流程
   - 错误链完整性
   - 用户界面错误显示

3. **错误场景测试**
   - 数据库连接失败
   - 权限错误
   - 资源不足
   - 参数错误

## 实现示例

### 增强的错误结构
```go
type Error struct {
    code      ErrorCode
    message   string
    details   map[string]interface{}
    cause     error
    stack     []uintptr
    timestamp time.Time
    component string
    operation string
}

func (e *Error) Error() string {
    return e.FormatUser()
}

func (e *Error) Code() ErrorCode {
    return e.code
}

func (e *Error) Cause() error {
    return e.cause
}

func (e *Error) Details() map[string]interface{} {
    return e.details
}

func (e *Error) FormatUser() string {
    switch e.code {
    case ErrCodeKeyNotFound:
        if cf, ok := e.details["column_family"].(string); ok {
            if key, ok := e.details["key"].(string); ok {
                return fmt.Sprintf("Key '%s' not found in column family '%s'", key, cf)
            }
        }
        return "Key not found"
    case ErrCodeColumnFamilyNotFound:
        if cf, ok := e.details["column_family"].(string); ok {
            return fmt.Sprintf("Column family '%s' does not exist", cf)
        }
        return "Column family not found"
    default:
        return e.message
    }
}

func (e *Error) FormatTechnical() string {
    var buf strings.Builder
    buf.WriteString(fmt.Sprintf("[%s] %s.%s: %s", 
        e.code, e.component, e.operation, e.message))
    
    if e.cause != nil {
        buf.WriteString(fmt.Sprintf(" (caused by: %v)", e.cause))
    }
    
    if len(e.details) > 0 {
        buf.WriteString(fmt.Sprintf(" details: %+v", e.details))
    }
    
    return buf.String()
}
```

### 错误创建和包装
```go
func NewKeyNotFoundError(cf, key string) *Error {
    return &Error{
        code:      ErrCodeKeyNotFound,
        message:   "key not found",
        details:   map[string]interface{}{"column_family": cf, "key": key},
        timestamp: time.Now(),
        component: "db",
        operation: "get",
    }
}

func WrapDatabaseError(err error, operation string, details map[string]interface{}) *Error {
    return &Error{
        code:      ErrCodeDatabaseError,
        message:   "database operation failed",
        details:   details,
        cause:     err,
        timestamp: time.Now(),
        component: "db",
        operation: operation,
        stack:     captureStack(),
    }
}

func Wrap(err error, code ErrorCode, message string) *Error {
    if e, ok := err.(*Error); ok {
        // 如果已经是我们的错误类型，添加到错误链
        return &Error{
            code:      code,
            message:   message,
            cause:     e,
            timestamp: time.Now(),
            stack:     captureStack(),
        }
    }
    
    return &Error{
        code:      code,
        message:   message,
        cause:     err,
        timestamp: time.Now(),
        stack:     captureStack(),
    }
}
```

### 数据库层使用示例
```go
func (d *DB) GetCF(cf, key string) (string, error) {
    h, ok := d.cfHandles[cf]
    if !ok {
        return "", NewColumnFamilyNotFoundError(cf)
    }
    
    val, err := d.db.GetCF(d.ro, h, []byte(key))
    if err != nil {
        return "", WrapDatabaseError(err, "get", map[string]interface{}{
            "column_family": cf,
            "key": key,
        })
    }
    defer val.Free()
    
    if !val.Exists() {
        return "", NewKeyNotFoundError(cf, key)
    }
    
    return string(val.Data()), nil
}
```

### 用户友好的错误处理
```go
func (h *Handler) handleError(err error, operation string) {
    if e, ok := err.(*errors.Error); ok {
        // 用户友好的错误信息
        fmt.Println("Error:", e.FormatUser())
        
        // 根据错误类型提供建议
        switch e.Code() {
        case errors.ErrCodeKeyNotFound:
            fmt.Println("Suggestion: Use 'listcf' to see available column families, or 'scan' to browse keys")
        case errors.ErrCodeColumnFamilyNotFound:
            fmt.Println("Suggestion: Use 'listcf' to see available column families, or 'createcf' to create a new one")
        case errors.ErrCodeDatabaseError:
            fmt.Println("Suggestion: Check database connection and file permissions")
        }
        
        // 详细模式（可通过配置控制）
        if h.config.VerboseErrors {
            fmt.Println("Technical details:", e.FormatTechnical())
        }
    } else {
        // 未知错误类型
        fmt.Printf("Operation %s failed: %v\n", operation, err)
    }
}
```

## 错误码定义

```go
const (
    // 用户操作错误 (1000-1999)
    ErrCodeKeyNotFound         ErrorCode = 1001
    ErrCodeColumnFamilyNotFound ErrorCode = 1002
    ErrCodeInvalidArgument      ErrorCode = 1003
    ErrCodeInvalidFormat       ErrorCode = 1004
    
    // 权限和状态错误 (2000-2999)
    ErrCodeReadOnlyMode        ErrorCode = 2001
    ErrCodeDatabaseClosed      ErrorCode = 2002
    ErrCodeResourceBusy        ErrorCode = 2003
    
    // 系统错误 (3000-3999)
    ErrCodeDatabaseError       ErrorCode = 3001
    ErrCodeFileSystemError     ErrorCode = 3002
    ErrCodeMemoryError         ErrorCode = 3003
    
    // 网络和I/O错误 (4000-4999)
    ErrCodeTimeoutError        ErrorCode = 4001
    ErrCodeConnectionError     ErrorCode = 4002
)
```

## 配置示例

```yaml
error_handling:
  verbose_errors: false      # 是否显示技术细节
  stack_trace: false         # 是否捕获调用栈
  max_error_details: 10      # 错误详情字段数限制
  error_suggestions: true    # 是否显示解决建议
```

## 风险评估

**低风险**
- 主要是改进现有错误处理
- 向后兼容现有错误类型
- 不影响核心业务逻辑

**潜在风险**
- 错误处理性能开销
- 错误信息过于详细可能泄露敏感信息

## 后续任务

- Task 02: 日志系统（记录详细错误信息）
- Task 09: 监控度量（错误统计和告警）
- Task 03: 命令架构（统一的错误处理）

## 参考资料

- [Go错误处理最佳实践](https://blog.golang.org/error-handling-and-go)
- [pkg/errors错误包装](https://github.com/pkg/errors)
- [错误设计模式](https://dave.cheney.net/2016/04/27/dont-just-check-errors-handle-them-gracefully)
- [用户友好的错误信息设计](https://uxdesign.cc/how-to-write-good-error-messages-858e4551cd4) 