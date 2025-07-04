# Task 08: 硬编码清理

**优先级**: 🟢 低优先级  
**状态**: ⏳ 待开始  
**预估工作量**: 2-3小时  
**负责人**: 待分配  

## 问题描述

代码中存在硬编码值和魔法数字，影响可维护性：

### 发现的问题
1. **时间戳解析中的魔法数字**:
   ```go
   case ts > 1e15: // Nanoseconds (16+ digits)
   case ts > 1e12: // Microseconds (13-15 digits)
   case ts > 1e9:  // Milliseconds (10-12 digits)
   case ts > 1e6:  // Seconds (7-9 digits)
   ```

2. **硬编码的字符串常量**:
   ```go
   cfNames := []string{"default"}  // 默认列族名
   readOnlyFlag := "[READ-ONLY]"  // 状态标识
   ```

3. **配置硬编码**:
   ```go
   watchInterval := 1*time.Second  // 默认监控间隔
   ```

4. **缓冲区大小和限制**:
   ```go
   // CSV写入、内存分配等使用硬编码大小
   ```

## 影响分析

- **可维护性**: 修改配置需要重新编译
- **可读性**: 魔法数字含义不明确
- **可测试性**: 硬编码值难以在测试中替换
- **国际化**: 硬编码字符串阻碍多语言支持

## 解决方案

### 1. 常量定义文件
```
internal/constants/
├── constants.go        # 通用常量
├── errors.go          # 错误信息常量
├── formats.go         # 格式化常量
└── defaults.go        # 默认值常量
```

### 2. 常量分类
- **时间相关常量**
- **大小和限制常量**
- **格式化常量**
- **默认值常量**
- **用户界面常量**

## 实施步骤

### Phase 1: 常量提取 (1小时)
1. 识别所有硬编码值
2. 按类别创建常量文件
3. 定义有意义的常量名
4. 添加文档注释

### Phase 2: 代码重构 (1小时)
1. 替换时间戳解析中的魔法数字
2. 替换硬编码字符串
3. 替换配置默认值
4. 替换缓冲区大小

### Phase 3: 测试和验证 (1小时)
1. 更新测试用例
2. 验证功能不变
3. 检查常量使用一致性
4. 添加常量测试

## 验收标准

### 功能要求
- [ ] 所有魔法数字替换为命名常量
- [ ] 硬编码字符串提取为常量
- [ ] 配置默认值集中管理
- [ ] 常量有清晰的文档说明

### 质量要求
- [ ] 常量命名符合Go惯例
- [ ] 常量按逻辑分组
- [ ] 所有常量有测试覆盖
- [ ] 代码可读性提升

### 维护性要求
- [ ] 新增常量有标准流程
- [ ] 常量修改影响范围可控
- [ ] 支持不同环境的常量覆盖

## 实现示例

### 常量定义
```go
// internal/constants/constants.go
package constants

import "time"

// 时间戳相关常量
const (
    // TimestampNanosecondThreshold 纳秒时间戳阈值（16位数字）
    TimestampNanosecondThreshold = 1e15
    
    // TimestampMicrosecondThreshold 微秒时间戳阈值（13-15位数字）
    TimestampMicrosecondThreshold = 1e12
    
    // TimestampMillisecondThreshold 毫秒时间戳阈值（10-12位数字）
    TimestampMillisecondThreshold = 1e9
    
    // TimestampSecondThreshold 秒时间戳阈值（7-9位数字）
    TimestampSecondThreshold = 1e6
    
    // TimestampMinValidValue 最小有效时间戳值
    TimestampMinValidValue = 1e6
    
    // TimestampMaxValidValue 最大有效时间戳值（约2286年）
    TimestampMaxValidValue = 1e12
)

// 时间格式常量
const (
    // TimeFormatDisplay 显示时间格式
    TimeFormatDisplay = "2006-01-02 15:04:05 UTC"
    
    // TimeFormatISO8601 ISO8601时间格式
    TimeFormatISO8601 = "2006-01-02T15:04:05Z"
    
    // TimeFormatLogFile 日志文件时间格式
    TimeFormatLogFile = "20060102_150405"
)

// 默认值常量
const (
    // DefaultColumnFamily 默认列族名称
    DefaultColumnFamily = "default"
    
    // DefaultWatchInterval 默认监控间隔
    DefaultWatchInterval = 1 * time.Second
    
    // DefaultShutdownTimeout 默认关闭超时
    DefaultShutdownTimeout = 30 * time.Second
    
    // DefaultReadTimeout 默认读取超时
    DefaultReadTimeout = 10 * time.Second
)

// 大小和限制常量
const (
    // MaxHistorySize 最大历史记录数量
    MaxHistorySize = 100
    
    // DefaultBufferSize 默认缓冲区大小
    DefaultBufferSize = 4096
    
    // MaxErrorMessageLength 最大错误信息长度
    MaxErrorMessageLength = 1024
    
    // MaxKeyLength 最大键长度
    MaxKeyLength = 1024
    
    // MaxValueLength 最大值长度
    MaxValueLength = 1024 * 1024 // 1MB
)

// 用户界面常量
const (
    // PromptDefault 默认提示符
    PromptDefault = "rocksdb[%s]> "
    
    // PromptReadOnly 只读模式提示符
    PromptReadOnly = "rocksdb[READ-ONLY][%s]> "
    
    // StatusReadOnly 只读状态标识
    StatusReadOnly = "[READ-ONLY]"
    
    // MessageOK 成功消息
    MessageOK = "OK"
    
    // MessageBye 退出消息
    MessageBye = "Bye."
)
```

### 错误消息常量
```go
// internal/constants/errors.go
package constants

// 错误消息模板
const (
    // ErrMsgKeyNotFound 键未找到错误消息
    ErrMsgKeyNotFound = "Key '%s' not found in column family '%s'"
    
    // ErrMsgColumnFamilyNotFound 列族未找到错误消息
    ErrMsgColumnFamilyNotFound = "Column family '%s' does not exist"
    
    // ErrMsgColumnFamilyExists 列族已存在错误消息
    ErrMsgColumnFamilyExists = "Column family '%s' already exists"
    
    // ErrMsgReadOnlyMode 只读模式错误消息
    ErrMsgReadOnlyMode = "Operation not allowed in read-only mode"
    
    // ErrMsgColumnFamilyEmpty 列族为空错误消息
    ErrMsgColumnFamilyEmpty = "Column family '%s' is empty"
    
    // ErrMsgDatabaseClosed 数据库已关闭错误消息
    ErrMsgDatabaseClosed = "Database is closed"
)

// 使用建议消息
const (
    // SuggestionKeyNotFound 键未找到的建议
    SuggestionKeyNotFound = "Use 'listcf' to see available column families, or 'scan' to browse keys"
    
    // SuggestionColumnFamilyNotFound 列族未找到的建议
    SuggestionColumnFamilyNotFound = "Use 'listcf' to see available column families, or 'createcf' to create a new one"
    
    // SuggestionDatabaseError 数据库错误的建议
    SuggestionDatabaseError = "Check database connection and file permissions"
    
    // SuggestionReadOnlyMode 只读模式的建议
    SuggestionReadOnlyMode = "Remove --read-only flag to enable write operations"
)
```

### 格式常量
```go
// internal/constants/formats.go
package constants

// CSV相关常量
const (
    // CSVSeparator CSV分隔符
    CSVSeparator = ","
    
    // CSVHeaderKey CSV键列头
    CSVHeaderKey = "Key"
    
    // CSVHeaderValue CSV值列头
    CSVHeaderValue = "Value"
    
    // CSVQuoteChar CSV引号字符
    CSVQuoteChar = '"'
)

// JSON相关常量
const (
    // JSONIndentPrefix JSON缩进前缀
    JSONIndentPrefix = ""
    
    // JSONIndentValue JSON缩进值
    JSONIndentValue = "  "
)

// 帮助信息格式
const (
    // HelpCommandFormat 命令帮助格式
    HelpCommandFormat = "  %-30s - %s"
    
    // UsageMessageFormat 用法消息格式
    UsageMessageFormat = "Usage: %s"
    
    // ExampleFormat 示例格式
    ExampleFormat = "Example: %s"
)
```

### 重构后的代码使用
```go
// 重构前
func parseTimestamp(key string) string {
    if ts, err := strconv.ParseInt(key, 10, 64); err == nil {
        var t time.Time
        switch {
        case ts > 1e15:
            t = time.Unix(0, ts)
        case ts > 1e12:
            t = time.Unix(0, ts*1000)
        case ts > 1e9:
            t = time.Unix(0, ts*1e6)
        case ts > 1e6:
            t = time.Unix(ts, 0)
        default:
            return ""
        }
        return t.UTC().Format("2006-01-02 15:04:05 UTC")
    }
    return ""
}

// 重构后
func parseTimestamp(key string) string {
    ts, err := strconv.ParseInt(key, 10, 64)
    if err != nil {
        return ""
    }
    
    var t time.Time
    switch {
    case ts > constants.TimestampNanosecondThreshold:
        t = time.Unix(0, ts)
    case ts > constants.TimestampMicrosecondThreshold:
        t = time.Unix(0, ts*1000)
    case ts > constants.TimestampMillisecondThreshold:
        t = time.Unix(0, ts*1e6)
    case ts > constants.TimestampSecondThreshold:
        t = time.Unix(ts, 0)
    default:
        return "" // 时间戳值太小，不是有效时间戳
    }
    
    return t.UTC().Format(constants.TimeFormatDisplay)
}
```

### 错误处理重构
```go
// 重构前
func handleError(err error, operation string, params ...string) {
    if errors.Is(err, db.ErrKeyNotFound) {
        if len(params) >= 2 {
            fmt.Printf("Key '%s' not found in column family '%s'\n", params[0], params[1])
        } else {
            fmt.Println("Key not found")
        }
    }
    // ...
}

// 重构后
func handleError(err error, operation string, params ...string) {
    if errors.Is(err, db.ErrKeyNotFound) {
        if len(params) >= 2 {
            fmt.Printf(constants.ErrMsgKeyNotFound+"\n", params[0], params[1])
            fmt.Printf("Suggestion: %s\n", constants.SuggestionKeyNotFound)
        } else {
            fmt.Println("Key not found")
        }
    }
    // ...
}
```

### 配置默认值重构
```go
// 重构前
func main() {
    watchInterval := flag.Duration("interval", 1*time.Second, "Watch interval")
    // ...
}

// 重构后
func main() {
    watchInterval := flag.Duration("interval", constants.DefaultWatchInterval, "Watch interval")
    // ...
}
```

## 常量测试

```go
// internal/constants/constants_test.go
func TestTimestampThresholds(t *testing.T) {
    // 验证时间戳阈值的逻辑关系
    assert.Greater(t, constants.TimestampNanosecondThreshold, constants.TimestampMicrosecondThreshold)
    assert.Greater(t, constants.TimestampMicrosecondThreshold, constants.TimestampMillisecondThreshold)
    assert.Greater(t, constants.TimestampMillisecondThreshold, constants.TimestampSecondThreshold)
}

func TestTimeFormats(t *testing.T) {
    now := time.Now()
    
    // 测试时间格式是否有效
    formatted := now.Format(constants.TimeFormatDisplay)
    parsed, err := time.Parse(constants.TimeFormatDisplay, formatted)
    assert.NoError(t, err)
    assert.Equal(t, now.UTC().Truncate(time.Second), parsed.Truncate(time.Second))
}

func TestDefaultValues(t *testing.T) {
    // 验证默认值合理性
    assert.Equal(t, "default", constants.DefaultColumnFamily)
    assert.Equal(t, 1*time.Second, constants.DefaultWatchInterval)
    assert.Greater(t, constants.DefaultShutdownTimeout, 0*time.Second)
}

func TestErrorMessages(t *testing.T) {
    // 验证错误消息模板
    msg := fmt.Sprintf(constants.ErrMsgKeyNotFound, "testkey", "testcf")
    assert.Contains(t, msg, "testkey")
    assert.Contains(t, msg, "testcf")
    assert.Contains(t, msg, "not found")
}
```

## 配置文件支持

```yaml
# 允许通过配置覆盖某些常量
defaults:
  column_family: "default"
  watch_interval: "1s"
  shutdown_timeout: "30s"

ui:
  prompt_format: "rocksdb[%s]> "
  readonly_indicator: "[READ-ONLY]"
  success_message: "OK"

limits:
  max_key_length: 1024
  max_value_length: 1048576
  max_history_size: 100
```

## 国际化准备

```go
// 为未来国际化做准备
type Messages struct {
    KeyNotFound           string
    ColumnFamilyNotFound  string
    OperationSuccess      string
    // ...
}

var DefaultMessages = Messages{
    KeyNotFound:          "Key '%s' not found in column family '%s'",
    ColumnFamilyNotFound: "Column family '%s' does not exist",
    OperationSuccess:     "OK",
}
```

## 检查清单

### 硬编码识别
- [ ] 所有数字字面量审查
- [ ] 字符串字面量审查
- [ ] 时间间隔硬编码
- [ ] 大小限制硬编码
- [ ] 格式字符串硬编码

### 常量分类
- [ ] 时间相关常量
- [ ] 大小和限制常量
- [ ] 用户界面常量
- [ ] 错误消息常量
- [ ] 格式化常量

### 代码更新
- [ ] 所有硬编码替换
- [ ] 测试用例更新
- [ ] 文档更新
- [ ] 示例代码更新

## 风险评估

**极低风险**
- 仅是代码重构，不改变逻辑
- 常量值保持不变
- 向后兼容

**潜在风险**
- 常量命名可能不够直观
- 过度抽象可能降低可读性

## 后续任务

- Task 01: 配置管理（将常量集成到配置系统）
- Task 03: 命令架构（使用常量改进命令处理）
- 国际化支持（基于常量系统）

## 参考资料

- [Go代码风格指南](https://github.com/golang/go/wiki/CodeReviewComments)
- [有效Go编程](https://golang.org/doc/effective_go.html#constants)
- [清洁代码](https://github.com/ryanmcdermott/clean-code-javascript#variables)
- [重构手册](https://refactoring.com/catalog/extractVariable.html) 